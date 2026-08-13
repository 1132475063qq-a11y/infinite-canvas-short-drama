package service

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"
)

const CanvasProjectionPatchKindFilmGenerationTask = "film_generation_task"

// BindFilmGenerationTaskCanvasProjectionRequest is the future Provider
// Gateway's final projection input. The binding is intentionally narrower
// than a Canvas document write: it can only associate an already-created
// film-generation Task with the exact Generation Request version selected by
// one existing generation node.
type BindFilmGenerationTaskCanvasProjectionRequest struct {
	CanvasID                         string `json:"canvasId"`
	NodeID                           string `json:"nodeId"`
	DomainProjectID                  string `json:"domainProjectId"`
	GenerationRequestArtifactID      string `json:"generationRequestArtifactId"`
	GenerationRequestArtifactVersion int    `json:"generationRequestArtifactVersion"`
	TaskID                           string `json:"taskId"`
}

// BindFilmGenerationTaskCanvasProjection persists a server-owned Task binding
// without rewriting the browser-synced Canvas payload. The future gateway must
// call the equivalent repository write in the same transaction that creates
// Task and billing data; this standalone method exists only for the binding
// contract and is deliberately not exposed as a browser write endpoint.
func (s *Service) BindFilmGenerationTaskCanvasProjection(userID string, req BindFilmGenerationTaskCanvasProjectionRequest) (*model.CanvasProjectionPatch, error) {
	canvasID := strings.TrimSpace(req.CanvasID)
	nodeID := strings.TrimSpace(req.NodeID)
	domainProjectID := strings.TrimSpace(req.DomainProjectID)
	artifactID := strings.TrimSpace(req.GenerationRequestArtifactID)
	taskID := strings.TrimSpace(req.TaskID)
	if canvasID == "" || nodeID == "" || domainProjectID == "" || artifactID == "" || req.GenerationRequestArtifactVersion < 1 || taskID == "" {
		return nil, BadAuthRequest("画布任务投影缺少 Canvas、节点、Generation Request 版本或 Task 标识")
	}

	canvas, err := s.repo.CanvasProjectForUser(userID, canvasID)
	if err != nil {
		return nil, err
	}
	if canvas.ProjectID != domainProjectID {
		return nil, BadAuthRequest("画布未关联指定影视项目，拒绝写入任务投影")
	}
	draft, err := s.FilmGenerationTaskDraft(userID, domainProjectID, artifactID)
	if err != nil {
		return nil, err
	}
	if !draft.RequestReady {
		return nil, BadAuthRequest("Generation Request 当前状态不能绑定生成任务：" + strings.Join(draft.Blockers, "；"))
	}
	if draft.GenerationRequestArtifactVersion != req.GenerationRequestArtifactVersion {
		return nil, BadAuthRequest("Generation Request 版本与投影绑定不一致")
	}
	task, err := s.repo.TaskForUser(userID, taskID)
	if err != nil {
		return nil, err
	}
	if err := validateFilmGenerationProjectionTask(task, canvasID, nodeID, draft); err != nil {
		return nil, err
	}
	if err := validateFilmGenerationProjectionNode(canvas.PayloadJSON, nodeID, draft); err != nil {
		return nil, err
	}

	now := time.Now()
	return s.repo.UpsertCanvasProjectionPatch(&model.CanvasProjectionPatch{
		ID:                    newID(),
		UserID:                userID,
		CanvasID:              canvasID,
		NodeID:                nodeID,
		PatchKind:             CanvasProjectionPatchKindFilmGenerationTask,
		TargetProjectID:       domainProjectID,
		TargetArtifactID:      draft.GenerationRequestArtifactID,
		TargetArtifactVersion: draft.GenerationRequestArtifactVersion,
		TaskID:                taskID,
		CreatedAt:             now,
		UpdatedAt:             now,
	}, now)
}

func validateFilmGenerationProjectionTask(task *model.Task, canvasID string, nodeID string, draft FilmGenerationTaskDraft) error {
	if task.ProjectID != canvasID {
		return BadAuthRequest("Task 不属于当前画布，拒绝绑定")
	}
	if task.Type != draft.TaskType || task.Operation != draft.Operation {
		return BadAuthRequest("Task 不是该 Generation Request 的影视生成任务")
	}
	var input struct {
		CanvasID                         string `json:"canvasId"`
		CanvasNodeID                     string `json:"canvasNodeId"`
		GenerationRequestArtifactID      string `json:"generationRequestArtifactId"`
		GenerationRequestArtifactVersion int    `json:"generationRequestArtifactVersion"`
	}
	if err := json.Unmarshal([]byte(task.InputJSON), &input); err != nil {
		return BadAuthRequest("影视生成 Task 输入格式无效，拒绝绑定")
	}
	if strings.TrimSpace(input.CanvasID) != canvasID || strings.TrimSpace(input.CanvasNodeID) != nodeID || strings.TrimSpace(input.GenerationRequestArtifactID) != draft.GenerationRequestArtifactID || input.GenerationRequestArtifactVersion != draft.GenerationRequestArtifactVersion {
		return BadAuthRequest("Task 输入与当前画布节点或 Generation Request 版本不一致")
	}
	return nil
}

func validateFilmGenerationProjectionNode(payloadJSON string, nodeID string, draft FilmGenerationTaskDraft) error {
	node, err := canvasProjectionNodeByID([]byte(payloadJSON), nodeID)
	if err != nil {
		return err
	}
	if !canvasGenerationNodeMatchesTarget(node, draft.ProjectID, draft.GenerationRequestArtifactID, draft.GenerationRequestArtifactVersion) {
		return BadAuthRequest("画布节点已指向其他 Generation Request 版本，拒绝覆盖")
	}
	return nil
}

// applyCanvasProjectionPatches overlays server-owned runtime facts at read
// time. Full-document client sync stores no task binding, so an older browser
// document cannot erase a completed server-side bind.
func applyCanvasProjectionPatches(raw json.RawMessage, patches []model.CanvasProjectionPatch, canvasProjectID string, updatedAt time.Time) (json.RawMessage, error) {
	// Existing documents may predate the Patch contract. Treat every persisted
	// generation-node taskId as browser-owned until a matching server Patch
	// proves it; this makes the read path safe before the next browser save.
	base, err := stripClientFilmGenerationTaskClaims(raw)
	if err != nil {
		return nil, err
	}
	if len(patches) == 0 {
		return base, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(base, &payload); err != nil {
		return nil, BadAuthRequest("画布数据格式无效，无法应用运行时投影")
	}
	nodes, ok := payload["nodes"].([]any)
	if !ok {
		return base, nil
	}
	changed := false
	currentProjectID := strings.TrimSpace(canvasProjectID)
	for _, patch := range patches {
		if patch.PatchKind != CanvasProjectionPatchKindFilmGenerationTask || strings.TrimSpace(patch.TaskID) == "" {
			continue
		}
		if currentProjectID == "" || currentProjectID != strings.TrimSpace(patch.TargetProjectID) {
			continue
		}
		node, unique := uniqueCanvasProjectionNode(nodes, patch.NodeID)
		if !unique || !canvasGenerationNodeMatchesTarget(node, patch.TargetProjectID, patch.TargetArtifactID, patch.TargetArtifactVersion) {
			continue
		}
		domainRef, _ := node["domainRef"].(map[string]any)
		domainRef["taskId"] = patch.TaskID
		changed = true
	}
	if !changed {
		return base, nil
	}
	if !updatedAt.IsZero() {
		payload["updatedAt"] = updatedAt.UTC().Format(time.RFC3339Nano)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(encoded), nil
}

// stripClientFilmGenerationTaskClaims ensures DomainRef.taskId stays a
// server-owned runtime projection field. Old browser snapshots may contain a
// previously read task ID, but it is intentionally removed before persistence
// and restored only by applyCanvasProjectionPatches when still valid.
func stripClientFilmGenerationTaskClaims(raw json.RawMessage) (json.RawMessage, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, BadAuthRequest("画布数据格式错误")
	}
	nodes, ok := payload["nodes"].([]any)
	if !ok {
		return raw, nil
	}
	changed := false
	for _, value := range nodes {
		node, ok := value.(map[string]any)
		if !ok || canvasProjectionString(node["filmKind"]) != "generation" {
			continue
		}
		domainRef, ok := node["domainRef"].(map[string]any)
		if !ok {
			continue
		}
		if _, exists := domainRef["taskId"]; exists {
			delete(domainRef, "taskId")
			changed = true
		}
	}
	if !changed {
		return raw, nil
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(encoded), nil
}

func canvasProjectionNodeByID(raw []byte, nodeID string) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, BadAuthRequest("画布数据格式无效，无法校验任务投影")
	}
	nodes, ok := payload["nodes"].([]any)
	if !ok {
		return nil, BadAuthRequest("画布不包含可绑定的生成节点")
	}
	node, unique := uniqueCanvasProjectionNode(nodes, nodeID)
	if !unique {
		return nil, BadAuthRequest("画布中的目标生成节点不存在或标识重复")
	}
	return node, nil
}

func uniqueCanvasProjectionNode(nodes []any, nodeID string) (map[string]any, bool) {
	var match map[string]any
	for _, value := range nodes {
		node, ok := value.(map[string]any)
		if !ok || canvasProjectionString(node["id"]) != nodeID {
			continue
		}
		if match != nil {
			return nil, false
		}
		match = node
	}
	return match, match != nil
}

func canvasGenerationNodeMatchesTarget(node map[string]any, projectID string, artifactID string, artifactVersion int) bool {
	if node == nil || canvasProjectionString(node["filmKind"]) != "generation" {
		return false
	}
	domainRef, ok := node["domainRef"].(map[string]any)
	if !ok {
		return false
	}
	version, validVersion := canvasProjectionInt(domainRef["artifactVersion"])
	return canvasProjectionString(domainRef["projectId"]) == projectID && canvasProjectionString(domainRef["artifactId"]) == artifactID && validVersion && version == artifactVersion
}

func canvasProjectionString(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func canvasProjectionInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		if typed < 1 || typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil || parsed < 1 {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

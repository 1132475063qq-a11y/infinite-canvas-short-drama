package service

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"
)

const (
	CanvasProjectionPatchKindFilmGenerationTask   = model.CanvasProjectionPatchKindFilmGenerationTask
	CanvasProjectionPatchKindFilmGenerationResult = model.CanvasProjectionPatchKindFilmGenerationResult
)

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
	if task.CanvasID != canvasID || task.DomainProjectID != draft.ProjectID {
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
	connections, _ := payload["connections"].([]any)
	taskPatchesByTaskID := uniqueFilmGenerationTaskPatchesByTaskID(patches)
	for _, patch := range patches {
		if strings.TrimSpace(patch.TaskID) == "" || currentProjectID == "" || currentProjectID != strings.TrimSpace(patch.TargetProjectID) {
			continue
		}
		switch patch.PatchKind {
		case CanvasProjectionPatchKindFilmGenerationTask:
			generationNode, unique := uniqueCanvasProjectionNode(nodes, patch.NodeID)
			if !unique || !canvasGenerationNodeMatchesTarget(generationNode, patch.TargetProjectID, patch.TargetArtifactID, patch.TargetArtifactVersion) {
				continue
			}
			domainRef, _ := generationNode["domainRef"].(map[string]any)
			applyCanvasProjectionExecutionFacts(domainRef, patch)
			changed = true
		case CanvasProjectionPatchKindFilmGenerationResult:
			resultNodeID := model.FilmGenerationResultCanvasNodeID(patch.TaskID)
			if patch.NodeID != resultNodeID {
				continue
			}
			taskPatch, trustedSource := taskPatchesByTaskID[patch.TaskID]
			if !trustedSource || taskPatch.TargetProjectID != patch.TargetProjectID || taskPatch.TargetArtifactID != patch.TargetArtifactID || taskPatch.TargetArtifactVersion != patch.TargetArtifactVersion {
				continue
			}
			generationNode, unique := uniqueCanvasProjectionNode(nodes, taskPatch.NodeID)
			if !unique || !canvasGenerationNodeMatchesTarget(generationNode, patch.TargetProjectID, patch.TargetArtifactID, patch.TargetArtifactVersion) {
				continue
			}
			media, available := canvasProjectionVideoMediaFromPatch(patch)
			if !available {
				continue
			}
			resultNode, found := uniqueCanvasProjectionNode(nodes, resultNodeID)
			canonical := canonicalFilmGenerationResultNode(resultNode, generationNode, resultNodeID, patch, media)
			if found {
				for index, node := range nodes {
					if canvasProjectionString(nodeMap(node)["id"]) == resultNodeID {
						nodes[index] = canonical
						break
					}
				}
			} else {
				nodes = append(nodes, canonical)
				payload["nodes"] = nodes
			}
			var connectionChanged bool
			connections, connectionChanged = ensureFilmGenerationResultConnection(connections, patch.TaskID, taskPatch.NodeID, resultNodeID)
			if connectionChanged {
				payload["connections"] = connections
			}
			changed = true
		}
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

// A Result patch owns the derived Result node ID. Its source generation node
// must come from the immutable Task patch for the same Task; target matching
// alone is insufficient because a Canvas may contain repeated projections.
func uniqueFilmGenerationTaskPatchesByTaskID(patches []model.CanvasProjectionPatch) map[string]model.CanvasProjectionPatch {
	result := make(map[string]model.CanvasProjectionPatch)
	ambiguous := make(map[string]struct{})
	for _, patch := range patches {
		if patch.PatchKind != CanvasProjectionPatchKindFilmGenerationTask {
			continue
		}
		taskID := strings.TrimSpace(patch.TaskID)
		if taskID == "" {
			continue
		}
		if _, invalid := ambiguous[taskID]; invalid {
			continue
		}
		if _, exists := result[taskID]; exists {
			delete(result, taskID)
			ambiguous[taskID] = struct{}{}
			continue
		}
		result[taskID] = patch
	}
	return result
}

// stripClientFilmGenerationTaskClaims ensures execution IDs stay server-owned
// runtime projection fields. Old browser snapshots may contain values read
// earlier, but they are removed before persistence and restored only from the
// current execution tables when the target still matches.
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
		if !ok {
			continue
		}
		domainRef, ok := node["domainRef"].(map[string]any)
		if ok && (canvasProjectionString(node["filmKind"]) == "generation" || canvasProjectionString(node["filmKind"]) == "result") {
			for _, field := range []string{"taskId", "generationAttemptId", "providerJobId", "resultId", "resourceId"} {
				if _, exists := domainRef[field]; exists {
					delete(domainRef, field)
					changed = true
				}
			}
		}
		if canvasProjectionString(node["filmKind"]) != "result" {
			continue
		}
		metadata, ok := node["metadata"].(map[string]any)
		if !ok {
			continue
		}
		for _, field := range []string{"content", "storageKey", "mimeType", "bytes", "naturalWidth", "naturalHeight", "durationMs", "taskId", "taskStatus", "taskProgress", "taskStage", "taskCreatedAt", "taskUpdatedAt"} {
			if _, exists := metadata[field]; exists {
				delete(metadata, field)
				changed = true
			}
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

type canvasProjectionVideoMedia struct {
	URL        string
	StorageKey string
	ResourceID string
	MimeType   string
	Width      int
	Height     int
	Bytes      int64
	DurationMS int64
}

func applyCanvasProjectionExecutionFacts(domainRef map[string]any, patch model.CanvasProjectionPatch) {
	domainRef["taskId"] = patch.TaskID
	if patch.GenerationAttemptID != "" {
		domainRef["generationAttemptId"] = patch.GenerationAttemptID
	}
	if patch.ProviderJobID != "" {
		domainRef["providerJobId"] = patch.ProviderJobID
	}
	if patch.ResultID != "" {
		domainRef["resultId"] = patch.ResultID
	}
}

func canvasProjectionVideoMediaFromPatch(patch model.CanvasProjectionPatch) (canvasProjectionVideoMedia, bool) {
	media := canvasProjectionVideoMedia{URL: strings.TrimSpace(patch.ResultURL)}
	if raw := strings.TrimSpace(patch.ResultPayload); raw != "" {
		var payload map[string]any
		if json.Unmarshal([]byte(raw), &payload) == nil {
			candidate := payload
			if video, ok := payload["video"].(map[string]any); ok {
				candidate = video
			}
			if media.URL == "" {
				media.URL = firstCanvasProjectionString(candidate, "dataUrl", "url", "content", "resultUrl", "outputUrl")
			}
			media.StorageKey = firstCanvasProjectionString(candidate, "storageKey")
			media.ResourceID = firstCanvasProjectionString(candidate, "resourceId")
			media.MimeType = firstCanvasProjectionString(candidate, "mimeType")
			media.Width = canvasProjectionNonNegativeInt(candidate["width"])
			media.Height = canvasProjectionNonNegativeInt(candidate["height"])
			media.Bytes = canvasProjectionNonNegativeInt64(candidate["bytes"])
			media.DurationMS = canvasProjectionNonNegativeInt64(candidate["durationMs"])
		}
	}
	if media.ResourceID == "" && strings.HasPrefix(media.StorageKey, "resource:") {
		media.ResourceID = strings.TrimPrefix(media.StorageKey, "resource:")
	}
	if media.ResourceID == "" {
		media.ResourceID = canvasProjectionResourceIDFromURL(media.URL)
	}
	if media.StorageKey == "" && media.ResourceID != "" {
		media.StorageKey = "resource:" + media.ResourceID
	}
	if media.MimeType == "" {
		media.MimeType = "video/mp4"
	}
	return media, isCanvasProjectionMediaURL(media.URL)
}

func canonicalFilmGenerationResultNode(existing map[string]any, generationNode map[string]any, resultNodeID string, patch model.CanvasProjectionPatch, media canvasProjectionVideoMedia) map[string]any {
	generationRef, _ := generationNode["domainRef"].(map[string]any)
	domainRef := map[string]any{
		"projectId":       patch.TargetProjectID,
		"artifactId":      patch.TargetArtifactID,
		"artifactVersion": strconv.Itoa(patch.TargetArtifactVersion),
	}
	for _, field := range []string{"unitId", "sceneId", "shotId"} {
		if value := canvasProjectionString(generationRef[field]); value != "" {
			domainRef[field] = value
		}
	}
	applyCanvasProjectionExecutionFacts(domainRef, patch)
	if media.ResourceID != "" {
		domainRef["resourceId"] = media.ResourceID
	}

	position := canvasProjectionResultPosition(generationNode)
	width, height := canvasProjectionVideoNodeSize(media)
	title := canvasProjectionResultTitle(generationNode)
	layout := map[string]any{"mode": "auto", "lane": "shot_pipeline"}
	metadata := map[string]any{}
	if existing != nil {
		if value := canvasProjectionString(existing["title"]); value != "" {
			title = value
		}
		if value, ok := existing["position"].(map[string]any); ok && canvasProjectionPositionValid(value) {
			position = value
		}
		if value, ok := existing["width"]; ok && canvasProjectionNonNegativeInt(value) > 0 {
			width = canvasProjectionNonNegativeInt(value)
		}
		if value, ok := existing["height"]; ok && canvasProjectionNonNegativeInt(value) > 0 {
			height = canvasProjectionNonNegativeInt(value)
		}
		if value, ok := existing["layout"].(map[string]any); ok {
			layout = value
		}
		if value, ok := existing["metadata"].(map[string]any); ok {
			for key, item := range value {
				metadata[key] = item
			}
		}
	}
	metadata["content"] = media.URL
	metadata["storageKey"] = media.StorageKey
	metadata["status"] = "success"
	metadata["mimeType"] = media.MimeType
	metadata["taskId"] = patch.TaskID
	metadata["taskStatus"] = "succeeded"
	metadata["taskProgress"] = 100
	metadata["taskStage"] = "任务完成"
	metadata["generationSourceNodeId"] = canvasProjectionString(generationNode["id"])
	if media.Width > 0 {
		metadata["naturalWidth"] = media.Width
	}
	if media.Height > 0 {
		metadata["naturalHeight"] = media.Height
	}
	if media.Bytes > 0 {
		metadata["bytes"] = media.Bytes
	}
	if media.DurationMS > 0 {
		metadata["durationMs"] = media.DurationMS
	}

	node := map[string]any{
		"id":        resultNodeID,
		"type":      "video",
		"filmKind":  "result",
		"domainRef": domainRef,
		"filmState": map[string]any{"lifecycle": "locked", "production": "generated", "evidence": "recorded", "attention": "none"},
		"layout":    layout,
		"title":     title,
		"position":  position,
		"width":     width,
		"height":    height,
		"metadata":  metadata,
	}
	if existing != nil {
		if parentID := canvasProjectionString(existing["parentId"]); parentID != "" {
			node["parentId"] = parentID
		}
	}
	return node
}

func ensureFilmGenerationResultConnection(connections []any, taskID string, fromNodeID string, toNodeID string) ([]any, bool) {
	canonicalID := "film-generation-result-edge:" + taskID
	canonical := map[string]any{
		"id":         canonicalID,
		"fromNodeId": fromNodeID,
		"toNodeId":   toNodeID,
		"edgeType":   "derivation",
		"filmPorts":  map[string]any{"from": "generation_job", "to": "generation_job"},
	}
	for index, value := range connections {
		connection, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if canvasProjectionString(connection["id"]) == canonicalID {
			connections[index] = canonical
			return connections, true
		}
		if canvasProjectionString(connection["fromNodeId"]) == fromNodeID && canvasProjectionString(connection["toNodeId"]) == toNodeID {
			connection["edgeType"] = "derivation"
			connection["filmPorts"] = map[string]any{"from": "generation_job", "to": "generation_job"}
			return connections, true
		}
	}
	return append(connections, canonical), true
}

func canvasProjectionResultPosition(generationNode map[string]any) map[string]any {
	position, _ := generationNode["position"].(map[string]any)
	x := canvasProjectionNumber(position["x"])
	y := canvasProjectionNumber(position["y"])
	width := canvasProjectionNumber(generationNode["width"])
	if width <= 0 {
		width = 320
	}
	return map[string]any{"x": x + width + 64, "y": y}
}

func canvasProjectionVideoNodeSize(media canvasProjectionVideoMedia) (int, int) {
	if media.Width > 0 && media.Height > 0 {
		width := media.Width
		if width > 480 {
			width = 480
		}
		height := int(float64(width) * float64(media.Height) / float64(media.Width))
		if height > 0 {
			return width, height
		}
	}
	return 360, 203
}

func canvasProjectionResultTitle(generationNode map[string]any) string {
	if title := canvasProjectionString(generationNode["title"]); title != "" {
		return title + " · 视频结果"
	}
	return "视频生成结果"
}

func canvasProjectionPositionValid(position map[string]any) bool {
	_, xOK := position["x"]
	_, yOK := position["y"]
	return xOK && yOK
}

func firstCanvasProjectionString(value map[string]any, keys ...string) string {
	for _, key := range keys {
		if result := canvasProjectionString(value[key]); result != "" {
			return result
		}
	}
	return ""
}

func canvasProjectionResourceIDFromURL(value string) string {
	const prefix = "/api/resources/"
	index := strings.Index(value, prefix)
	if index < 0 {
		return ""
	}
	rest := strings.TrimPrefix(value[index:], prefix)
	id, _, _ := strings.Cut(rest, "/")
	return strings.TrimSpace(id)
}

func isCanvasProjectionMediaURL(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.HasPrefix(value, "/api/resources/") || strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://")
}

func canvasProjectionNonNegativeInt(value any) int {
	return int(canvasProjectionNonNegativeInt64(value))
}

func canvasProjectionNonNegativeInt64(value any) int64 {
	switch typed := value.(type) {
	case float64:
		if typed > 0 {
			return int64(typed)
		}
	case float32:
		if typed > 0 {
			return int64(typed)
		}
	case int:
		if typed > 0 {
			return int64(typed)
		}
	case int64:
		if typed > 0 {
			return typed
		}
	case json.Number:
		if parsed, err := typed.Int64(); err == nil && parsed > 0 {
			return parsed
		}
	case string:
		if parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

func canvasProjectionNumber(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		parsed, _ := typed.Float64()
		return parsed
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed
	default:
		return 0
	}
}

func nodeMap(value any) map[string]any {
	node, _ := value.(map[string]any)
	return node
}

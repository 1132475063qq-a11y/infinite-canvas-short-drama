package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
)

const FilmGenerationTaskDraftSchemaVersion = 1

const (
	FilmGenerationTaskDraftStateAwaitingRequestReview = "awaiting_request_review"
	FilmGenerationTaskDraftStateAwaitingProviderRoute = "awaiting_provider_route"
	FilmGenerationTaskDraftStateNotSubmittable        = "not_submittable"
)

// FilmGenerationGatewayInput is the provider-independent part of a future
// Task.InputJSON. It intentionally has no provider config, channel, API key,
// headers, or model selection. Those fields are resolved inside the future
// Provider Gateway immediately before it atomically creates a queued Task.
type FilmGenerationGatewayInput struct {
	SchemaVersion                    int                             `json:"schemaVersion"`
	Mode                             string                          `json:"mode"`
	Prompt                           string                          `json:"prompt"`
	AspectRatio                      string                          `json:"aspectRatio,omitempty"`
	DurationMS                       int64                           `json:"durationMs,omitempty"`
	OutputIntent                     string                          `json:"outputIntent"`
	GenerationRequestArtifactID      string                          `json:"generationRequestArtifactId"`
	GenerationRequestArtifactVersion int                             `json:"generationRequestArtifactVersion"`
	PromptArtifactID                 string                          `json:"promptArtifactId"`
	PromptArtifactVersion            int                             `json:"promptArtifactVersion"`
	SourceRefs                       []string                        `json:"sourceRefs"`
	SpatialPackArtifactID            string                          `json:"spatialPackArtifactId,omitempty"`
	SpatialPackArtifactVersion       int                             `json:"spatialPackArtifactVersion,omitempty"`
	SpatialGateArtifactID            string                          `json:"spatialGateArtifactId,omitempty"`
	SpatialGateArtifactVersion       int                             `json:"spatialGateArtifactVersion,omitempty"`
	CameraAnchorID                   string                          `json:"cameraAnchorId,omitempty"`
	ViewID                           string                          `json:"viewId,omitempty"`
	SpatialContext                   *LockedSceneAssetPackProjection `json:"spatialContext,omitempty"`
}

// FilmGenerationTaskDraft is an auditable, read-only bridge between the
// immutable Generation Request and the mutable Task runtime. It is not a Task
// row and must never be claimed by the worker, billed, or sent to a provider.
type FilmGenerationTaskDraft struct {
	SchemaVersion                    int                             `json:"schemaVersion"`
	GenerationRequestArtifactID      string                          `json:"generationRequestArtifactId"`
	GenerationRequestArtifactVersion int                             `json:"generationRequestArtifactVersion"`
	GenerationRequestStatus          string                          `json:"generationRequestStatus"`
	ProjectID                        string                          `json:"projectId"`
	UnitID                           string                          `json:"unitId,omitempty"`
	SceneID                          string                          `json:"sceneId,omitempty"`
	ShotID                           string                          `json:"shotId"`
	TaskType                         string                          `json:"taskType"`
	Operation                        string                          `json:"operation"`
	RequestReady                     bool                            `json:"requestReady"`
	ProviderRouteResolved            bool                            `json:"providerRouteResolved"`
	SubmissionAllowed                bool                            `json:"submissionAllowed"`
	SubmissionState                  string                          `json:"submissionState"`
	Blockers                         []string                        `json:"blockers"`
	RequestFingerprint               string                          `json:"requestFingerprint"`
	SpatialGateReady                 bool                            `json:"spatialGateReady"`
	SpatialContinuityGate            *SceneSpatialGate               `json:"spatialContinuityGate,omitempty"`
	SpatialGateArtifactID            string                          `json:"spatialGateArtifactId,omitempty"`
	SpatialGateArtifactVersion       int                             `json:"spatialGateArtifactVersion,omitempty"`
	SpatialPackArtifactID            string                          `json:"spatialPackArtifactId,omitempty"`
	SpatialPackArtifactVersion       int                             `json:"spatialPackArtifactVersion,omitempty"`
	CameraAnchorID                   string                          `json:"cameraAnchorId,omitempty"`
	ViewID                           string                          `json:"viewId,omitempty"`
	SpatialContext                   *LockedSceneAssetPackProjection `json:"spatialContext,omitempty"`
	GatewayInput                     FilmGenerationGatewayInput      `json:"gatewayInput"`
}

// FilmGenerationTaskDraft returns an exact immutable Generation Request
// version. It deliberately does not call CreateTask: CreateTask queues work,
// reserves billing and can be claimed by the existing worker before a Provider
// Gateway has chosen a server-side route.
func (s *Service) FilmGenerationTaskDraft(userID string, projectID string, artifactID string) (FilmGenerationTaskDraft, error) {
	project, err := s.repo.ProjectForUser(userID, strings.TrimSpace(projectID))
	if err != nil {
		return FilmGenerationTaskDraft{}, err
	}
	if project.Type != model.ProjectTypeShortDrama {
		return FilmGenerationTaskDraft{}, BadAuthRequest("Generation Request 任务合同只适用于短剧项目")
	}

	artifact, err := s.repo.FilmArtifactForProject(project.ID, strings.TrimSpace(artifactID))
	if err != nil {
		return FilmGenerationTaskDraft{}, err
	}
	if artifact.ArtifactType != FilmArtifactTypeGenerationRequest {
		return FilmGenerationTaskDraft{}, BadAuthRequest("指定 Artifact 不是 Generation Request")
	}
	if strings.TrimSpace(artifact.ShotID) == "" {
		return FilmGenerationTaskDraft{}, BadAuthRequest("Generation Request 缺少镜头关联")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(artifact.PayloadJSON), &payload); err != nil {
		return FilmGenerationTaskDraft{}, BadAuthRequest("Generation Request 内容格式无效")
	}
	if err := validateGenerationRequestPayload(payload); err != nil {
		return FilmGenerationTaskDraft{}, BadAuthRequest("Generation Request 内容不符合任务合同：" + err.Error())
	}
	promptArtifactVersion, ok := positivePayloadInt(payload["promptArtifactVersion"])
	if !ok {
		return FilmGenerationTaskDraft{}, BadAuthRequest("Generation Request 缺少有效的 Prompt Pack 版本")
	}
	sourceRefs, err := decodeFilmGenerationSourceRefs(artifact.SourceRefsJSON)
	if err != nil {
		return FilmGenerationTaskDraft{}, err
	}
	promptArtifactID := payloadString(payload, "promptArtifactId")
	if !filmSourceRefExists(sourceRefs, promptArtifactID) {
		return FilmGenerationTaskDraft{}, BadAuthRequest("Generation Request 来源记录缺少 Prompt Pack")
	}
	promptArtifact, err := s.repo.FilmArtifactForProject(project.ID, promptArtifactID)
	if err != nil {
		return FilmGenerationTaskDraft{}, BadAuthRequest("Generation Request 引用的 Prompt Pack 版本不存在")
	}
	if promptArtifact.ArtifactType != FilmArtifactTypeVideoPromptPack || promptArtifact.ShotID != artifact.ShotID || promptArtifact.ObjectVersion != promptArtifactVersion {
		return FilmGenerationTaskDraft{}, BadAuthRequest("Generation Request 引用的 Prompt Pack 与冻结来源不一致")
	}
	var promptPayload map[string]any
	if err := json.Unmarshal([]byte(promptArtifact.PayloadJSON), &promptPayload); err != nil || payloadString(promptPayload, "compiledPrompt") != payloadString(payload, "compiledPrompt") {
		return FilmGenerationTaskDraft{}, BadAuthRequest("Generation Request 的冻结提示词与 Prompt Pack 版本不一致")
	}
	var spatialProjection *LockedSceneAssetPackProjection
	if strings.TrimSpace(artifact.SceneID) != "" {
		if promptArtifact.SceneID != artifact.SceneID {
			return FilmGenerationTaskDraft{}, BadAuthRequest("场景镜头引用的 Prompt Pack 没有同一场景归属")
		}
		projection, projectionErr := decodeLockedSceneAssetPackProjection(promptPayload)
		if projectionErr != nil {
			return FilmGenerationTaskDraft{}, BadAuthRequest("Generation Request 的 Prompt Pack 空间投影无效：" + projectionErr.Error())
		}
		if projection.SceneID != artifact.SceneID {
			return FilmGenerationTaskDraft{}, BadAuthRequest("Generation Request 的 Prompt Pack 空间投影与镜头场景不一致")
		}
		for _, key := range []string{"spatialPackArtifactId", "spatialGateArtifactId", "cameraAnchorId", "viewId"} {
			if payloadString(payload, key) != payloadString(promptPayload, key) {
				return FilmGenerationTaskDraft{}, BadAuthRequest("Generation Request 的空间来源与 Prompt Pack 版本不一致")
			}
		}
		spatialProjection = &projection
	}

	mediaType := strings.ToLower(payloadString(payload, "mediaType"))
	gatewayInput := FilmGenerationGatewayInput{
		SchemaVersion:                    FilmGenerationTaskDraftSchemaVersion,
		Mode:                             mediaType,
		Prompt:                           payloadString(payload, "compiledPrompt"),
		AspectRatio:                      payloadString(payload, "aspectRatio"),
		DurationMS:                       generationRequestDurationMS(payload),
		OutputIntent:                     payloadString(payload, "outputIntent"),
		GenerationRequestArtifactID:      artifact.ID,
		GenerationRequestArtifactVersion: artifact.ObjectVersion,
		PromptArtifactID:                 promptArtifactID,
		PromptArtifactVersion:            promptArtifactVersion,
		SourceRefs:                       sourceRefs,
	}
	if spatialProjection != nil {
		gatewayInput.SpatialPackArtifactID = spatialProjection.PackArtifactID
		gatewayInput.SpatialPackArtifactVersion = spatialProjection.PackArtifactVersion
		gatewayInput.SpatialGateArtifactID = spatialProjection.GateArtifactID
		gatewayInput.SpatialGateArtifactVersion = spatialProjection.GateArtifactVersion
		gatewayInput.CameraAnchorID = spatialProjection.CameraAnchorID
		gatewayInput.ViewID = spatialProjection.ViewID
		gatewayInput.SpatialContext = spatialProjection
		if !filmSourceRefExists(sourceRefs, spatialProjection.PackArtifactID) || !filmSourceRefExists(sourceRefs, spatialProjection.GateArtifactID) {
			return FilmGenerationTaskDraft{}, BadAuthRequest("Generation Request 来源记录缺少锁定场景 Pack/Gate")
		}
	}
	requestReady, submissionState, blockers := filmGenerationRequestReadiness(artifact.Status, project.Status)
	var spatialGate *SceneSpatialGate
	var spatialGateArtifact *model.FilmArtifact
	var spatialPackArtifact *model.FilmArtifact
	spatialGateReady := true
	if strings.TrimSpace(artifact.SceneID) != "" {
		spatialGateReady = false
		spatialGate, spatialGateArtifact, spatialPackArtifact, err = s.sceneSpatialGateForGeneration(project.ID, artifact.SceneID)
		if err != nil {
			return FilmGenerationTaskDraft{}, err
		}
		if spatialGate == nil || spatialGate.Status != SceneSpatialGatePass {
			requestReady = false
			if submissionState == FilmGenerationTaskDraftStateAwaitingProviderRoute {
				submissionState = FilmGenerationTaskDraftStateAwaitingRequestReview
			}
			if spatialGate == nil {
				blockers = append(blockers, "场景尚未建立 Spatial Continuity Gate；必须先保存场景空间资产包")
			} else {
				blockers = append(blockers, "Spatial Continuity Gate 当前为 "+string(spatialGate.Status)+"，只有 PASS 才能进入正式镜头生成")
			}
		} else if spatialProjection == nil || spatialPackArtifact == nil || spatialGateArtifact == nil || spatialProjection.PackArtifactID != spatialPackArtifact.ID || spatialProjection.PackArtifactVersion != spatialPackArtifact.ObjectVersion || spatialProjection.GateArtifactID != spatialGateArtifact.ID || spatialProjection.GateArtifactVersion != spatialGateArtifact.ObjectVersion {
			requestReady = false
			if submissionState == FilmGenerationTaskDraftStateAwaitingProviderRoute {
				submissionState = FilmGenerationTaskDraftStateAwaitingRequestReview
			}
			blockers = append(blockers, "Prompt Pack 锁定的空间版本已不是当前 PASS 场景资产版本，请重新编译 Prompt Pack")
		} else {
			spatialGateReady = true
		}
		if spatialGateArtifact != nil {
			sourceRefs = appendUniqueFilmSourceRef(sourceRefs, spatialGateArtifact.ID)
		}
		if spatialPackArtifact != nil {
			sourceRefs = appendUniqueFilmSourceRef(sourceRefs, spatialPackArtifact.ID)
		}
	}
	gatewayInput.SourceRefs = sourceRefs

	return FilmGenerationTaskDraft{
		SchemaVersion:                    FilmGenerationTaskDraftSchemaVersion,
		GenerationRequestArtifactID:      artifact.ID,
		GenerationRequestArtifactVersion: artifact.ObjectVersion,
		GenerationRequestStatus:          artifact.Status,
		ProjectID:                        project.ID,
		UnitID:                           artifact.UnitID,
		SceneID:                          artifact.SceneID,
		ShotID:                           artifact.ShotID,
		TaskType:                         filmGenerationTaskType(mediaType),
		Operation:                        "film_generation",
		RequestReady:                     requestReady,
		ProviderRouteResolved:            false,
		SubmissionAllowed:                false,
		SubmissionState:                  submissionState,
		Blockers:                         blockers,
		RequestFingerprint:               filmGenerationRequestFingerprint(artifact, gatewayInput),
		SpatialGateReady:                 spatialGateReady,
		SpatialContinuityGate:            spatialGate,
		SpatialGateArtifactID:            filmArtifactID(spatialGateArtifact),
		SpatialGateArtifactVersion:       filmArtifactVersion(spatialGateArtifact),
		SpatialPackArtifactID:            filmArtifactID(spatialPackArtifact),
		SpatialPackArtifactVersion:       filmArtifactVersion(spatialPackArtifact),
		CameraAnchorID:                   gatewayInput.CameraAnchorID,
		ViewID:                           gatewayInput.ViewID,
		SpatialContext:                   gatewayInput.SpatialContext,
		GatewayInput:                     gatewayInput,
	}, nil
}

func filmArtifactID(artifact *model.FilmArtifact) string {
	if artifact == nil {
		return ""
	}
	return artifact.ID
}

func filmArtifactVersion(artifact *model.FilmArtifact) int {
	if artifact == nil {
		return 0
	}
	return artifact.ObjectVersion
}

// sceneSpatialGateForGeneration reads and re-validates the latest scene pack
// and its derived gate. A missing or tampered gate is represented as
// UNCERTAIN, never as an implicit PASS.
func (s *Service) sceneSpatialGateForGeneration(projectID string, sceneID string) (*SceneSpatialGate, *model.FilmArtifact, *model.FilmArtifact, error) {
	packArtifact, packErr := s.repo.LatestFilmArtifactForScope(projectID, model.FilmArtifactScopeScene, sceneID, FilmArtifactTypeSceneAssetPack)
	if errors.Is(packErr, gorm.ErrRecordNotFound) {
		return &SceneSpatialGate{SchemaVersion: SceneSpatialContractSchemaVersion, SceneID: sceneID, Status: SceneSpatialGateUncertain, Issues: []SceneSpatialIssue{{Code: "SPATIAL_GATE_NOT_PASS", Severity: SceneSpatialIssueBlocker, Message: "场景没有已保存的空间资产包"}}, Validator: "scene-spatial-gate/1"}, nil, nil, nil
	}
	if packErr != nil {
		return nil, nil, nil, packErr
	}
	pack, err := DecodeSceneAssetPackPayload([]byte(packArtifact.PayloadJSON))
	if err != nil {
		return &SceneSpatialGate{SchemaVersion: SceneSpatialContractSchemaVersion, SceneID: sceneID, Status: SceneSpatialGateUncertain, Issues: []SceneSpatialIssue{{Code: "SPATIAL_GATE_NOT_PASS", Severity: SceneSpatialIssueBlocker, Message: "场景空间资产包无法解析"}}, Validator: "scene-spatial-gate/1"}, nil, packArtifact, nil
	}
	validated := ValidateSceneAssetPack(pack, nil)
	validated.PackArtifactID = packArtifact.ID
	validated.PackArtifactVersion = packArtifact.ObjectVersion
	validated.PackVersion = packArtifact.ObjectVersion
	gateArtifact, gateErr := s.repo.LatestFilmArtifactForScope(projectID, model.FilmArtifactScopeScene, sceneID, FilmArtifactTypeSpatialContinuityGate)
	if errors.Is(gateErr, gorm.ErrRecordNotFound) {
		validated.Status = SceneSpatialGateUncertain
		validated.Issues = append(validated.Issues, SceneSpatialIssue{Code: "SPATIAL_GATE_NOT_PASS", Severity: SceneSpatialIssueBlocker, Message: "场景空间资产包没有对应的门禁结果"})
		return &validated, nil, packArtifact, nil
	}
	if gateErr != nil {
		return nil, nil, nil, gateErr
	}
	var stored SceneSpatialGate
	if err := json.Unmarshal([]byte(gateArtifact.PayloadJSON), &stored); err != nil {
		validated.Status = SceneSpatialGateUncertain
		validated.Issues = append(validated.Issues, SceneSpatialIssue{Code: "SPATIAL_GATE_NOT_PASS", Severity: SceneSpatialIssueBlocker, Message: "空间门禁结果无法解析"})
		return &validated, gateArtifact, packArtifact, nil
	}
	if stored.SchemaVersion != SceneSpatialContractSchemaVersion || stored.Validator != "scene-spatial-gate/1" || stored.SceneID != sceneID || stored.PackArtifactID != packArtifact.ID || stored.PackArtifactVersion != packArtifact.ObjectVersion || stored.PackVersion != packArtifact.ObjectVersion || (stored.Status == SceneSpatialGatePass && gateArtifact.Status != "ready") {
		stored.Status = worseSceneSpatialStatus(SceneSpatialGateUncertain, stored.Status)
		stored.Issues = append(stored.Issues, SceneSpatialIssue{Code: "SPATIAL_GATE_NOT_PASS", Severity: SceneSpatialIssueBlocker, Message: "空间门禁结果与当前场景资产版本不一致"})
	}
	if worseSceneSpatialStatus(stored.Status, validated.Status) != stored.Status {
		stored.Status = validated.Status
		stored.Issues = append(stored.Issues, SceneSpatialIssue{Code: "SPATIAL_GATE_NOT_PASS", Severity: SceneSpatialIssueBlocker, Message: "当前场景资产未通过重新校验"})
	}
	if !sceneAssetPackStatusReady(packArtifact.Status) {
		stored.Status = worseSceneSpatialStatus(stored.Status, SceneSpatialGateUncertain)
		stored.Issues = appendSceneSpatialIssue(stored.Issues, SceneSpatialIssue{Code: "SCENE_PACK_NOT_READY", Severity: SceneSpatialIssueBlocker, Path: "status", Message: "场景空间资产包尚未标记为 ready/locked"})
	}
	return &stored, gateArtifact, packArtifact, nil
}

func appendSceneSpatialIssue(issues []SceneSpatialIssue, issue SceneSpatialIssue) []SceneSpatialIssue {
	for _, existing := range issues {
		if existing.Code == issue.Code && existing.Path == issue.Path {
			return issues
		}
	}
	return append(issues, issue)
}

func decodeFilmGenerationSourceRefs(value string) ([]string, error) {
	var decoded []string
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return nil, BadAuthRequest("Generation Request 来源记录格式无效")
	}
	refs := make([]string, 0, len(decoded))
	for _, ref := range decoded {
		refs = appendUniqueFilmSourceRef(refs, strings.TrimSpace(ref))
	}
	if len(refs) == 0 {
		return nil, BadAuthRequest("Generation Request 缺少来源版本记录")
	}
	return refs, nil
}

func filmSourceRefExists(refs []string, wanted string) bool {
	for _, ref := range refs {
		if ref == wanted {
			return true
		}
	}
	return false
}

func positivePayloadInt(value any) (int, bool) {
	number, ok := value.(float64)
	if !ok || number < 1 || number != float64(int(number)) {
		return 0, false
	}
	return int(number), true
}

func generationRequestDurationMS(payload map[string]any) int64 {
	value, ok := payload["durationMs"].(float64)
	if !ok || value <= 0 || value != float64(int64(value)) {
		return 0
	}
	return int64(value)
}

func filmGenerationTaskType(mediaType string) string {
	switch mediaType {
	case "image":
		return "canvas_image"
	case "audio":
		return "canvas_audio"
	default:
		return "canvas_video"
	}
}

func filmGenerationRequestReadiness(requestStatus string, projectStatus model.ProjectStatus) (bool, string, []string) {
	if projectStatus == model.ProjectStatusArchived {
		return false, FilmGenerationTaskDraftStateNotSubmittable, []string{"项目已归档，不能创建新的生成任务"}
	}
	switch requestStatus {
	case "ready", "locked":
		return true, FilmGenerationTaskDraftStateAwaitingProviderRoute, []string{"尚未配置 Provider Gateway 路由；当前任务合同只读，不会创建 Task"}
	case "draft", "review":
		return false, FilmGenerationTaskDraftStateAwaitingRequestReview, []string{"Generation Request 尚未准备完成；请先完成评审并保存为 ready 或 locked"}
	case "superseded", "archived":
		return false, FilmGenerationTaskDraftStateNotSubmittable, []string{"Generation Request 已失效或归档，不能提交"}
	default:
		return false, FilmGenerationTaskDraftStateNotSubmittable, []string{"Generation Request 状态无法提交"}
	}
}

func filmGenerationRequestFingerprint(artifact *model.FilmArtifact, input FilmGenerationGatewayInput) string {
	value := struct {
		ArtifactID      string                     `json:"artifactId"`
		ArtifactVersion int                        `json:"artifactVersion"`
		Payload         FilmGenerationGatewayInput `json:"payload"`
	}{
		ArtifactID:      artifact.ID,
		ArtifactVersion: artifact.ObjectVersion,
		Payload:         input,
	}
	encoded, _ := json.Marshal(value)
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:])
}

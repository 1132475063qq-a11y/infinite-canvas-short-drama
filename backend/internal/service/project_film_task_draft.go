package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"infinite-canvas/backend/internal/model"
)

const FilmGenerationTaskDraftSchemaVersion = 1

const (
	FilmGenerationTaskDraftStateAwaitingRequestReview = "awaiting_request_review"
	FilmGenerationTaskDraftStateAwaitingProviderRoute = "awaiting_provider_route"
	FilmGenerationTaskDraftStateNotSubmittable         = "not_submittable"
)

// FilmGenerationGatewayInput is the provider-independent part of a future
// Task.InputJSON. It intentionally has no provider config, channel, API key,
// headers, or model selection. Those fields are resolved inside the future
// Provider Gateway immediately before it atomically creates a queued Task.
type FilmGenerationGatewayInput struct {
	SchemaVersion                    int      `json:"schemaVersion"`
	Mode                             string   `json:"mode"`
	Prompt                           string   `json:"prompt"`
	AspectRatio                      string   `json:"aspectRatio,omitempty"`
	DurationMS                       int64    `json:"durationMs,omitempty"`
	OutputIntent                     string   `json:"outputIntent"`
	GenerationRequestArtifactID      string   `json:"generationRequestArtifactId"`
	GenerationRequestArtifactVersion int      `json:"generationRequestArtifactVersion"`
	PromptArtifactID                 string   `json:"promptArtifactId"`
	PromptArtifactVersion            int      `json:"promptArtifactVersion"`
	SourceRefs                       []string `json:"sourceRefs"`
}

// FilmGenerationTaskDraft is an auditable, read-only bridge between the
// immutable Generation Request and the mutable Task runtime. It is not a Task
// row and must never be claimed by the worker, billed, or sent to a provider.
type FilmGenerationTaskDraft struct {
	SchemaVersion                    int                              `json:"schemaVersion"`
	GenerationRequestArtifactID      string                           `json:"generationRequestArtifactId"`
	GenerationRequestArtifactVersion int                              `json:"generationRequestArtifactVersion"`
	GenerationRequestStatus          string                           `json:"generationRequestStatus"`
	ProjectID                        string                           `json:"projectId"`
	UnitID                           string                           `json:"unitId,omitempty"`
	SceneID                          string                           `json:"sceneId,omitempty"`
	ShotID                           string                           `json:"shotId"`
	TaskType                         string                           `json:"taskType"`
	Operation                        string                           `json:"operation"`
	RequestReady                     bool                             `json:"requestReady"`
	ProviderRouteResolved            bool                             `json:"providerRouteResolved"`
	SubmissionAllowed                bool                             `json:"submissionAllowed"`
	SubmissionState                  string                           `json:"submissionState"`
	Blockers                         []string                         `json:"blockers"`
	RequestFingerprint               string                           `json:"requestFingerprint"`
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
	requestReady, submissionState, blockers := filmGenerationRequestReadiness(artifact.Status, project.Status)

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
		GatewayInput:                     gatewayInput,
	}, nil
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

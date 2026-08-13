package service

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
)

const (
	FilmArtifactTypeActing          = "acting"
	FilmArtifactTypeVideoPromptPack = "video_prompt_pack"
	// GenerationRequest 是提交给 Provider Gateway 前冻结的请求快照；它不是运行中的 Task。
	FilmArtifactTypeGenerationRequest = "generation_request"
)

type SaveProjectFilmArtifactRequest struct {
	ShotID             string         `json:"shotId"`
	ArtifactType       string         `json:"artifactType"`
	Status             string         `json:"status"`
	ResponsibleAgentID string         `json:"responsibleAgentId"`
	Payload            map[string]any `json:"payload"`
}

func (s *Service) SaveProjectFilmArtifact(userID string, projectID string, req SaveProjectFilmArtifactRequest) (model.FilmArtifact, error) {
	if _, err := s.repo.ProjectForUser(userID, projectID); err != nil {
		return model.FilmArtifact{}, err
	}
	shotID := strings.TrimSpace(req.ShotID)
	if shotID == "" {
		return model.FilmArtifact{}, BadAuthRequest("影视 Artifact 必须关联镜头")
	}
	shot, err := s.repo.ShotForProject(projectID, shotID)
	if err != nil {
		return model.FilmArtifact{}, err
	}
	artifactType := strings.TrimSpace(req.ArtifactType)
	payload := req.Payload
	var promptArtifact *model.FilmArtifact
	if artifactType == FilmArtifactTypeGenerationRequest {
		payload, promptArtifact, err = s.normalizeGenerationRequestPayload(projectID, *shot, req.Payload)
		if err != nil {
			return model.FilmArtifact{}, err
		}
	}
	if err := validateFilmArtifactPayload(artifactType, payload); err != nil {
		return model.FilmArtifact{}, err
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "draft"
	}
	if !validFilmArtifactStatus(status) {
		return model.FilmArtifact{}, BadAuthRequest("不支持的影视 Artifact 状态")
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return model.FilmArtifact{}, BadAuthRequest("影视 Artifact 内容格式无效")
	}
	sourceRefs, err := s.filmArtifactSourceRefs(projectID, *shot, artifactType, promptArtifact)
	if err != nil {
		return model.FilmArtifact{}, err
	}
	sourceRefsJSON, err := json.Marshal(sourceRefs)
	if err != nil {
		return model.FilmArtifact{}, err
	}
	now := time.Now()
	artifact := model.FilmArtifact{
		ID:                 newID(),
		ProjectID:          projectID,
		UnitID:             shot.UnitID,
		SceneID:            shot.SceneID,
		ShotID:             shot.ID,
		ArtifactType:       artifactType,
		Status:             status,
		ResponsibleAgentID: strings.TrimSpace(req.ResponsibleAgentID),
		PayloadJSON:        string(payloadJSON),
		SourceRefsJSON:     string(sourceRefsJSON),
		AuthorityRefsJSON:  "[]",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.repo.SaveFilmArtifactVersion(&artifact); err != nil {
		return model.FilmArtifact{}, err
	}
	return artifact, nil
}

func (s *Service) normalizeGenerationRequestPayload(projectID string, shot model.Shot, payload map[string]any) (map[string]any, *model.FilmArtifact, error) {
	promptArtifactID := payloadString(payload, "promptArtifactId")
	if promptArtifactID == "" {
		return nil, nil, BadAuthRequest("Generation Request 必须引用已保存的 Prompt Pack")
	}
	promptArtifact, err := s.repo.LatestFilmArtifact(projectID, shot.ID, FilmArtifactTypeVideoPromptPack)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, BadAuthRequest("请先保存包含最终提示词的 Prompt Pack")
	}
	if err != nil {
		return nil, nil, err
	}
	if promptArtifact.ID != promptArtifactID {
		return nil, nil, BadAuthRequest("Prompt Pack 已更新，请刷新 Generation Request 后再保存")
	}

	var promptPayload map[string]any
	if err := json.Unmarshal([]byte(promptArtifact.PayloadJSON), &promptPayload); err != nil {
		return nil, nil, BadAuthRequest("Prompt Pack 内容格式无效，无法编译 Generation Request")
	}
	compiledPrompt := payloadString(promptPayload, "compiledPrompt")
	if compiledPrompt == "" {
		return nil, nil, BadAuthRequest("请先在 Prompt Pack 中填写并保存最终编译提示词")
	}

	// 只允许 Prompt Pack 成为编译提示词的权威来源，避免请求版本与提示词版本漂移。
	normalized := make(map[string]any, len(payload)+2)
	for key, value := range payload {
		normalized[key] = value
	}
	normalized["promptArtifactId"] = promptArtifact.ID
	normalized["promptArtifactVersion"] = promptArtifact.ObjectVersion
	normalized["compiledPrompt"] = compiledPrompt
	return normalized, promptArtifact, nil
}

func (s *Service) filmArtifactSourceRefs(projectID string, shot model.Shot, artifactType string, promptArtifact *model.FilmArtifact) ([]string, error) {
	refs := make([]string, 0, 8)
	if shot.ContractArtifactID != "" {
		refs = appendUniqueFilmSourceRef(refs, shot.ContractArtifactID)
	}
	if artifactType == FilmArtifactTypeVideoPromptPack {
		acting, err := s.repo.LatestFilmArtifact(projectID, shot.ID, FilmArtifactTypeActing)
		if err == nil {
			refs = appendUniqueFilmSourceRef(refs, acting.ID)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if artifactType == FilmArtifactTypeGenerationRequest {
		if promptArtifact == nil {
			return nil, BadAuthRequest("Generation Request 缺少 Prompt Pack 事实来源")
		}
		refs = appendUniqueFilmSourceRef(refs, promptArtifact.ID)
	}
	if artifactType == FilmArtifactTypeVideoPromptPack || artifactType == FilmArtifactTypeGenerationRequest {
		references, err := s.repo.ProjectShotAssetReferences(projectID)
		if err != nil {
			return nil, err
		}
		for _, reference := range references {
			if reference.ShotID == shot.ID {
				refs = appendUniqueFilmSourceRef(refs, reference.AssetVersionID)
			}
		}
	}
	return refs, nil
}

func appendUniqueFilmSourceRef(refs []string, value string) []string {
	if value == "" {
		return refs
	}
	for _, ref := range refs {
		if ref == value {
			return refs
		}
	}
	return append(refs, value)
}

func validateFilmArtifactPayload(artifactType string, payload map[string]any) error {
	if payload == nil {
		return BadAuthRequest("影视 Artifact 内容不能为空")
	}
	switch artifactType {
	case FilmArtifactTypeActing:
		for _, forbidden := range []string{"appearance", "physique", "clothing", "voice", "camera", "lens", "shotSize"} {
			if _, exists := payload[forbidden]; exists {
				return BadAuthRequest("Acting 不能修改外貌、声音或镜头领域事实")
			}
		}
		return nil
	case FilmArtifactTypeVideoPromptPack:
		return nil
	case FilmArtifactTypeGenerationRequest:
		return validateGenerationRequestPayload(payload)
	default:
		return BadAuthRequest("不支持的影视 Artifact 类型")
	}
}

func validateGenerationRequestPayload(payload map[string]any) error {
	if containsGenerationSecret(payload) {
		return BadAuthRequest("Generation Request 不能包含 API Key、Token 或其他 Provider 密钥")
	}
	for key := range payload {
		if !isAllowedGenerationRequestPayloadKey(key) {
			return BadAuthRequest("Generation Request 包含不支持的字段：" + key)
		}
	}
	mediaType := strings.ToLower(payloadString(payload, "mediaType"))
	if mediaType != "image" && mediaType != "video" && mediaType != "audio" {
		return BadAuthRequest("Generation Request 必须指定 image、video 或 audio 媒体类型")
	}
	if payloadString(payload, "promptArtifactId") == "" || payloadString(payload, "compiledPrompt") == "" {
		return BadAuthRequest("Generation Request 必须包含已编译 Prompt Pack 的版本引用")
	}
	if mediaType != "audio" && payloadString(payload, "aspectRatio") == "" {
		return BadAuthRequest("Generation Request 必须指定画幅比例")
	}
	if payloadString(payload, "outputIntent") == "" {
		return BadAuthRequest("Generation Request 必须说明输出用途")
	}
	if mediaType == "video" && !positiveDurationMS(payload["durationMs"]) {
		return BadAuthRequest("视频 Generation Request 必须指定有效时长")
	}
	return nil
}

// Generation Request is a provider-independent production fact, not a generic
// provider payload. Keeping this allowlist small prevents browser-supplied
// channel configuration or unreviewed execution options from being frozen into
// an Artifact before the Provider Gateway owns that translation.
func isAllowedGenerationRequestPayloadKey(key string) bool {
	switch key {
	case "promptArtifactId", "promptArtifactVersion", "compiledPrompt", "mediaType", "aspectRatio", "durationMs", "outputIntent":
		return true
	default:
		return false
	}
}

func payloadString(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}

func positiveDurationMS(value any) bool {
	const maxDurationMS = int64(120000)
	switch duration := value.(type) {
	case float64:
		return duration > 0 && duration <= float64(maxDurationMS) && duration == float64(int64(duration))
	case float32:
		return duration > 0 && duration <= float32(maxDurationMS) && duration == float32(int64(duration))
	case int:
		return duration > 0 && int64(duration) <= maxDurationMS
	case int64:
		return duration > 0 && duration <= maxDurationMS
	case json.Number:
		duration, err := duration.Int64()
		return err == nil && duration > 0 && duration <= maxDurationMS
	default:
		return false
	}
}

func containsGenerationSecret(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if isGenerationSecretKey(key) || containsGenerationSecret(nested) {
				return true
			}
		}
	case []any:
		for _, nested := range typed {
			if containsGenerationSecret(nested) {
				return true
			}
		}
	}
	return false
}

func isGenerationSecretKey(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", " ", "").Replace(key))
	if strings.Contains(normalized, "apikey") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "credential") ||
		strings.Contains(normalized, "providerkey") ||
		strings.Contains(normalized, "privatekey") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "bearer") {
		return true
	}
	switch normalized {
	case "authorization", "token", "accesstoken", "refreshtoken":
		return true
	default:
		return false
	}
}

func validFilmArtifactStatus(status string) bool {
	switch status {
	case "draft", "ready", "review", "locked", "superseded", "archived":
		return true
	default:
		return false
	}
}

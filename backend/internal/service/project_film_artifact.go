package service

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/gorm"
)

const (
	FilmArtifactTypeActing          = "acting"
	FilmArtifactTypeVideoPromptPack = "video_prompt_pack"
	// GenerationRequest 是提交给 Provider Gateway 前冻结的请求快照；它不是运行中的 Task。
	FilmArtifactTypeGenerationRequest = "generation_request"
	// SceneAssetPack is a scene-scoped spatial source of truth. The derived
	// gate is stored separately so its decision can be audited without
	// pretending that a validation result is part of the source payload.
	FilmArtifactTypeSceneAssetPack        = "scene_asset_pack"
	FilmArtifactTypeSpatialContinuityGate = "spatial_continuity_gate"
)

type SaveProjectFilmArtifactRequest struct {
	ShotID             string         `json:"shotId"`
	ArtifactType       string         `json:"artifactType"`
	Status             string         `json:"status"`
	ResponsibleAgentID string         `json:"responsibleAgentId"`
	Payload            map[string]any `json:"payload"`
}

type SaveProjectSceneAssetPackRequest struct {
	SceneID            string         `json:"sceneId"`
	Status             string         `json:"status"`
	ResponsibleAgentID string         `json:"responsibleAgentId"`
	ExpectedVersion    *int           `json:"expectedVersion"`
	Payload            map[string]any `json:"payload"`
}

type SceneAssetPackSaveResult struct {
	PackArtifact model.FilmArtifact `json:"packArtifact"`
	GateArtifact model.FilmArtifact `json:"gateArtifact"`
	Gate         SceneSpatialGate   `json:"gate"`
}

type SceneAssetPackDetail struct {
	PackArtifact *model.FilmArtifact `json:"packArtifact,omitempty"`
	GateArtifact *model.FilmArtifact `json:"gateArtifact,omitempty"`
	Gate         SceneSpatialGate    `json:"gate"`
}

// ProjectSceneAssetPack returns the latest source pack and derived gate. A
// scene without a pack is a normal, explicitly uncertain state for the UI.
func (s *Service) ProjectSceneAssetPack(userID string, projectID string, sceneID string) (SceneAssetPackDetail, error) {
	project, err := s.repo.ProjectForUser(userID, projectID)
	if err != nil {
		return SceneAssetPackDetail{}, err
	}
	scene, err := s.repo.SceneForProject(project.ID, strings.TrimSpace(sceneID))
	if err != nil {
		return SceneAssetPackDetail{}, err
	}
	gate, gateArtifact, packArtifact, err := s.sceneSpatialGateForGeneration(project.ID, scene.ID)
	if err != nil {
		return SceneAssetPackDetail{}, err
	}
	return SceneAssetPackDetail{PackArtifact: packArtifact, GateArtifact: gateArtifact, Gate: *gate}, nil
}

// SaveProjectSceneAssetPack appends the source spatial pack and a derived
// Spatial Continuity Gate atomically. A draft/review pack may be incomplete;
// ready/locked packs are only accepted when the gate is PASS.
func (s *Service) SaveProjectSceneAssetPack(userID string, projectID string, req SaveProjectSceneAssetPackRequest) (SceneAssetPackSaveResult, error) {
	project, err := s.repo.ProjectForUser(userID, projectID)
	if err != nil {
		return SceneAssetPackSaveResult{}, err
	}
	if project.Type != model.ProjectTypeShortDrama {
		return SceneAssetPackSaveResult{}, BadAuthRequest("场景空间资产只适用于短剧项目")
	}
	sceneID := strings.TrimSpace(req.SceneID)
	if sceneID == "" {
		return SceneAssetPackSaveResult{}, BadAuthRequest("场景空间资产必须关联场景")
	}
	scene, err := s.repo.SceneForProject(project.ID, sceneID)
	if err != nil {
		return SceneAssetPackSaveResult{}, err
	}
	if req.Payload == nil {
		return SceneAssetPackSaveResult{}, BadAuthRequest("场景空间资产内容不能为空")
	}
	if req.ExpectedVersion == nil || *req.ExpectedVersion < 0 {
		return SceneAssetPackSaveResult{}, BadAuthRequest("场景空间资产缺少有效的预期版本，请刷新后重试")
	}
	payloadJSON, err := json.Marshal(req.Payload)
	if err != nil {
		return SceneAssetPackSaveResult{}, BadAuthRequest("场景空间资产内容格式无效")
	}
	pack, err := DecodeSceneAssetPackPayload(payloadJSON)
	if err != nil {
		return SceneAssetPackSaveResult{}, BadAuthRequest("场景空间资产内容格式无效：" + err.Error())
	}
	if pack.SceneID == "" {
		pack.SceneID = scene.ID
	}
	if pack.SceneID != scene.ID {
		return SceneAssetPackSaveResult{}, BadAuthRequest("场景空间资产与目标场景不一致")
	}
	if pack.SchemaVersion == 0 {
		pack.SchemaVersion = SceneSpatialContractSchemaVersion
	}
	baselinePack, err := s.sceneSpatialContinuityBaseline(project.ID, scene.ID)
	if err != nil {
		return SceneAssetPackSaveResult{}, err
	}
	gate := ValidateSceneAssetPack(pack, baselinePack)
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "draft"
	}
	if !validFilmArtifactStatus(status) {
		return SceneAssetPackSaveResult{}, BadAuthRequest("不支持的场景空间资产状态")
	}
	if (status == "ready" || status == "locked") && gate.Status != SceneSpatialGatePass {
		return SceneAssetPackSaveResult{}, BadAuthRequest("场景空间门禁未通过，不能将空间资产标记为 ready/locked：" + string(gate.Status))
	}
	if gate.Status == SceneSpatialGatePass && !sceneAssetPackStatusReady(status) {
		gate.Status = SceneSpatialGateUncertain
		gate.Issues = append(gate.Issues, SceneSpatialIssue{Code: "SCENE_PACK_NOT_READY", Severity: SceneSpatialIssueBlocker, Path: "status", Message: "空间结构已通过校验，但场景资产包尚未标记为 ready/locked"})
	}
	packPayload, err := EncodeSceneAssetPackPayload(pack)
	if err != nil {
		return SceneAssetPackSaveResult{}, BadAuthRequest("场景空间资产编码失败")
	}
	gate.PackVersion = 0 // repository fills the exact committed version.
	gatePayload, err := json.Marshal(gate)
	if err != nil {
		return SceneAssetPackSaveResult{}, BadAuthRequest("空间门禁结果编码失败")
	}
	now := time.Now()
	packArtifact := model.FilmArtifact{
		ID: newID(), ProjectID: project.ID, UnitID: scene.UnitID, SceneID: scene.ID,
		Scope: model.FilmArtifactScopeScene, ScopeID: scene.ID, ArtifactType: FilmArtifactTypeSceneAssetPack,
		Status: status, ResponsibleAgentID: strings.TrimSpace(req.ResponsibleAgentID), PayloadJSON: string(packPayload),
		SourceRefsJSON: "[]", AuthorityRefsJSON: "[]", CreatedAt: now, UpdatedAt: now,
	}
	gateArtifact := model.FilmArtifact{
		ID: newID(), ProjectID: project.ID, UnitID: scene.UnitID, SceneID: scene.ID,
		Scope: model.FilmArtifactScopeScene, ScopeID: scene.ID, ArtifactType: FilmArtifactTypeSpatialContinuityGate,
		Status: spatialGateArtifactStatus(gate.Status), ResponsibleAgentID: "scene-spatial-gate/1", PayloadJSON: string(gatePayload),
		SourceRefsJSON: "[]", AuthorityRefsJSON: "[]", CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.SaveSceneAssetPackAndGate(&packArtifact, &gateArtifact, *req.ExpectedVersion); err != nil {
		if errors.Is(err, repository.ErrFilmArtifactVersionChanged) {
			return SceneAssetPackSaveResult{}, Conflict("场景空间资产已被其他操作更新，请刷新后重试")
		}
		return SceneAssetPackSaveResult{}, err
	}
	gate.PackArtifactID = packArtifact.ID
	gate.PackArtifactVersion = packArtifact.ObjectVersion
	gate.PackVersion = packArtifact.ObjectVersion
	return SceneAssetPackSaveResult{PackArtifact: packArtifact, GateArtifact: gateArtifact, Gate: gate}, nil
}

// sceneSpatialContinuityBaseline keeps failed or incomplete drafts from
// becoming the canonical comparison source. Only the latest persisted PASS
// gate may establish continuity locks for the next version.
func (s *Service) sceneSpatialContinuityBaseline(projectID string, sceneID string) (*SceneAssetPack, error) {
	gateArtifact, err := s.repo.LatestFilmArtifactForScopeWithStatus(projectID, model.FilmArtifactScopeScene, sceneID, FilmArtifactTypeSpatialContinuityGate, "ready")
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var gate SceneSpatialGate
	if err := json.Unmarshal([]byte(gateArtifact.PayloadJSON), &gate); err != nil || gate.Status != SceneSpatialGatePass || strings.TrimSpace(gate.PackArtifactID) == "" || gate.PackArtifactVersion < 1 {
		return nil, Conflict("最近一次通过的空间门禁记录不完整，请先修复场景空间事实")
	}
	packArtifact, err := s.repo.FilmArtifactForProject(projectID, gate.PackArtifactID)
	if err != nil || packArtifact.Scope != model.FilmArtifactScopeScene || packArtifact.ScopeID != sceneID || packArtifact.ArtifactType != FilmArtifactTypeSceneAssetPack || packArtifact.ObjectVersion != gate.PackArtifactVersion || !sceneAssetPackStatusReady(packArtifact.Status) {
		return nil, Conflict("最近一次通过的空间门禁与场景资产版本不一致，请先修复场景空间事实")
	}
	pack, err := DecodeSceneAssetPackPayload([]byte(packArtifact.PayloadJSON))
	if err != nil {
		return nil, Conflict("最近一次通过的场景空间资产无法解析，请先修复场景空间事实")
	}
	return &pack, nil
}

func spatialGateArtifactStatus(status SceneSpatialGateStatus) string {
	if status == SceneSpatialGatePass {
		return "ready"
	}
	return "review"
}

func sceneAssetPackStatusReady(status string) bool {
	return status == "ready" || status == "locked"
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
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "draft"
	}
	if !validFilmArtifactStatus(status) {
		return model.FilmArtifact{}, BadAuthRequest("不支持的影视 Artifact 状态")
	}
	payload := req.Payload
	var promptArtifact *model.FilmArtifact
	if artifactType == FilmArtifactTypeVideoPromptPack {
		payload, err = s.normalizeVideoPromptPackPayload(projectID, *shot, req.Payload, status)
		if err != nil {
			return model.FilmArtifact{}, err
		}
	} else if artifactType == FilmArtifactTypeGenerationRequest {
		payload, promptArtifact, err = s.normalizeGenerationRequestPayload(projectID, *shot, req.Payload)
		if err != nil {
			return model.FilmArtifact{}, err
		}
	}
	if err := validateFilmArtifactPayload(artifactType, payload); err != nil {
		return model.FilmArtifact{}, err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return model.FilmArtifact{}, BadAuthRequest("影视 Artifact 内容格式无效")
	}
	sourceRefs, err := s.filmArtifactSourceRefs(projectID, *shot, artifactType, promptArtifact, payload)
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

// normalizeVideoPromptPackPayload is the only write path that can create a
// scene-linked Prompt Pack. An empty draft/review is deliberately allowed so
// production nodes can be created before prompt authoring begins. As soon as
// there is creative prose, the server resolves the exact PASS Pack/Gate and
// compiles the immutable spatial projection into the final prompt.
func (s *Service) normalizeVideoPromptPackPayload(projectID string, shot model.Shot, payload map[string]any, status string) (map[string]any, error) {
	if payload == nil {
		return nil, BadAuthRequest("Prompt Pack 内容不能为空")
	}
	normalized := cloneFilmPayload(payload)
	for _, key := range spatialPromptFieldKeys() {
		delete(normalized, key)
	}
	creative := promptCreativeText(payload)
	if creative == "" {
		if status != "draft" && status != "review" {
			return nil, BadAuthRequest("未编译的 Prompt Pack 只能保存为 draft/review；请先填写创作提示词并完成空间编译")
		}
		// A blank draft must not retain a browser-provided final prompt. It has
		// no server-owned spatial projection and therefore cannot feed a
		// Generation Request until it is compiled on a later save.
		delete(normalized, "compiledPrompt")
		return normalized, nil
	}
	if strings.TrimSpace(shot.SceneID) == "" {
		normalized["creativePrompt"] = creative
		normalized["compiledPrompt"] = creative
		return normalized, nil
	}

	gate, gateArtifact, packArtifact, err := s.sceneSpatialGateForGeneration(projectID, shot.SceneID)
	if err != nil {
		return nil, err
	}
	if gate == nil || gate.Status != SceneSpatialGatePass || gateArtifact == nil || packArtifact == nil {
		status := SceneSpatialGateUncertain
		if gate != nil {
			status = gate.Status
		}
		return nil, BadAuthRequest("场景空间门禁当前为 " + string(status) + "，Prompt Pack 只能消费 PASS 且 ready/locked 的场景资产包")
	}
	pack, err := DecodeSceneAssetPackPayload([]byte(packArtifact.PayloadJSON))
	if err != nil {
		return nil, BadAuthRequest("场景空间资产包无法解析，不能编译 Prompt Pack")
	}
	projection, err := BuildLockedSceneAssetPackProjection(pack, *packArtifact, *gateArtifact, *gate, payloadString(payload, "cameraAnchorId"), payloadString(payload, "viewId"))
	if err != nil {
		return nil, BadAuthRequest("Prompt Pack 空间视角无效：" + err.Error())
	}
	compiled, err := CompileLockedScenePrompt(creative, projection)
	if err != nil {
		return nil, BadAuthRequest("Prompt Pack 缺少可编译的创作提示词")
	}
	normalized["creativePrompt"] = creative
	normalized["compiledPrompt"] = compiled
	normalized["lockedSceneAssetPack"] = projection
	normalized["spatialPackArtifactId"] = projection.PackArtifactID
	normalized["spatialPackArtifactVersion"] = projection.PackArtifactVersion
	normalized["spatialGateArtifactId"] = projection.GateArtifactID
	normalized["spatialGateArtifactVersion"] = projection.GateArtifactVersion
	normalized["spatialContractSchemaVersion"] = projection.SchemaVersion
	normalized["cameraAnchorId"] = projection.CameraAnchorID
	normalized["viewId"] = projection.ViewID
	normalized["spatialBindingStatus"] = string(projection.ContinuityStatus)
	return normalized, nil
}

func cloneFilmPayload(payload map[string]any) map[string]any {
	cloned := make(map[string]any, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}
	return cloned
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

	// 只允许 Prompt Pack 成为编译提示词和空间约束的权威来源，避免请求
	// 版本、提示词版本与场景版本漂移。
	normalized := cloneFilmPayload(payload)
	for _, key := range spatialPromptFieldKeys() {
		delete(normalized, key)
	}
	normalized["promptArtifactId"] = promptArtifact.ID
	normalized["promptArtifactVersion"] = promptArtifact.ObjectVersion
	normalized["compiledPrompt"] = compiledPrompt
	if strings.TrimSpace(shot.SceneID) != "" {
		projection, projectionErr := decodeLockedSceneAssetPackProjection(promptPayload)
		if projectionErr != nil {
			return nil, nil, BadAuthRequest("场景镜头的 Prompt Pack 尚未包含锁定空间投影：" + projectionErr.Error())
		}
		if projection.SceneID != shot.SceneID {
			return nil, nil, BadAuthRequest("Prompt Pack 锁定空间投影与镜头场景不一致")
		}
		gate, gateArtifact, packArtifact, gateErr := s.sceneSpatialGateForGeneration(projectID, shot.SceneID)
		if gateErr != nil {
			return nil, nil, gateErr
		}
		if gate == nil || gate.Status != SceneSpatialGatePass || gateArtifact == nil || packArtifact == nil || projection.PackArtifactID != packArtifact.ID || projection.PackArtifactVersion != packArtifact.ObjectVersion || projection.GateArtifactID != gateArtifact.ID || projection.GateArtifactVersion != gateArtifact.ObjectVersion {
			return nil, nil, Conflict("Prompt Pack 锁定的场景空间版本已变化，请重新保存 Prompt Pack 后再创建 Generation Request")
		}
		for _, key := range spatialPromptFieldKeys() {
			if value, exists := promptPayload[key]; exists {
				normalized[key] = value
			}
		}
	}
	return normalized, promptArtifact, nil
}

func (s *Service) filmArtifactSourceRefs(projectID string, shot model.Shot, artifactType string, promptArtifact *model.FilmArtifact, payload map[string]any) ([]string, error) {
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
		// Use the exact server-generated spatial references carried by the
		// payload, rather than querying "latest" a second time after compile.
		for _, key := range []string{"spatialPackArtifactId", "spatialGateArtifactId"} {
			refs = appendUniqueFilmSourceRef(refs, payloadString(payload, key))
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
		if containsGenerationSecret(payload) {
			return BadAuthRequest("Prompt Pack 不能包含 API Key、Token 或其他 Provider 密钥")
		}
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
	if mediaType != "audio" && !validGenerationAspectRatio(payloadString(payload, "aspectRatio")) {
		return BadAuthRequest("Generation Request 必须指定有效画幅比例")
	}
	if payloadString(payload, "outputIntent") == "" {
		return BadAuthRequest("Generation Request 必须说明输出用途")
	}
	if mediaType == "video" && !positiveDurationMS(payload["durationMs"]) {
		return BadAuthRequest("视频 Generation Request 必须指定有效时长")
	}
	return nil
}

func validGenerationAspectRatio(value string) bool {
	value = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(value, "×", "x")))
	separator := ":"
	if strings.Contains(value, "x") {
		separator = "x"
	}
	parts := strings.Split(value, separator)
	if len(parts) != 2 {
		return false
	}
	width, widthErr := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	height, heightErr := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	return widthErr == nil && heightErr == nil && width > 0 && height > 0 && !math.IsInf(width, 0) && !math.IsInf(height, 0) && !math.IsNaN(width) && !math.IsNaN(height)
}

// Generation Request is a provider-independent production fact, not a generic
// provider payload. Keeping this allowlist small prevents browser-supplied
// channel configuration or unreviewed execution options from being frozen into
// an Artifact before the Provider Gateway owns that translation.
func isAllowedGenerationRequestPayloadKey(key string) bool {
	switch key {
	case "promptArtifactId", "promptArtifactVersion", "compiledPrompt", "mediaType", "aspectRatio", "durationMs", "outputIntent",
		"lockedSceneAssetPack", "spatialPackArtifactId", "spatialPackArtifactVersion",
		"spatialGateArtifactId", "spatialGateArtifactVersion", "spatialContractSchemaVersion", "cameraAnchorId", "viewId", "spatialBindingStatus":
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
		durationMS, err := duration.Int64()
		return err == nil && durationMS > 0 && durationMS <= maxDurationMS
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

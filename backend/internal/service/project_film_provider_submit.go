package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"
)

const FilmGenerationTaskSubmissionSchemaVersion = 1

// SubmitFilmGenerationTaskRequest deliberately accepts no endpoint,
// credential, header, provider-job ID or arbitrary provider option. The
// selected channel/model must come from the redacted server catalog.
type SubmitFilmGenerationTaskRequest struct {
	CanvasID           string `json:"canvasId"`
	CanvasNodeID       string `json:"canvasNodeId"`
	RequestFingerprint string `json:"requestFingerprint"`
	ChannelID          string `json:"channelId"`
	Model              string `json:"model"`
}

type FilmGenerationTaskSubmission struct {
	SchemaVersion                    int         `json:"schemaVersion"`
	GenerationRequestArtifactID      string      `json:"generationRequestArtifactId"`
	GenerationRequestArtifactVersion int         `json:"generationRequestArtifactVersion"`
	RequestFingerprint               string      `json:"requestFingerprint"`
	CanvasID                         string      `json:"canvasId"`
	CanvasNodeID                     string      `json:"canvasNodeId"`
	ProjectionRevision               int         `json:"projectionRevision"`
	Created                          bool        `json:"created"`
	IdempotentReplay                 bool        `json:"idempotentReplay"`
	Task                             *model.Task `json:"task"`
}

// SubmitFilmGenerationTask is the first write boundary of the Film Provider
// Gateway. It produces one normal queued Task and never calls a provider
// synchronously. The Worker can only observe the Task after the same database
// transaction has reserved billing and installed the server-owned projection.
func (s *Service) SubmitFilmGenerationTask(userID string, projectID string, artifactID string, req SubmitFilmGenerationTaskRequest) (*FilmGenerationTaskSubmission, error) {
	if err := s.RequireFeature(FeatureShortDrama); err != nil {
		return nil, err
	}
	projectID = strings.TrimSpace(projectID)
	artifactID = strings.TrimSpace(artifactID)
	canvasID := strings.TrimSpace(req.CanvasID)
	canvasNodeID := strings.TrimSpace(req.CanvasNodeID)
	requestFingerprint := strings.TrimSpace(req.RequestFingerprint)
	channelID := strings.TrimSpace(req.ChannelID)
	modelKey := strings.TrimPrefix(strings.TrimSpace(req.Model), "models/")
	if projectID == "" || artifactID == "" || canvasID == "" || canvasNodeID == "" || requestFingerprint == "" || channelID == "" || modelKey == "" {
		return nil, BadAuthRequest("影视生成提交缺少画布、节点、请求指纹、系统渠道或模型")
	}

	draft, err := s.FilmGenerationTaskDraft(userID, projectID, artifactID)
	if err != nil {
		return nil, err
	}
	if !draft.RequestReady {
		return nil, BadAuthRequest("Generation Request 当前不能提交：" + strings.Join(draft.Blockers, "；"))
	}
	if requestFingerprint != draft.RequestFingerprint {
		return nil, Conflict("Generation Request 已变化，请刷新任务合同和渠道列表后重试")
	}
	if draft.GatewayInput.Mode != "image" && draft.GatewayInput.Mode != "video" {
		return nil, BadAuthRequest("当前原子提交只开放图片和视频 Generation Request")
	}

	canvas, err := s.repo.CanvasProjectForUser(userID, canvasID)
	if err != nil {
		return nil, err
	}
	if canvas.ProjectID != draft.ProjectID {
		return nil, BadAuthRequest("画布未关联当前影视项目，拒绝提交")
	}
	if err := validateFilmGenerationProjectionNode(canvas.PayloadJSON, canvasNodeID, draft); err != nil {
		return nil, err
	}
	if existing, found, existingErr := s.existingFilmGenerationSubmission(userID, canvasID, canvasNodeID, draft); existingErr != nil {
		return nil, existingErr
	} else if found {
		return existing, nil
	}

	resolution, err := s.resolveFilmGenerationProviderRoute(draft, channelID, modelKey)
	if err != nil {
		return nil, BadAuthRequest("所选系统渠道或模型当前不可用，请刷新渠道列表后重试")
	}
	if !resolution.channelModel.PriceConfigured {
		return nil, BadAuthRequest("所选模型尚未配置用户积分价格")
	}
	if strings.TrimSpace(resolution.channel.BaseURL) == "" || strings.TrimSpace(resolution.channel.APIKey) == "" || (requiresFilmProviderSecret(resolution.channelModel.Protocol) && strings.TrimSpace(resolution.channel.SecretKey) == "") {
		return nil, BadAuthRequest("所选系统渠道尚未完成服务端配置")
	}

	input, err := filmGenerationRuntimeInput(draft, resolution, canvasID, canvasNodeID)
	if err != nil {
		return nil, err
	}
	if err := s.ValidateTaskCapability(input); err != nil {
		return nil, err
	}
	if containsInlineMediaDataURL(input) {
		return nil, BadAuthRequest("影视生成任务不能包含内嵌媒体，请先保存为项目资源")
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, BadAuthRequest("影视生成任务输入格式无效")
	}

	policy, err := s.RuntimePolicy()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	task := model.Task{
		ID: newID(), UserID: userID, ProjectID: canvasID, Type: draft.TaskType,
		Status: model.TaskStatusQueued, Stage: "等待队列调度", Progress: 5,
		Prompt: draft.GatewayInput.Prompt, Operation: draft.Operation,
		Provider: model.TaskProviderFilmGateway, Model: resolution.channelModel.ModelKey,
		InputJSON: string(inputJSON), CreatedAt: now, UpdatedAt: now,
	}
	billingOrder, err := s.filmGenerationBillingOrder(userID, &task, draft, resolution, canvasNodeID)
	if err != nil {
		return nil, err
	}
	if billingOrder != nil {
		task.BillingOrderID = billingOrder.ID
	}
	attempt, err := filmGenerationAttemptFromRuntimeInput(task, input, now)
	if err != nil {
		return nil, err
	}
	artifact, err := s.repo.FilmArtifactForProject(draft.ProjectID, draft.GenerationRequestArtifactID)
	if err != nil {
		return nil, err
	}
	patch := model.CanvasProjectionPatch{
		ID: newID(), UserID: userID, CanvasID: canvasID, NodeID: canvasNodeID,
		PatchKind: CanvasProjectionPatchKindFilmGenerationTask,
		TargetProjectID: draft.ProjectID, TargetArtifactID: draft.GenerationRequestArtifactID,
		TargetArtifactVersion: draft.GenerationRequestArtifactVersion, TaskID: task.ID,
		CreatedAt: now, UpdatedAt: now,
	}
	write := repository.FilmGenerationAtomicCreateInput{
		UserID: userID, ProjectID: draft.ProjectID, CanvasPayloadJSON: canvas.PayloadJSON,
		GenerationRequestArtifact: *artifact, Channel: *resolution.channel, ChannelModel: *resolution.channelModel,
		Task: &task, Attempt: attempt, BillingOrder: billingOrder, Patch: &patch, ActiveTaskLimit: policy.Task.ActiveTaskLimit,
	}
	created, err := s.createFilmGenerationTaskWithinStorageQuota(write, policy)
	if err != nil {
		return nil, s.mapFilmGenerationSubmitError(err, policy)
	}
	if err := validateFilmGenerationProjectionTask(&created.Task, canvasID, canvasNodeID, draft); err != nil {
		return nil, err
	}
	if created.Created {
		s.recordActivity(userID, "task", 1)
		_ = s.log(userID, created.Task.ID, "info", "影视生成任务已原子进入队列", "")
	}
	return filmGenerationSubmission(draft, created.Task, created.Patch, created.Created), nil
}

func (s *Service) existingFilmGenerationSubmission(userID string, canvasID string, canvasNodeID string, draft FilmGenerationTaskDraft) (*FilmGenerationTaskSubmission, bool, error) {
	patches, err := s.repo.CanvasProjectionPatchesForCanvas(userID, canvasID)
	if err != nil {
		return nil, false, err
	}
	for _, patch := range patches {
		if patch.NodeID != canvasNodeID || patch.PatchKind != CanvasProjectionPatchKindFilmGenerationTask || patch.TargetProjectID != draft.ProjectID || patch.TargetArtifactID != draft.GenerationRequestArtifactID || patch.TargetArtifactVersion != draft.GenerationRequestArtifactVersion {
			continue
		}
		task, taskErr := s.repo.TaskForUser(userID, patch.TaskID)
		if taskErr != nil {
			return nil, false, taskErr
		}
		if err := validateFilmGenerationProjectionTask(task, canvasID, canvasNodeID, draft); err != nil {
			return nil, false, err
		}
		attempt, attemptErr := s.repo.GenerationAttemptForTaskNumber(task.ID, 1)
		if attemptErr != nil || attempt.UserID != userID || attempt.DomainProjectID != draft.ProjectID || attempt.CanvasID != canvasID || attempt.CanvasNodeID != canvasNodeID || attempt.GenerationRequestArtifactID != draft.GenerationRequestArtifactID || attempt.GenerationRequestArtifactVersion != draft.GenerationRequestArtifactVersion || attempt.RequestFingerprint != draft.RequestFingerprint {
			return nil, false, Conflict("服务器中的影视生成执行事实链不完整，请由管理员核对后再提交")
		}
		return filmGenerationSubmission(draft, *task, patch, false), true, nil
	}
	return nil, false, nil
}

func filmGenerationSubmission(draft FilmGenerationTaskDraft, task model.Task, patch model.CanvasProjectionPatch, created bool) *FilmGenerationTaskSubmission {
	return &FilmGenerationTaskSubmission{
		SchemaVersion: FilmGenerationTaskSubmissionSchemaVersion,
		GenerationRequestArtifactID: draft.GenerationRequestArtifactID,
		GenerationRequestArtifactVersion: draft.GenerationRequestArtifactVersion,
		RequestFingerprint: draft.RequestFingerprint,
		CanvasID: patch.CanvasID, CanvasNodeID: patch.NodeID,
		ProjectionRevision: patch.Revision, Created: created, IdempotentReplay: !created,
		Task: taskForOutput(task),
	}
}

func filmGenerationRuntimeInput(draft FilmGenerationTaskDraft, resolution filmGenerationProviderRouteResolution, canvasID string, canvasNodeID string) (map[string]any, error) {
	config, executionMetadata, err := filmGenerationRuntimeConfig(draft, resolution)
	if err != nil {
		return nil, err
	}
	metadata := map[string]any{
		"domainProjectId": draft.ProjectID,
		"canvasId": canvasID,
		"canvasNodeId": canvasNodeID,
		"unitId": draft.UnitID,
		"sceneId": draft.SceneID,
		"shotId": draft.ShotID,
		"outputIntent": draft.GatewayInput.OutputIntent,
		"requestFingerprint": draft.RequestFingerprint,
		"generationRequestArtifactId": draft.GenerationRequestArtifactID,
		"generationRequestArtifactVersion": draft.GenerationRequestArtifactVersion,
		"promptArtifactId": draft.GatewayInput.PromptArtifactID,
		"promptArtifactVersion": draft.GatewayInput.PromptArtifactVersion,
		"sourceRefs": append([]string(nil), draft.GatewayInput.SourceRefs...),
		"providerRoute": map[string]any{
			"channelId": resolution.channel.ID,
			"channelModelId": resolution.channelModel.ID,
			"model": resolution.channelModel.ModelKey,
			"capability": resolution.channelModel.Capability,
			"protocol": string(resolution.channelModel.Protocol),
			"capabilityVersion": resolution.channelModel.CapabilityVersion,
			"priceVersion": resolution.channelModel.PriceVersion,
		},
	}
	for key, value := range executionMetadata {
		metadata[key] = value
	}
	return map[string]any{
		"mode": draft.GatewayInput.Mode,
		"prompt": draft.GatewayInput.Prompt,
		"config": config,
		"metadata": metadata,
		"canvasId": canvasID,
		"canvasNodeId": canvasNodeID,
		"domainProjectId": draft.ProjectID,
		"generationRequestArtifactId": draft.GenerationRequestArtifactID,
		"generationRequestArtifactVersion": draft.GenerationRequestArtifactVersion,
	}, nil
}

func filmGenerationRuntimeConfig(draft FilmGenerationTaskDraft, resolution filmGenerationProviderRouteResolution) (map[string]any, map[string]any, error) {
	config := map[string]any{"channelId": resolution.channel.ID, "model": resolution.channelModel.ModelKey}
	metadata := map[string]any{}
	capability, err := DecodeModelCapabilityConfig(resolution.channelModel.CapabilityConfigJSON)
	if err != nil {
		return nil, nil, BadAuthRequest("所选模型能力配置无效")
	}
	if draft.GatewayInput.Mode == "image" {
		profile := DefaultImageCapabilityConfig(string(resolution.channelModel.Protocol), resolution.channelModel.ModelKey)
		if capability != nil && capability.Image != nil {
			profile = capability.Image
		}
		if err := validateImageCapabilityConfig(profile); err != nil {
			return nil, nil, err
		}
		size, err := filmGenerationImageSize(profile.Size, draft.GatewayInput.AspectRatio)
		if err != nil {
			return nil, nil, err
		}
		if size != "" {
			config["size"] = size
		}
		if profile.Quality.Supported && strings.TrimSpace(profile.Quality.Default) != "" {
			config["quality"] = profile.Quality.Default
		}
		if profile.TransparentBackground.Supported {
			config["transparentBackground"] = strconv.FormatBool(profile.TransparentBackground.Default)
		}
		config["count"] = "1"
		return config, metadata, nil
	}
	if capability == nil || capability.Video == nil {
		return nil, nil, BadAuthRequest("所选视频模型尚未配置能力参数")
	}
	profile := capability.Video
	if err := validateVideoCapabilityConfig(profile); err != nil {
		return nil, nil, err
	}
	if draft.GatewayInput.DurationMS <= 0 || draft.GatewayInput.DurationMS%1000 != 0 {
		return nil, nil, BadAuthRequest("视频 Generation Request 必须使用整秒时长")
	}
	seconds := int(draft.GatewayInput.DurationMS / 1000)
	if !videoDurationAllowed(profile.Duration, seconds) {
		return nil, nil, BadAuthRequest("Generation Request 时长不在所选模型支持范围内")
	}
	if !videoRatioAllowed(profile.Ratios, draft.GatewayInput.AspectRatio) {
		return nil, nil, BadAuthRequest("Generation Request 画幅不在所选模型支持范围内")
	}
	config["size"] = draft.GatewayInput.AspectRatio
	config["videoSeconds"] = strconv.Itoa(seconds)
	config["vquality"] = profile.DefaultResolution
	if profile.GenerateAudio.Supported {
		config["videoGenerateAudio"] = strconv.FormatBool(profile.GenerateAudio.Default)
	}
	if profile.Watermark.Supported {
		config["videoWatermark"] = strconv.FormatBool(profile.Watermark.Default)
	}
	metadata["videoEditOperation"] = profile.DefaultOperation
	return config, metadata, nil
}

func filmGenerationImageSize(profile ImageSizeConfig, aspectRatio string) (string, error) {
	if profile.Parameter == "none" {
		return "", nil
	}
	aspectRatio = strings.TrimSpace(aspectRatio)
	if containsCapabilityString(profile.Values, aspectRatio) {
		return aspectRatio, nil
	}
	for _, candidate := range profile.Values {
		if videoRatioAllowed([]string{aspectRatio}, candidate) {
			return candidate, nil
		}
	}
	if profile.AllowCustom {
		return aspectRatio, nil
	}
	return "", BadAuthRequest("Generation Request 画幅不在所选图片模型支持范围内")
}

func (s *Service) filmGenerationBillingOrder(userID string, task *model.Task, draft FilmGenerationTaskDraft, resolution filmGenerationProviderRouteResolution, canvasNodeID string) (*model.BillingOrder, error) {
	enabled, err := s.FeatureEnabled(FeatureCredits)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, nil
	}
	quantity := int64(1)
	if draft.GatewayInput.Mode == "video" {
		quantity = draft.GatewayInput.DurationMS / 1000
	}
	return s.newBillingOrder(
		userID, task.ID, filmGenerationIdempotencyKey(userID, task.ProjectID, canvasNodeID, draft.GenerationRequestArtifactID, draft.GenerationRequestArtifactVersion),
		resolution.channel.ID, resolution.channelModel.ModelKey, draft.GatewayInput.Mode, draft.Operation, quantity, tokenBillingEstimate{},
	)
}

func filmGenerationIdempotencyKey(userID string, canvasID string, canvasNodeID string, artifactID string, artifactVersion int) string {
	value := fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%d", userID, canvasID, canvasNodeID, artifactID, artifactVersion)
	hash := sha256.Sum256([]byte(value))
	return "film_generation:" + hex.EncodeToString(hash[:])
}

func (s *Service) createFilmGenerationTaskWithinStorageQuota(input repository.FilmGenerationAtomicCreateInput, policy RuntimePolicySetting) (*repository.FilmGenerationAtomicCreateResult, error) {
	s.storageMu.Lock()
	defer s.storageMu.Unlock()
	usage, err := s.repo.UserStorageUsage(input.UserID)
	if err != nil {
		return nil, err
	}
	incomingBytes := int64(len([]byte(input.Task.Prompt)) + len([]byte(input.Task.InputJSON)) + len([]byte(input.Task.Error)))
	if input.Attempt != nil {
		incomingBytes += int64(len(input.Attempt.RequestFingerprint) + len(input.Attempt.Model) + len(input.Attempt.Capability) + len(input.Attempt.Protocol) + len(input.Attempt.Error))
	}
	if err := validateTaskStorageQuotaWithPolicy(usage, incomingBytes, policy.Resource); err != nil {
		return nil, err
	}
	return s.repo.CreateFilmGenerationTaskAtomic(input)
}

func (s *Service) mapFilmGenerationSubmitError(err error, policy RuntimePolicySetting) error {
	switch {
	case errors.Is(err, repository.ErrFilmGenerationTargetChanged):
		return Conflict("画布节点或 Generation Request 已变化，请刷新后重试")
	case errors.Is(err, repository.ErrFilmGenerationRouteChanged):
		return Conflict("系统渠道配置在提交期间发生变化，请刷新渠道列表后重试")
	case errors.Is(err, repository.ErrFilmGenerationNodeBusy):
		return Conflict("该画布节点已有排队或运行中的生成任务，请等待完成后再创建新版本")
	case errors.Is(err, repository.ErrActiveTaskLimit):
		return BadAuthRequest(fmt.Sprintf("同时排队或运行的任务最多 %d 个，请等待已有任务完成", policy.Task.ActiveTaskLimit))
	case errors.Is(err, repository.ErrInsufficientCredits):
		return BadAuthRequest("积分不足，请先使用兑换码充值")
	default:
		return err
	}
}

// validateTaskProviderRouteSnapshot protects an already-queued film task from
// silently switching protocol or capability after an administrator edits a
// channel model. Backend channel endpoint/credential rotation remains allowed
// because those values are intentionally resolved only at execution time.
func (s *Service) validateTaskProviderRouteSnapshot(config providerConfig, metadata map[string]interface{}) error {
	route, ok := metadata["providerRoute"].(map[string]interface{})
	if !ok {
		return nil
	}
	channelID := metadataString(route, "channelId")
	channelModelID := metadataString(route, "channelModelId")
	modelKey := strings.TrimPrefix(metadataString(route, "model"), "models/")
	protocol := metadataString(route, "protocol")
	capability := metadataString(route, "capability")
	capabilityVersion := firstInt64(route, "capabilityVersion")
	_, hasCapabilityVersion := route["capabilityVersion"]
	if channelID == "" || channelModelID == "" || modelKey == "" || protocol == "" || capability == "" || !hasCapabilityVersion {
		return errors.New("影视生成任务的 Provider 路由快照不完整")
	}
	if strings.TrimSpace(config.ChannelID) != channelID || strings.TrimPrefix(strings.TrimSpace(config.Model), "models/") != modelKey {
		return errors.New("影视生成任务的 Provider 路由与执行配置不一致")
	}
	item, err := s.repo.ChannelModelByID(channelID, channelModelID)
	if err != nil || !item.Enabled || item.ModelKey != modelKey || string(item.Protocol) != protocol || item.Capability != capability || item.CapabilityVersion != capabilityVersion {
		return errors.New("影视生成任务绑定的 Provider 能力版本已变化，请创建新的 Generation Request 后重试")
	}
	return nil
}

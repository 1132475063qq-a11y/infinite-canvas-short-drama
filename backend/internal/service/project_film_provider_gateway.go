package service

import (
	"strings"

	"infinite-canvas/backend/internal/model"
)

const FilmGenerationProviderRouteCatalogSchemaVersion = 1

const (
	FilmGenerationProviderRouteStateAwaitingRequestReview = "awaiting_request_review"
	FilmGenerationProviderRouteStateRoutesAvailable       = "provider_routes_available"
	FilmGenerationProviderRouteStateConfigurationRequired = "provider_route_configuration_required"
	FilmGenerationProviderRouteStateUnavailable           = "provider_route_unavailable"
)

// FilmGenerationProviderRoute is a public, non-secret route candidate. It is
// deliberately not a provider config: the browser can inspect readiness but
// never receives Base URL, credentials, headers, or a provider job handle.
type FilmGenerationProviderRoute struct {
	ChannelID              string   `json:"channelId"`
	ChannelName            string   `json:"channelName"`
	Model                  string   `json:"model"`
	ModelDisplayName       string   `json:"modelDisplayName"`
	Capability             string   `json:"capability"`
	Protocol               string   `json:"protocol"`
	BillingMode            string   `json:"billingMode"`
	UnitPriceMicrocredits  int64    `json:"unitPriceMicrocredits,omitempty"`
	PriceConfigured        bool     `json:"priceConfigured"`
	CapabilityVersion      int64    `json:"capabilityVersion,omitempty"`
	ProviderReady          bool     `json:"providerReady"`
	BillingReady           bool     `json:"billingReady"`
	RouteReady             bool     `json:"routeReady"`
	Blockers               []string `json:"blockers"`
}

// FilmGenerationProviderRouteCatalog is a read-only discovery response. The
// eventual submit endpoint must resolve the selected route again inside its
// own transaction; a catalog response is never an authorization to queue work.
type FilmGenerationProviderRouteCatalog struct {
	SchemaVersion                    int                           `json:"schemaVersion"`
	GenerationRequestArtifactID      string                        `json:"generationRequestArtifactId"`
	GenerationRequestArtifactVersion int                           `json:"generationRequestArtifactVersion"`
	Mode                             string                        `json:"mode"`
	RequestReady                     bool                          `json:"requestReady"`
	State                            string                        `json:"state"`
	Blockers                         []string                      `json:"blockers"`
	RequestFingerprint               string                        `json:"requestFingerprint"`
	Routes                           []FilmGenerationProviderRoute `json:"routes"`
}

type filmGenerationProviderRouteResolution struct {
	config       providerConfig
	channel      *model.ModelChannel
	channelModel *model.ChannelModel
}

// FilmGenerationProviderRoutes exposes only eligible backend system-channel
// candidates for an immutable request. It is intentionally side-effect free:
// no Task, billing order, ProviderJob, Result, or Canvas DomainRef is written.
func (s *Service) FilmGenerationProviderRoutes(userID string, projectID string, artifactID string) (FilmGenerationProviderRouteCatalog, error) {
	draft, err := s.FilmGenerationTaskDraft(userID, projectID, artifactID)
	if err != nil {
		return FilmGenerationProviderRouteCatalog{}, err
	}
	catalog := FilmGenerationProviderRouteCatalog{
		SchemaVersion:                    FilmGenerationProviderRouteCatalogSchemaVersion,
		GenerationRequestArtifactID:      draft.GenerationRequestArtifactID,
		GenerationRequestArtifactVersion: draft.GenerationRequestArtifactVersion,
		Mode:                             draft.GatewayInput.Mode,
		RequestReady:                     draft.RequestReady,
		RequestFingerprint:               draft.RequestFingerprint,
		Routes:                           []FilmGenerationProviderRoute{},
	}
	if !draft.RequestReady {
		// Preserve terminal request states (for example, a superseded artifact or
		// archived project) instead of presenting every non-ready request as one
		// that merely needs review.
		if draft.SubmissionState == FilmGenerationTaskDraftStateNotSubmittable {
			catalog.State = FilmGenerationTaskDraftStateNotSubmittable
		} else {
			catalog.State = FilmGenerationProviderRouteStateAwaitingRequestReview
		}
		catalog.Blockers = append([]string(nil), draft.Blockers...)
		return catalog, nil
	}

	channels, err := s.repo.SystemChannels(false)
	if err != nil {
		return FilmGenerationProviderRouteCatalog{}, err
	}
	readyCount := 0
	for index := range channels {
		channel := &channels[index]
		models, modelErr := s.repo.ChannelModels(channel.ID, false)
		if modelErr != nil {
			return FilmGenerationProviderRouteCatalog{}, modelErr
		}
		for modelIndex := range models {
			channelModel := &models[modelIndex]
			if channelModel.Capability != draft.GatewayInput.Mode || !stringInSlice(channelModel.ModelKey, channelModelNames(*channel)) {
				continue
			}
			route := s.filmGenerationProviderRouteForCandidate(draft, channel, channelModel)
			catalog.Routes = append(catalog.Routes, route)
			if route.RouteReady {
				readyCount++
			}
		}
	}

	switch {
	case readyCount > 0:
		catalog.State = FilmGenerationProviderRouteStateRoutesAvailable
	case len(catalog.Routes) > 0:
		catalog.State = FilmGenerationProviderRouteStateConfigurationRequired
		catalog.Blockers = []string{"候选系统模型尚未同时满足服务端配置和计费要求，请由管理员完成渠道配置"}
	default:
		catalog.State = FilmGenerationProviderRouteStateUnavailable
		catalog.Blockers = []string{"当前没有与该生成类型匹配的已启用系统渠道模型"}
	}
	return catalog, nil
}

func (s *Service) filmGenerationProviderRouteForCandidate(draft FilmGenerationTaskDraft, channel *model.ModelChannel, channelModel *model.ChannelModel) FilmGenerationProviderRoute {
	route := FilmGenerationProviderRoute{
		ChannelID:             channel.ID,
		ChannelName:           channel.Name,
		Model:                 channelModel.ModelKey,
		ModelDisplayName:      firstNonEmpty(channelModel.DisplayName, channelModel.ModelKey),
		Capability:            channelModel.Capability,
		Protocol:              string(channelModel.Protocol),
		BillingMode:           channelModel.BillingMode,
		PriceConfigured:       channelModel.PriceConfigured,
		CapabilityVersion:     channelModel.CapabilityVersion,
		BillingReady:          channelModel.PriceConfigured,
		Blockers:              []string{},
	}
	if channelModel.PriceConfigured {
		route.UnitPriceMicrocredits = channelModel.UnitPriceMicrocredits
	} else {
		route.Blockers = append(route.Blockers, "模型尚未配置用户积分价格")
	}
	resolution, err := s.resolveFilmGenerationProviderRoute(draft, channel.ID, channelModel.ModelKey)
	if err != nil {
		// 不向非管理员泄漏渠道 URL、鉴权头或密钥缺失细节。
		route.Blockers = append(route.Blockers, "服务端渠道契约尚未完成，暂不能提交")
		return route
	}
	if _, _, err := filmGenerationRuntimeConfig(draft, resolution); err != nil {
		route.Blockers = append(route.Blockers, "模型能力不支持当前请求的画幅、时长或生成模式")
	}
	if strings.TrimSpace(channel.BaseURL) == "" || strings.TrimSpace(channel.APIKey) == "" {
		route.Blockers = append(route.Blockers, "服务端渠道配置尚不完整")
	} else if requiresFilmProviderSecret(channelModel.Protocol) && strings.TrimSpace(channel.SecretKey) == "" {
		route.Blockers = append(route.Blockers, "服务端渠道配置尚不完整")
	}
	if draft.GatewayInput.Mode == "video" && (draft.GatewayInput.DurationMS <= 0 || draft.GatewayInput.DurationMS%1000 != 0) {
		route.Blockers = append(route.Blockers, "视频 Generation Request 必须使用整秒时长后才能提交到当前任务运行时")
	}
	route.ProviderReady = len(route.Blockers) == 0 || (len(route.Blockers) == 1 && route.Blockers[0] == "模型尚未配置用户积分价格")
	route.RouteReady = route.ProviderReady && route.BillingReady
	return route
}

// resolveFilmGenerationProviderRoute is shared preparation for the future
// atomic submit path. Keeping it server-side ensures a browser-selected model
// never supplies its own endpoint, API key, secret key, or outbound headers.
func (s *Service) resolveFilmGenerationProviderRoute(draft FilmGenerationTaskDraft, channelID string, modelKey string) (filmGenerationProviderRouteResolution, error) {
	config, err := s.resolveProviderConfig(providerConfig{ChannelID: strings.TrimSpace(channelID), Model: strings.TrimSpace(modelKey)})
	if err != nil {
		return filmGenerationProviderRouteResolution{}, err
	}
	channel, err := s.repo.SystemChannel(config.ChannelID)
	if err != nil {
		return filmGenerationProviderRouteResolution{}, err
	}
	channelModel, err := s.repo.ChannelModelByKey(channel.ID, config.Model)
	if err != nil {
		return filmGenerationProviderRouteResolution{}, err
	}
	if channelModel.Capability != draft.GatewayInput.Mode {
		return filmGenerationProviderRouteResolution{}, BadAuthRequest("所选系统模型与 Generation Request 媒体类型不匹配")
	}
	if err := validateGenerationInterface(draft.GatewayInput.Mode, config.InterfaceType); err != nil {
		return filmGenerationProviderRouteResolution{}, BadAuthRequest(err.Error())
	}
	return filmGenerationProviderRouteResolution{config: config, channel: channel, channelModel: channelModel}, nil
}

func requiresFilmProviderSecret(protocol model.ChannelInterfaceType) bool {
	return protocol == model.ChannelInterfaceVolcengineJiMengImage || protocol == model.ChannelInterfaceVolcengineJiMengVideo
}

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"infinite-canvas/backend/internal/model"
)

type ChannelModelsRequest struct {
	BaseURL   string           `json:"baseUrl"`
	APIKey    string           `json:"apiKey"`
	APIFormat string           `json:"apiFormat"`
	Headers   []OutboundHeader `json:"headers"`
}

type channelModelsPayload struct {
	Data   []channelModelItem `json:"data"`
	Models []channelModelItem `json:"models"`
	Error  *providerError     `json:"error"`
	Code   *int               `json:"code"`
	Msg    string             `json:"msg"`
}

type channelModelItem struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	SupportedEndpointTypes []string `json:"supported_endpoint_types"`
}

func (s *Service) FetchChannelModels(ctx context.Context, actor *model.User, input ChannelModelsRequest) ([]string, error) {
	items, err := s.FetchChannelModelCatalog(ctx, actor, input)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(items))
	models := make([]string, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.ID)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		models = append(models, name)
	}
	sort.Strings(models)
	return models, nil
}

// ChannelModelCatalogItem 是前端自定义渠道拉取模型目录后的最小合同；
// supportedEndpointTypes 由上游 /models 返回，前端据此推导能力与协议，不依赖模型名猜测。
type ChannelModelCatalogItem struct {
	ID                     string                     `json:"id"`
	DisplayName            string                     `json:"displayName,omitempty"`
	SupportedEndpointTypes []string                   `json:"supportedEndpointTypes,omitempty"`
	Capability             string                     `json:"capability,omitempty"`
	Protocol               model.ChannelInterfaceType `json:"protocol,omitempty"`
	CapabilityConfig       *ModelCapabilityConfig     `json:"capabilityConfig,omitempty"`
}

func (s *Service) FetchChannelModelCatalog(ctx context.Context, actor *model.User, input ChannelModelsRequest) ([]ChannelModelCatalogItem, error) {
	if actor == nil || strings.TrimSpace(actor.ID) == "" {
		return nil, Unauthorized("请先登录")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	apiKey := strings.TrimSpace(input.APIKey)
	if baseURL == "" {
		return nil, BadAuthRequest("请填写 Base URL")
	}
	if apiKey == "" {
		return nil, BadAuthRequest("请填写 API Key")
	}
	apiFormat := strings.ToLower(strings.TrimSpace(input.APIFormat))
	if apiFormat == "" {
		apiFormat = "openai"
	}
	if apiFormat != "openai" && apiFormat != "gemini" {
		return nil, BadAuthRequest("接口协议不支持拉取模型")
	}
	headers, err := NormalizeOutboundHeaders(input.Headers)
	if err != nil {
		return nil, err
	}

	modelsTarget := apiURL(baseURL, "/models")
	if apiFormat == "gemini" {
		if !strings.HasSuffix(strings.ToLower(baseURL), "/v1beta") {
			baseURL += "/v1beta"
		}
		modelsTarget = baseURL + "/models"
	}
	if _, err := ValidateOutboundURL(modelsTarget); err != nil {
		return nil, err
	}

	// /models is the standard OpenAI/Gemini catalog. Some video gateways expose
	// their live provider directory only from /health, so both endpoints are
	// read and merged. Either usable response is sufficient for discovery.
	modelsData, modelsErr := fetchChannelCatalogDocument(ctx, modelsTarget, apiFormat, apiKey, headers)
	var catalog []ChannelModelCatalogItem
	if modelsErr == nil {
		items, parseErr := parseChannelModelsCatalog(modelsData, apiFormat)
		if parseErr != nil {
			modelsErr = parseErr
		} else {
			catalog = mergeChannelModelCatalogItems(catalog, items)
		}
	}

	var healthErr error
	if apiFormat == "openai" {
		healthTarget := apiURL(baseURL, "/health")
		if _, err := ValidateOutboundURL(healthTarget); err != nil {
			return nil, err
		}
		healthData, fetchErr := fetchChannelCatalogDocument(ctx, healthTarget, apiFormat, apiKey, headers)
		if fetchErr != nil {
			healthErr = fetchErr
		} else if items, parseErr := parseChannelHealthVideoCatalog(healthData); parseErr != nil {
			healthErr = parseErr
		} else {
			catalog = mergeChannelModelCatalogItems(catalog, items)
		}
	}

	if modelsErr != nil && healthErr != nil {
		return nil, channelModelsUpstreamError(modelsErr)
	}
	if modelsErr != nil && apiFormat != "openai" {
		return nil, channelModelsUpstreamError(modelsErr)
	}
	return catalog, nil
}

func fetchChannelCatalogDocument(ctx context.Context, target string, apiFormat string, apiKey string, headers []OutboundHeader) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, BadAuthRequest("模型服务地址无效")
	}
	if apiFormat == "gemini" {
		request.Header.Set("x-goog-api-key", apiKey)
	} else {
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}
	ApplyOutboundHeaders(request, headers)
	data, _, err := doBinary(request)
	return data, err
}

func parseChannelModelsCatalog(data []byte, apiFormat string) ([]ChannelModelCatalogItem, error) {
	var payload channelModelsPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, &AuthError{Status: http.StatusBadGateway, Message: "模型服务返回的不是有效 JSON"}
	}
	if payload.Error != nil && strings.TrimSpace(payload.Error.Message) != "" {
		return nil, &AuthError{Status: http.StatusBadGateway, Message: payload.Error.Message}
	}
	if payload.Code != nil && *payload.Code != 0 {
		return nil, &AuthError{Status: http.StatusBadGateway, Message: firstNonEmpty(strings.TrimSpace(payload.Msg), "模型服务返回失败")}
	}

	items := payload.Data
	if apiFormat == "gemini" {
		items = payload.Models
	}
	catalog := make([]ChannelModelCatalogItem, 0, len(items))
	for _, item := range items {
		name := strings.TrimPrefix(strings.TrimSpace(firstNonEmpty(item.ID, item.Name)), "models/")
		if name == "" {
			continue
		}
		catalog = append(catalog, ChannelModelCatalogItem{
			ID:                     name,
			DisplayName:            firstNonEmpty(strings.TrimSpace(item.Name), name),
			SupportedEndpointTypes: normalizeCatalogEndpointTypes(item.SupportedEndpointTypes),
		})
	}
	return catalog, nil
}

// healthVideoProvider carries only runtime capability facts. In particular,
// upstream pointsCost is intentionally not modeled: it is never a user price.
type healthVideoProvider struct {
	ID                 string
	DisplayName        string
	AllowedDurations   []int
	DefaultDuration    int
	AllowedRatios      []string
	DefaultRatio       string
	AllowedResolutions []string
	DefaultResolution  string
}

func parseChannelHealthVideoCatalog(data []byte) ([]ChannelModelCatalogItem, error) {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, &AuthError{Status: http.StatusBadGateway, Message: "模型服务返回的不是有效 JSON"}
	}
	if message := channelCatalogPayloadError(payload); message != "" {
		return nil, &AuthError{Status: http.StatusBadGateway, Message: message}
	}
	providers := make(map[string]healthVideoProvider)
	collectHealthVideoProviders(payload, providers, 0)
	catalog := make([]ChannelModelCatalogItem, 0, len(providers))
	for _, provider := range providers {
		if provider.ID == "" {
			continue
		}
		catalog = append(catalog, ChannelModelCatalogItem{
			ID:               provider.ID,
			DisplayName:      firstNonEmpty(provider.DisplayName, provider.ID),
			Capability:       "video",
			Protocol:         model.ChannelInterfaceVolcengineArkVideo,
			CapabilityConfig: videoCapabilityConfigFromHealth(provider),
		})
	}
	return catalog, nil
}

func channelCatalogPayloadError(payload map[string]any) string {
	if errorValue, ok := payload["error"].(map[string]any); ok {
		if message := catalogString(errorValue, "message", "msg"); message != "" {
			return message
		}
	}
	if code, ok := catalogPositiveInt(payload["code"]); ok && code != 0 {
		return firstNonEmpty(catalogString(payload, "msg", "message"), "模型服务返回失败")
	}
	return ""
}

func collectHealthVideoProviders(value any, providers map[string]healthVideoProvider, depth int) {
	if depth > 6 {
		return
	}
	object, ok := value.(map[string]any)
	if !ok {
		return
	}
	for key, nested := range object {
		if strings.EqualFold(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""), "videoproviders") {
			collectHealthVideoProviderCollection(nested, providers)
		}
	}
	for _, nested := range object {
		if child, ok := nested.(map[string]any); ok {
			collectHealthVideoProviders(child, providers, depth+1)
		}
	}
}

func collectHealthVideoProviderCollection(value any, providers map[string]healthVideoProvider) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if id, ok := item.(string); ok {
				addHealthVideoProvider(id, nil, providers)
				continue
			}
			addHealthVideoProvider("", item, providers)
		}
	case map[string]any:
		if looksLikeHealthVideoProvider(typed) {
			addHealthVideoProvider("", typed, providers)
			return
		}
		for key, item := range typed {
			if key == "items" || key == "data" || key == "providers" || key == "models" {
				collectHealthVideoProviderCollection(item, providers)
				continue
			}
			addHealthVideoProvider(key, item, providers)
		}
	case string:
		addHealthVideoProvider(typed, nil, providers)
	}
}

func looksLikeHealthVideoProvider(value map[string]any) bool {
	for _, key := range []string{"id", "provider", "providerId", "allowedDurations", "allowed_durations", "allowedRatios", "allowed_ratios", "resolution"} {
		if _, exists := value[key]; exists {
			return true
		}
	}
	return false
}

func addHealthVideoProvider(fallbackID string, value any, providers map[string]healthVideoProvider) {
	definition, _ := value.(map[string]any)
	id := strings.TrimPrefix(strings.TrimSpace(firstNonEmpty(catalogString(definition, "id", "key", "provider", "providerId", "provider_id", "model", "modelId"), fallbackID)), "models/")
	if id == "" {
		return
	}
	provider := healthVideoProvider{
		ID:                 id,
		DisplayName:        catalogString(definition, "displayName", "display_name", "label", "title", "name"),
		AllowedDurations:   catalogPositiveInts(firstCatalogValue(definition, "allowedDurations", "allowed_durations", "durations", "supportedDurations", "supported_durations")),
		DefaultDuration:    catalogInteger(firstCatalogValue(definition, "defaultDuration", "default_duration", "duration")),
		AllowedRatios:      catalogStrings(firstCatalogValue(definition, "allowedRatios", "allowed_ratios", "ratios", "supportedRatios", "supported_ratios")),
		DefaultRatio:       catalogString(definition, "defaultRatio", "default_ratio", "ratio", "aspectRatio", "aspect_ratio"),
		AllowedResolutions: catalogResolutions(firstCatalogValue(definition, "allowedResolutions", "allowed_resolutions", "resolutions", "supportedResolutions", "supported_resolutions", "resolution")),
		DefaultResolution:  catalogResolution(catalogString(definition, "defaultResolution", "default_resolution", "resolution")),
	}
	if existing, exists := providers[id]; exists {
		provider = mergeHealthVideoProvider(existing, provider)
	}
	providers[id] = provider
}

func mergeHealthVideoProvider(existing healthVideoProvider, next healthVideoProvider) healthVideoProvider {
	if next.DisplayName == "" {
		next.DisplayName = existing.DisplayName
	}
	if len(next.AllowedDurations) == 0 {
		next.AllowedDurations = existing.AllowedDurations
	}
	if next.DefaultDuration == 0 {
		next.DefaultDuration = existing.DefaultDuration
	}
	if len(next.AllowedRatios) == 0 {
		next.AllowedRatios = existing.AllowedRatios
	}
	if next.DefaultRatio == "" {
		next.DefaultRatio = existing.DefaultRatio
	}
	if len(next.AllowedResolutions) == 0 {
		next.AllowedResolutions = existing.AllowedResolutions
	}
	if next.DefaultResolution == "" {
		next.DefaultResolution = existing.DefaultResolution
	}
	return next
}

func videoCapabilityConfigFromHealth(provider healthVideoProvider) *ModelCapabilityConfig {
	profile := DefaultModelCapabilityConfigForModel(string(model.ChannelInterfaceVolcengineArkVideo), provider.ID).Video
	if len(provider.AllowedDurations) > 0 {
		profile.Duration = VideoDurationConfig{Selection: "enum", Values: append([]int(nil), provider.AllowedDurations...), Default: provider.AllowedDurations[0]}
		if containsCapabilityInt(profile.Duration.Values, provider.DefaultDuration) {
			profile.Duration.Default = provider.DefaultDuration
		}
	}
	if len(provider.AllowedRatios) > 0 {
		profile.Ratios = append([]string(nil), provider.AllowedRatios...)
		profile.DefaultRatio = preferredCatalogValue(profile.Ratios, provider.DefaultRatio, "16:9")
	}
	if len(provider.AllowedResolutions) > 0 {
		profile.Resolutions = append([]string(nil), provider.AllowedResolutions...)
		profile.DefaultResolution = preferredCatalogValue(profile.Resolutions, provider.DefaultResolution, "720p")
	}
	return &ModelCapabilityConfig{Version: 1, Video: profile}
}

func mergeChannelModelCatalogItems(current []ChannelModelCatalogItem, additions []ChannelModelCatalogItem) []ChannelModelCatalogItem {
	byID := make(map[string]int, len(current)+len(additions))
	for index := range current {
		byID[current[index].ID] = index
	}
	for _, item := range additions {
		item.ID = strings.TrimPrefix(strings.TrimSpace(item.ID), "models/")
		if item.ID == "" {
			continue
		}
		if index, exists := byID[item.ID]; exists {
			current[index] = mergeChannelModelCatalogItem(current[index], item)
			continue
		}
		byID[item.ID] = len(current)
		current = append(current, item)
	}
	sort.Slice(current, func(left int, right int) bool { return current[left].ID < current[right].ID })
	return current
}

func mergeChannelModelCatalogItem(existing ChannelModelCatalogItem, next ChannelModelCatalogItem) ChannelModelCatalogItem {
	if next.DisplayName != "" {
		existing.DisplayName = next.DisplayName
	}
	existing.SupportedEndpointTypes = normalizeCatalogEndpointTypes(append(existing.SupportedEndpointTypes, next.SupportedEndpointTypes...))
	if next.Capability != "" {
		existing.Capability = next.Capability
		existing.Protocol = next.Protocol
		existing.CapabilityConfig = next.CapabilityConfig
	}
	return existing
}

func firstCatalogValue(value map[string]any, keys ...string) any {
	for _, key := range keys {
		if item, exists := value[key]; exists {
			return item
		}
	}
	return nil
}

func catalogString(value map[string]any, keys ...string) string {
	for _, key := range keys {
		if item, ok := value[key].(string); ok && strings.TrimSpace(item) != "" {
			return strings.TrimSpace(item)
		}
	}
	return ""
}

func catalogStrings(value any) []string {
	var values []string
	switch typed := value.(type) {
	case string:
		values = strings.Split(typed, ",")
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				values = append(values, text)
			}
		}
	}
	return normalizeCatalogEndpointTypes(values)
}

func catalogPositiveInts(value any) []int {
	values := make([]int, 0)
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if number := catalogInteger(item); number > 0 {
				values = append(values, number)
			}
		}
	case string:
		for _, item := range strings.Split(typed, ",") {
			if number := catalogInteger(item); number > 0 {
				values = append(values, number)
			}
		}
	}
	sort.Ints(values)
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}

func catalogResolutions(value any) []string {
	values := catalogStrings(value)
	for index := range values {
		values[index] = catalogResolution(values[index])
	}
	return normalizeCatalogEndpointTypes(values)
}

func catalogResolution(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	if _, err := strconv.Atoi(value); err == nil {
		return value + "p"
	}
	return value
}

func catalogInteger(value any) int {
	switch typed := value.(type) {
	case float64:
		if typed > 0 && typed == float64(int(typed)) {
			return int(typed)
		}
	case int:
		return typed
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return 0
}

func catalogPositiveInt(value any) (int, bool) {
	if number := catalogInteger(value); number != 0 {
		return number, true
	}
	return 0, false
}

func containsCapabilityInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func preferredCatalogValue(values []string, preferred string, fallback string) string {
	for _, candidate := range []string{preferred, fallback, values[0]} {
		if containsCapabilityString(values, candidate) {
			return candidate
		}
	}
	return values[0]
}

func normalizeCatalogEndpointTypes(values []string) []string {
	seen := make(map[string]bool, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		item := strings.TrimSpace(value)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		normalized = append(normalized, item)
	}
	return normalized
}

func channelModelsUpstreamError(err error) error {
	var authErr *AuthError
	if errors.As(err, &authErr) {
		return authErr
	}
	var httpErr providerHTTPError
	if !errors.As(err, &httpErr) {
		return &AuthError{Status: http.StatusBadGateway, Message: "连接模型服务失败：" + err.Error()}
	}
	switch httpErr.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &AuthError{Status: http.StatusBadGateway, Message: "模型服务鉴权失败，请检查 API Key"}
	case http.StatusNotFound:
		return &AuthError{Status: http.StatusBadGateway, Message: "模型服务未提供 /models 接口"}
	case http.StatusTooManyRequests:
		return &AuthError{Status: http.StatusBadGateway, Message: "模型服务请求过于频繁或额度不足"}
	default:
		return &AuthError{Status: http.StatusBadGateway, Message: fmt.Sprintf("模型服务请求失败：HTTP %d", httpErr.StatusCode)}
	}
}

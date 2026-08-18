package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"infinite-canvas/backend/internal/model"
)

func TestFetchAdminChannelModelsImportsHealthVideoProvidersWithoutModelsEndpoint(t *testing.T) {
	t.Setenv("CANVAS_ALLOW_PRIVATE_UPSTREAMS", "true")
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/aggc/api/v3/models":
			http.NotFound(writer, request)
		case "/aggc/api/v3/health":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{
				"defaults": {
					"videoProviders": {
						"aggc": {
							"displayName": "AGGC",
							"allowedDurations": [5, 10],
							"defaultDuration": 10,
							"allowedRatios": ["9:16", "16:9"],
							"defaultRatio": "9:16",
							"allowedResolutions": ["720", "1080"],
							"defaultResolution": "1080",
							"pointsCost": 0.3
						},
						"aggc_1080": {
							"allowedDurations": [5, 8],
							"defaultDuration": 8,
							"allowedRatios": ["16:9"],
							"allowedResolutions": ["1080"],
							"defaultResolution": "1080",
							"pointsCost": 0.5
						}
					}
				}
			}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer upstream.Close()

	svc, db := newChannelModelTestService(t)
	admin := &model.User{ID: "admin", Role: model.UserRoleAdmin}
	channel := model.ModelChannel{
		ID: "health-video-channel", UserID: admin.ID, Scope: model.ChannelScopeSystem, Enabled: true,
		Name: "Video Gateway", BaseURL: upstream.URL + "/aggc/api/v3", APIKey: "test-key", APIFormat: "openai", ModelsJSON: "[]",
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatal(err)
	}

	result, err := svc.FetchAdminChannelModels(context.Background(), admin, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Models, []string{"aggc", "aggc_1080"}) || result.Added != 2 {
		t.Fatalf("FetchAdminChannelModels() = %#v, want two health providers", result)
	}

	var imported []model.ChannelModel
	if err := db.Where("channel_id = ?", channel.ID).Order("model_key asc").Find(&imported).Error; err != nil {
		t.Fatal(err)
	}
	if len(imported) != 2 {
		t.Fatalf("imported channel models = %#v", imported)
	}
	for _, item := range imported {
		if item.Capability != "video" || item.Protocol != model.ChannelInterfaceVolcengineArkVideo || item.BillingMode != "fixed_request" {
			t.Fatalf("catalog contract = %#v", item)
		}
		if item.Enabled || item.PriceConfigured || item.UnitPriceMicrocredits != 0 || item.InputTokenPriceMicrocredits != 0 || item.OutputTokenPriceMicrocredits != 0 || item.CachedTokenPriceMicrocredits != 0 {
			t.Fatalf("discovered model must remain disabled and unpriced: %#v", item)
		}
		if strings.Contains(item.CapabilityConfigJSON, "pointsCost") {
			t.Fatalf("upstream pointsCost must not become customer pricing: %s", item.CapabilityConfigJSON)
		}
	}
	assertImportedHealthVideoCapability(t, imported[0], []int{5, 10}, "9:16", []string{"9:16", "16:9"}, "1080p", []string{"720p", "1080p"})
	assertImportedHealthVideoCapability(t, imported[1], []int{5, 8}, "16:9", []string{"16:9"}, "1080p", []string{"1080p"})

	var preserved model.ChannelModel
	if err := db.First(&preserved, "channel_id = ? AND model_key = ?", channel.ID, "aggc").Error; err != nil {
		t.Fatal(err)
	}
	preserved.Enabled = true
	preserved.PriceConfigured = true
	preserved.BillingMode = "per_second"
	preserved.UnitPriceMicrocredits = 300000
	preserved.CapabilityVersion = 42
	preserved.CapabilityConfigJSON = `{"version":1,"video":{"duration":{"selection":"enum","values":[42],"default":42}}}`
	if err := db.Save(&preserved).Error; err != nil {
		t.Fatal(err)
	}

	result, err = svc.FetchAdminChannelModels(context.Background(), admin, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 0 {
		t.Fatalf("second catalog fetch added %d models, want 0", result.Added)
	}
	var reloaded model.ChannelModel
	if err := db.First(&reloaded, "id = ?", preserved.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !reloaded.Enabled || !reloaded.PriceConfigured || reloaded.BillingMode != "per_second" || reloaded.UnitPriceMicrocredits != 300000 || reloaded.CapabilityVersion != 42 || reloaded.CapabilityConfigJSON != preserved.CapabilityConfigJSON {
		t.Fatalf("existing administrator configuration was overwritten: %#v", reloaded)
	}
}

func assertImportedHealthVideoCapability(t *testing.T, item model.ChannelModel, durations []int, defaultRatio string, ratios []string, defaultResolution string, resolutions []string) {
	t.Helper()
	var capability ModelCapabilityConfig
	if err := json.Unmarshal([]byte(item.CapabilityConfigJSON), &capability); err != nil {
		t.Fatal(err)
	}
	if capability.Video == nil {
		t.Fatalf("%s does not have a video capability configuration", item.ModelKey)
	}
	video := capability.Video
	if video.Duration.Selection != "enum" || !reflect.DeepEqual(video.Duration.Values, durations) || video.Duration.Default != durations[len(durations)-1] {
		t.Fatalf("%s duration capability = %#v", item.ModelKey, video.Duration)
	}
	if video.DefaultRatio != defaultRatio || !reflect.DeepEqual(video.Ratios, ratios) || video.DefaultResolution != defaultResolution || !reflect.DeepEqual(video.Resolutions, resolutions) {
		t.Fatalf("%s video capability = %#v", item.ModelKey, video)
	}
}

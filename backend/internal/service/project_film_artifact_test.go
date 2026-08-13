package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestValidateFilmArtifactPayloadKeepsActingInItsDomain(t *testing.T) {
	if err := validateFilmArtifactPayload(FilmArtifactTypeActing, map[string]any{
		"objective": "conceal mistake",
		"obstacle":  "XiaoPi questions him",
		"tactic":    "dismiss then redirect",
		"beat":      "4",
	}); err != nil {
		t.Fatal(err)
	}
	if err := validateFilmArtifactPayload(FilmArtifactTypeActing, map[string]any{"camera": "slow push"}); err == nil {
		t.Fatal("acting must not be allowed to override camera facts")
	}
	if err := validateFilmArtifactPayload("unknown", map[string]any{}); err == nil {
		t.Fatal("unknown artifact types must be rejected")
	}
}

func TestValidateGenerationRequestPayloadRequiresCompiledPromptAndRejectsSecrets(t *testing.T) {
	valid := map[string]any{
		"mediaType":        "video",
		"promptArtifactId": "prompt-pack-v2",
		"compiledPrompt":   "A tense close-up in the clinic.",
		"aspectRatio":      "9:16",
		"durationMs":       float64(5000),
		"outputIntent":     "首轮镜头生成",
	}
	if err := validateFilmArtifactPayload(FilmArtifactTypeGenerationRequest, valid); err != nil {
		t.Fatal(err)
	}
	missingPrompt := map[string]any{}
	for key, value := range valid {
		missingPrompt[key] = value
	}
	missingPrompt["compiledPrompt"] = ""
	if err := validateFilmArtifactPayload(FilmArtifactTypeGenerationRequest, missingPrompt); err == nil {
		t.Fatal("generation request must require a compiled prompt snapshot")
	}
	withSecret := map[string]any{}
	for key, value := range valid {
		withSecret[key] = value
	}
	withSecret["apiKey"] = "must-not-be-persisted"
	if err := validateFilmArtifactPayload(FilmArtifactTypeGenerationRequest, withSecret); err == nil {
		t.Fatal("generation request must reject provider credentials")
	}
	withNestedToken := map[string]any{}
	for key, value := range valid {
		withNestedToken[key] = value
	}
	withNestedToken["provider"] = map[string]any{"bearerToken": "must-not-be-persisted"}
	if err := validateFilmArtifactPayload(FilmArtifactTypeGenerationRequest, withNestedToken); err == nil {
		t.Fatal("generation request must reject nested provider tokens")
	}
	withUnownedExecutionField := map[string]any{}
	for key, value := range valid {
		withUnownedExecutionField[key] = value
	}
	withUnownedExecutionField["model"] = "browser-must-not-select-provider-model"
	if err := validateFilmArtifactPayload(FilmArtifactTypeGenerationRequest, withUnownedExecutionField); err == nil {
		t.Fatal("generation request must not persist browser-owned provider execution fields")
	}
}

func TestSaveGenerationRequestFreezesTheCurrentPromptPackSnapshot(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.Shot{}, &model.FilmArtifact{}, &model.ShotAssetReference{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	project := model.Project{ID: "project-1", UserID: "user-1", Name: "寄生广告", Type: model.ProjectTypeShortDrama, AspectRatio: "9:16", Status: model.ProjectStatusActive, CreatedAt: now, UpdatedAt: now}
	shot := model.Shot{ID: "shot-1", ProjectID: project.ID, Title: "SC01-SH001", DurationMs: 5000, Status: "draft", ContractArtifactID: "contract-1", CreatedAt: now, UpdatedAt: now}
	promptPack := model.FilmArtifact{
		ID: "prompt-pack-v2", ProjectID: project.ID, ShotID: shot.ID, ArtifactType: FilmArtifactTypeVideoPromptPack, ObjectVersion: 2, Status: "ready",
		PayloadJSON: `{"compiledPrompt":"A tense close-up in the clinic."}`, CreatedAt: now, UpdatedAt: now,
	}
	contract := model.FilmArtifact{ID: "contract-1", ProjectID: project.ID, ShotID: shot.ID, ArtifactType: "shot_contract", ObjectVersion: 1, Status: "ready", PayloadJSON: `{}`, CreatedAt: now, UpdatedAt: now}
	assetReference := model.ShotAssetReference{ID: "reference-1", ShotID: shot.ID, AssetVersionID: "asset-version-1", Role: "character", Status: "active", CreatedAt: now}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&shot).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&contract).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&promptPack).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&assetReference).Error; err != nil {
		t.Fatal(err)
	}

	svc := New(repository.New(db), t.TempDir())
	artifact, err := svc.SaveProjectFilmArtifact(project.UserID, project.ID, SaveProjectFilmArtifactRequest{
		ShotID: shot.ID, ArtifactType: FilmArtifactTypeGenerationRequest, Payload: map[string]any{
			"promptArtifactId": promptPack.ID,
			"compiledPrompt":   "client content must be ignored",
			"mediaType":        "video",
			"aspectRatio":      "9:16",
			"durationMs":       float64(5000),
			"outputIntent":     "首轮镜头生成",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.ArtifactType != FilmArtifactTypeGenerationRequest || artifact.ObjectVersion != 1 {
		t.Fatalf("unexpected generation request identity: %#v", artifact)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(artifact.PayloadJSON), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["compiledPrompt"] != "A tense close-up in the clinic." || payload["promptArtifactId"] != promptPack.ID || payload["promptArtifactVersion"] != float64(promptPack.ObjectVersion) {
		t.Fatalf("generation request did not freeze the authoritative prompt snapshot: %#v", payload)
	}
	var sourceRefs []string
	if err := json.Unmarshal([]byte(artifact.SourceRefsJSON), &sourceRefs); err != nil {
		t.Fatal(err)
	}
	if !hasFilmArtifactRef(sourceRefs, contract.ID) || !hasFilmArtifactRef(sourceRefs, promptPack.ID) || !hasFilmArtifactRef(sourceRefs, assetReference.AssetVersionID) {
		t.Fatalf("generation request source refs are incomplete: %#v", sourceRefs)
	}
}

func TestFilmGenerationTaskDraftUsesExactRequestVersionWithoutCreatingTask(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.Shot{}, &model.FilmArtifact{}, &model.Task{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	project := model.Project{ID: "project-draft", UserID: "user-draft", Name: "寄生广告", Type: model.ProjectTypeShortDrama, AspectRatio: "9:16", Status: model.ProjectStatusActive, CreatedAt: now, UpdatedAt: now}
	shot := model.Shot{ID: "shot-draft", ProjectID: project.ID, Title: "SC01-SH001", DurationMs: 5000, Status: "ready", CreatedAt: now, UpdatedAt: now}
	promptPackV1 := model.FilmArtifact{ID: "prompt-pack-v1", ProjectID: project.ID, ShotID: shot.ID, ArtifactType: FilmArtifactTypeVideoPromptPack, ObjectVersion: 1, Status: "ready", PayloadJSON: `{"compiledPrompt":"Frozen prompt v1"}`, CreatedAt: now, UpdatedAt: now}
	requestV1 := model.FilmArtifact{
		ID: "generation-request-v1", ProjectID: project.ID, ShotID: shot.ID, ArtifactType: FilmArtifactTypeGenerationRequest, ObjectVersion: 1, Status: "ready",
		PayloadJSON: `{"promptArtifactId":"prompt-pack-v1","promptArtifactVersion":1,"compiledPrompt":"Frozen prompt v1","mediaType":"video","aspectRatio":"9:16","durationMs":5000,"outputIntent":"首轮镜头生成"}`,
		SourceRefsJSON: `["shot-contract-v1","prompt-pack-v1","asset-version-v1"]`, CreatedAt: now, UpdatedAt: now,
	}
	requestV2 := model.FilmArtifact{
		ID: "generation-request-v2", ProjectID: project.ID, ShotID: shot.ID, ArtifactType: FilmArtifactTypeGenerationRequest, ObjectVersion: 2, Status: "draft",
		PayloadJSON: `{"promptArtifactId":"prompt-pack-v2","promptArtifactVersion":2,"compiledPrompt":"Newer prompt must not replace v1","mediaType":"video","aspectRatio":"9:16","durationMs":6000,"outputIntent":"第二次生成"}`,
		SourceRefsJSON: `["shot-contract-v1","prompt-pack-v2"]`, CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second),
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&shot).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&promptPackV1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&requestV1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&requestV2).Error; err != nil {
		t.Fatal(err)
	}

	svc := New(repository.New(db), t.TempDir())
	draft, err := svc.FilmGenerationTaskDraft(project.UserID, project.ID, requestV1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if draft.GenerationRequestArtifactID != requestV1.ID || draft.GenerationRequestArtifactVersion != requestV1.ObjectVersion {
		t.Fatalf("task draft did not use the exact requested version: %#v", draft)
	}
	if draft.TaskType != "canvas_video" || draft.Operation != "film_generation" || draft.GatewayInput.Prompt != "Frozen prompt v1" || draft.GatewayInput.DurationMS != 5000 {
		t.Fatalf("unexpected provider-independent task contract: %#v", draft)
	}
	if !draft.RequestReady || draft.ProviderRouteResolved || draft.SubmissionAllowed || draft.SubmissionState != FilmGenerationTaskDraftStateAwaitingProviderRoute {
		t.Fatalf("unexpected task readiness state: %#v", draft)
	}
	if !hasFilmArtifactRef(draft.GatewayInput.SourceRefs, "shot-contract-v1") || !hasFilmArtifactRef(draft.GatewayInput.SourceRefs, "prompt-pack-v1") || !hasFilmArtifactRef(draft.GatewayInput.SourceRefs, "asset-version-v1") || draft.RequestFingerprint == "" {
		t.Fatalf("task draft provenance or fingerprint is incomplete: %#v", draft)
	}
	serialized, err := json.Marshal(draft)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(serialized), "apiKey") || strings.Contains(string(serialized), "secretKey") || strings.Contains(string(serialized), "baseUrl") {
		t.Fatalf("task draft must not expose provider credentials or routing: %s", serialized)
	}
	var taskCount int64
	if err := db.Model(&model.Task{}).Count(&taskCount).Error; err != nil {
		t.Fatal(err)
	}
	if taskCount != 0 {
		t.Fatalf("reading a film task draft must not create a queued Task, got %d", taskCount)
	}
}

func TestFilmGenerationProviderRoutesExposeOnlyRedactedSystemCandidates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.Shot{}, &model.FilmArtifact{}, &model.Task{}, &model.ModelChannel{}, &model.ChannelModel{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	project := model.Project{ID: "project-routes", UserID: "user-routes", Name: "寄生广告", Type: model.ProjectTypeShortDrama, AspectRatio: "9:16", Status: model.ProjectStatusActive, CreatedAt: now, UpdatedAt: now}
	shot := model.Shot{ID: "shot-routes", ProjectID: project.ID, Title: "SC01-SH001", DurationMs: 5000, Status: "ready", CreatedAt: now, UpdatedAt: now}
	promptPack := model.FilmArtifact{ID: "prompt-pack-routes", ProjectID: project.ID, ShotID: shot.ID, ArtifactType: FilmArtifactTypeVideoPromptPack, ObjectVersion: 1, Status: "ready", PayloadJSON: `{"compiledPrompt":"Frozen route prompt"}`, CreatedAt: now, UpdatedAt: now}
	request := model.FilmArtifact{ID: "generation-request-routes", ProjectID: project.ID, ShotID: shot.ID, ArtifactType: FilmArtifactTypeGenerationRequest, ObjectVersion: 1, Status: "ready", PayloadJSON: `{"promptArtifactId":"prompt-pack-routes","promptArtifactVersion":1,"compiledPrompt":"Frozen route prompt","mediaType":"video","aspectRatio":"9:16","durationMs":5000,"outputIntent":"首轮镜头生成"}`, SourceRefsJSON: `["prompt-pack-routes"]`, CreatedAt: now, UpdatedAt: now}
	channel := model.ModelChannel{ID: "channel-routes", Scope: model.ChannelScopeSystem, Enabled: true, Name: "受控视频渠道", BaseURL: "https://private-provider.example/v1", APIKey: "server-only-api-key", APIFormat: "openai", ModelsJSON: `["video-model"]`, CreatedAt: now, UpdatedAt: now}
	videoModel := model.ChannelModel{ID: "model-routes", ChannelID: channel.ID, ModelKey: "video-model", DisplayName: "受控视频模型", Capability: "video", Protocol: model.ChannelInterfaceNewAPIVideo, BillingMode: "per_second", UnitPriceMicrocredits: 100_000, PriceConfigured: true, Enabled: true, CapabilityVersion: 3, CreatedAt: now, UpdatedAt: now}
	for _, item := range []any{&project, &shot, &promptPack, &request, &channel, &videoModel} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}

	svc := New(repository.New(db), t.TempDir())
	catalog, err := svc.FilmGenerationProviderRoutes(project.UserID, project.ID, request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.State != FilmGenerationProviderRouteStateRoutesAvailable || !catalog.RequestReady || len(catalog.Routes) != 1 {
		t.Fatalf("unexpected provider route catalog: %#v", catalog)
	}
	route := catalog.Routes[0]
	if !route.RouteReady || !route.ProviderReady || !route.BillingReady || route.ChannelID != channel.ID || route.Model != videoModel.ModelKey || route.Protocol != string(model.ChannelInterfaceNewAPIVideo) {
		t.Fatalf("unexpected provider route: %#v", route)
	}
	serialized, err := json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private-provider.example", "server-only-api-key", "baseUrl", "apiKey", "secretKey", "headers"} {
		if strings.Contains(string(serialized), forbidden) {
			t.Fatalf("provider route catalog leaked %q: %s", forbidden, serialized)
		}
	}
	var taskCount int64
	if err := db.Model(&model.Task{}).Count(&taskCount).Error; err != nil {
		t.Fatal(err)
	}
	if taskCount != 0 {
		t.Fatalf("provider route discovery must not create a Task, got %d", taskCount)
	}
}

func hasFilmArtifactRef(refs []string, wanted string) bool {
	for _, ref := range refs {
		if ref == wanted {
			return true
		}
	}
	return false
}

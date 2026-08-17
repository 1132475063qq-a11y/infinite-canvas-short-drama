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

func TestScenePromptPackCompilesExactLockedSpatialProjection(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.Scene{}, &model.Shot{}, &model.FilmArtifact{}, &model.ShotAssetReference{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	project := model.Project{ID: "project-prompt-spatial", UserID: "user-prompt-spatial", Name: "寄生广告", Type: model.ProjectTypeShortDrama, AspectRatio: "9:16", Status: model.ProjectStatusActive, CreatedAt: now, UpdatedAt: now}
	scene := model.Scene{ID: "scene-clinic-prompt", ProjectID: project.ID, UnitID: "unit-clinic", Title: "黑市诊所", Status: "ready", CreatedAt: now, UpdatedAt: now}
	shot := model.Shot{ID: "shot-clinic-prompt", ProjectID: project.ID, UnitID: scene.UnitID, SceneID: scene.ID, Title: "SC05-SH001", DurationMs: 5000, Status: "ready", CreatedAt: now, UpdatedAt: now}
	for _, item := range []any{&project, &scene, &shot} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := New(repository.New(db), t.TempDir())
	pack := blackMarketClinicSpatialPack()
	pack.SceneID = scene.ID
	pack.SceneManifest.SceneID = scene.ID
	bindSpatialPackScene(&pack, scene.ID)
	packPayload := mustSpatialPayload(t, pack)
	savedPack, err := svc.SaveProjectSceneAssetPack(project.UserID, project.ID, SaveProjectSceneAssetPackRequest{SceneID: scene.ID, Status: "ready", ExpectedVersion: intPointer(0), Payload: packPayload})
	if err != nil {
		t.Fatal(err)
	}
	if savedPack.Gate.Status != SceneSpatialGatePass {
		t.Fatalf("fixture must pass before prompt compilation: %#v", savedPack.Gate)
	}

	prompt, err := svc.SaveProjectFilmArtifact(project.UserID, project.ID, SaveProjectFilmArtifactRequest{
		ShotID: shot.ID, ArtifactType: FilmArtifactTypeVideoPromptPack, Status: "ready",
		Payload: map[string]any{
			"creativePrompt":       "小皮从操作台抬头看向富豪，保持二维硬边阴影。",
			"compiledPrompt":       "浏览器伪造的最终提示词",
			"viewId":               "V01",
			"cameraAnchorId":       "C1",
			"lockedSceneAssetPack": map[string]any{"sceneId": "attacker-scene"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var promptPayload map[string]any
	if err := json.Unmarshal([]byte(prompt.PayloadJSON), &promptPayload); err != nil {
		t.Fatal(err)
	}
	compiled := payloadString(promptPayload, "compiledPrompt")
	if !strings.Contains(compiled, lockedSceneAssetPackMarker) || strings.Contains(compiled, "浏览器伪造") || !strings.Contains(compiled, "clinic-door-anchor") {
		t.Fatalf("Prompt Pack was not compiled from the locked spatial projection: %s", compiled)
	}
	if payloadString(promptPayload, "spatialPackArtifactId") != savedPack.PackArtifact.ID || payloadString(promptPayload, "spatialGateArtifactId") != savedPack.GateArtifact.ID || payloadString(promptPayload, "viewId") != "V01" || payloadString(promptPayload, "cameraAnchorId") != "C1" {
		t.Fatalf("Prompt Pack spatial provenance is not exact: %#v", promptPayload)
	}
	projection, projectionErr := decodeLockedSceneAssetPackProjection(promptPayload)
	if projectionErr != nil || projection.ViewAsset == nil || projection.ViewAsset.AssetID != "clinic-master" {
		t.Fatalf("Prompt Pack must expose the master scene asset when it satisfies V01: projection=%#v error=%v", projection, projectionErr)
	}
	var promptRefs []string
	if err := json.Unmarshal([]byte(prompt.SourceRefsJSON), &promptRefs); err != nil {
		t.Fatal(err)
	}
	if !filmSourceRefExists(promptRefs, savedPack.PackArtifact.ID) || !filmSourceRefExists(promptRefs, savedPack.GateArtifact.ID) {
		t.Fatalf("Prompt Pack source refs omitted exact scene versions: %#v", promptRefs)
	}

	request, err := svc.SaveProjectFilmArtifact(project.UserID, project.ID, SaveProjectFilmArtifactRequest{
		ShotID: shot.ID, ArtifactType: FilmArtifactTypeGenerationRequest, Status: "ready",
		Payload: map[string]any{
			"promptArtifactId": prompt.ID,
			"compiledPrompt":   "another forged prompt",
			"mediaType":        "video",
			"aspectRatio":      "9:16",
			"durationMs":       float64(5000),
			"outputIntent":     "黑市诊所连续性测试",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var requestPayload map[string]any
	if err := json.Unmarshal([]byte(request.PayloadJSON), &requestPayload); err != nil {
		t.Fatal(err)
	}
	if payloadString(requestPayload, "compiledPrompt") != compiled || payloadString(requestPayload, "spatialPackArtifactId") != savedPack.PackArtifact.ID || payloadString(requestPayload, "spatialGateArtifactId") != savedPack.GateArtifact.ID {
		t.Fatalf("Generation Request did not freeze Prompt Pack spatial snapshot: %#v", requestPayload)
	}

	draft, err := svc.FilmGenerationTaskDraft(project.UserID, project.ID, request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !draft.RequestReady || !draft.SpatialGateReady || draft.GatewayInput.SpatialContext == nil || draft.GatewayInput.ViewID != "V01" {
		t.Fatalf("PASS scene prompt should reach the provider-independent draft: %#v", draft)
	}

	drifted := blackMarketClinicSpatialPack()
	drifted.SceneID = scene.ID
	drifted.SceneManifest.SceneID = scene.ID
	bindSpatialPackScene(&drifted, scene.ID)
	drifted.FixedAnchors[0].Position.X += 1
	if _, err := svc.SaveProjectSceneAssetPack(project.UserID, project.ID, SaveProjectSceneAssetPackRequest{SceneID: scene.ID, Status: "draft", ExpectedVersion: intPointer(1), Payload: mustSpatialPayload(t, drifted)}); err != nil {
		t.Fatal(err)
	}
	draft, err = svc.FilmGenerationTaskDraft(project.UserID, project.ID, request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if draft.RequestReady || draft.SpatialGateReady {
		t.Fatalf("a changed scene pack must block the old spatial Prompt Pack: %#v", draft)
	}
}

func TestScenePromptPackRejectsAmbiguousViewSelection(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.Scene{}, &model.Shot{}, &model.FilmArtifact{}, &model.ShotAssetReference{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	project := model.Project{ID: "project-prompt-ambiguous", UserID: "user-prompt-ambiguous", Name: "寄生广告", Type: model.ProjectTypeShortDrama, AspectRatio: "9:16", Status: model.ProjectStatusActive, CreatedAt: now, UpdatedAt: now}
	scene := model.Scene{ID: "scene-ambiguous", ProjectID: project.ID, UnitID: "unit-ambiguous", Title: "黑市诊所", Status: "ready", CreatedAt: now, UpdatedAt: now}
	shot := model.Shot{ID: "shot-ambiguous", ProjectID: project.ID, UnitID: scene.UnitID, SceneID: scene.ID, Title: "SC05-SH001", DurationMs: 5000, Status: "ready", CreatedAt: now, UpdatedAt: now}
	for _, item := range []any{&project, &scene, &shot} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := New(repository.New(db), t.TempDir())
	pack := blackMarketClinicSpatialPack()
	pack.SceneID = scene.ID
	pack.SceneManifest.SceneID = scene.ID
	bindSpatialPackScene(&pack, scene.ID)
	if _, err := svc.SaveProjectSceneAssetPack(project.UserID, project.ID, SaveProjectSceneAssetPackRequest{SceneID: scene.ID, Status: "ready", ExpectedVersion: intPointer(0), Payload: mustSpatialPayload(t, pack)}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SaveProjectFilmArtifact(project.UserID, project.ID, SaveProjectFilmArtifactRequest{ShotID: shot.ID, ArtifactType: FilmArtifactTypeVideoPromptPack, Payload: map[string]any{"creativePrompt": "没有指定机位"}}); err == nil {
		t.Fatal("a scene-linked Prompt Pack must not guess among multiple views")
	}
}

func TestScenePromptPackAllowsUncompiledDraftBeforeSpatialGate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.Scene{}, &model.Shot{}, &model.FilmArtifact{}, &model.ShotAssetReference{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	project := model.Project{ID: "project-prompt-draft", UserID: "user-prompt-draft", Name: "寄生广告", Type: model.ProjectTypeShortDrama, AspectRatio: "9:16", Status: model.ProjectStatusActive, CreatedAt: now, UpdatedAt: now}
	scene := model.Scene{ID: "scene-prompt-draft", ProjectID: project.ID, UnitID: "unit-prompt-draft", Title: "黑市诊所", Status: "draft", CreatedAt: now, UpdatedAt: now}
	shot := model.Shot{ID: "shot-prompt-draft", ProjectID: project.ID, UnitID: scene.UnitID, SceneID: scene.ID, Title: "SC05-SH001", DurationMs: 5000, Status: "draft", CreatedAt: now, UpdatedAt: now}
	for _, item := range []any{&project, &scene, &shot} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := New(repository.New(db), t.TempDir())
	blankPayload := map[string]any{
		"creativePrompt":       "",
		"compiledPrompt":       "",
		"cameraAnchorId":       "browser-camera",
		"viewId":               "browser-view",
		"lockedSceneAssetPack": map[string]any{"sceneId": "browser-scene"},
	}

	draft, err := svc.SaveProjectFilmArtifact(project.UserID, project.ID, SaveProjectFilmArtifactRequest{
		ShotID: shot.ID, ArtifactType: FilmArtifactTypeVideoPromptPack, Status: "draft", Payload: blankPayload,
	})
	if err != nil {
		t.Fatalf("blank Prompt Pack draft should not require a spatial gate: %v", err)
	}
	var draftPayload map[string]any
	if err := json.Unmarshal([]byte(draft.PayloadJSON), &draftPayload); err != nil {
		t.Fatal(err)
	}
	for _, key := range append(spatialPromptFieldKeys(), "compiledPrompt") {
		if _, exists := draftPayload[key]; exists {
			t.Fatalf("uncompiled Prompt Pack draft retained server-owned field %q: %#v", key, draftPayload)
		}
	}

	if _, err := svc.SaveProjectFilmArtifact(project.UserID, project.ID, SaveProjectFilmArtifactRequest{
		ShotID: shot.ID, ArtifactType: FilmArtifactTypeVideoPromptPack, Status: "review", Payload: blankPayload,
	}); err != nil {
		t.Fatalf("blank Prompt Pack review should be allowed before spatial compilation: %v", err)
	}
	if _, err := svc.SaveProjectFilmArtifact(project.UserID, project.ID, SaveProjectFilmArtifactRequest{
		ShotID: shot.ID, ArtifactType: FilmArtifactTypeVideoPromptPack, Status: "ready", Payload: blankPayload,
	}); err == nil {
		t.Fatal("an uncompiled Prompt Pack must not be marked ready")
	}
	if _, err := svc.SaveProjectFilmArtifact(project.UserID, project.ID, SaveProjectFilmArtifactRequest{
		ShotID: shot.ID, ArtifactType: FilmArtifactTypeVideoPromptPack, Status: "draft", Payload: map[string]any{"creativePrompt": "小皮抬头看向富豪"},
	}); err == nil {
		t.Fatal("a scene-linked Prompt Pack with creative prose must still require a PASS spatial gate")
	}
}

func mustSpatialPayload(t *testing.T, pack SceneAssetPack) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func bindSpatialPackScene(pack *SceneAssetPack, sceneID string) {
	for index := range pack.FixedAnchors {
		pack.FixedAnchors[index].SceneID = sceneID
	}
	for index := range pack.MovableAnchors {
		pack.MovableAnchors[index].SceneID = sceneID
	}
	if pack.CameraAnchorPlan != nil {
		for index := range pack.CameraAnchorPlan.Anchors {
			pack.CameraAnchorPlan.Anchors[index].SceneID = sceneID
		}
	}
}

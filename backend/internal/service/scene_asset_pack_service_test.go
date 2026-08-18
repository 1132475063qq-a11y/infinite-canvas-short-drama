package service

import (
	"encoding/json"
	"testing"
	"time"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSaveProjectSceneAssetPackPersistsVersionedGateAndBlocksDriftedGeneration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.Scene{}, &model.Shot{}, &model.FilmArtifact{}, &model.ShotAssetReference{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	project := model.Project{ID: "project-spatial", UserID: "user-spatial", Name: "寄生广告", Type: model.ProjectTypeShortDrama, AspectRatio: "9:16", Status: model.ProjectStatusActive, CreatedAt: now, UpdatedAt: now}
	scene := model.Scene{ID: "scene-clinic", ProjectID: project.ID, Code: "SC05", Title: "黑市诊所", UnitID: "unit-5", Status: "ready", CreatedAt: now, UpdatedAt: now}
	shot := model.Shot{ID: "shot-spatial", ProjectID: project.ID, UnitID: scene.UnitID, SceneID: scene.ID, Title: "SC05-SH001", DurationMs: 5000, Status: "ready", CreatedAt: now, UpdatedAt: now}
	for _, item := range []any{&project, &scene, &shot} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := New(repository.New(db), t.TempDir())
	pack := blackMarketClinicSpatialPack()
	pack.SceneID = scene.ID
	pack.SceneManifest.SceneID = scene.ID
	packJSON, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	var packPayload map[string]any
	if err := json.Unmarshal(packJSON, &packPayload); err != nil {
		t.Fatal(err)
	}
	first, err := svc.SaveProjectSceneAssetPack(project.UserID, project.ID, SaveProjectSceneAssetPackRequest{SceneID: scene.ID, Status: "ready", ExpectedVersion: intPointer(0), Payload: packPayload})
	if err != nil {
		t.Fatal(err)
	}
	if first.PackArtifact.ObjectVersion != 1 || first.GateArtifact.ObjectVersion != 1 || first.Gate.Status != SceneSpatialGatePass {
		t.Fatalf("unexpected first spatial versions/gate: %#v", first)
	}
	var storedGatePayload SceneSpatialGate
	if err := json.Unmarshal([]byte(first.GateArtifact.PayloadJSON), &storedGatePayload); err != nil {
		t.Fatal(err)
	}
	if storedGatePayload.PackVersion != first.PackArtifact.ObjectVersion {
		t.Fatalf("gate did not freeze pack version: %#v", storedGatePayload)
	}
	if storedGatePayload.PackArtifactID != first.PackArtifact.ID || storedGatePayload.PackArtifactVersion != first.PackArtifact.ObjectVersion {
		t.Fatalf("gate did not freeze the exact pack artifact: %#v", storedGatePayload)
	}
	if _, err := svc.SaveProjectSceneAssetPack(project.UserID, project.ID, SaveProjectSceneAssetPackRequest{SceneID: scene.ID, Status: "draft", ExpectedVersion: intPointer(0), Payload: packPayload}); err == nil {
		t.Fatal("stale scene asset pack save should be rejected")
	}
	prompt, err := svc.SaveProjectFilmArtifact(project.UserID, project.ID, SaveProjectFilmArtifactRequest{
		ShotID: shot.ID, ArtifactType: FilmArtifactTypeVideoPromptPack, Status: "ready",
		Payload: map[string]any{"creativePrompt": "黑市诊所连续性测试", "viewId": "V01"},
	})
	if err != nil {
		t.Fatal(err)
	}
	request, err := svc.SaveProjectFilmArtifact(project.UserID, project.ID, SaveProjectFilmArtifactRequest{
		ShotID: shot.ID, ArtifactType: FilmArtifactTypeGenerationRequest, Status: "ready",
		Payload: map[string]any{"promptArtifactId": prompt.ID, "mediaType": "image", "aspectRatio": "9:16", "outputIntent": "场景测试"},
	})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := svc.FilmGenerationTaskDraft(project.UserID, project.ID, request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !draft.RequestReady || !draft.SpatialGateReady || draft.SpatialGateArtifactID != first.GateArtifact.ID {
		t.Fatalf("PASS spatial gate should permit draft: %#v", draft)
	}

	drifted := blackMarketClinicSpatialPack()
	drifted.SceneID = scene.ID
	drifted.SceneManifest.SceneID = scene.ID
	drifted.FixedAnchors[0].Position.X += 2
	driftedJSON, err := json.Marshal(drifted)
	if err != nil {
		t.Fatal(err)
	}
	var driftedPayload map[string]any
	if err := json.Unmarshal(driftedJSON, &driftedPayload); err != nil {
		t.Fatal(err)
	}
	second, err := svc.SaveProjectSceneAssetPack(project.UserID, project.ID, SaveProjectSceneAssetPackRequest{SceneID: scene.ID, Status: "draft", ExpectedVersion: intPointer(1), Payload: driftedPayload})
	if err != nil {
		t.Fatal(err)
	}
	if second.Gate.Status != SceneSpatialGateFail || second.PackArtifact.ObjectVersion != 2 {
		t.Fatalf("drifted pack should be persisted as failed draft: %#v", second)
	}
	draft, err = svc.FilmGenerationTaskDraft(project.UserID, project.ID, request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if draft.RequestReady || draft.SpatialGateReady || draft.SpatialContinuityGate == nil || draft.SpatialContinuityGate.Status != SceneSpatialGateFail {
		t.Fatalf("failed spatial gate must block generation: %#v", draft)
	}

	corrected := blackMarketClinicSpatialPack()
	corrected.SceneID = scene.ID
	corrected.SceneManifest.SceneID = scene.ID
	correctedJSON, err := json.Marshal(corrected)
	if err != nil {
		t.Fatal(err)
	}
	var correctedPayload map[string]any
	if err := json.Unmarshal(correctedJSON, &correctedPayload); err != nil {
		t.Fatal(err)
	}
	third, err := svc.SaveProjectSceneAssetPack(project.UserID, project.ID, SaveProjectSceneAssetPackRequest{SceneID: scene.ID, Status: "ready", ExpectedVersion: intPointer(2), Payload: correctedPayload})
	if err != nil {
		t.Fatal(err)
	}
	if third.Gate.Status != SceneSpatialGatePass || third.PackArtifact.ObjectVersion != 3 {
		t.Fatalf("corrected pack should compare with the latest PASS baseline: %#v", third)
	}
	promptV3, err := svc.SaveProjectFilmArtifact(project.UserID, project.ID, SaveProjectFilmArtifactRequest{
		ShotID: shot.ID, ArtifactType: FilmArtifactTypeVideoPromptPack, Status: "ready",
		Payload: map[string]any{"creativePrompt": "修复后的黑市诊所连续性测试", "viewId": "V01"},
	})
	if err != nil {
		t.Fatal(err)
	}
	requestV3, err := svc.SaveProjectFilmArtifact(project.UserID, project.ID, SaveProjectFilmArtifactRequest{
		ShotID: shot.ID, ArtifactType: FilmArtifactTypeGenerationRequest, Status: "ready",
		Payload: map[string]any{"promptArtifactId": promptV3.ID, "mediaType": "image", "aspectRatio": "9:16", "outputIntent": "场景测试"},
	})
	if err != nil {
		t.Fatal(err)
	}
	draft, err = svc.FilmGenerationTaskDraft(project.UserID, project.ID, requestV3.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !draft.RequestReady || !draft.SpatialGateReady || draft.SpatialPackArtifactID != third.PackArtifact.ID {
		t.Fatalf("corrected PASS pack should reopen generation: %#v", draft)
	}

	draftOnly, err := svc.SaveProjectSceneAssetPack(project.UserID, project.ID, SaveProjectSceneAssetPackRequest{SceneID: scene.ID, Status: "draft", ExpectedVersion: intPointer(3), Payload: correctedPayload})
	if err != nil {
		t.Fatal(err)
	}
	if draftOnly.Gate.Status != SceneSpatialGateUncertain || draftOnly.PackArtifact.ObjectVersion != 4 {
		t.Fatalf("a structurally valid draft must remain gated: %#v", draftOnly)
	}
	draft, err = svc.FilmGenerationTaskDraft(project.UserID, project.ID, requestV3.ID)
	if err != nil {
		t.Fatal(err)
	}
	if draft.RequestReady || draft.SpatialGateReady || draft.SpatialContinuityGate == nil || draft.SpatialContinuityGate.Status != SceneSpatialGateUncertain {
		t.Fatalf("a draft pack must not open formal generation: %#v", draft)
	}
}

func intPointer(value int) *int {
	return &value
}

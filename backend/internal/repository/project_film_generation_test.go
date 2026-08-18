package repository

import (
	"errors"
	"testing"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLockFilmGenerationTargetRejectsSpatialPackTOCTOU(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.CanvasProject{}, &model.Scene{}, &model.Shot{}, &model.FilmArtifact{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	project := model.Project{ID: "project-spatial-lock", UserID: "user-spatial-lock", Type: model.ProjectTypeShortDrama, Status: model.ProjectStatusActive, CreatedAt: now, UpdatedAt: now}
	canvas := model.CanvasProject{ID: "canvas-spatial-lock", UserID: project.UserID, ProjectID: project.ID, PayloadJSON: `{"nodes":[]}`, CreatedAt: now, UpdatedAt: now}
	scene := model.Scene{ID: "scene-spatial-lock", ProjectID: project.ID, Title: "Clinic", CreatedAt: now, UpdatedAt: now}
	shot := model.Shot{ID: "shot-spatial-lock", ProjectID: project.ID, SceneID: scene.ID, CreatedAt: now, UpdatedAt: now}
	request := model.FilmArtifact{ID: "request-spatial-lock", ProjectID: project.ID, SceneID: scene.ID, ShotID: shot.ID, Scope: model.FilmArtifactScopeShot, ScopeID: shot.ID, ArtifactType: "generation_request", ObjectVersion: 1, Status: "ready", PayloadJSON: `{}`, SourceRefsJSON: `[]`, CreatedAt: now, UpdatedAt: now}
	packV1 := model.FilmArtifact{ID: "pack-spatial-lock-v1", ProjectID: project.ID, SceneID: scene.ID, Scope: model.FilmArtifactScopeScene, ScopeID: scene.ID, ArtifactType: "scene_asset_pack", ObjectVersion: 1, Status: "ready", PayloadJSON: `{}`, CreatedAt: now, UpdatedAt: now}
	gateV1 := model.FilmArtifact{ID: "gate-spatial-lock-v1", ProjectID: project.ID, SceneID: scene.ID, Scope: model.FilmArtifactScopeScene, ScopeID: scene.ID, ArtifactType: "spatial_continuity_gate", ObjectVersion: 1, Status: "ready", PayloadJSON: `{"status":"PASS","packArtifactId":"pack-spatial-lock-v1","packArtifactVersion":1,"packVersion":1}`, CreatedAt: now, UpdatedAt: now}
	for _, item := range []any{&project, &canvas, &scene, &shot, &request, &packV1, &gateV1} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	input := FilmGenerationAtomicCreateInput{
		UserID: project.UserID, ProjectID: project.ID, CanvasPayloadJSON: canvas.PayloadJSON,
		GenerationRequestArtifact: request,
		Patch:                     &model.CanvasProjectionPatch{CanvasID: canvas.ID},
		SceneID:                   scene.ID, SpatialGateArtifactID: gateV1.ID, SpatialGateArtifactVersion: gateV1.ObjectVersion,
	}
	if err := db.Transaction(func(tx *gorm.DB) error { return lockFilmGenerationTarget(tx, input) }); err != nil {
		t.Fatalf("current PASS gate rejected: %v", err)
	}
	packV2 := packV1
	packV2.ID = "pack-spatial-lock-v2"
	packV2.ObjectVersion = 2
	if err := db.Create(&packV2).Error; err != nil {
		t.Fatal(err)
	}
	err = db.Transaction(func(tx *gorm.DB) error { return lockFilmGenerationTarget(tx, input) })
	if !errors.Is(err, ErrFilmGenerationTargetChanged) {
		t.Fatalf("spatial pack changed after draft; got %v, want ErrFilmGenerationTargetChanged", err)
	}
}

func TestSaveSceneAssetPackAndGateRejectsStaleExpectedVersion(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.Scene{}, &model.FilmArtifact{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	project := model.Project{ID: "project-spatial-version", UserID: "user-spatial-version", Type: model.ProjectTypeShortDrama, Status: model.ProjectStatusActive, CreatedAt: now, UpdatedAt: now}
	scene := model.Scene{ID: "scene-spatial-version", ProjectID: project.ID, Title: "Clinic", CreatedAt: now, UpdatedAt: now}
	for _, item := range []any{&project, &scene} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := New(db)
	packV1 := model.FilmArtifact{ID: "pack-spatial-version-v1", ProjectID: project.ID, SceneID: scene.ID, Scope: model.FilmArtifactScopeScene, ScopeID: scene.ID, ArtifactType: "scene_asset_pack", Status: "draft", PayloadJSON: `{}`, CreatedAt: now, UpdatedAt: now}
	gateV1 := model.FilmArtifact{ID: "gate-spatial-version-v1", ProjectID: project.ID, SceneID: scene.ID, Scope: model.FilmArtifactScopeScene, ScopeID: scene.ID, ArtifactType: "spatial_continuity_gate", Status: "review", PayloadJSON: `{"status":"UNCERTAIN"}`, CreatedAt: now, UpdatedAt: now}
	if err := repo.SaveSceneAssetPackAndGate(&packV1, &gateV1, 0); err != nil {
		t.Fatal(err)
	}

	packStale := packV1
	packStale.ID = "pack-spatial-version-stale"
	packStale.ObjectVersion = 0
	gateStale := gateV1
	gateStale.ID = "gate-spatial-version-stale"
	gateStale.ObjectVersion = 0
	err = repo.SaveSceneAssetPackAndGate(&packStale, &gateStale, 0)
	if !errors.Is(err, ErrFilmArtifactVersionChanged) {
		t.Fatalf("stale spatial write error = %v, want ErrFilmArtifactVersionChanged", err)
	}
	var artifactCount int64
	if err := db.Model(&model.FilmArtifact{}).Count(&artifactCount).Error; err != nil {
		t.Fatal(err)
	}
	if artifactCount != 2 {
		t.Fatalf("stale write must be atomic; artifact count = %d, want 2", artifactCount)
	}
}

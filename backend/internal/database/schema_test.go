package database

import (
	"testing"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrateSchemaBackfillsFilmArtifactScopeBeforeUniqueIndex(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.FilmArtifact{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	legacy := []model.FilmArtifact{
		{ID: "legacy-shot-a", ProjectID: "project-1", ShotID: "shot-a", ArtifactType: "shot_contract", ObjectVersion: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "legacy-shot-b", ProjectID: "project-1", ShotID: "shot-b", ArtifactType: "shot_contract", ObjectVersion: 1, CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := MigrateSchema(db); err != nil {
		t.Fatal(err)
	}
	var stored []model.FilmArtifact
	if err := db.Order("id asc").Find(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if len(stored) != 2 || stored[0].Scope != model.FilmArtifactScopeShot || stored[0].ScopeID == "" || stored[1].Scope != model.FilmArtifactScopeShot || stored[1].ScopeID == "" {
		t.Fatalf("legacy artifact scopes were not backfilled: %#v", stored)
	}
	duplicate := model.FilmArtifact{ID: "duplicate", ProjectID: "project-1", Scope: model.FilmArtifactScopeShot, ScopeID: "shot-a", ShotID: "shot-a", ArtifactType: "shot_contract", ObjectVersion: 1, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("scope version unique index did not reject a duplicate")
	}
}

func TestMigrateSchemaBackfillsFilmTaskProjectAndCanvasIDs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Task{}, &model.GenerationAttempt{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	task := model.Task{ID: "legacy-film-task", UserID: "user-1", ProjectID: "canvas-1", Provider: model.TaskProviderFilmGateway, CreatedAt: now, UpdatedAt: now}
	attempt := model.GenerationAttempt{ID: "legacy-film-attempt", UserID: task.UserID, TaskID: task.ID, AttemptNumber: 1, DomainProjectID: "project-1", CanvasID: "canvas-1", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&attempt).Error; err != nil {
		t.Fatal(err)
	}
	if err := MigrateSchema(db); err != nil {
		t.Fatal(err)
	}
	var stored model.Task
	if err := db.First(&stored, "id = ?", task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.DomainProjectID != attempt.DomainProjectID || stored.CanvasID != attempt.CanvasID || stored.ProjectID != task.ProjectID {
		t.Fatalf("film task IDs were not backfilled compatibly: %#v", stored)
	}
}

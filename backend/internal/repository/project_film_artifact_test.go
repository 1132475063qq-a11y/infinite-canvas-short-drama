package repository

import (
	"testing"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSaveShotWithContractAdvancesVersionAtomically(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.Shot{}, &model.FilmArtifact{}); err != nil {
		t.Fatal(err)
	}
	repo := New(db)
	now := time.Now()
	shot := model.Shot{ID: "shot-1", ProjectID: "project-1", Title: "SC01-SH001", Status: "draft", CreatedAt: now, UpdatedAt: now}
	first := model.FilmArtifact{ID: "contract-1", ProjectID: shot.ProjectID, ShotID: shot.ID, ArtifactType: "shot_contract", Status: "draft", PayloadJSON: `{"shotSize":"MS"}`, CreatedAt: now, UpdatedAt: now}
	if err := repo.SaveShotWithContract(&shot, true, &first); err != nil {
		t.Fatal(err)
	}
	if shot.ContractArtifactID != first.ID || shot.ContractVersion != 1 || first.ObjectVersion != 1 {
		t.Fatalf("unexpected first contract pointer: %#v %#v", shot, first)
	}

	second := model.FilmArtifact{ID: "contract-2", ProjectID: shot.ProjectID, ShotID: shot.ID, ArtifactType: "shot_contract", Status: "ready", PayloadJSON: `{"shotSize":"MCU"}`, CreatedAt: now, UpdatedAt: now}
	shot.Title = "SC01-SH001 updated"
	shot.Status = "ready"
	shot.UpdatedAt = now.Add(time.Second)
	if err := repo.SaveShotWithContract(&shot, false, &second); err != nil {
		t.Fatal(err)
	}
	if shot.ContractArtifactID != second.ID || shot.ContractVersion != 2 || second.ObjectVersion != 2 {
		t.Fatalf("unexpected second contract pointer: %#v %#v", shot, second)
	}
	var persisted model.Shot
	if err := db.First(&persisted, "id = ?", shot.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.Title != shot.Title || persisted.Status != shot.Status {
		t.Fatalf("shot fields were not updated with the contract: %#v", persisted)
	}
	var count int64
	if err := db.Model(&model.FilmArtifact{}).Where("shot_id = ?", shot.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected two immutable contract versions, got %d", count)
	}
}

func TestSaveFilmArtifactVersionAdvancesProjectRevision(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.Shot{}, &model.FilmArtifact{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	project := model.Project{ID: "project-1", UserID: "user-1", Name: "Fixture", Revision: 4, CreatedAt: now, UpdatedAt: now}
	shot := model.Shot{ID: "shot-1", ProjectID: project.ID, Title: "SC01-SH001", Status: "draft", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&shot).Error; err != nil {
		t.Fatal(err)
	}
	repo := New(db)
	first := model.FilmArtifact{ID: "acting-1", ProjectID: project.ID, ShotID: shot.ID, ArtifactType: "acting", Status: "draft", PayloadJSON: `{"objective":"conceal"}`, CreatedAt: now, UpdatedAt: now}
	second := model.FilmArtifact{ID: "acting-2", ProjectID: project.ID, ShotID: shot.ID, ArtifactType: "acting", Status: "ready", PayloadJSON: `{"objective":"redirect"}`, CreatedAt: now, UpdatedAt: now.Add(time.Second)}
	if err := repo.SaveFilmArtifactVersion(&first); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveFilmArtifactVersion(&second); err != nil {
		t.Fatal(err)
	}
	if first.ObjectVersion != 1 || second.ObjectVersion != 2 {
		t.Fatalf("unexpected versions: %d, %d", first.ObjectVersion, second.ObjectVersion)
	}
	if err := db.First(&project, "id = ?", project.ID).Error; err != nil {
		t.Fatal(err)
	}
	if project.Revision != 6 {
		t.Fatalf("expected project revision 6, got %d", project.Revision)
	}
}

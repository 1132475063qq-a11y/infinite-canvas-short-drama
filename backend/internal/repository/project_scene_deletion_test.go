package repository

import (
	"testing"

	"infinite-canvas/backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeleteProjectRemovesScenesInTransaction(t *testing.T) {
	repo, db := newProjectDeletionRepository(t)
	project := model.Project{ID: "project-delete", UserID: "user-1", Name: "待删除项目"}
	scene := model.Scene{ID: "scene-delete", ProjectID: project.ID, Code: "SC01", Title: "待删除场景"}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&scene).Error; err != nil {
		t.Fatal(err)
	}

	if err := repo.DeleteProject(project.UserID, project.ID); err != nil {
		t.Fatal(err)
	}

	assertSceneCount(t, db, "project_id = ?", []any{project.ID}, 0)
}

func TestDeleteProjectRemovesFilmArtifactsInTransaction(t *testing.T) {
	repo, db := newProjectDeletionRepository(t)
	project := model.Project{ID: "project-artifact-delete", UserID: "user-1", Name: "合同删除项目"}
	shot := model.Shot{ID: "shot-artifact-delete", ProjectID: project.ID, Title: "SC01-SH001"}
	artifact := model.FilmArtifact{ID: "artifact-delete", ProjectID: project.ID, ShotID: shot.ID, ArtifactType: "shot_contract", ObjectVersion: 1}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&shot).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&artifact).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteProject(project.UserID, project.ID); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.FilmArtifact{}).Where("project_id = ?", project.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected project film artifacts deleted, got %d", count)
	}
}

func TestDeleteProjectRemovesCanvasProjectionPatchesInTransaction(t *testing.T) {
	repo, db := newProjectDeletionRepository(t)
	project := model.Project{ID: "project-projection-delete", UserID: "user-1", Name: "任务投影删除项目"}
	canvas := model.CanvasProject{ID: "canvas-projection-delete", UserID: project.UserID, ProjectID: project.ID, Title: "生产画布", PayloadJSON: `{}`}
	patch := model.CanvasProjectionPatch{
		ID:                    "patch-delete",
		UserID:                project.UserID,
		CanvasID:              canvas.ID,
		NodeID:                "generation-node",
		PatchKind:             "film_generation_task",
		TargetProjectID:       project.ID,
		TargetArtifactID:      "generation-request-v1",
		TargetArtifactVersion: 1,
		TaskID:                "task-delete",
		Revision:              1,
	}
	for _, record := range []any{&project, &canvas, &patch} {
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := repo.DeleteProject(project.UserID, project.ID); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.CanvasProjectionPatch{}).Where("target_project_id = ?", project.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected project canvas projection patches deleted, got %d", count)
	}
}

func TestDeleteProjectUnitRemovesOnlyUnitScenes(t *testing.T) {
	repo, db := newProjectDeletionRepository(t)
	project := model.Project{ID: "project-unit-delete", UserID: "user-1", Name: "章节删除项目"}
	unit := model.ProjectUnit{ID: "unit-delete", ProjectID: project.ID, Title: "待删除章节"}
	unitScene := model.Scene{ID: "scene-unit-delete", ProjectID: project.ID, UnitID: unit.ID, Code: "SC01", Title: "章节场景"}
	projectScene := model.Scene{ID: "scene-keep", ProjectID: project.ID, Code: "SC02", Title: "项目场景"}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&unit).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]model.Scene{unitScene, projectScene}).Error; err != nil {
		t.Fatal(err)
	}

	if err := repo.DeleteProjectUnit(project.ID, unit.ID); err != nil {
		t.Fatal(err)
	}

	assertSceneCount(t, db, "project_id = ? AND unit_id = ?", []any{project.ID, unit.ID}, 0)
	assertSceneCount(t, db, "id = ? AND project_id = ?", []any{projectScene.ID, project.ID}, 1)
}

func newProjectDeletionRepository(t *testing.T) (*Repository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.Project{},
		&model.ProjectUnit{},
		&model.CanvasProject{},
		&model.CanvasProjectionPatch{},
		&model.CanvasUnitLink{},
		&model.Scene{},
		&model.Shot{},
		&model.FilmArtifact{},
		&model.EcommerceArtifact{},
		&model.ShotAssetReference{},
		&model.WorkflowInstance{},
		&model.WorkflowStepInstance{},
		&model.WorkflowStepTask{},
		&model.ProjectAssetLink{},
		&model.ProjectAssetCandidate{},
	); err != nil {
		t.Fatal(err)
	}
	return New(db), db
}

func assertSceneCount(t *testing.T, db *gorm.DB, query string, args []any, expected int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.Scene{}).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != expected {
		t.Fatalf("expected %d scenes, got %d", expected, count)
	}
}

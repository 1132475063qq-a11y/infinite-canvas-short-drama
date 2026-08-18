package repository

import (
	"testing"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSaveEcommerceArtifactVersionIsAppendOnly(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.EcommerceArtifact{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	project := model.Project{ID: "project-1", UserID: "user-1", Name: "电商原型", Type: model.ProjectTypeEcommerce, Revision: 3, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	repo := New(db)
	first := model.EcommerceArtifact{ID: "dna-1", ProjectID: project.ID, ArtifactKey: "product-dna", ArtifactType: "product_dna", SchemaVersion: 1, Lifecycle: "finalized", Evidence: "recorded", PayloadJSON: `{"color":"cream"}`, SourceRefsJSON: `["asset-1"]`, CreatedAt: now, UpdatedAt: now}
	if err := repo.SaveEcommerceArtifactVersion(&first); err != nil {
		t.Fatal(err)
	}
	second := model.EcommerceArtifact{ID: "dna-2", ProjectID: project.ID, ArtifactKey: "product-dna", ArtifactType: "product_dna", SchemaVersion: 1, Lifecycle: "draft", Evidence: "inferred", PayloadJSON: `{"color":"warm cream"}`, SourceRefsJSON: `["asset-1"]`, CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)}
	if err := repo.SaveEcommerceArtifactVersion(&second); err != nil {
		t.Fatal(err)
	}
	if first.Revision != 1 || second.Revision != 2 {
		t.Fatalf("unexpected revisions: %d, %d", first.Revision, second.Revision)
	}
	var count int64
	if err := db.Model(&model.EcommerceArtifact{}).Where("artifact_key = ?", "product-dna").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected two immutable revisions, got %d", count)
	}
	latest, err := repo.LatestEcommerceArtifact(project.ID, "product-dna", "product_dna")
	if err != nil {
		t.Fatal(err)
	}
	if latest.ID != second.ID || latest.Lifecycle != "draft" {
		t.Fatalf("unexpected latest artifact: %#v", latest)
	}
	if err := db.First(&project, "id = ?", project.ID).Error; err != nil {
		t.Fatal(err)
	}
	if project.Revision != 5 {
		t.Fatalf("expected project revision 5, got %d", project.Revision)
	}
}

func TestEcommerceArtifactsAreIsolatedByArtifactKey(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.EcommerceArtifact{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	project := model.Project{ID: "project-1", UserID: "user-1", Name: "电商原型", Type: model.ProjectTypeEcommerce, Revision: 1, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	repo := New(db)
	for _, item := range []model.EcommerceArtifact{
		{ID: "dna-1", ProjectID: project.ID, ArtifactKey: "product-a", ArtifactType: "product_dna", SchemaVersion: 1, Lifecycle: "draft", Evidence: "recorded", PayloadJSON: `{}`, SourceRefsJSON: `["asset-a"]`, CreatedAt: now, UpdatedAt: now},
		{ID: "dna-2", ProjectID: project.ID, ArtifactKey: "product-b", ArtifactType: "product_dna", SchemaVersion: 1, Lifecycle: "draft", Evidence: "recorded", PayloadJSON: `{}`, SourceRefsJSON: `["asset-b"]`, CreatedAt: now, UpdatedAt: now},
	} {
		if err := repo.SaveEcommerceArtifactVersion(&item); err != nil {
			t.Fatal(err)
		}
	}
	items, err := repo.ProjectEcommerceArtifacts(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ArtifactKey == items[1].ArtifactKey {
		t.Fatalf("artifacts from separate keys were not isolated: %#v", items)
	}
}

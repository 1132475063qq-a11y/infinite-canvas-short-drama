package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"
	"infinite-canvas/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestProjectSceneAssetPackRoutesPersistAndReadGate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "scene-asset-pack.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.AuthSession{}, &model.Project{}, &model.Scene{}, &model.FilmArtifact{}); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	user := model.User{ID: "user-handler-spatial", Username: "spatial-handler", DisplayName: "Spatial Handler", Role: model.UserRoleUser, Status: model.UserStatusActive, CreatedAt: now, UpdatedAt: now}
	token := "scene-asset-pack-handler-token"
	tokenHash := sha256.Sum256([]byte(token))
	session := model.AuthSession{ID: "session-handler-spatial", UserID: user.ID, TokenHash: hex.EncodeToString(tokenHash[:]), ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now}
	project := model.Project{ID: "project-handler-spatial", UserID: user.ID, Name: "空间接口测试", Type: model.ProjectTypeShortDrama, AspectRatio: "9:16", Status: model.ProjectStatusActive, CreatedAt: now, UpdatedAt: now}
	scene := model.Scene{ID: "scene-handler-spatial", ProjectID: project.ID, Code: "SC01", Title: "测试场景", Status: "ready", CreatedAt: now, UpdatedAt: now}
	for _, item := range []any{&user, &session, &project, &scene} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}

	svc := service.New(repository.New(db), t.TempDir())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")
	RegisterProjectRoutes(api, svc)

	postBody := []byte(`{"sceneId":"client-must-not-override-path","status":"draft","expectedVersion":0,"payload":{}}`)
	postRequest := httptest.NewRequest(http.MethodPost, "/api/projects/"+project.ID+"/scenes/"+scene.ID+"/asset-pack", bytes.NewReader(postBody))
	postRequest.Header.Set("Content-Type", "application/json")
	postRequest.AddCookie(&http.Cookie{Name: service.SessionCookieName, Value: session.ID + "." + token})
	postResponse := httptest.NewRecorder()
	router.ServeHTTP(postResponse, postRequest)
	if postResponse.Code != http.StatusOK {
		t.Fatalf("POST asset pack status = %d, body = %s", postResponse.Code, postResponse.Body.String())
	}
	var saved sceneAssetPackEnvelope
	if err := json.Unmarshal(postResponse.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Code != 0 || saved.Data.SceneAssetPack.PackArtifact == nil || saved.Data.SceneAssetPack.Gate.Status != service.SceneSpatialGateUncertain {
		t.Fatalf("unexpected POST response: %#v", saved)
	}
	if saved.Data.SceneAssetPack.PackArtifact.SceneID != scene.ID || saved.Data.SceneAssetPack.PackArtifact.Scope != model.FilmArtifactScopeScene || saved.Data.SceneAssetPack.PackArtifact.ScopeID != scene.ID {
		t.Fatalf("path scene must own the persisted artifact: %#v", saved.Data.SceneAssetPack.PackArtifact)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/scenes/"+scene.ID+"/asset-pack", nil)
	getRequest.AddCookie(&http.Cookie{Name: service.SessionCookieName, Value: session.ID + "." + token})
	getResponse := httptest.NewRecorder()
	router.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET asset pack status = %d, body = %s", getResponse.Code, getResponse.Body.String())
	}
	var loaded sceneAssetPackEnvelope
	if err := json.Unmarshal(getResponse.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Code != 0 || loaded.Data.SceneAssetPack.PackArtifact == nil || loaded.Data.SceneAssetPack.GateArtifact == nil {
		t.Fatalf("unexpected GET response: %#v", loaded)
	}
	if loaded.Data.SceneAssetPack.Gate.PackArtifactID != loaded.Data.SceneAssetPack.PackArtifact.ID || loaded.Data.SceneAssetPack.Gate.PackArtifactVersion != loaded.Data.SceneAssetPack.PackArtifact.ObjectVersion {
		t.Fatalf("gate did not bind the exact pack artifact: %#v", loaded.Data.SceneAssetPack)
	}
}

type sceneAssetPackEnvelope struct {
	Code int `json:"code"`
	Data struct {
		SceneAssetPack struct {
			PackArtifact *model.FilmArtifact      `json:"packArtifact"`
			GateArtifact *model.FilmArtifact      `json:"gateArtifact"`
			Gate         service.SceneSpatialGate `json:"gate"`
		} `json:"sceneAssetPack"`
	} `json:"data"`
}

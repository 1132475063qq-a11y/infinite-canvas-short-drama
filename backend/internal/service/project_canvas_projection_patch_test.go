package service

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFilmGenerationTaskProjectionKeepsTaskBindingOutsideCanvasPayload(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.Project{},
		&model.Shot{},
		&model.FilmArtifact{},
		&model.CanvasProject{},
		&model.CanvasProjectionPatch{},
		&model.Task{},
	); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	project := model.Project{
		ID:          "project-projection",
		UserID:      "user-projection",
		Name:        "寄生广告",
		Type:        model.ProjectTypeShortDrama,
		AspectRatio: "9:16",
		Status:      model.ProjectStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	shot := model.Shot{
		ID:         "shot-projection",
		ProjectID:  project.ID,
		Title:      "SC01-SH001",
		DurationMs: 5000,
		Status:     "ready",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	promptPack := model.FilmArtifact{
		ID:            "prompt-pack-projection-v1",
		ProjectID:     project.ID,
		ShotID:        shot.ID,
		ArtifactType:  FilmArtifactTypeVideoPromptPack,
		ObjectVersion: 1,
		Status:        "ready",
		PayloadJSON:   `{"compiledPrompt":"Frozen prompt v1"}`,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	generationRequest := model.FilmArtifact{
		ID:            "generation-request-projection-v1",
		ProjectID:     project.ID,
		ShotID:        shot.ID,
		ArtifactType:  FilmArtifactTypeGenerationRequest,
		ObjectVersion: 1,
		Status:        "ready",
		PayloadJSON:   `{"promptArtifactId":"prompt-pack-projection-v1","promptArtifactVersion":1,"compiledPrompt":"Frozen prompt v1","mediaType":"video","aspectRatio":"9:16","durationMs":5000,"outputIntent":"首轮镜头生成"}`,
		SourceRefsJSON: `["prompt-pack-projection-v1"]`,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	canvasPayload := canvasProjectionTestPayload("canvas-projection", project.ID, "generation-node", generationRequest.ID, 1, "")
	canvas := model.CanvasProject{
		ID:          "canvas-projection",
		UserID:      project.UserID,
		ProjectID:   project.ID,
		Title:       "SC01 生产画布",
		PayloadJSON: canvasPayload,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	taskInput, err := json.Marshal(map[string]any{
		"canvasId":                        canvas.ID,
		"canvasNodeId":                    "generation-node",
		"generationRequestArtifactId":      generationRequest.ID,
		"generationRequestArtifactVersion": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	task := model.Task{
		ID:        "task-projection",
		UserID:    project.UserID,
		ProjectID: canvas.ID,
		Type:      "canvas_video",
		Status:    model.TaskStatusQueued,
		Operation: "film_generation",
		InputJSON: string(taskInput),
		CreatedAt: now,
		UpdatedAt: now,
	}
	for _, record := range []any{&project, &shot, &promptPack, &generationRequest, &canvas, &task} {
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}

	svc := New(repository.New(db), t.TempDir())
	bindRequest := BindFilmGenerationTaskCanvasProjectionRequest{
		CanvasID:                        canvas.ID,
		NodeID:                          "generation-node",
		DomainProjectID:                 project.ID,
		GenerationRequestArtifactID:     generationRequest.ID,
		GenerationRequestArtifactVersion: generationRequest.ObjectVersion,
		TaskID:                          task.ID,
	}
	patch, err := svc.BindFilmGenerationTaskCanvasProjection(project.UserID, bindRequest)
	if err != nil {
		t.Fatal(err)
	}
	if patch.TaskID != task.ID || patch.TargetArtifactID != generationRequest.ID || patch.TargetArtifactVersion != generationRequest.ObjectVersion || patch.Revision != 1 {
		t.Fatalf("unexpected server-owned projection patch: %#v", patch)
	}
	if err := db.Model(&model.FilmArtifact{}).Where("id = ?", generationRequest.ID).Update("status", "draft").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.BindFilmGenerationTaskCanvasProjection(project.UserID, bindRequest); err == nil {
		t.Fatal("non-ready Generation Request must not bind a Task")
	}
	if err := db.Model(&model.FilmArtifact{}).Where("id = ?", generationRequest.ID).Update("status", "ready").Error; err != nil {
		t.Fatal(err)
	}

	storedCanvas, err := repository.New(db).CanvasProjectForUser(project.UserID, canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(storedCanvas.PayloadJSON, task.ID) {
		t.Fatalf("base Canvas payload must not persist the server Task binding: %s", storedCanvas.PayloadJSON)
	}

	// The legacy all-canvas replacement path is also a full browser snapshot.
	// It must retain the server-owned patch while this Canvas ID still exists.
	if err := repository.New(db).ReplaceCanvasProjects(project.UserID, []model.CanvasProject{canvas}); err != nil {
		t.Fatal(err)
	}

	runtimePayload, err := svc.UserCanvasProject(project.UserID, canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	if taskID := canvasProjectionTaskID(t, runtimePayload, "generation-node"); taskID != task.ID {
		t.Fatalf("runtime Canvas projection did not expose the approved Task binding: got %q want %q", taskID, task.ID)
	}

	// Simulate a later full-document browser sync that changes the node to a
	// newer request and also submits a forged stale taskId. The client-side
	// taskId is stripped before storage, while the exact v1 patch must not be
	// overlaid onto the v2 node.
	newerBrowserPayload := canvasProjectionTestPayload(canvas.ID, project.ID, "generation-node", "generation-request-projection-v2", 2, "browser-injected-task")
	parsedCanvas, err := canvasProjectFromJSON(project.UserID, json.RawMessage(newerBrowserPayload))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(parsedCanvas.PayloadJSON, "browser-injected-task") {
		t.Fatalf("browser-supplied taskId must be stripped before persistence: %s", parsedCanvas.PayloadJSON)
	}
	if err := repository.New(db).UpsertCanvasProject(&parsedCanvas); err != nil {
		t.Fatal(err)
	}

	runtimePayload, err = svc.UserCanvasProject(project.UserID, canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	if taskID := canvasProjectionTaskID(t, runtimePayload, "generation-node"); taskID != "" {
		t.Fatalf("v1 Task binding must not leak onto a newer Generation Request node: got %q", taskID)
	}
}

func TestCanvasProjectionDoesNotTrustPersistedGenerationTaskIDWithoutPatch(t *testing.T) {
	raw := json.RawMessage(canvasProjectionTestPayload("canvas-legacy", "project-legacy", "generation-node", "generation-request-v1", 1, "legacy-browser-task"))
	projected, err := applyCanvasProjectionPatches(raw, nil, "project-legacy", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if taskID := canvasProjectionTaskID(t, projected, "generation-node"); taskID != "" {
		t.Fatalf("persisted browser taskId must not be trusted without a server Patch: got %q", taskID)
	}
}

func TestCanvasProjectionDoesNotApplyAfterCanvasMovesToAnotherProject(t *testing.T) {
	raw := json.RawMessage(canvasProjectionTestPayload("canvas-moved", "project-original", "generation-node", "generation-request-v1", 1, ""))
	projected, err := applyCanvasProjectionPatches(raw, []model.CanvasProjectionPatch{{
		PatchKind:             CanvasProjectionPatchKindFilmGenerationTask,
		TargetProjectID:       "project-original",
		TargetArtifactID:      "generation-request-v1",
		TargetArtifactVersion: 1,
		TaskID:                "task-original",
	}}, "project-other", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if taskID := canvasProjectionTaskID(t, projected, "generation-node"); taskID != "" {
		t.Fatalf("Task binding must not apply after the Canvas moves to another project: got %q", taskID)
	}
}

func canvasProjectionTestPayload(canvasID string, projectID string, nodeID string, artifactID string, artifactVersion int, taskID string) string {
	domainRef := map[string]any{
		"projectId":       projectID,
		"artifactId":      artifactID,
		// FilmNodeDomainRef serializes versions as strings in the Canvas UI.
		"artifactVersion": strconv.Itoa(artifactVersion),
	}
	if taskID != "" {
		domainRef["taskId"] = taskID
	}
	payload := map[string]any{
		"id":        canvasID,
		"projectId": projectID,
		"title":     "SC01 生产画布",
		"createdAt": "2026-08-13T00:00:00Z",
		"updatedAt": "2026-08-13T00:00:00Z",
		"nodes": []any{map[string]any{
			"id":        nodeID,
			"filmKind":  "generation",
			"domainRef": domainRef,
		}},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

func canvasProjectionTaskID(t *testing.T, raw json.RawMessage, nodeID string) string {
	t.Helper()
	node, err := canvasProjectionNodeByID(raw, nodeID)
	if err != nil {
		t.Fatal(err)
	}
	domainRef, ok := node["domainRef"].(map[string]any)
	if !ok {
		t.Fatal("generation node is missing domainRef")
	}
	return canvasProjectionString(domainRef["taskId"])
}

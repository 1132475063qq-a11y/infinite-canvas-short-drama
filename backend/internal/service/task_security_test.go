package service

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNormalizeTaskInputMakesTypedProviderConfigBillable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemSetting{}, &model.ChannelModel{}); err != nil {
		t.Fatal(err)
	}
	channelModel := model.ChannelModel{
		ID: "model-1", ChannelID: "channel-1", ModelKey: "text-model", Capability: "text",
		BillingMode: "fixed_request", UnitPriceMicrocredits: 100_000, PriceConfigured: true, Enabled: true,
	}
	if err := db.Create(&channelModel).Error; err != nil {
		t.Fatal(err)
	}
	svc := &Service{repo: repository.New(db)}
	input, err := normalizeTaskInput(map[string]any{
		"mode":   "text",
		"config": providerConfig{ChannelID: "channel-1", Model: "text-model", APIKey: "system"},
	})
	if err != nil {
		t.Fatal(err)
	}
	order, err := svc.taskBillingOrder("user-1", &model.Task{ID: "task-1", Type: "agent_storyboard"}, input)
	if err != nil {
		t.Fatal(err)
	}
	if order == nil || order.ChannelID != "channel-1" || order.AmountMicrocredits != 100_000 {
		t.Fatalf("taskBillingOrder() = %#v", order)
	}
}

func TestNormalizeTaskInputStillAllowsSecretProtection(t *testing.T) {
	input, err := normalizeTaskInput(map[string]any{
		"config": providerConfig{BaseURL: "https://example.com", APIKey: "private-key", Model: "text-model"},
	})
	if err != nil {
		t.Fatal(err)
	}
	config, ok := input["config"].(map[string]any)
	if !ok || config["apiKey"] != "private-key" {
		t.Fatalf("normalized config = %#v", input["config"])
	}
	svc := &Service{dataDir: t.TempDir()}
	if err := svc.protectTaskSecrets(input); err != nil {
		t.Fatal(err)
	}
	protected, _ := config["apiKey"].(string)
	if protected == "private-key" || !strings.HasPrefix(protected, encryptedSettingPrefix) {
		t.Fatalf("protected apiKey = %q", protected)
	}
}

func TestTaskInputRejectsInlineMedia(t *testing.T) {
	input, err := normalizeTaskInput(map[string]any{
		"referenceImages": []providerMedia{{DataURL: testReferenceImageDataURL}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsInlineMediaDataURL(input) {
		t.Fatal("containsInlineMediaDataURL() = false")
	}
}

func TestCreateTaskRejectsCustomProviderByDefault(t *testing.T) {
	svc, _ := newTaskChannelPolicyTestService(t)

	_, err := svc.CreateTask("user-1", CreateTaskRequest{
		Prompt: "test",
		Type:   "canvas_image_text_to_image",
		Input: map[string]any{
			"mode": "image",
			"config": map[string]any{
				"baseUrl": "https://provider.example/v1",
				"apiKey":  "user-key",
				"model":   "image-model",
			},
		},
	})
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.Status != 403 {
		t.Fatalf("CreateTask() error = %#v, want 403", err)
	}
}

func TestCreateTaskAllowsCustomProviderWhenEnabled(t *testing.T) {
	svc, _ := newTaskChannelPolicyTestService(t)
	actor := &model.User{ID: "admin-1", Role: model.UserRoleAdmin}
	if _, err := svc.UpdateFeatureAvailability(actor, FeatureAvailability{
		ShortDramaEnabled: true, TaskCenterEnabled: true, CreditsEnabled: true, CustomChannelsEnabled: true,
	}); err != nil {
		t.Fatal(err)
	}

	task, err := svc.CreateTask("user-1", CreateTaskRequest{
		Prompt: "test",
		Type:   "canvas_image_text_to_image",
		Input: map[string]any{
			"mode": "image",
			"config": map[string]any{
				"baseUrl": "https://provider.example/v1",
				"apiKey":  "user-key",
				"model":   "image-model",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if task == nil || task.Status != model.TaskStatusQueued {
		t.Fatalf("CreateTask() = %#v", task)
	}
}

func TestCreateTaskAllowsManagedSystemChannelByDefault(t *testing.T) {
	svc, db := newTaskChannelPolicyTestService(t)
	actor := &model.User{ID: "admin-1", Role: model.UserRoleAdmin}
	if _, err := svc.UpdateFeatureAvailability(actor, FeatureAvailability{
		ShortDramaEnabled: true, TaskCenterEnabled: true, CreditsEnabled: false, CustomChannelsEnabled: false,
	}); err != nil {
		t.Fatal(err)
	}
	channel := model.ModelChannel{
		ID: "channel-1", Scope: model.ChannelScopeSystem, Enabled: true, Name: "managed",
		BaseURL: "https://provider.example/v1", APIKey: "system-key", APIFormat: "openai", ModelsJSON: `["image-model"]`,
	}
	channelModel := model.ChannelModel{
		ID: "model-1", ChannelID: channel.ID, ModelKey: "image-model", Capability: "image",
		Protocol: model.ChannelInterfaceOpenAIImage, BillingMode: "fixed_request", PriceConfigured: true, Enabled: true,
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&channelModel).Error; err != nil {
		t.Fatal(err)
	}

	task, err := svc.CreateTask("user-1", CreateTaskRequest{
		Prompt: "test",
		Type:   "canvas_image_text_to_image",
		Input: map[string]any{
			"mode":   "image",
			"config": map[string]any{"channelId": channel.ID, "model": channelModel.ModelKey},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if task == nil || task.Status != model.TaskStatusQueued {
		t.Fatalf("CreateTask() = %#v", task)
	}
}

func TestResolveProviderConfigRejectsCustomProviderByDefault(t *testing.T) {
	svc, _ := newTaskChannelPolicyTestService(t)

	_, err := svc.resolveProviderConfig(providerConfig{
		BaseURL: "https://provider.example/v1",
		APIKey:  "user-key",
		Model:   "image-model",
	})
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.Status != 403 {
		t.Fatalf("resolveProviderConfig() error = %#v, want 403", err)
	}
}

func TestRetryTaskRejectsLegacyCustomProviderByDefault(t *testing.T) {
	svc, db := newTaskChannelPolicyTestService(t)
	input := map[string]any{
		"mode": "image",
		"config": map[string]any{
			"baseUrl": "https://provider.example/v1",
			"apiKey":  "legacy-user-key",
			"model":   "image-model",
		},
	}
	if err := svc.protectTaskSecrets(input); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	task := model.Task{ID: "task-legacy", UserID: "user-1", Type: "canvas_image_text_to_image", Status: model.TaskStatusFailed, Prompt: "test", InputJSON: string(encoded)}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}

	_, err = svc.RetryTask(task.UserID, task.ID)
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.Status != 403 {
		t.Fatalf("RetryTask() error = %#v, want 403", err)
	}
}

func newTaskChannelPolicyTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(
		&model.SystemSetting{}, &model.AdminAuditEvent{}, &model.ModelChannel{}, &model.ChannelModel{},
		&model.Asset{}, &model.CanvasProject{}, &model.Session{}, &model.Message{}, &model.Task{},
		&model.TaskLog{}, &model.Result{}, &model.ApiCallLog{}, &model.TaskTextDelta{},
	); err != nil {
		t.Fatal(err)
	}
	return New(repository.New(db), t.TempDir()), db
}

func TestCreateSessionRemovesDraftWhenTaskCreationFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemSetting{}, &model.Asset{}, &model.CanvasProject{}, &model.Session{}, &model.Message{}, &model.Task{}, &model.TaskLog{}, &model.Result{}, &model.ApiCallLog{}); err != nil {
		t.Fatal(err)
	}
	for range 5 {
		if err := db.Create(&model.Task{ID: newID(), UserID: "user-1", Status: model.TaskStatusQueued, Prompt: "queued"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := &Service{repo: repository.New(db), dataDir: t.TempDir()}
	if _, err := svc.CreateSession("user-1", CreateSessionRequest{Prompt: "new session"}); err == nil {
		t.Fatal("CreateSession() error = nil")
	}
	var sessionCount int64
	var messageCount int64
	if err := db.Model(&model.Session{}).Where("user_id = ?", "user-1").Count(&sessionCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.Message{}).Where("user_id = ?", "user-1").Count(&messageCount).Error; err != nil {
		t.Fatal(err)
	}
	if sessionCount != 0 || messageCount != 0 {
		t.Fatalf("draft counts = sessions:%d messages:%d", sessionCount, messageCount)
	}
}

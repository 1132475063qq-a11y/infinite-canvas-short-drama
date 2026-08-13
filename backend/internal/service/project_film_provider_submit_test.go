package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type filmGenerationSubmitFixture struct {
	db      *gorm.DB
	service *Service
	project model.Project
	request model.FilmArtifact
	canvas  model.CanvasProject
	channel model.ModelChannel
	model   model.ChannelModel
}

func TestSubmitFilmGenerationTaskCreatesBillingTaskAndProjectionAtomically(t *testing.T) {
	fixture := newFilmGenerationSubmitFixture(t, 1_000_000)
	draft, err := fixture.service.FilmGenerationTaskDraft(fixture.project.UserID, fixture.project.ID, fixture.request.ID)
	if err != nil {
		t.Fatal(err)
	}
	req := SubmitFilmGenerationTaskRequest{
		CanvasID: fixture.canvas.ID, CanvasNodeID: "generation-node-1",
		RequestFingerprint: draft.RequestFingerprint, ChannelID: fixture.channel.ID, Model: fixture.model.ModelKey,
	}
	submission, err := fixture.service.SubmitFilmGenerationTask(fixture.project.UserID, fixture.project.ID, fixture.request.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	if !submission.Created || submission.IdempotentReplay || submission.ProjectionRevision != 1 || submission.Task == nil || submission.Task.Status != model.TaskStatusQueued {
		t.Fatalf("unexpected first submission: %#v", submission)
	}

	var storedTask model.Task
	if err := fixture.db.First(&storedTask, "id = ?", submission.Task.ID).Error; err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"server-only-api-key", "private-provider.example", "apiKey", "secretKey", "baseUrl", "headers"} {
		if strings.Contains(storedTask.InputJSON, forbidden) {
			t.Fatalf("stored Task leaked provider field %q: %s", forbidden, storedTask.InputJSON)
		}
	}
	var storedInput map[string]any
	if err := json.Unmarshal([]byte(storedTask.InputJSON), &storedInput); err != nil {
		t.Fatal(err)
	}
	config, _ := storedInput["config"].(map[string]any)
	if config["channelId"] != fixture.channel.ID || config["model"] != fixture.model.ModelKey || config["videoSeconds"] != "5" || config["size"] != "9:16" {
		t.Fatalf("unexpected non-secret runtime config: %#v", config)
	}

	var patch model.CanvasProjectionPatch
	if err := fixture.db.First(&patch, "canvas_id = ? AND node_id = ?", fixture.canvas.ID, req.CanvasNodeID).Error; err != nil {
		t.Fatal(err)
	}
	if patch.TaskID != storedTask.ID || patch.TargetArtifactID != fixture.request.ID || patch.TargetArtifactVersion != fixture.request.ObjectVersion || patch.Revision != 1 {
		t.Fatalf("unexpected projection patch: %#v", patch)
	}
	var attempt model.GenerationAttempt
	if err := fixture.db.First(&attempt, "task_id = ? AND attempt_number = ?", storedTask.ID, 1).Error; err != nil {
		t.Fatal(err)
	}
	if attempt.Status != model.GenerationAttemptStatusQueued || attempt.GenerationRequestArtifactID != fixture.request.ID || attempt.GenerationRequestArtifactVersion != fixture.request.ObjectVersion || attempt.CanvasNodeID != req.CanvasNodeID || attempt.ChannelID != fixture.channel.ID || attempt.ChannelModelID != fixture.model.ID || attempt.BillingOrderID != storedTask.BillingOrderID {
		t.Fatalf("unexpected initial generation attempt: %#v", attempt)
	}
	var order model.BillingOrder
	if err := fixture.db.First(&order, "id = ?", storedTask.BillingOrderID).Error; err != nil {
		t.Fatal(err)
	}
	if order.Status != model.BillingStatusReserved || order.AmountMicrocredits != 500_000 || order.TaskID != storedTask.ID {
		t.Fatalf("unexpected billing order: %#v", order)
	}
	var account model.CreditAccount
	if err := fixture.db.First(&account, "user_id = ?", fixture.project.UserID).Error; err != nil {
		t.Fatal(err)
	}
	if account.AvailableMicrocredits != 500_000 || account.ReservedMicrocredits != 500_000 {
		t.Fatalf("unexpected account reservation: %#v", account)
	}

	replayed, err := fixture.service.SubmitFilmGenerationTask(fixture.project.UserID, fixture.project.ID, fixture.request.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Created || !replayed.IdempotentReplay || replayed.Task == nil || replayed.Task.ID != storedTask.ID || replayed.ProjectionRevision != 1 {
		t.Fatalf("unexpected idempotent replay: %#v", replayed)
	}
	for value, label := range map[any]string{&model.Task{}: "tasks", &model.GenerationAttempt{}: "generation attempts", &model.BillingOrder{}: "billing orders", &model.CanvasProjectionPatch{}: "projection patches"} {
		var count int64
		if err := fixture.db.Model(value).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("%s count = %d, want 1", label, count)
		}
	}
}

func TestSubmitFilmGenerationTaskRollsBackWhenCreditReservationFails(t *testing.T) {
	fixture := newFilmGenerationSubmitFixture(t, 0)
	draft, err := fixture.service.FilmGenerationTaskDraft(fixture.project.UserID, fixture.project.ID, fixture.request.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.service.SubmitFilmGenerationTask(fixture.project.UserID, fixture.project.ID, fixture.request.ID, SubmitFilmGenerationTaskRequest{
		CanvasID: fixture.canvas.ID, CanvasNodeID: "generation-node-1",
		RequestFingerprint: draft.RequestFingerprint, ChannelID: fixture.channel.ID, Model: fixture.model.ModelKey,
	})
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.Status != 400 || !strings.Contains(authErr.Message, "积分不足") {
		t.Fatalf("unexpected insufficient-credit error: %#v", err)
	}
	for value, label := range map[any]string{&model.Task{}: "tasks", &model.GenerationAttempt{}: "generation attempts", &model.BillingOrder{}: "billing orders", &model.CreditLedgerEntry{}: "credit ledger", &model.CanvasProjectionPatch{}: "projection patches"} {
		var count int64
		if err := fixture.db.Model(value).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s count = %d after rollback, want 0", label, count)
		}
	}
}

func TestSubmitFilmGenerationTaskRejectsCorruptedIdempotentFactChain(t *testing.T) {
	fixture := newFilmGenerationSubmitFixture(t, 1_000_000)
	draft, err := fixture.service.FilmGenerationTaskDraft(fixture.project.UserID, fixture.project.ID, fixture.request.ID)
	if err != nil {
		t.Fatal(err)
	}
	req := SubmitFilmGenerationTaskRequest{
		CanvasID: fixture.canvas.ID, CanvasNodeID: "generation-node-1",
		RequestFingerprint: draft.RequestFingerprint, ChannelID: fixture.channel.ID, Model: fixture.model.ModelKey,
	}
	submission, err := fixture.service.SubmitFilmGenerationTask(fixture.project.UserID, fixture.project.ID, fixture.request.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.db.Delete(&model.GenerationAttempt{}, "task_id = ?", submission.Task.ID).Error; err != nil {
		t.Fatal(err)
	}
	_, err = fixture.service.SubmitFilmGenerationTask(fixture.project.UserID, fixture.project.ID, fixture.request.ID, req)
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.Status != 409 || !strings.Contains(authErr.Message, "事实链不完整") {
		t.Fatalf("corrupted idempotent replay error = %#v", err)
	}
}

func TestTaskProviderRouteSnapshotRejectsCapabilityDrift(t *testing.T) {
	fixture := newFilmGenerationSubmitFixture(t, 0)
	draft, err := fixture.service.FilmGenerationTaskDraft(fixture.project.UserID, fixture.project.ID, fixture.request.ID)
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := fixture.service.resolveFilmGenerationProviderRoute(draft, fixture.channel.ID, fixture.model.ModelKey)
	if err != nil {
		t.Fatal(err)
	}
	input, err := filmGenerationRuntimeInput(draft, resolution, fixture.canvas.ID, "generation-node-1")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var runtimeInput canvasGenerationInput
	if err := json.Unmarshal(encoded, &runtimeInput); err != nil {
		t.Fatal(err)
	}
	if err := fixture.service.validateTaskProviderRouteSnapshot(runtimeInput.Config, runtimeInput.Metadata); err != nil {
		t.Fatalf("current route snapshot rejected: %v", err)
	}
	if err := fixture.db.Model(&model.ChannelModel{}).Where("id = ?", fixture.model.ID).Update("capability_version", fixture.model.CapabilityVersion+1).Error; err != nil {
		t.Fatal(err)
	}
	if err := fixture.service.validateTaskProviderRouteSnapshot(runtimeInput.Config, runtimeInput.Metadata); err == nil {
		t.Fatal("provider route capability drift must reject worker execution")
	}
}

func TestFilmGenerationExecutionFactsFollowClaimProviderAndSuccess(t *testing.T) {
	fixture := newFilmGenerationSubmitFixture(t, 1_000_000)
	draft, err := fixture.service.FilmGenerationTaskDraft(fixture.project.UserID, fixture.project.ID, fixture.request.ID)
	if err != nil {
		t.Fatal(err)
	}
	submission, err := fixture.service.SubmitFilmGenerationTask(fixture.project.UserID, fixture.project.ID, fixture.request.ID, SubmitFilmGenerationTaskRequest{
		CanvasID: fixture.canvas.ID, CanvasNodeID: "generation-node-1",
		RequestFingerprint: draft.RequestFingerprint, ChannelID: fixture.channel.ID, Model: fixture.model.ModelKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	repo := repository.New(fixture.db)
	claimed, err := repo.ClaimNextTask("worker-film-success", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if claimed == nil || claimed.ID != submission.Task.ID || claimed.Attempts != 1 {
		t.Fatalf("unexpected claimed task: %#v", claimed)
	}
	attempt, err := repo.GenerationAttemptForTaskNumber(claimed.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Status != model.GenerationAttemptStatusRunning || attempt.StartedAt == nil {
		t.Fatalf("attempt was not claimed with task: %#v", attempt)
	}
	providerObservedAt := time.Now().UTC()
	if err := fixture.service.LogAPICall(model.ApiCallLog{
		UserID: fixture.project.UserID, TaskID: claimed.ID, BillingOrderID: claimed.BillingOrderID,
		AttemptNumber: claimed.Attempts,
		ChannelID: fixture.channel.ID, Capability: "video", RequestKind: "create", Model: fixture.model.ModelKey,
		Status: model.ApiCallStatusSucceeded, ProviderStatus: "queued", ProviderRequestID: "provider-job-success",
		CreatedAt: providerObservedAt,
	}); err != nil {
		t.Fatal(err)
	}
	if err := fixture.service.refreshTaskProviderState(claimed); err != nil {
		t.Fatal(err)
	}
	resultJSON := []byte(`{"video":"https://cdn.example/film-success.mp4"}`)
	if err := fixture.service.saveTaskCompletionWithinStorageQuota(claimed, resultJSON, nil, false); err != nil {
		t.Fatal(err)
	}
	attempt, err = repo.GenerationAttemptForTaskNumber(claimed.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Status != model.GenerationAttemptStatusSucceeded || attempt.CompletedAt == nil {
		t.Fatalf("attempt did not finish with task: %#v", attempt)
	}
	var providerJob model.ProviderJob
	if err := fixture.db.First(&providerJob, "generation_attempt_id = ? AND provider_request_id = ?", attempt.ID, "provider-job-success").Error; err != nil {
		t.Fatal(err)
	}
	if providerJob.Status != model.ProviderJobStatusSucceeded || providerJob.CompletedAt == nil {
		t.Fatalf("provider job did not reconcile to success: %#v", providerJob)
	}
	var result model.Result
	if err := fixture.db.First(&result, "attempt_id = ? AND kind = ?", attempt.ID, model.ResultKindFilmGeneration).Error; err != nil {
		t.Fatal(err)
	}
	if result.TaskID != claimed.ID || result.URL != "https://cdn.example/film-success.mp4" || result.GenerationRequestArtifactID != fixture.request.ID {
		t.Fatalf("unexpected film result: %#v", result)
	}
	history, err := fixture.service.ProjectFilmGenerationExecutions(fixture.project.UserID, fixture.project.ID, fixture.request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Attempts) != 1 || history.Attempts[0].GenerationRequestArtifactVersion != fixture.request.ObjectVersion || len(history.Attempts[0].ProviderJobs) != 1 || len(history.Attempts[0].Results) != 1 {
		t.Fatalf("unexpected execution history: %#v", history)
	}
	runtimeCanvas, err := fixture.service.UserCanvasProject(fixture.project.UserID, fixture.canvas.ID)
	if err != nil {
		t.Fatal(err)
	}
	node, err := canvasProjectionNodeByID(runtimeCanvas, "generation-node-1")
	if err != nil {
		t.Fatal(err)
	}
	domainRef, _ := node["domainRef"].(map[string]any)
	if domainRef["taskId"] != claimed.ID || domainRef["generationAttemptId"] != attempt.ID || domainRef["providerJobId"] != providerJob.ID || domainRef["resultId"] != result.ID {
		t.Fatalf("runtime projection is missing execution facts: %#v", domainRef)
	}
}

func TestFilmGenerationRetryPreservesUncertainAttemptAndQueuesNextAttempt(t *testing.T) {
	fixture := newFilmGenerationSubmitFixture(t, 1_000_000)
	draft, err := fixture.service.FilmGenerationTaskDraft(fixture.project.UserID, fixture.project.ID, fixture.request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.SubmitFilmGenerationTask(fixture.project.UserID, fixture.project.ID, fixture.request.ID, SubmitFilmGenerationTaskRequest{
		CanvasID: fixture.canvas.ID, CanvasNodeID: "generation-node-1",
		RequestFingerprint: draft.RequestFingerprint, ChannelID: fixture.channel.ID, Model: fixture.model.ModelKey,
	}); err != nil {
		t.Fatal(err)
	}
	repo := repository.New(fixture.db)
	claimed, err := repo.ClaimNextTask("worker-film-failure", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordTaskProviderObservation(repository.ProviderJobObservation{
		TaskID: claimed.ID, AttemptNumber: claimed.Attempts, ProviderRequestID: "provider-job-uncertain", PollStage: "accepted",
		Status: model.ProviderJobStatusAccepted, ObservedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := fixture.service.refreshTaskProviderState(claimed); err != nil {
		t.Fatal(err)
	}
	failedAt := time.Now().UTC()
	updated, err := repo.UpdateTaskTerminalState(claimed.ID, model.TaskStatusRunning, model.TaskStatusFailed, "上游状态待核对", "poll connection ended", failedAt)
	if err != nil || !updated {
		t.Fatalf("failed to mark task terminal: updated=%v err=%v", updated, err)
	}
	firstAttempt, err := repo.GenerationAttemptForTaskNumber(claimed.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if firstAttempt.Status != model.GenerationAttemptStatusUncertain {
		t.Fatalf("non-terminal provider job must leave attempt uncertain: %#v", firstAttempt)
	}
	retried, err := fixture.service.RetryTask(fixture.project.UserID, claimed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retried.Status != model.TaskStatusQueued || retried.Attempts != 1 {
		t.Fatalf("unexpected retried task: %#v", retried)
	}
	secondAttempt, err := repo.GenerationAttemptForTaskNumber(claimed.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if secondAttempt.Status != model.GenerationAttemptStatusQueued || secondAttempt.BillingOrderID == "" || secondAttempt.BillingOrderID == firstAttempt.BillingOrderID {
		t.Fatalf("retry did not create an independent queued attempt: %#v", secondAttempt)
	}
	reclaimed, err := repo.ClaimNextTask("worker-film-retry", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed.ID != claimed.ID || reclaimed.Attempts != 2 {
		t.Fatalf("unexpected retry claim: %#v", reclaimed)
	}
	secondAttempt, err = repo.GenerationAttemptForTaskNumber(claimed.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if secondAttempt.Status != model.GenerationAttemptStatusRunning || secondAttempt.StartedAt == nil {
		t.Fatalf("retry attempt was not claimed: %#v", secondAttempt)
	}
}

func TestFilmGenerationLeaseRecoveryStartsNewAttempt(t *testing.T) {
	fixture := newFilmGenerationSubmitFixture(t, 1_000_000)
	draft, err := fixture.service.FilmGenerationTaskDraft(fixture.project.UserID, fixture.project.ID, fixture.request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.SubmitFilmGenerationTask(fixture.project.UserID, fixture.project.ID, fixture.request.ID, SubmitFilmGenerationTaskRequest{
		CanvasID: fixture.canvas.ID, CanvasNodeID: "generation-node-1",
		RequestFingerprint: draft.RequestFingerprint, ChannelID: fixture.channel.ID, Model: fixture.model.ModelKey,
	}); err != nil {
		t.Fatal(err)
	}
	repo := repository.New(fixture.db)
	firstClaim, err := repo.ClaimNextTask("worker-before-expiry", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	expired := time.Now().Add(-time.Minute)
	if err := fixture.db.Model(&model.Task{}).Where("id = ?", firstClaim.ID).Updates(map[string]any{"lease_expires_at": expired, "updated_at": expired}).Error; err != nil {
		t.Fatal(err)
	}
	secondClaim, err := repo.ClaimNextTask("worker-after-expiry", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if secondClaim.ID != firstClaim.ID || secondClaim.Attempts != 2 {
		t.Fatalf("unexpected recovery claim: %#v", secondClaim)
	}
	firstAttempt, err := repo.GenerationAttemptForTaskNumber(firstClaim.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	secondAttempt, err := repo.GenerationAttemptForTaskNumber(firstClaim.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if firstAttempt.Status != model.GenerationAttemptStatusUncertain || firstAttempt.CompletedAt == nil || secondAttempt.Status != model.GenerationAttemptStatusRunning || secondAttempt.StartedAt == nil {
		t.Fatalf("lease recovery did not preserve both attempts: first=%#v second=%#v", firstAttempt, secondAttempt)
	}
	lateObservedAt := time.Now().UTC()
	if err := repo.RecordTaskProviderObservation(repository.ProviderJobObservation{
		TaskID: firstClaim.ID, AttemptNumber: firstClaim.Attempts, ProviderRequestID: "late-provider-job",
		PollStage: "accepted", Status: model.ProviderJobStatusAccepted, ObservedAt: lateObservedAt,
	}); err != nil {
		t.Fatal(err)
	}
	var lateJob model.ProviderJob
	if err := fixture.db.First(&lateJob, "provider_request_id = ?", "late-provider-job").Error; err != nil {
		t.Fatal(err)
	}
	if lateJob.GenerationAttemptID != firstAttempt.ID {
		t.Fatalf("late provider observation attached to the wrong attempt: %#v", lateJob)
	}
	currentTask, err := repo.Task(firstClaim.ID)
	if err != nil {
		t.Fatal(err)
	}
	if currentTask.ProviderRequestID == "late-provider-job" {
		t.Fatalf("late provider observation overwrote the current attempt pointer: %#v", currentTask)
	}
	staleUpdated, staleErr := repo.UpdateTaskTerminalStateForLease(firstClaim.ID, model.TaskStatusRunning, model.TaskStatusFailed, "stale worker failed", "late failure", time.Now().UTC(), firstClaim.LeaseOwner)
	if staleErr != nil || staleUpdated {
		t.Fatalf("stale worker changed terminal state: updated=%v err=%v", staleUpdated, staleErr)
	}
	if err := fixture.service.saveTaskCompletionWithinStorageQuota(firstClaim, []byte(`{"video":"https://cdn.example/stale.mp4"}`), nil, false); !errors.Is(err, repository.ErrTaskStateConflict) {
		t.Fatalf("stale worker completion error = %v, want task state conflict", err)
	}
	var resultCount int64
	if err := fixture.db.Model(&model.Result{}).Where("task_id = ?", firstClaim.ID).Count(&resultCount).Error; err != nil {
		t.Fatal(err)
	}
	if resultCount != 0 {
		t.Fatalf("stale worker persisted %d results", resultCount)
	}
}

func TestFilmGenerationQueuedCancellationPreservesAttemptBeforeRetry(t *testing.T) {
	fixture := newFilmGenerationSubmitFixture(t, 1_000_000)
	draft, err := fixture.service.FilmGenerationTaskDraft(fixture.project.UserID, fixture.project.ID, fixture.request.ID)
	if err != nil {
		t.Fatal(err)
	}
	submission, err := fixture.service.SubmitFilmGenerationTask(fixture.project.UserID, fixture.project.ID, fixture.request.ID, SubmitFilmGenerationTaskRequest{
		CanvasID: fixture.canvas.ID, CanvasNodeID: "generation-node-1",
		RequestFingerprint: draft.RequestFingerprint, ChannelID: fixture.channel.ID, Model: fixture.model.ModelKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.CancelTask(context.Background(), fixture.project.UserID, submission.Task.ID); err != nil {
		t.Fatal(err)
	}
	repo := repository.New(fixture.db)
	firstAttempt, err := repo.GenerationAttemptForTaskNumber(submission.Task.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if firstAttempt.Status != model.GenerationAttemptStatusCancelled || firstAttempt.StartedAt != nil || firstAttempt.CompletedAt == nil {
		t.Fatalf("queued cancellation did not preserve attempt 1: %#v", firstAttempt)
	}
	retried, err := fixture.service.RetryTask(fixture.project.UserID, submission.Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	secondAttempt, err := repo.GenerationAttemptForTaskNumber(submission.Task.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if retried.Status != model.TaskStatusQueued || retried.Attempts != 1 || secondAttempt.Status != model.GenerationAttemptStatusQueued || secondAttempt.BillingOrderID == firstAttempt.BillingOrderID {
		t.Fatalf("queued cancellation retry lost execution history: task=%#v attempt=%#v", retried, secondAttempt)
	}
}

func newFilmGenerationSubmitFixture(t *testing.T, availableCredits int64) filmGenerationSubmitFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.SystemSetting{}, &model.Project{}, &model.Shot{}, &model.FilmArtifact{},
		&model.Asset{}, &model.CanvasProject{}, &model.CanvasProjectionPatch{},
		&model.Session{}, &model.Message{}, &model.Task{}, &model.GenerationAttempt{}, &model.ProviderJob{}, &model.TaskLog{}, &model.Result{},
		&model.ApiCallLog{}, &model.TaskTextDelta{}, &model.ModelChannel{}, &model.ChannelModel{},
		&model.CreditAccount{}, &model.BillingOrder{}, &model.CreditLedgerEntry{},
	); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	project := model.Project{ID: "project-submit", UserID: "user-submit", Name: "寄生广告", Type: model.ProjectTypeShortDrama, AspectRatio: "9:16", Status: model.ProjectStatusActive, CreatedAt: now, UpdatedAt: now}
	shot := model.Shot{ID: "shot-submit", ProjectID: project.ID, Title: "SC01-SH001", DurationMs: 5000, Status: "ready", CreatedAt: now, UpdatedAt: now}
	promptPack := model.FilmArtifact{ID: "prompt-pack-submit", ProjectID: project.ID, ShotID: shot.ID, ArtifactType: FilmArtifactTypeVideoPromptPack, ObjectVersion: 1, Status: "ready", PayloadJSON: `{"compiledPrompt":"A tense close-up in the clinic."}`, CreatedAt: now, UpdatedAt: now}
	request := model.FilmArtifact{ID: "generation-request-submit", ProjectID: project.ID, ShotID: shot.ID, ArtifactType: FilmArtifactTypeGenerationRequest, ObjectVersion: 1, Status: "ready", PayloadJSON: `{"promptArtifactId":"prompt-pack-submit","promptArtifactVersion":1,"compiledPrompt":"A tense close-up in the clinic.","mediaType":"video","aspectRatio":"9:16","durationMs":5000,"outputIntent":"首轮镜头生成"}`, SourceRefsJSON: `["prompt-pack-submit"]`, CreatedAt: now, UpdatedAt: now}
	canvas := model.CanvasProject{ID: "canvas-submit", UserID: project.UserID, ProjectID: project.ID, Title: "SC01 Production Canvas", PayloadJSON: `{"nodes":[{"id":"generation-node-1","filmKind":"generation","domainRef":{"projectId":"project-submit","artifactId":"generation-request-submit","artifactVersion":1}}]}`, CreatedAt: now, UpdatedAt: now}
	channel := model.ModelChannel{ID: "channel-submit", Scope: model.ChannelScopeSystem, Enabled: true, Name: "受控视频渠道", BaseURL: "https://private-provider.example/v1", APIKey: "server-only-api-key", APIFormat: "openai", ModelsJSON: `["video-model"]`, CreatedAt: now, UpdatedAt: now}
	capabilityJSON, err := json.Marshal(DefaultModelCapabilityConfigForModel(string(model.ChannelInterfaceNewAPIVideo), "video-model"))
	if err != nil {
		t.Fatal(err)
	}
	channelModel := model.ChannelModel{ID: "model-submit", ChannelID: channel.ID, ModelKey: "video-model", DisplayName: "受控视频模型", Capability: "video", Protocol: model.ChannelInterfaceNewAPIVideo, BillingMode: "per_second", UnitPriceMicrocredits: 100_000, PriceConfigured: true, Enabled: true, PriceVersion: 2, CapabilityConfigJSON: string(capabilityJSON), CapabilityVersion: 3, CreatedAt: now, UpdatedAt: now}
	account := model.CreditAccount{UserID: project.UserID, AvailableMicrocredits: availableCredits, CreatedAt: now, UpdatedAt: now}
	for _, item := range []any{&project, &shot, &promptPack, &request, &canvas, &channel, &channelModel, &account} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	return filmGenerationSubmitFixture{db: db, service: New(repository.New(db), t.TempDir()), project: project, request: request, canvas: canvas, channel: channel, model: channelModel}
}

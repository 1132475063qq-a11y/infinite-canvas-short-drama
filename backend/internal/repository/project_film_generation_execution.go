package repository

import (
	"errors"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrGenerationAttemptMissing  = errors.New("generation attempt is missing")
	ErrGenerationAttemptConflict = errors.New("generation attempt state conflict")
)

type ProviderJobObservation struct {
	TaskID            string
	AttemptNumber     int
	ProviderRequestID string
	PollStage         string
	ProviderStatus    string
	Status            model.ProviderJobStatus
	Error             string
	NextPollAt        *time.Time
	ObservedAt        time.Time
}

type FilmGenerationExecutionRows struct {
	Attempts     []model.GenerationAttempt
	ProviderJobs []model.ProviderJob
	Results      []model.Result
}

func createQueuedGenerationAttempt(tx *gorm.DB, attempt *model.GenerationAttempt) error {
	if attempt == nil || attempt.ID == "" || attempt.TaskID == "" || attempt.UserID == "" || attempt.AttemptNumber < 1 || attempt.DomainProjectID == "" || attempt.CanvasID == "" || attempt.CanvasNodeID == "" || attempt.GenerationRequestArtifactID == "" || attempt.GenerationRequestArtifactVersion < 1 || attempt.RequestFingerprint == "" || attempt.ChannelID == "" || attempt.ChannelModelID == "" || attempt.Model == "" || attempt.Capability == "" || attempt.Protocol == "" {
		return gorm.ErrInvalidData
	}
	if attempt.Status != model.GenerationAttemptStatusQueued {
		return gorm.ErrInvalidData
	}
	return tx.Create(attempt).Error
}

// claimGenerationAttempt is part of the Task claim transaction. A Worker
// lease recovery becomes a new attempt and leaves the expired execution as
// uncertain instead of silently overwriting it.
func claimGenerationAttempt(tx *gorm.DB, task *model.Task, now time.Time) error {
	if task == nil || task.Provider != model.TaskProviderFilmGateway {
		return nil
	}
	if task.Attempts < 1 {
		return ErrGenerationAttemptMissing
	}

	var attempt model.GenerationAttempt
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&attempt, "task_id = ? AND attempt_number = ?", task.ID, task.Attempts).Error
	if err == nil {
		if attempt.Status != model.GenerationAttemptStatusQueued {
			return ErrGenerationAttemptConflict
		}
		result := tx.Model(&model.GenerationAttempt{}).
			Where("id = ? AND status = ?", attempt.ID, model.GenerationAttemptStatusQueued).
			Updates(map[string]any{
				"status": model.GenerationAttemptStatusRunning, "billing_order_id": task.BillingOrderID,
				"started_at": &now, "updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrGenerationAttemptConflict
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) || task.Attempts == 1 {
		return ErrGenerationAttemptMissing
	}

	var previous model.GenerationAttempt
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&previous, "task_id = ? AND attempt_number = ?", task.ID, task.Attempts-1).Error; err != nil {
		return ErrGenerationAttemptMissing
	}
	if previous.Status != model.GenerationAttemptStatusRunning {
		return ErrGenerationAttemptConflict
	}
	leaseError := "Worker lease expired before a terminal state was recorded"
	closed := tx.Model(&model.GenerationAttempt{}).Where("id = ? AND status = ?", previous.ID, model.GenerationAttemptStatusRunning).Updates(map[string]any{
		"status": model.GenerationAttemptStatusUncertain, "error": leaseError,
		"completed_at": &now, "updated_at": now,
	})
	if closed.Error != nil {
		return closed.Error
	}
	if closed.RowsAffected != 1 {
		return ErrGenerationAttemptConflict
	}
	current := previous
	current.ID = newRepositoryID()
	current.AttemptNumber = task.Attempts
	current.BillingOrderID = task.BillingOrderID
	current.Status = model.GenerationAttemptStatusRunning
	current.Error = ""
	current.StartedAt = &now
	current.CompletedAt = nil
	current.CreatedAt = now
	current.UpdatedAt = now
	return tx.Create(&current).Error
}

func queueGenerationRetryAttempt(tx *gorm.DB, task *model.Task, attempt *model.GenerationAttempt, now time.Time) error {
	if task == nil || task.Provider != model.TaskProviderFilmGateway {
		if attempt != nil {
			return gorm.ErrInvalidData
		}
		return nil
	}
	if attempt == nil || attempt.TaskID != task.ID || attempt.UserID != task.UserID || attempt.BillingOrderID != task.BillingOrderID {
		return gorm.ErrInvalidData
	}
	if attempt.BillingOrderID != "" {
		var order model.BillingOrder
		if err := tx.First(&order, "id = ? AND user_id = ? AND task_id = ?", attempt.BillingOrderID, task.UserID, task.ID).Error; err != nil {
			return err
		}
		if order.ChannelID != attempt.ChannelID || order.ChannelModelID != attempt.ChannelModelID || order.Model != attempt.Model || order.Capability != attempt.Capability || order.PriceVersion != attempt.PriceVersion {
			return gorm.ErrInvalidData
		}
	}
	var latestNumber int
	if err := tx.Model(&model.GenerationAttempt{}).Where("task_id = ?", task.ID).Select("COALESCE(MAX(attempt_number), 0)").Scan(&latestNumber).Error; err != nil {
		return err
	}
	if task.Attempts != latestNumber {
		if err := tx.Model(&model.Task{}).Where("id = ?", task.ID).Updates(map[string]any{"attempts": latestNumber, "updated_at": now}).Error; err != nil {
			return err
		}
		task.Attempts = latestNumber
	}
	attempt.AttemptNumber = latestNumber + 1
	attempt.Status = model.GenerationAttemptStatusQueued
	attempt.StartedAt = nil
	attempt.CompletedAt = nil
	attempt.CreatedAt = now
	attempt.UpdatedAt = now
	return createQueuedGenerationAttempt(tx, attempt)
}

func finishGenerationAttemptForTask(tx *gorm.DB, task *model.Task, taskStatus model.TaskStatus, errorText string, completedAt time.Time) error {
	if task == nil || task.Provider != model.TaskProviderFilmGateway {
		return nil
	}
	attempt, err := latestGenerationAttemptForTask(tx, task.ID)
	if err != nil {
		return err
	}
	status := generationAttemptStatusForTask(taskStatus)
	if taskStatus == model.TaskStatusFailed && attempt.Status == model.GenerationAttemptStatusUncertain {
		status = model.GenerationAttemptStatusUncertain
	}
	if taskStatus == model.TaskStatusFailed && strings.TrimSpace(task.ProviderRequestID) != "" {
		var providerJob model.ProviderJob
		jobErr := tx.First(&providerJob, "generation_attempt_id = ? AND provider_request_id = ?", attempt.ID, task.ProviderRequestID).Error
		if jobErr != nil && !errors.Is(jobErr, gorm.ErrRecordNotFound) {
			return jobErr
		}
		if errors.Is(jobErr, gorm.ErrRecordNotFound) || !providerJobStatusTerminal(providerJob.Status) {
			status = model.GenerationAttemptStatusUncertain
		}
	}
	updates := map[string]any{
		"status": status, "error": errorText, "completed_at": &completedAt, "updated_at": completedAt,
	}
	result := tx.Model(&model.GenerationAttempt{}).
		Where("id = ? AND status IN ?", attempt.ID, []model.GenerationAttemptStatus{
			model.GenerationAttemptStatusQueued,
			model.GenerationAttemptStatusRunning,
			model.GenerationAttemptStatusFailed,
			model.GenerationAttemptStatusUncertain,
		}).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrGenerationAttemptConflict
	}
	if taskStatus == model.TaskStatusSucceeded {
		providerUpdates := map[string]any{"status": model.ProviderJobStatusSucceeded, "last_error": "", "completed_at": &completedAt, "last_observed_at": completedAt, "updated_at": completedAt}
		query := tx.Model(&model.ProviderJob{}).Where("generation_attempt_id = ?", attempt.ID)
		if strings.TrimSpace(task.ProviderRequestID) != "" {
			query = query.Where("provider_request_id = ?", task.ProviderRequestID)
		} else {
			query = query.Where("status NOT IN ?", []model.ProviderJobStatus{model.ProviderJobStatusFailed, model.ProviderJobStatusCancelled})
		}
		if err := query.Updates(providerUpdates).Error; err != nil {
			return err
		}
	}
	return nil
}

func generationAttemptStatusForTask(status model.TaskStatus) model.GenerationAttemptStatus {
	switch status {
	case model.TaskStatusSucceeded:
		return model.GenerationAttemptStatusSucceeded
	case model.TaskStatusFailed:
		return model.GenerationAttemptStatusFailed
	case model.TaskStatusCancelled:
		return model.GenerationAttemptStatusCancelled
	case model.TaskStatusRunning:
		return model.GenerationAttemptStatusRunning
	default:
		return model.GenerationAttemptStatusQueued
	}
}

func latestGenerationAttemptForTask(tx *gorm.DB, taskID string) (*model.GenerationAttempt, error) {
	var attempt model.GenerationAttempt
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("task_id = ?", taskID).Order("attempt_number desc").First(&attempt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGenerationAttemptMissing
		}
		return nil, err
	}
	return &attempt, nil
}

func (r *Repository) GenerationAttemptForTaskNumber(taskID string, attemptNumber int) (*model.GenerationAttempt, error) {
	var attempt model.GenerationAttempt
	if err := r.db.First(&attempt, "task_id = ? AND attempt_number = ?", taskID, attemptNumber).Error; err != nil {
		return nil, err
	}
	return &attempt, nil
}

// RecordTaskProviderObservation keeps the mutable Task pointer and immutable
// attempt-scoped ProviderJob evidence in one database transaction.
func (r *Repository) RecordTaskProviderObservation(observation ProviderJobObservation) error {
	if strings.TrimSpace(observation.TaskID) == "" {
		return gorm.ErrInvalidData
	}
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = time.Now()
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		var task model.Task
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, "id = ?", observation.TaskID).Error; err != nil {
			return err
		}
		updates := map[string]any{"poll_stage": observation.PollStage, "next_poll_at": observation.NextPollAt, "updated_at": observation.ObservedAt}
		if strings.TrimSpace(observation.ProviderRequestID) != "" {
			updates["provider_request_id"] = strings.TrimSpace(observation.ProviderRequestID)
		}
		if task.Provider != model.TaskProviderFilmGateway {
			return tx.Model(&model.Task{}).Where("id = ?", task.ID).Updates(updates).Error
		}
		var attempt *model.GenerationAttempt
		var err error
		if observation.AttemptNumber > 0 {
			var exact model.GenerationAttempt
			if loadErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&exact, "task_id = ? AND attempt_number = ?", task.ID, observation.AttemptNumber).Error; loadErr != nil {
				return loadErr
			}
			attempt = &exact
		} else {
			attempt, err = latestGenerationAttemptForTask(tx, task.ID)
		}
		if err != nil {
			return err
		}
		// A late request from an expired lease still belongs to its original
		// attempt, but it must not overwrite the mutable pointer of the task
		// currently owned by a newer attempt.
		if attempt.AttemptNumber == task.Attempts {
			if err := tx.Model(&model.Task{}).Where("id = ?", task.ID).Updates(updates).Error; err != nil {
				return err
			}
		}
		if strings.TrimSpace(observation.ProviderRequestID) == "" {
			if observation.Status == model.ProviderJobStatusUncertain {
				updates := map[string]any{"status": model.GenerationAttemptStatusUncertain, "error": strings.TrimSpace(observation.Error), "updated_at": observation.ObservedAt}
				if err := tx.Model(&model.GenerationAttempt{}).
					Where("id = ? AND status IN ?", attempt.ID, []model.GenerationAttemptStatus{model.GenerationAttemptStatusQueued, model.GenerationAttemptStatusRunning, model.GenerationAttemptStatusUncertain}).
					Updates(updates).Error; err != nil {
					return err
				}
			}
			return nil
		}
		return upsertProviderJobObservation(tx, *attempt, observation)
	})
}

func upsertProviderJobObservation(tx *gorm.DB, attempt model.GenerationAttempt, observation ProviderJobObservation) error {
	requestID := strings.TrimSpace(observation.ProviderRequestID)
	var existing model.ProviderJob
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&existing, "generation_attempt_id = ? AND provider_request_id = ?", attempt.ID, requestID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	status := observation.Status
	if status == "" {
		status = model.ProviderJobStatusAccepted
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		job := model.ProviderJob{
			ID: newRepositoryID(), UserID: attempt.UserID, TaskID: attempt.TaskID, GenerationAttemptID: attempt.ID,
			ProviderRequestID: requestID, ChannelID: attempt.ChannelID, ChannelModelID: attempt.ChannelModelID,
			Model: attempt.Model, Capability: attempt.Capability, Protocol: attempt.Protocol,
			Status: status, ProviderStatus: strings.TrimSpace(observation.ProviderStatus), PollStage: strings.TrimSpace(observation.PollStage),
			LastError: strings.TrimSpace(observation.Error), FirstObservedAt: observation.ObservedAt, LastObservedAt: observation.ObservedAt,
			CreatedAt: observation.ObservedAt, UpdatedAt: observation.ObservedAt,
		}
		if providerJobStatusTerminal(status) {
			job.CompletedAt = &observation.ObservedAt
		}
		return tx.Create(&job).Error
	}
	if observation.ObservedAt.Before(existing.LastObservedAt) {
		return nil
	}
	if providerJobStatusTerminal(existing.Status) {
		// A later positive reconciliation may recover a previously failed or
		// cancelled upstream job. Once succeeded, however, noisy later polls
		// must never downgrade the recorded success.
		if existing.Status == model.ProviderJobStatusSucceeded || status != model.ProviderJobStatusSucceeded {
			status = existing.Status
		}
	} else if existing.Status == model.ProviderJobStatusRunning && status == model.ProviderJobStatusAccepted {
		status = existing.Status
	} else if existing.Status == model.ProviderJobStatusCancellationRequested && !providerJobStatusTerminal(status) {
		status = existing.Status
	}
	updates := map[string]any{
		"status": status, "provider_status": strings.TrimSpace(observation.ProviderStatus),
		"poll_stage": strings.TrimSpace(observation.PollStage), "last_error": strings.TrimSpace(observation.Error),
		"last_observed_at": observation.ObservedAt, "updated_at": observation.ObservedAt,
	}
	if providerJobStatusTerminal(status) {
		updates["completed_at"] = &observation.ObservedAt
	}
	return tx.Model(&model.ProviderJob{}).Where("id = ?", existing.ID).Updates(updates).Error
}

func providerJobStatusTerminal(status model.ProviderJobStatus) bool {
	return status == model.ProviderJobStatusSucceeded || status == model.ProviderJobStatusFailed || status == model.ProviderJobStatusCancelled
}

func updateProviderJobCancellation(tx *gorm.DB, taskID string, status model.ProviderCancelStatus, now time.Time) error {
	var task model.Task
	if err := tx.Select("provider").First(&task, "id = ?", taskID).Error; err != nil {
		return err
	}
	if task.Provider != model.TaskProviderFilmGateway {
		return nil
	}
	attempt, err := latestGenerationAttemptForTask(tx, taskID)
	if errors.Is(err, ErrGenerationAttemptMissing) {
		return nil
	}
	if err != nil {
		return err
	}
	providerStatus := model.ProviderJobStatusCancellationRequested
	if status == model.ProviderCancelStatusConfirmed {
		providerStatus = model.ProviderJobStatusCancelled
	} else if status == model.ProviderCancelStatusUncertain {
		providerStatus = model.ProviderJobStatusUncertain
	}
	updates := map[string]any{"status": providerStatus, "last_observed_at": now, "updated_at": now}
	if providerJobStatusTerminal(providerStatus) {
		updates["completed_at"] = &now
	}
	return tx.Model(&model.ProviderJob{}).
		Where("generation_attempt_id = ? AND status NOT IN ?", attempt.ID, []model.ProviderJobStatus{model.ProviderJobStatusSucceeded, model.ProviderJobStatusFailed}).
		Updates(updates).Error
}

func (r *Repository) FilmGenerationExecutionRows(userID string, projectID string, artifactID string) (FilmGenerationExecutionRows, error) {
	rows := FilmGenerationExecutionRows{}
	if err := r.db.Where("user_id = ? AND domain_project_id = ? AND generation_request_artifact_id = ?", userID, projectID, artifactID).
		Order("attempt_number asc").Find(&rows.Attempts).Error; err != nil {
		return rows, err
	}
	if len(rows.Attempts) == 0 {
		return rows, nil
	}
	attemptIDs := make([]string, 0, len(rows.Attempts))
	for _, attempt := range rows.Attempts {
		attemptIDs = append(attemptIDs, attempt.ID)
	}
	if err := r.db.Where("generation_attempt_id IN ?", attemptIDs).Order("first_observed_at asc").Find(&rows.ProviderJobs).Error; err != nil {
		return rows, err
	}
	if err := r.db.Where("attempt_id IN ? AND kind = ?", attemptIDs, model.ResultKindFilmGeneration).Order("created_at asc").Find(&rows.Results).Error; err != nil {
		return rows, err
	}
	return rows, nil
}

func enrichCanvasProjectionRuntimeFacts(db *gorm.DB, patches []model.CanvasProjectionPatch) error {
	if len(patches) == 0 {
		return nil
	}
	taskIDs := make([]string, 0, len(patches))
	for _, patch := range patches {
		if (patch.PatchKind == model.CanvasProjectionPatchKindFilmGenerationTask || patch.PatchKind == model.CanvasProjectionPatchKindFilmGenerationResult) && patch.TaskID != "" {
			taskIDs = append(taskIDs, patch.TaskID)
		}
	}
	if len(taskIDs) == 0 {
		return nil
	}
	var attempts []model.GenerationAttempt
	if err := db.Where("task_id IN ?", taskIDs).Order("attempt_number desc").Find(&attempts).Error; err != nil {
		return err
	}
	latestByTask := make(map[string]model.GenerationAttempt, len(taskIDs))
	for _, attempt := range attempts {
		if _, exists := latestByTask[attempt.TaskID]; !exists {
			latestByTask[attempt.TaskID] = attempt
		}
	}
	attemptIDs := make([]string, 0, len(latestByTask))
	for _, attempt := range latestByTask {
		attemptIDs = append(attemptIDs, attempt.ID)
	}
	jobsByAttempt := map[string]string{}
	resultsByAttempt := map[string]model.Result{}
	if len(attemptIDs) > 0 {
		var jobs []model.ProviderJob
		if err := db.Where("generation_attempt_id IN ?", attemptIDs).Order("last_observed_at desc").Find(&jobs).Error; err != nil {
			return err
		}
		for _, job := range jobs {
			if _, exists := jobsByAttempt[job.GenerationAttemptID]; !exists {
				jobsByAttempt[job.GenerationAttemptID] = job.ID
			}
		}
		var results []model.Result
		if err := db.Where("attempt_id IN ? AND kind = ?", attemptIDs, model.ResultKindFilmGeneration).Order("created_at desc").Find(&results).Error; err != nil {
			return err
		}
		for _, result := range results {
			if _, exists := resultsByAttempt[result.AttemptID]; !exists {
				resultsByAttempt[result.AttemptID] = result
			}
		}
	}
	for index := range patches {
		attempt, exists := latestByTask[patches[index].TaskID]
		if !exists {
			continue
		}
		patches[index].GenerationAttemptID = attempt.ID
		patches[index].ProviderJobID = jobsByAttempt[attempt.ID]
		if result, exists := resultsByAttempt[attempt.ID]; exists {
			patches[index].ResultID = result.ID
			patches[index].ResultURL = result.URL
			patches[index].ResultPayload = result.Payload
		}
	}
	return nil
}

// createFilmGenerationResultProjection records the derived Result node in the
// same completion transaction as Task, Attempt, ProviderJob and Result. The
// browser can keep saving a full Canvas snapshot without erasing this fact.
func createFilmGenerationResultProjection(tx *gorm.DB, task *model.Task, results []model.Result, completedAt time.Time) error {
	if task == nil || task.Provider != model.TaskProviderFilmGateway || strings.TrimSpace(task.CanvasID) == "" {
		return nil
	}
	var filmResult *model.Result
	for index := range results {
		if results[index].TaskID == task.ID && results[index].Kind == model.ResultKindFilmGeneration {
			filmResult = &results[index]
			break
		}
	}
	if filmResult == nil {
		return nil
	}

	var taskPatch model.CanvasProjectionPatch
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&taskPatch,
		"user_id = ? AND canvas_id = ? AND task_id = ? AND patch_kind = ?",
		task.UserID, task.CanvasID, task.ID, model.CanvasProjectionPatchKindFilmGenerationTask,
	).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Legacy Film tasks may have completed before the canvas-binding contract
		// existed. Keep their accounting and result history readable.
		return nil
	}
	if err != nil {
		return err
	}
	if taskPatch.TargetProjectID == "" || taskPatch.TargetArtifactID == "" || taskPatch.TargetArtifactVersion < 1 {
		return gorm.ErrInvalidData
	}
	if filmResult.DomainProjectID != taskPatch.TargetProjectID || filmResult.GenerationRequestArtifactID != taskPatch.TargetArtifactID || filmResult.GenerationRequestArtifactVersion != taskPatch.TargetArtifactVersion {
		return gorm.ErrInvalidData
	}

	patch := model.CanvasProjectionPatch{
		ID:                    newRepositoryID(),
		UserID:                task.UserID,
		CanvasID:              task.CanvasID,
		NodeID:                model.FilmGenerationResultCanvasNodeID(task.ID),
		PatchKind:             model.CanvasProjectionPatchKindFilmGenerationResult,
		TargetProjectID:       taskPatch.TargetProjectID,
		TargetArtifactID:      taskPatch.TargetArtifactID,
		TargetArtifactVersion: taskPatch.TargetArtifactVersion,
		TaskID:                task.ID,
		CreatedAt:             completedAt,
		UpdatedAt:             completedAt,
	}
	projected, changed, err := upsertCanvasProjectionPatch(tx, &patch)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	updated := tx.Model(&model.CanvasProject{}).Where("id = ? AND user_id = ?", projected.CanvasID, projected.UserID).Update("updated_at", completedAt)
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

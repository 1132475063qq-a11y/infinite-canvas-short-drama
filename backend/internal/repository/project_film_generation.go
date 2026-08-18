package repository

import (
	"encoding/json"
	"errors"
	"strings"

	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrFilmGenerationTargetChanged  = errors.New("film generation target changed")
	ErrFilmGenerationRouteChanged   = errors.New("film generation provider route changed")
	ErrFilmGenerationNodeBusy       = errors.New("film generation node already has an active task")
	ErrFilmGenerationPatchCorrupted = errors.New("film generation projection patch is corrupted")
)

// FilmGenerationAtomicCreateInput is the repository boundary for the only
// write that may expose a queued film-generation task to the Worker. The
// expected rows are snapshots resolved by the service; this transaction locks
// and compares the current rows before reserving credits or creating anything.
type FilmGenerationAtomicCreateInput struct {
	UserID                    string
	ProjectID                 string
	CanvasPayloadJSON         string
	GenerationRequestArtifact model.FilmArtifact
	Channel                   model.ModelChannel
	ChannelModel              model.ChannelModel
	Task                      *model.Task
	Attempt                   *model.GenerationAttempt
	BillingOrder              *model.BillingOrder
	Patch                     *model.CanvasProjectionPatch
	ActiveTaskLimit           int
	// Scene spatial facts are optional for legacy shot records. When present,
	// the transaction locks and rechecks the exact PASS gate version so a pack
	// update between draft and submit cannot slip through (TOCTOU protection).
	SceneID                    string
	SpatialGateArtifactID      string
	SpatialGateArtifactVersion int
}

type FilmGenerationAtomicCreateResult struct {
	Task    model.Task
	Patch   model.CanvasProjectionPatch
	Created bool
}

// CreateFilmGenerationTaskAtomic serializes submissions per Canvas row and
// commits Task, optional credit reservation and Canvas Projection Patch in one
// database transaction. A duplicate submission for the exact Artifact version
// returns the existing Task without reserving credits again.
func (r *Repository) CreateFilmGenerationTaskAtomic(input FilmGenerationAtomicCreateInput) (*FilmGenerationAtomicCreateResult, error) {
	if input.Task == nil || input.Attempt == nil || input.Patch == nil || strings.TrimSpace(input.UserID) == "" || strings.TrimSpace(input.ProjectID) == "" {
		return nil, gorm.ErrInvalidData
	}
	if input.Task.UserID != input.UserID || input.Patch.UserID != input.UserID || input.Attempt.UserID != input.UserID || input.Task.DomainProjectID != input.ProjectID || input.Task.CanvasID != input.Patch.CanvasID || input.Task.ID != input.Patch.TaskID || input.Attempt.TaskID != input.Task.ID || input.Patch.TargetProjectID != input.ProjectID || input.Attempt.DomainProjectID != input.ProjectID || input.Attempt.CanvasID != input.Patch.CanvasID || input.Attempt.CanvasNodeID != input.Patch.NodeID || input.Patch.TargetArtifactID != input.GenerationRequestArtifact.ID || input.Attempt.GenerationRequestArtifactID != input.GenerationRequestArtifact.ID || input.Patch.TargetArtifactVersion != input.GenerationRequestArtifact.ObjectVersion || input.Attempt.GenerationRequestArtifactVersion != input.GenerationRequestArtifact.ObjectVersion {
		return nil, gorm.ErrInvalidData
	}
	if input.Task.Provider != model.TaskProviderFilmGateway || input.Task.Model != input.ChannelModel.ModelKey || input.Attempt.AttemptNumber != 1 || input.Attempt.Status != model.GenerationAttemptStatusQueued || input.Attempt.ChannelID != input.Channel.ID || input.Attempt.ChannelModelID != input.ChannelModel.ID || input.Attempt.Model != input.ChannelModel.ModelKey || input.Attempt.Capability != input.ChannelModel.Capability || input.Attempt.Protocol != string(input.ChannelModel.Protocol) || input.Attempt.CapabilityVersion != input.ChannelModel.CapabilityVersion || input.Attempt.PriceVersion != input.ChannelModel.PriceVersion {
		return nil, gorm.ErrInvalidData
	}
	if input.Attempt.BillingOrderID != input.Task.BillingOrderID {
		return nil, gorm.ErrInvalidData
	}
	if input.BillingOrder != nil && (input.BillingOrder.UserID != input.UserID || input.BillingOrder.TaskID != input.Task.ID || input.Task.BillingOrderID != input.BillingOrder.ID || input.Attempt.BillingOrderID != input.BillingOrder.ID) {
		return nil, gorm.ErrInvalidData
	}
	result := &FilmGenerationAtomicCreateResult{}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := lockFilmGenerationTarget(tx, input); err != nil {
			return err
		}
		if err := lockFilmGenerationRoute(tx, input); err != nil {
			return err
		}

		var existingPatch model.CanvasProjectionPatch
		patchErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&existingPatch,
			"user_id = ? AND canvas_id = ? AND node_id = ? AND patch_kind = ?",
			input.Patch.UserID, input.Patch.CanvasID, input.Patch.NodeID, input.Patch.PatchKind,
		).Error
		if patchErr != nil && !errors.Is(patchErr, gorm.ErrRecordNotFound) {
			return patchErr
		}
		if patchErr == nil {
			var existingTask model.Task
			taskErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&existingTask, "id = ? AND user_id = ?", existingPatch.TaskID, input.UserID).Error
			if taskErr != nil {
				return ErrFilmGenerationPatchCorrupted
			}
			if sameFilmGenerationTarget(existingPatch, *input.Patch) {
				var existingAttempt model.GenerationAttempt
				if err := tx.First(&existingAttempt, "task_id = ? AND attempt_number = ?", existingTask.ID, 1).Error; err != nil || existingTask.DomainProjectID != input.ProjectID || existingTask.CanvasID != existingPatch.CanvasID || existingAttempt.UserID != input.UserID || existingAttempt.DomainProjectID != input.ProjectID || existingAttempt.CanvasID != existingPatch.CanvasID || existingAttempt.CanvasNodeID != existingPatch.NodeID || existingAttempt.GenerationRequestArtifactID != existingPatch.TargetArtifactID || existingAttempt.GenerationRequestArtifactVersion != existingPatch.TargetArtifactVersion || existingAttempt.RequestFingerprint != input.Attempt.RequestFingerprint {
					return ErrFilmGenerationPatchCorrupted
				}
				result.Task = existingTask
				result.Patch = existingPatch
				result.Created = false
				return nil
			}
			if existingTask.Status == model.TaskStatusQueued || existingTask.Status == model.TaskStatusRunning {
				return ErrFilmGenerationNodeBusy
			}
		}

		if err := enforceActiveTaskLimit(tx, input.UserID, input.ActiveTaskLimit); err != nil {
			return err
		}
		if input.BillingOrder != nil {
			if err := reserveBillingOrder(tx, input.BillingOrder); err != nil {
				return err
			}
		}
		if err := tx.Create(input.Task).Error; err != nil {
			return err
		}
		if err := createQueuedGenerationAttempt(tx, input.Attempt); err != nil {
			return err
		}
		patch, _, err := upsertCanvasProjectionPatch(tx, input.Patch)
		if err != nil {
			return err
		}
		updated := tx.Model(&model.CanvasProject{}).
			Where("id = ? AND user_id = ?", input.Patch.CanvasID, input.UserID).
			Update("updated_at", input.Patch.UpdatedAt)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return ErrFilmGenerationTargetChanged
		}
		result.Task = *input.Task
		result.Patch = patch
		result.Created = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func lockFilmGenerationTarget(tx *gorm.DB, input FilmGenerationAtomicCreateInput) error {
	var project model.Project
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&project, "id = ? AND user_id = ?", input.ProjectID, input.UserID).Error; err != nil {
		return ErrFilmGenerationTargetChanged
	}
	if project.Type != model.ProjectTypeShortDrama || project.Status == model.ProjectStatusArchived {
		return ErrFilmGenerationTargetChanged
	}

	var canvas model.CanvasProject
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&canvas, "id = ? AND user_id = ?", input.Patch.CanvasID, input.UserID).Error; err != nil {
		return ErrFilmGenerationTargetChanged
	}
	if canvas.ProjectID != input.ProjectID || canvas.PayloadJSON != input.CanvasPayloadJSON {
		return ErrFilmGenerationTargetChanged
	}

	var artifact model.FilmArtifact
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&artifact, "id = ? AND project_id = ?", input.GenerationRequestArtifact.ID, input.ProjectID).Error; err != nil {
		return ErrFilmGenerationTargetChanged
	}
	expected := input.GenerationRequestArtifact
	if artifact.ArtifactType != "generation_request" || artifact.ObjectVersion != expected.ObjectVersion || artifact.Status != expected.Status || artifact.ShotID != expected.ShotID || artifact.SceneID != expected.SceneID || artifact.UnitID != expected.UnitID || artifact.PayloadJSON != expected.PayloadJSON || artifact.SourceRefsJSON != expected.SourceRefsJSON || (artifact.Status != "ready" && artifact.Status != "locked") {
		return ErrFilmGenerationTargetChanged
	}
	if input.SceneID != "" {
		if expected.SceneID != input.SceneID || input.SpatialGateArtifactID == "" || input.SpatialGateArtifactVersion < 1 {
			return ErrFilmGenerationTargetChanged
		}
		var scene model.Scene
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&scene, "id = ? AND project_id = ?", input.SceneID, input.ProjectID).Error; err != nil {
			return ErrFilmGenerationTargetChanged
		}
		var gate model.FilmArtifact
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&gate, "id = ? AND project_id = ? AND scope = ? AND scope_id = ? AND artifact_type = ?", input.SpatialGateArtifactID, input.ProjectID, model.FilmArtifactScopeScene, input.SceneID, "spatial_continuity_gate").Error; err != nil {
			return ErrFilmGenerationTargetChanged
		}
		if gate.ObjectVersion != input.SpatialGateArtifactVersion || gate.Status != "ready" {
			return ErrFilmGenerationTargetChanged
		}
		var gatePayload struct {
			Status              string `json:"status"`
			PackArtifactID      string `json:"packArtifactId"`
			PackArtifactVersion int    `json:"packArtifactVersion"`
			PackVersion         int    `json:"packVersion"`
		}
		if err := json.Unmarshal([]byte(gate.PayloadJSON), &gatePayload); err != nil || gatePayload.Status != "PASS" {
			return ErrFilmGenerationTargetChanged
		}
		var latestPack model.FilmArtifact
		if err := tx.Where("project_id = ? AND scope = ? AND scope_id = ? AND artifact_type = ?", input.ProjectID, model.FilmArtifactScopeScene, input.SceneID, "scene_asset_pack").Order("object_version desc").First(&latestPack).Error; err != nil || (latestPack.Status != "ready" && latestPack.Status != "locked") || gatePayload.PackArtifactID != latestPack.ID || gatePayload.PackArtifactVersion != latestPack.ObjectVersion || gatePayload.PackVersion != latestPack.ObjectVersion {
			return ErrFilmGenerationTargetChanged
		}
	}
	return nil
}

func lockFilmGenerationRoute(tx *gorm.DB, input FilmGenerationAtomicCreateInput) error {
	var channel model.ModelChannel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&channel, "id = ? AND scope = ? AND enabled = ?", input.Channel.ID, model.ChannelScopeSystem, true).Error; err != nil {
		return ErrFilmGenerationRouteChanged
	}
	expectedChannel := input.Channel
	if channel.BaseURL != expectedChannel.BaseURL || channel.APIKey != expectedChannel.APIKey || channel.SecretKey != expectedChannel.SecretKey || channel.APIFormat != expectedChannel.APIFormat || channel.HeadersJSON != expectedChannel.HeadersJSON || channel.ModelsJSON != expectedChannel.ModelsJSON {
		return ErrFilmGenerationRouteChanged
	}

	var channelModel model.ChannelModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&channelModel, "id = ? AND channel_id = ? AND model_key = ? AND enabled = ?", input.ChannelModel.ID, channel.ID, input.ChannelModel.ModelKey, true).Error; err != nil {
		return ErrFilmGenerationRouteChanged
	}
	expectedModel := input.ChannelModel
	if channelModel.Capability != expectedModel.Capability || channelModel.Protocol != expectedModel.Protocol || channelModel.BillingMode != expectedModel.BillingMode || channelModel.UnitPriceMicrocredits != expectedModel.UnitPriceMicrocredits || channelModel.InputTokenPriceMicrocredits != expectedModel.InputTokenPriceMicrocredits || channelModel.OutputTokenPriceMicrocredits != expectedModel.OutputTokenPriceMicrocredits || channelModel.CachedTokenPriceMicrocredits != expectedModel.CachedTokenPriceMicrocredits || channelModel.PriceConfigured != expectedModel.PriceConfigured || channelModel.PriceVersion != expectedModel.PriceVersion || channelModel.CapabilityConfigJSON != expectedModel.CapabilityConfigJSON || channelModel.CapabilityVersion != expectedModel.CapabilityVersion || !channelModel.PriceConfigured {
		return ErrFilmGenerationRouteChanged
	}
	if input.BillingOrder != nil && (input.BillingOrder.ChannelID != channel.ID || input.BillingOrder.ChannelModelID != channelModel.ID || input.BillingOrder.Model != channelModel.ModelKey || input.BillingOrder.Capability != channelModel.Capability || input.BillingOrder.BillingMode != channelModel.BillingMode || input.BillingOrder.PriceVersion != channelModel.PriceVersion || input.BillingOrder.UnitPriceMicrocredits != channelModel.UnitPriceMicrocredits) {
		return ErrFilmGenerationRouteChanged
	}
	return nil
}

func sameFilmGenerationTarget(existing model.CanvasProjectionPatch, expected model.CanvasProjectionPatch) bool {
	return existing.TargetProjectID == expected.TargetProjectID && existing.TargetArtifactID == expected.TargetArtifactID && existing.TargetArtifactVersion == expected.TargetArtifactVersion
}

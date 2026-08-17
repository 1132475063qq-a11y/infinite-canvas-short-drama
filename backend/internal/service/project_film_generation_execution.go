package service

import (
	"errors"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"
)

const FilmGenerationExecutionHistorySchemaVersion = 1

type FilmGenerationExecutionHistory struct {
	SchemaVersion                    int                              `json:"schemaVersion"`
	ProjectID                        string                           `json:"projectId"`
	GenerationRequestArtifactID      string                           `json:"generationRequestArtifactId"`
	GenerationRequestArtifactVersion int                              `json:"generationRequestArtifactVersion"`
	Attempts                         []FilmGenerationAttemptExecution `json:"attempts"`
}

type FilmGenerationAttemptExecution struct {
	ID                               string                         `json:"id"`
	TaskID                           string                         `json:"taskId"`
	AttemptNumber                    int                            `json:"attemptNumber"`
	GenerationRequestArtifactVersion int                            `json:"generationRequestArtifactVersion"`
	CanvasID                         string                         `json:"canvasId"`
	CanvasNodeID                     string                         `json:"canvasNodeId"`
	UnitID                           string                         `json:"unitId,omitempty"`
	SceneID                          string                         `json:"sceneId,omitempty"`
	ShotID                           string                         `json:"shotId,omitempty"`
	RequestFingerprint               string                         `json:"requestFingerprint"`
	BillingOrderID                   string                         `json:"billingOrderId,omitempty"`
	ChannelID                        string                         `json:"channelId"`
	ChannelModelID                   string                         `json:"channelModelId"`
	Model                            string                         `json:"model"`
	Capability                       string                         `json:"capability"`
	Protocol                         string                         `json:"protocol"`
	CapabilityVersion                int64                          `json:"capabilityVersion"`
	PriceVersion                     int64                          `json:"priceVersion"`
	Status                           model.GenerationAttemptStatus `json:"status"`
	Error                            string                         `json:"error,omitempty"`
	StartedAt                        *time.Time                     `json:"startedAt,omitempty"`
	CompletedAt                      *time.Time                     `json:"completedAt,omitempty"`
	CreatedAt                        time.Time                      `json:"createdAt"`
	UpdatedAt                        time.Time                      `json:"updatedAt"`
	ProviderJobs                     []FilmProviderJobExecution     `json:"providerJobs"`
	Results                          []FilmGenerationResult         `json:"results"`
}

type FilmProviderJobExecution struct {
	ID                string                  `json:"id"`
	ProviderRequestID string                  `json:"providerRequestId"`
	Status            model.ProviderJobStatus `json:"status"`
	ProviderStatus    string                  `json:"providerStatus,omitempty"`
	PollStage         string                  `json:"pollStage,omitempty"`
	LastError         string                  `json:"lastError,omitempty"`
	FirstObservedAt   time.Time               `json:"firstObservedAt"`
	LastObservedAt    time.Time               `json:"lastObservedAt"`
	CompletedAt       *time.Time              `json:"completedAt,omitempty"`
}

type FilmGenerationResult struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	URL       string    `json:"url,omitempty"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"createdAt"`
}

// filmGenerationAttemptFromRuntimeInput freezes the exact backend-resolved
// request and route used by one queue execution. It accepts the public Task
// input only; provider credentials are deliberately not part of this record.
func filmGenerationAttemptFromRuntimeInput(task model.Task, input map[string]any, now time.Time) (*model.GenerationAttempt, error) {
	if task.Provider != model.TaskProviderFilmGateway || task.ID == "" || task.UserID == "" {
		return nil, errors.New("影视生成任务缺少受控 Provider Gateway 标识")
	}
	metadata, ok := input["metadata"].(map[string]any)
	if !ok {
		return nil, errors.New("影视生成任务缺少执行元数据")
	}
	route, ok := metadata["providerRoute"].(map[string]any)
	if !ok {
		return nil, errors.New("影视生成任务缺少 Provider 路由快照")
	}
	artifactVersion := int(firstInt64(metadata, "generationRequestArtifactVersion"))
	attempt := &model.GenerationAttempt{
		ID:                               newID(),
		UserID:                           task.UserID,
		TaskID:                           task.ID,
		AttemptNumber:                    1,
		DomainProjectID:                  metadataString(metadata, "domainProjectId"),
		CanvasID:                         metadataString(metadata, "canvasId"),
		CanvasNodeID:                     metadataString(metadata, "canvasNodeId"),
		UnitID:                           metadataString(metadata, "unitId"),
		SceneID:                          metadataString(metadata, "sceneId"),
		ShotID:                           metadataString(metadata, "shotId"),
		GenerationRequestArtifactID:      metadataString(metadata, "generationRequestArtifactId"),
		GenerationRequestArtifactVersion: artifactVersion,
		RequestFingerprint:               metadataString(metadata, "requestFingerprint"),
		BillingOrderID:                   task.BillingOrderID,
		ChannelID:                        metadataString(route, "channelId"),
		ChannelModelID:                   metadataString(route, "channelModelId"),
		Model:                            strings.TrimPrefix(metadataString(route, "model"), "models/"),
		Capability:                       metadataString(route, "capability"),
		Protocol:                         metadataString(route, "protocol"),
		CapabilityVersion:                firstInt64(route, "capabilityVersion"),
		PriceVersion:                     firstInt64(route, "priceVersion"),
		Status:                           model.GenerationAttemptStatusQueued,
		CreatedAt:                        now,
		UpdatedAt:                        now,
	}
	if attempt.DomainProjectID == "" || attempt.CanvasID == "" || attempt.CanvasNodeID == "" || attempt.GenerationRequestArtifactID == "" || attempt.GenerationRequestArtifactVersion < 1 || attempt.RequestFingerprint == "" || attempt.ChannelID == "" || attempt.ChannelModelID == "" || attempt.Model == "" || attempt.Capability == "" || attempt.Protocol == "" {
		return nil, errors.New("影视生成任务的 Generation Request 或 Provider 路由快照不完整")
	}
	if task.DomainProjectID != attempt.DomainProjectID || task.CanvasID != attempt.CanvasID || strings.TrimPrefix(task.Model, "models/") != attempt.Model {
		return nil, errors.New("影视生成任务与 Attempt 路由快照不一致")
	}
	return attempt, nil
}

func providerJobStatusFromAPICall(log model.ApiCallLog) model.ProviderJobStatus {
	raw := strings.ToLower(strings.TrimSpace(log.ProviderStatus))
	switch raw {
	case "success", "succeeded", "completed", "complete", "done":
		return model.ProviderJobStatusSucceeded
	case "failed", "failure", "error", "expired":
		return model.ProviderJobStatusFailed
	case "cancelled", "canceled":
		return model.ProviderJobStatusCancelled
	case "running", "processing", "in_progress", "in-progress":
		return model.ProviderJobStatusRunning
	case "queued", "pending", "accepted", "submitted":
		return model.ProviderJobStatusAccepted
	}
	if log.Status == model.ApiCallStatusFailed {
		if log.StatusCode >= 400 && log.StatusCode < 500 && log.StatusCode != 408 {
			return model.ProviderJobStatusFailed
		}
		return model.ProviderJobStatusUncertain
	}
	switch strings.ToLower(strings.TrimSpace(log.RequestKind)) {
	case "download":
		return model.ProviderJobStatusSucceeded
	case "poll":
		return model.ProviderJobStatusRunning
	default:
		return model.ProviderJobStatusAccepted
	}
}

// ProjectFilmGenerationExecutions returns backend facts only. It never reads
// execution IDs from the browser Canvas document and never exposes provider
// credentials, endpoint configuration or raw API call bodies.
func (s *Service) ProjectFilmGenerationExecutions(userID string, projectID string, artifactID string) (FilmGenerationExecutionHistory, error) {
	if err := s.RequireFeature(FeatureShortDrama); err != nil {
		return FilmGenerationExecutionHistory{}, err
	}
	projectID = strings.TrimSpace(projectID)
	artifactID = strings.TrimSpace(artifactID)
	project, err := s.repo.ProjectForUser(userID, projectID)
	if err != nil {
		return FilmGenerationExecutionHistory{}, err
	}
	if project.Type != model.ProjectTypeShortDrama {
		return FilmGenerationExecutionHistory{}, BadAuthRequest("生成执行历史只适用于短剧项目")
	}
	artifact, err := s.repo.FilmArtifactForProject(project.ID, artifactID)
	if err != nil {
		return FilmGenerationExecutionHistory{}, err
	}
	if artifact.ArtifactType != FilmArtifactTypeGenerationRequest {
		return FilmGenerationExecutionHistory{}, BadAuthRequest("指定 Artifact 不是 Generation Request")
	}
	rows, err := s.repo.FilmGenerationExecutionRows(userID, project.ID, artifact.ID)
	if err != nil {
		return FilmGenerationExecutionHistory{}, err
	}
	history := FilmGenerationExecutionHistory{
		SchemaVersion: FilmGenerationExecutionHistorySchemaVersion,
		ProjectID: project.ID,
		GenerationRequestArtifactID: artifact.ID,
		GenerationRequestArtifactVersion: artifact.ObjectVersion,
		Attempts: make([]FilmGenerationAttemptExecution, 0, len(rows.Attempts)),
	}
	attemptIndex := make(map[string]int, len(rows.Attempts))
	for _, attempt := range rows.Attempts {
		attemptIndex[attempt.ID] = len(history.Attempts)
		history.Attempts = append(history.Attempts, FilmGenerationAttemptExecution{
			ID: attempt.ID, TaskID: attempt.TaskID, AttemptNumber: attempt.AttemptNumber,
			GenerationRequestArtifactVersion: attempt.GenerationRequestArtifactVersion,
			CanvasID: attempt.CanvasID, CanvasNodeID: attempt.CanvasNodeID,
			UnitID: attempt.UnitID, SceneID: attempt.SceneID, ShotID: attempt.ShotID,
			RequestFingerprint: attempt.RequestFingerprint, BillingOrderID: attempt.BillingOrderID,
			ChannelID: attempt.ChannelID, ChannelModelID: attempt.ChannelModelID,
			Model: attempt.Model, Capability: attempt.Capability, Protocol: attempt.Protocol,
			CapabilityVersion: attempt.CapabilityVersion, PriceVersion: attempt.PriceVersion,
			Status: attempt.Status, Error: attempt.Error,
			StartedAt: attempt.StartedAt, CompletedAt: attempt.CompletedAt,
			CreatedAt: attempt.CreatedAt, UpdatedAt: attempt.UpdatedAt,
			ProviderJobs: []FilmProviderJobExecution{}, Results: []FilmGenerationResult{},
		})
	}
	for _, job := range rows.ProviderJobs {
		index, ok := attemptIndex[job.GenerationAttemptID]
		if !ok {
			continue
		}
		history.Attempts[index].ProviderJobs = append(history.Attempts[index].ProviderJobs, FilmProviderJobExecution{
			ID: job.ID, ProviderRequestID: job.ProviderRequestID, Status: job.Status,
			ProviderStatus: job.ProviderStatus, PollStage: job.PollStage, LastError: job.LastError,
			FirstObservedAt: job.FirstObservedAt, LastObservedAt: job.LastObservedAt, CompletedAt: job.CompletedAt,
		})
	}
	for _, result := range rows.Results {
		index, ok := attemptIndex[result.AttemptID]
		if !ok {
			continue
		}
		history.Attempts[index].Results = append(history.Attempts[index].Results, FilmGenerationResult{
			ID: result.ID, Kind: result.Kind, URL: result.URL, Payload: result.Payload, CreatedAt: result.CreatedAt,
		})
	}
	return history, nil
}

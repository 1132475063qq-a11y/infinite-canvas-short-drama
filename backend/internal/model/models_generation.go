package model

import "time"

// GenerationAttempt is one backend-owned execution attempt of a versioned
// Film Generation Request. Task remains the queue item; this row preserves the
// retry/lease-recovery history that a mutable Task cannot represent.
type GenerationAttempt struct {
	ID                               string                  `json:"id" gorm:"primaryKey;size:36"`
	UserID                           string                  `json:"userId" gorm:"index;size:36"`
	TaskID                           string                  `json:"taskId" gorm:"index;size:36;uniqueIndex:idx_generation_attempt_number,priority:1"`
	AttemptNumber                    int                     `json:"attemptNumber" gorm:"uniqueIndex:idx_generation_attempt_number,priority:2"`
	DomainProjectID                  string                  `json:"domainProjectId" gorm:"index;size:36"`
	CanvasID                         string                  `json:"canvasId" gorm:"index;size:80"`
	CanvasNodeID                     string                  `json:"canvasNodeId" gorm:"index;size:120"`
	UnitID                           string                  `json:"unitId,omitempty" gorm:"index;size:36"`
	SceneID                          string                  `json:"sceneId,omitempty" gorm:"index;size:36"`
	ShotID                           string                  `json:"shotId,omitempty" gorm:"index;size:36"`
	GenerationRequestArtifactID      string                  `json:"generationRequestArtifactId" gorm:"index;size:36"`
	GenerationRequestArtifactVersion int                     `json:"generationRequestArtifactVersion"`
	RequestFingerprint               string                  `json:"requestFingerprint" gorm:"size:64"`
	BillingOrderID                   string                  `json:"billingOrderId,omitempty" gorm:"index;size:36"`
	ChannelID                        string                  `json:"channelId" gorm:"index;size:36"`
	ChannelModelID                   string                  `json:"channelModelId" gorm:"index;size:36"`
	Model                            string                  `json:"model" gorm:"size:120"`
	Capability                       string                  `json:"capability" gorm:"size:32"`
	Protocol                         string                  `json:"protocol" gorm:"size:32"`
	CapabilityVersion                int64                   `json:"capabilityVersion"`
	PriceVersion                     int64                   `json:"priceVersion"`
	Status                           GenerationAttemptStatus `json:"status" gorm:"index;size:24"`
	Error                            string                  `json:"error,omitempty" gorm:"type:text"`
	StartedAt                        *time.Time              `json:"startedAt,omitempty"`
	CompletedAt                      *time.Time              `json:"completedAt,omitempty"`
	CreatedAt                        time.Time               `json:"createdAt" gorm:"index"`
	UpdatedAt                        time.Time               `json:"updatedAt"`
}

// ProviderJob records only a real upstream job identity and normalized state.
// Request/response bodies stay in the existing redacted ApiCallLog.
type ProviderJob struct {
	ID                  string            `json:"id" gorm:"primaryKey;size:36"`
	UserID              string            `json:"userId" gorm:"index;size:36"`
	TaskID              string            `json:"taskId" gorm:"index;size:36"`
	GenerationAttemptID string            `json:"generationAttemptId" gorm:"index;size:36;uniqueIndex:idx_provider_job_attempt_request,priority:1"`
	ProviderRequestID   string            `json:"providerRequestId" gorm:"index;size:160;uniqueIndex:idx_provider_job_attempt_request,priority:2"`
	ChannelID           string            `json:"channelId" gorm:"index;size:36"`
	ChannelModelID      string            `json:"channelModelId" gorm:"index;size:36"`
	Model               string            `json:"model" gorm:"size:120"`
	Capability          string            `json:"capability" gorm:"size:32"`
	Protocol            string            `json:"protocol" gorm:"size:32"`
	Status              ProviderJobStatus `json:"status" gorm:"index;size:32"`
	ProviderStatus      string            `json:"providerStatus,omitempty" gorm:"size:64"`
	PollStage           string            `json:"pollStage,omitempty" gorm:"size:32"`
	LastError           string            `json:"lastError,omitempty" gorm:"type:text"`
	FirstObservedAt     time.Time         `json:"firstObservedAt"`
	LastObservedAt      time.Time         `json:"lastObservedAt" gorm:"index"`
	CompletedAt         *time.Time        `json:"completedAt,omitempty"`
	CreatedAt           time.Time         `json:"createdAt"`
	UpdatedAt           time.Time         `json:"updatedAt"`
}

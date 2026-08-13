package model

import "time"

// EcommerceArtifact 是电商创意域的独立事实源。
// 它不能复用 FilmArtifact，因为两条 Domain 的版本、Skill 和 QA 语义不同。
type EcommerceArtifact struct {
	ID                 string    `json:"id" gorm:"primaryKey;size:36"`
	ProjectID          string    `json:"projectId" gorm:"index;size:36;uniqueIndex:idx_ecommerce_artifact_version,priority:1"`
	ArtifactKey        string    `json:"artifactKey" gorm:"size:180;uniqueIndex:idx_ecommerce_artifact_version,priority:2"`
	ArtifactType       string    `json:"artifactType" gorm:"index;size:64;uniqueIndex:idx_ecommerce_artifact_version,priority:3"`
	SchemaVersion      int       `json:"schemaVersion" gorm:"index"`
	Revision           int       `json:"revision" gorm:"uniqueIndex:idx_ecommerce_artifact_version,priority:4"`
	Lifecycle          string    `json:"lifecycle" gorm:"index;size:24"`
	Evidence           string    `json:"evidence" gorm:"index;size:24"`
	ResponsibleAgentID string    `json:"responsibleAgentId,omitempty" gorm:"index;size:80"`
	SkillRef           string    `json:"skillRef,omitempty" gorm:"size:160"`
	PayloadJSON        string    `json:"payloadJson" gorm:"type:text"`
	SourceRefsJSON     string    `json:"sourceRefsJson" gorm:"type:text"`
	AuthorityRefsJSON  string    `json:"authorityRefsJson" gorm:"type:text"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

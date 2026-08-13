package service

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
)

const (
	FilmArtifactTypeActing          = "acting"
	FilmArtifactTypeVideoPromptPack = "video_prompt_pack"
)

type SaveProjectFilmArtifactRequest struct {
	ShotID             string         `json:"shotId"`
	ArtifactType       string         `json:"artifactType"`
	Status             string         `json:"status"`
	ResponsibleAgentID string         `json:"responsibleAgentId"`
	Payload            map[string]any `json:"payload"`
}

func (s *Service) SaveProjectFilmArtifact(userID string, projectID string, req SaveProjectFilmArtifactRequest) (model.FilmArtifact, error) {
	if _, err := s.repo.ProjectForUser(userID, projectID); err != nil {
		return model.FilmArtifact{}, err
	}
	shotID := strings.TrimSpace(req.ShotID)
	if shotID == "" {
		return model.FilmArtifact{}, BadAuthRequest("影视 Artifact 必须关联镜头")
	}
	shot, err := s.repo.ShotForProject(projectID, shotID)
	if err != nil {
		return model.FilmArtifact{}, err
	}
	artifactType := strings.TrimSpace(req.ArtifactType)
	if err := validateFilmArtifactPayload(artifactType, req.Payload); err != nil {
		return model.FilmArtifact{}, err
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "draft"
	}
	if !validFilmArtifactStatus(status) {
		return model.FilmArtifact{}, BadAuthRequest("不支持的影视 Artifact 状态")
	}
	payloadJSON, err := json.Marshal(req.Payload)
	if err != nil {
		return model.FilmArtifact{}, BadAuthRequest("影视 Artifact 内容格式无效")
	}
	sourceRefs, err := s.filmArtifactSourceRefs(projectID, *shot, artifactType)
	if err != nil {
		return model.FilmArtifact{}, err
	}
	sourceRefsJSON, err := json.Marshal(sourceRefs)
	if err != nil {
		return model.FilmArtifact{}, err
	}
	now := time.Now()
	artifact := model.FilmArtifact{
		ID:                 newID(),
		ProjectID:          projectID,
		UnitID:             shot.UnitID,
		SceneID:            shot.SceneID,
		ShotID:             shot.ID,
		ArtifactType:       artifactType,
		Status:             status,
		ResponsibleAgentID: strings.TrimSpace(req.ResponsibleAgentID),
		PayloadJSON:        string(payloadJSON),
		SourceRefsJSON:     string(sourceRefsJSON),
		AuthorityRefsJSON:  "[]",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.repo.SaveFilmArtifactVersion(&artifact); err != nil {
		return model.FilmArtifact{}, err
	}
	return artifact, nil
}

func (s *Service) filmArtifactSourceRefs(projectID string, shot model.Shot, artifactType string) ([]string, error) {
	refs := make([]string, 0, 8)
	if shot.ContractArtifactID != "" {
		refs = append(refs, shot.ContractArtifactID)
	}
	if artifactType == FilmArtifactTypeVideoPromptPack {
		acting, err := s.repo.LatestFilmArtifact(projectID, shot.ID, FilmArtifactTypeActing)
		if err == nil {
			refs = append(refs, acting.ID)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		references, err := s.repo.ProjectShotAssetReferences(projectID)
		if err != nil {
			return nil, err
		}
		for _, reference := range references {
			if reference.ShotID == shot.ID {
				refs = append(refs, reference.AssetVersionID)
			}
		}
	}
	return refs, nil
}

func validateFilmArtifactPayload(artifactType string, payload map[string]any) error {
	if payload == nil {
		return BadAuthRequest("影视 Artifact 内容不能为空")
	}
	switch artifactType {
	case FilmArtifactTypeActing:
		for _, forbidden := range []string{"appearance", "physique", "clothing", "voice", "camera", "lens", "shotSize"} {
			if _, exists := payload[forbidden]; exists {
				return BadAuthRequest("Acting 不能修改外貌、声音或镜头领域事实")
			}
		}
		return nil
	case FilmArtifactTypeVideoPromptPack:
		return nil
	default:
		return BadAuthRequest("不支持的影视 Artifact 类型")
	}
}

func validFilmArtifactStatus(status string) bool {
	switch status {
	case "draft", "ready", "review", "locked", "superseded", "archived":
		return true
	default:
		return false
	}
}

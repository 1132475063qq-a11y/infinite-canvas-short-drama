package database

import (
	"infinite-canvas/backend/internal/model"

	"gorm.io/gorm"
)

// Models 是应用持久化表的唯一清单，服务启动和跨数据库迁移必须共用它。
func Models() []any {
	return []any{
		&model.User{},
		&model.AuthSession{},
		&model.UserIdentity{},
		&model.OAuthState{},
		&model.EmailVerificationCode{},
		&model.ModelChannel{},
		&model.ChannelModel{},
		&model.ApiCallLog{},
		&model.ModelPricing{},
		&model.CreditAccount{},
		&model.CreditLedgerEntry{},
		&model.BillingOrder{},
		&model.RedeemBatch{},
		&model.RedeemCode{},
		&model.AdminAuditEvent{},
		&model.UserDailyActivity{},
		&model.SystemSetting{},
		&model.UserOSSSetting{},
		&model.UserDailyUploadUsage{},
		&model.Skill{},
		&model.UserSkillState{},
		&model.Resource{},
		&model.Asset{},
		&model.ProjectAssetLink{},
		&model.ProjectAssetCandidate{},
		&model.AssetVersion{},
		&model.AssetRepresentation{},
		&model.VoiceProfile{},
		&model.CharacterVoiceBinding{},
		&model.Project{},
		&model.StyleProfile{},
		&model.ProjectUnit{},
		&model.CanvasUnitLink{},
		&model.Scene{},
		&model.Shot{},
		&model.FilmArtifact{},
		&model.EcommerceArtifact{},
		&model.ShotAssetReference{},
		&model.WorkflowTemplateVersion{},
		&model.WorkflowInstance{},
		&model.WorkflowStepInstance{},
		&model.WorkflowStepTask{},
		&model.CanvasProject{},
		&model.CanvasProjectionPatch{},
		&model.CanvasShare{},
		&model.PromptTemplate{},
		&model.UserPromptCustomization{},
		&model.Announcement{},
		&model.UserAnnouncementRead{},
		&model.Task{},
		&model.GenerationAttempt{},
		&model.ProviderJob{},
		&model.TaskTextDelta{},
		&model.Session{},
		&model.Message{},
		&model.TaskLog{},
		&model.SessionFile{},
		&model.Result{},
	}
}

func MigrateSchema(db *gorm.DB) error {
	// 旧表只保存 Updream 目录状态，与本地技能主键没有可迁移关系；首次升级时按产品要求清空重建。
	if db.Migrator().HasColumn(&model.UserSkillState{}, "skill_dir") && !db.Migrator().HasColumn(&model.UserSkillState{}, "skill_id") {
		if err := db.Migrator().DropTable(&model.UserSkillState{}); err != nil {
			return err
		}
	}
	if err := db.AutoMigrate(Models()...); err != nil {
		return err
	}
	// Film tasks historically overloaded tasks.project_id with a Canvas ID.
	// GenerationAttempt already stores both identities, so use its newest row
	// to backfill the explicit columns without rewriting the legacy API field.
	if db.Migrator().HasTable(&model.Task{}) && db.Migrator().HasTable(&model.GenerationAttempt{}) {
		var attempts []model.GenerationAttempt
		if err := db.Select("task_id", "attempt_number", "domain_project_id", "canvas_id").
			Where("domain_project_id <> '' AND canvas_id <> ''").
			Order("task_id asc, attempt_number desc").Find(&attempts).Error; err != nil {
			return err
		}
		seen := make(map[string]struct{}, len(attempts))
		for _, attempt := range attempts {
			if _, exists := seen[attempt.TaskID]; exists {
				continue
			}
			seen[attempt.TaskID] = struct{}{}
			if err := db.Model(&model.Task{}).
				Where("id = ? AND provider = ? AND (domain_project_id = '' OR canvas_id = '')", attempt.TaskID, model.TaskProviderFilmGateway).
				Updates(map[string]any{"domain_project_id": attempt.DomainProjectID, "canvas_id": attempt.CanvasID}).Error; err != nil {
				return err
			}
		}
	}
	// 逻辑删除后的同名模型允许重新添加，旧唯一索引不能继续覆盖已删除记录。
	if err := db.Exec("DROP INDEX IF EXISTS idx_channel_model_key").Error; err != nil {
		return err
	}
	if err := db.Exec("DROP INDEX IF EXISTS idx_users_email").Error; err != nil {
		return err
	}
	// Film artifacts now use an explicit scope (project/unit/scene/shot).
	// Remove the pre-scope unique index so old nullable ShotID semantics cannot
	// shadow scene-level versions during migration.
	if err := db.Exec("DROP INDEX IF EXISTS idx_film_artifact_version").Error; err != nil {
		return err
	}
	if err := db.Exec("DROP INDEX IF EXISTS idx_film_artifact_scope_version").Error; err != nil {
		return err
	}
	// Backfill the explicit scope before installing its unique index. This is
	// intentionally SQL rather than a Go loop so a large existing project is
	// migrated in one transaction-sized set of updates per scope.
	if db.Migrator().HasTable(&model.FilmArtifact{}) {
		for _, statement := range []string{
			"UPDATE film_artifacts SET scope = 'shot', scope_id = shot_id WHERE COALESCE(scope, '') = '' AND COALESCE(shot_id, '') <> ''",
			"UPDATE film_artifacts SET scope = 'scene', scope_id = scene_id WHERE COALESCE(scope, '') = '' AND COALESCE(scene_id, '') <> ''",
			"UPDATE film_artifacts SET scope = 'unit', scope_id = unit_id WHERE COALESCE(scope, '') = '' AND COALESCE(unit_id, '') <> ''",
			"UPDATE film_artifacts SET scope = 'project', scope_id = project_id WHERE COALESCE(scope, '') = ''",
		} {
			if err := db.Exec(statement).Error; err != nil {
				return err
			}
		}
		if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_film_artifact_scope_version ON film_artifacts(project_id, scope, scope_id, artifact_type, object_version)").Error; err != nil {
			return err
		}
	}
	return db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_nonempty ON users(lower(email)) WHERE email <> ''").Error
}

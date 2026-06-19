package svcmodels

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/LingByte/SoulNexus/pkg/logger"
)

// MigrateAgentsBoundJsTemplate backfills bound_js_template_source_id when legacy data
// stored a JS template id in agents.js_source_id (before loader id / template id split).
// For those rows a fresh loader client id is generated so loader.js URLs keep working.
func MigrateAgentsBoundJsTemplate(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("agents") {
		return nil
	}
	if !db.Migrator().HasColumn(&Agent{}, "BoundJsTemplateSourceID") {
		return nil
	}
	if !db.Migrator().HasTable((JSTemplate{}).TableName()) {
		return nil
	}

	var templateIDs []string
	if err := db.Model(&JSTemplate{}).Where("js_source_id <> ''").Pluck("js_source_id", &templateIDs).Error; err != nil {
		return err
	}
	if len(templateIDs) == 0 {
		return nil
	}

	var agents []Agent
	if err := db.Where("js_source_id IN ? AND (bound_js_template_source_id = '' OR bound_js_template_source_id IS NULL)", templateIDs).
		Find(&agents).Error; err != nil {
		return err
	}

	for _, agent := range agents {
		tplID := strings.TrimSpace(agent.JsSourceID)
		if tplID == "" {
			continue
		}
		newLoaderID := utils.SnowflakeUtil.GenID()
		if err := db.Model(&Agent{}).Where("id = ?", agent.ID).Updates(map[string]any{
			"bound_js_template_source_id": tplID,
			"js_source_id":                newLoaderID,
		}).Error; err != nil {
			logger.Warn("agent js template migrate",
				zap.Int64("agentId", agent.ID),
				zap.String("templateId", tplID),
				zap.Error(err),
			)
			continue
		}
		logger.Info("agent js template migrated",
			zap.Int64("agentId", agent.ID),
			zap.String("boundTemplateId", tplID),
			zap.String("newLoaderId", newLoaderID),
		)
	}
	return nil
}

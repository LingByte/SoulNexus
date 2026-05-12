package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// 历史遗留：部分老表（chat_messages / llm_usage 等）由早期建表 SQL 创建为
// utf8mb3 / utf8mb3_general_ci，导致写入 emoji 或 mb4 字符时 GORM 报：
//   Error 3988 (HY000): Conversion from collation utf8mb4_unicode_ci into
//   utf8mb3_general_ci impossible for parameter
// 这里在启动时对几个高风险表执行一次幂等的 ALTER TABLE 转换：utf8mb4 / utf8mb4_unicode_ci。
// 已经是 utf8mb4 的表会被 MySQL 直接跳过，几乎零代价。

import (
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/LingByte/SoulNexus/pkg/logger"
)

// 需要强制 utf8mb4 的高风险表（含用户文本 / Emoji）。
var charsetMigrateTables = []string{
	"chat_messages",
	"llm_usage",
	"llm_usage_user_daily",
	"llm_usage_user_model_daily",
	"speech_usage",
	"internal_notifications",
	"mail_logs",
	"sms_logs",
	"announcements",
	"operation_logs",
}

// MigrateTextCharset 把指定表（若存在且非 utf8mb4）整体转成 utf8mb4 / utf8mb4_unicode_ci。
// 仅在 MySQL 上生效；其他驱动直接跳过。
func MigrateTextCharset(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if name := db.Dialector.Name(); strings.ToLower(name) != "mysql" {
		return nil
	}
	mig := db.Migrator()
	for _, tbl := range charsetMigrateTables {
		if !mig.HasTable(tbl) {
			continue
		}
		var currentCharset string
		// 查表当前默认字符集（来自 INFORMATION_SCHEMA）。
		err := db.Raw(`
			SELECT ccsa.CHARACTER_SET_NAME
			FROM INFORMATION_SCHEMA.TABLES t
			JOIN INFORMATION_SCHEMA.COLLATION_CHARACTER_SET_APPLICABILITY ccsa
			  ON ccsa.COLLATION_NAME = t.TABLE_COLLATION
			WHERE t.TABLE_SCHEMA = DATABASE() AND t.TABLE_NAME = ?
		`, tbl).Scan(&currentCharset).Error
		if err != nil {
			logger.Warn("charset-migrate inspect table", zap.String("table", tbl), zap.Error(err))
			continue
		}
		if strings.EqualFold(currentCharset, "utf8mb4") {
			continue
		}
		// 走 ALTER TABLE ... CONVERT TO CHARACTER SET 把所有文本列一次性升级。
		stmt := "ALTER TABLE `" + tbl + "` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
		if err := db.Exec(stmt).Error; err != nil {
			logger.Warn("charset-migrate alter table",
				zap.String("table", tbl),
				zap.String("from", currentCharset),
				zap.Error(err),
			)
			continue
		}
		logger.Info("charset-migrate converted table",
			zap.String("table", tbl),
			zap.String("from", currentCharset),
			zap.String("to", "utf8mb4"),
		)
	}
	// 历史遗留：llm_usage.request_type 老 schema 是 varchar(20)，遇到 openapi_openai_chat_completions
	// (31 字符) 会触发 Error 1406 Data too long。这里在 GORM AutoMigrate 之外，额外强制加宽到 64 字符。
	if mig.HasTable("llm_usage") {
		var curLen int
		err := db.Raw(`
			SELECT CHARACTER_MAXIMUM_LENGTH
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'llm_usage' AND COLUMN_NAME = 'request_type'
		`).Scan(&curLen).Error
		if err == nil && curLen > 0 && curLen < 64 {
			if err := db.Exec("ALTER TABLE `llm_usage` MODIFY COLUMN `request_type` VARCHAR(64) NOT NULL").Error; err != nil {
				logger.Warn("widen llm_usage.request_type", zap.Int("from", curLen), zap.Error(err))
			} else {
				logger.Info("widen llm_usage.request_type", zap.Int("from", curLen), zap.Int("to", 64))
			}
		}
	}
	return nil
}

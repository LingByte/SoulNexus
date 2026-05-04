package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"

	"gorm.io/gorm"
)

// MigrateLegacyAssistantsNaming 将旧库 assistants / assistant_id 迁移为 agents / agent_id（幂等）。
func MigrateLegacyAssistantsNaming(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	switch db.Dialector.Name() {
	case "sqlite":
		return migrateAssistantsSQLite(db)
	case "mysql":
		return migrateAssistantsMySQL(db)
	case "postgres":
		return migrateAssistantsPostgres(db)
	default:
		return nil
	}
}

func migrateAssistantsSQLite(db *gorm.DB) error {
	var n int64
	if err := db.Raw(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='assistants'`).Scan(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		var agentsCount int64
		_ = db.Raw(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='agents'`).Scan(&agentsCount).Error
		if agentsCount == 0 {
			if err := db.Exec(`ALTER TABLE assistants RENAME TO agents`).Error; err != nil {
				return fmt.Errorf("rename assistants->agents: %w", err)
			}
		}
	}
	renameCol := func(table, from, to string) error {
		var cnt int64
		q := fmt.Sprintf(`SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name=?`, table)
		if err := db.Raw(q, from).Scan(&cnt).Error; err != nil {
			return err
		}
		if cnt == 0 {
			return nil
		}
		var existsTo int64
		q2 := fmt.Sprintf(`SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name=?`, table)
		if err := db.Raw(q2, to).Scan(&existsTo).Error; err != nil {
			return err
		}
		if existsTo > 0 {
			return nil
		}
		stmt := fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s`, table, from, to)
		return db.Exec(stmt).Error
	}
	for _, t := range []struct{ table, from, to string }{
		{"chat_sessions", "assistant_id", "agent_id"},
		{"usage_records", "assistant_id", "agent_id"},
		{"devices", "assistant_id", "agent_id"},
		{"voiceprints", "assistant_id", "agent_id"},
		{"call_recordings", "assistant_id", "agent_id"},
	} {
		if err := renameCol(t.table, t.from, t.to); err != nil {
			return fmt.Errorf("%s: %w", t.table, err)
		}
	}
	return nil
}

func migrateAssistantsMySQL(db *gorm.DB) error {
	var n int64
	if err := db.Raw(`
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = 'assistants'`).Scan(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		var agentsN int64
		_ = db.Raw(`
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = 'agents'`).Scan(&agentsN).Error
		if agentsN == 0 {
			if err := db.Exec(`RENAME TABLE assistants TO agents`).Error; err != nil {
				return fmt.Errorf("rename assistants->agents: %w", err)
			}
		}
	}
	renameCol := func(table, from, to string) error {
		var cnt int64
		if err := db.Raw(`
SELECT COUNT(*) FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`, table, from).Scan(&cnt).Error; err != nil {
			return err
		}
		if cnt == 0 {
			return nil
		}
		var existsTo int64
		if err := db.Raw(`
SELECT COUNT(*) FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`, table, to).Scan(&existsTo).Error; err != nil {
			return err
		}
		if existsTo > 0 {
			return nil
		}
		stmt := fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s`, table, from, to)
		return db.Exec(stmt).Error
	}
	for _, t := range []struct{ table, from, to string }{
		{"chat_sessions", "assistant_id", "agent_id"},
		{"usage_records", "assistant_id", "agent_id"},
		{"devices", "assistant_id", "agent_id"},
		{"voiceprints", "assistant_id", "agent_id"},
		{"call_recordings", "assistant_id", "agent_id"},
	} {
		if err := renameCol(t.table, t.from, t.to); err != nil {
			return fmt.Errorf("%s: %w", t.table, err)
		}
	}
	return nil
}

func migrateAssistantsPostgres(db *gorm.DB) error {
	var n int64
	if err := db.Raw(`
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = current_schema() AND table_name = 'assistants'`).Scan(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		var agentsN int64
		_ = db.Raw(`
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = current_schema() AND table_name = 'agents'`).Scan(&agentsN).Error
		if agentsN == 0 {
			if err := db.Exec(`ALTER TABLE assistants RENAME TO agents`).Error; err != nil {
				return fmt.Errorf("rename assistants->agents: %w", err)
			}
		}
	}
	renameCol := func(table, from, to string) error {
		var cnt int64
		if err := db.Raw(`
SELECT COUNT(*) FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?`, table, from).Scan(&cnt).Error; err != nil {
			return err
		}
		if cnt == 0 {
			return nil
		}
		var existsTo int64
		if err := db.Raw(`
SELECT COUNT(*) FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?`, table, to).Scan(&existsTo).Error; err != nil {
			return err
		}
		if existsTo > 0 {
			return nil
		}
		stmt := fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s`, table, from, to)
		return db.Exec(stmt).Error
	}
	for _, t := range []struct{ table, from, to string }{
		{"chat_sessions", "assistant_id", "agent_id"},
		{"usage_records", "assistant_id", "agent_id"},
		{"devices", "assistant_id", "agent_id"},
		{"voiceprints", "assistant_id", "agent_id"},
		{"call_recordings", "assistant_id", "agent_id"},
	} {
		if err := renameCol(t.table, t.from, t.to); err != nil {
			return fmt.Errorf("%s: %w", t.table, err)
		}
	}
	return nil
}

package svcmodels

import (
	"strings"

	"gorm.io/gorm"
)

// MigrateAgentsWidenTextColumns expands agents text columns before GORM AutoMigrate.
// Without this, shrinking to varchar(N) fails when legacy rows exceed the limit
// (MySQL Error 1406: Data too long for column).
func MigrateAgentsWidenTextColumns(db *gorm.DB) error {
	if db == nil || strings.ToLower(db.Dialector.Name()) != "mysql" {
		return nil
	}
	if !db.Migrator().HasTable("agents") {
		return nil
	}
	stmts := []string{
		"ALTER TABLE `agents` MODIFY COLUMN `description` TEXT",
		"ALTER TABLE `agents` MODIFY COLUMN `opening_statement` TEXT",
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			// Column may not exist yet on very old schemas; ignore unknown column.
			if strings.Contains(strings.ToLower(err.Error()), "unknown column") {
				continue
			}
			return err
		}
	}
	return nil
}

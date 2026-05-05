package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/LingByte/SoulNexus/pkg/logger"
)

// MigrateGroupTenancyResources ensures every account has a personal org and moves legacy
// user_id-scoped rows to group_id (+ created_by). Safe to run on every startup (idempotent).
func MigrateGroupTenancyResources(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	var users []User
	if err := db.Find(&users).Error; err != nil {
		return fmt.Errorf("list users for tenancy migrate: %w", err)
	}
	for i := range users {
		if _, err := EnsurePersonalGroupForUser(db, users[i].ID); err != nil {
			logger.Warn("ensure personal group", zap.Uint("userId", users[i].ID), zap.Error(err))
		}
	}

	personalOf := func(userID uint) uint {
		id, err := PersonalGroupIDForUser(db, userID)
		if err != nil || id == 0 {
			g, err := EnsurePersonalGroupForUser(db, userID)
			if err != nil {
				return 0
			}
			return g.ID
		}
		return id
	}

	steps := []func(*gorm.DB, func(uint) uint) error{
		migrateDevicesTenancy,
		migrateAgentsTenancy,
		migrateKnowledgeBasesTenancy,
		migrateWorkflowDefinitionsTenancy,
		migrateJSTemplatesTenancy,
		migrateVoiceTrainingTenancy,
		migrateVoiceClonesTenancy,
		migrateVoiceSynthesesTenancy,
		migrateCallRecordingsTenancy,
		migrateUserCredentialsTenancy,
		migrateWorkflowPluginsTenancy,
		migrateWorkflowPluginInstallationsTenancy,
	}

	for _, step := range steps {
		if err := step(db, personalOf); err != nil {
			logger.Warn("group tenancy migrate step", zap.Error(err))
		}
	}
	return nil
}

type legacyDeviceRow struct {
	ID      string `gorm:"column:id"`
	UserID  uint   `gorm:"column:user_id"`
	GroupID *uint  `gorm:"column:group_id"`
}

func (legacyDeviceRow) TableName() string { return (&Device{}).TableName() }

func migrateDevicesTenancy(db *gorm.DB, personalOf func(uint) uint) error {
	if !db.Migrator().HasTable((&Device{}).TableName()) {
		return nil
	}
	if !db.Migrator().HasColumn(&legacyDeviceRow{}, "UserID") {
		return nil
	}
	var rows []legacyDeviceRow
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		gid := uint(0)
		if r.GroupID != nil && *r.GroupID != 0 {
			gid = *r.GroupID
		} else {
			gid = personalOf(r.UserID)
		}
		if gid == 0 {
			continue
		}
		if err := db.Model(&Device{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
			"group_id":    gid,
			"created_by": r.UserID,
		}).Error; err != nil {
			return err
		}
	}
	return db.Migrator().DropColumn(&legacyDeviceRow{}, "UserID")
}

type legacyAgentRow struct {
	ID      int64  `gorm:"column:id"`
	UserID  uint   `gorm:"column:user_id"`
	GroupID *uint  `gorm:"column:group_id"`
}

func (legacyAgentRow) TableName() string { return (&Agent{}).TableName() }

func migrateAgentsTenancy(db *gorm.DB, personalOf func(uint) uint) error {
	if !db.Migrator().HasTable((&Agent{}).TableName()) {
		return nil
	}
	if !db.Migrator().HasColumn(&legacyAgentRow{}, "UserID") {
		return nil
	}
	var rows []legacyAgentRow
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		gid := uint(0)
		if r.GroupID != nil && *r.GroupID != 0 {
			gid = *r.GroupID
		} else {
			gid = personalOf(r.UserID)
		}
		if gid == 0 {
			continue
		}
		if err := db.Model(&Agent{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
			"group_id":    gid,
			"created_by": r.UserID,
		}).Error; err != nil {
			return err
		}
	}
	return db.Migrator().DropColumn(&legacyAgentRow{}, "UserID")
}

type legacyKBRow struct {
	ID      uint   `gorm:"column:id"`
	UserID  uint   `gorm:"column:user_id"`
	GroupID *uint  `gorm:"column:group_id"`
}

func (legacyKBRow) TableName() string { return (&KnowledgeBase{}).TableName() }

func migrateKnowledgeBasesTenancy(db *gorm.DB, personalOf func(uint) uint) error {
	if !db.Migrator().HasTable((&KnowledgeBase{}).TableName()) {
		return nil
	}
	if !db.Migrator().HasColumn(&legacyKBRow{}, "UserID") {
		return nil
	}
	var rows []legacyKBRow
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		gid := uint(0)
		if r.GroupID != nil && *r.GroupID != 0 {
			gid = *r.GroupID
		} else {
			gid = personalOf(r.UserID)
		}
		if gid == 0 {
			continue
		}
		if err := db.Model(&KnowledgeBase{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
			"group_id":    gid,
			"created_by": r.UserID,
		}).Error; err != nil {
			return err
		}
	}
	return db.Migrator().DropColumn(&legacyKBRow{}, "UserID")
}

type legacyWFRow struct {
	ID      uint   `gorm:"column:id"`
	UserID  uint   `gorm:"column:user_id"`
	GroupID *uint  `gorm:"column:group_id"`
}

func (legacyWFRow) TableName() string { return "workflow_definitions" }

func migrateWorkflowDefinitionsTenancy(db *gorm.DB, personalOf func(uint) uint) error {
	if !db.Migrator().HasTable("workflow_definitions") {
		return nil
	}
	if !db.Migrator().HasColumn(&legacyWFRow{}, "UserID") {
		return nil
	}
	var rows []legacyWFRow
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		gid := uint(0)
		if r.GroupID != nil && *r.GroupID != 0 {
			gid = *r.GroupID
		} else {
			gid = personalOf(r.UserID)
		}
		if gid == 0 {
			continue
		}
		if err := db.Model(&WorkflowDefinition{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
			"group_id":     gid,
			"creator_uid": r.UserID,
		}).Error; err != nil {
			return err
		}
	}
	return db.Migrator().DropColumn(&legacyWFRow{}, "UserID")
}

type legacyJSRow struct {
	ID     string `gorm:"column:id"`
	UserID uint   `gorm:"column:user_id"`
	GroupID *uint `gorm:"column:group_id"`
}

func (legacyJSRow) TableName() string { return (&JSTemplate{}).TableName() }

func migrateJSTemplatesTenancy(db *gorm.DB, personalOf func(uint) uint) error {
	if !db.Migrator().HasTable((&JSTemplate{}).TableName()) {
		return nil
	}
	if !db.Migrator().HasColumn(&legacyJSRow{}, "UserID") {
		return nil
	}
	var rows []legacyJSRow
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		gid := uint(0)
		if r.GroupID != nil && *r.GroupID != 0 {
			gid = *r.GroupID
		} else {
			gid = personalOf(r.UserID)
		}
		if gid == 0 {
			continue
		}
		if err := db.Model(&JSTemplate{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
			"group_id":    gid,
			"created_by": r.UserID,
		}).Error; err != nil {
			return err
		}
	}
	return db.Migrator().DropColumn(&legacyJSRow{}, "UserID")
}

type legacyVTTaskRow struct {
	ID      uint   `gorm:"column:id"`
	UserID  uint   `gorm:"column:user_id"`
	GroupID *uint  `gorm:"column:group_id"`
}

func (legacyVTTaskRow) TableName() string { return "voice_training_tasks" }

func migrateVoiceTrainingTenancy(db *gorm.DB, personalOf func(uint) uint) error {
	if !db.Migrator().HasTable("voice_training_tasks") {
		return nil
	}
	if !db.Migrator().HasColumn(&legacyVTTaskRow{}, "UserID") {
		return nil
	}
	var rows []legacyVTTaskRow
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		gid := uint(0)
		if r.GroupID != nil && *r.GroupID != 0 {
			gid = *r.GroupID
		} else {
			gid = personalOf(r.UserID)
		}
		if gid == 0 {
			continue
		}
		if err := db.Model(&VoiceTrainingTask{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
			"group_id":    gid,
			"created_by": r.UserID,
		}).Error; err != nil {
			return err
		}
	}
	return db.Migrator().DropColumn(&legacyVTTaskRow{}, "UserID")
}

type legacyVCloneRow struct {
	ID      uint   `gorm:"column:id"`
	UserID  uint   `gorm:"column:user_id"`
	GroupID *uint  `gorm:"column:group_id"`
}

func (legacyVCloneRow) TableName() string { return "voice_clones" }

func migrateVoiceClonesTenancy(db *gorm.DB, personalOf func(uint) uint) error {
	if !db.Migrator().HasTable("voice_clones") {
		return nil
	}
	if !db.Migrator().HasColumn(&legacyVCloneRow{}, "UserID") {
		return nil
	}
	var rows []legacyVCloneRow
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		gid := uint(0)
		if r.GroupID != nil && *r.GroupID != 0 {
			gid = *r.GroupID
		} else {
			gid = personalOf(r.UserID)
		}
		if gid == 0 {
			continue
		}
		if err := db.Model(&VoiceClone{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
			"group_id":    gid,
			"created_by": r.UserID,
		}).Error; err != nil {
			return err
		}
	}
	return db.Migrator().DropColumn(&legacyVCloneRow{}, "UserID")
}

type legacyVSynthRow struct {
	ID     uint `gorm:"column:id"`
	UserID uint `gorm:"column:user_id"`
}

func (legacyVSynthRow) TableName() string { return "voice_syntheses" }

func migrateVoiceSynthesesTenancy(db *gorm.DB, personalOf func(uint) uint) error {
	if !db.Migrator().HasTable("voice_syntheses") {
		return nil
	}
	if !db.Migrator().HasColumn(&legacyVSynthRow{}, "UserID") {
		return nil
	}
	var rows []legacyVSynthRow
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		gid := personalOf(r.UserID)
		if gid == 0 {
			continue
		}
		if err := db.Model(&VoiceSynthesis{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
			"group_id":    gid,
			"created_by": r.UserID,
		}).Error; err != nil {
			return err
		}
	}
	return db.Migrator().DropColumn(&legacyVSynthRow{}, "UserID")
}

type legacyCallRecRow struct {
	ID     uint `gorm:"column:id"`
	UserID uint `gorm:"column:user_id"`
}

func (legacyCallRecRow) TableName() string { return (&CallRecording{}).TableName() }

func migrateCallRecordingsTenancy(db *gorm.DB, personalOf func(uint) uint) error {
	if !db.Migrator().HasTable((&CallRecording{}).TableName()) {
		return nil
	}
	if !db.Migrator().HasColumn(&legacyCallRecRow{}, "UserID") {
		return nil
	}
	var rows []legacyCallRecRow
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		gid := personalOf(r.UserID)
		if gid == 0 {
			continue
		}
		if err := db.Model(&CallRecording{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
			"group_id":    gid,
			"created_by": r.UserID,
		}).Error; err != nil {
			return err
		}
	}
	return db.Migrator().DropColumn(&legacyCallRecRow{}, "UserID")
}

type legacyCredRow struct {
	ID     uint `gorm:"column:id"`
	UserID uint `gorm:"column:user_id"`
}

func (legacyCredRow) TableName() string { return (&UserCredential{}).TableName() }

func migrateUserCredentialsTenancy(db *gorm.DB, personalOf func(uint) uint) error {
	if !db.Migrator().HasTable((&UserCredential{}).TableName()) {
		return nil
	}
	if !db.Migrator().HasColumn(&legacyCredRow{}, "UserID") {
		return nil
	}
	var rows []legacyCredRow
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		gid := personalOf(r.UserID)
		if gid == 0 {
			continue
		}
		if err := db.Model(&UserCredential{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
			"group_id":    gid,
			"created_by": r.UserID,
		}).Error; err != nil {
			return err
		}
	}
	return db.Migrator().DropColumn(&legacyCredRow{}, "UserID")
}

type legacyWPluginRow struct {
	ID      uint   `gorm:"column:id"`
	UserID  uint   `gorm:"column:user_id"`
	GroupID *uint  `gorm:"column:group_id"`
}

func (legacyWPluginRow) TableName() string { return "workflow_plugins" }

func migrateWorkflowPluginsTenancy(db *gorm.DB, personalOf func(uint) uint) error {
	if !db.Migrator().HasTable("workflow_plugins") {
		return nil
	}
	if !db.Migrator().HasColumn(&legacyWPluginRow{}, "UserID") {
		return nil
	}
	var rows []legacyWPluginRow
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		gid := uint(0)
		if r.GroupID != nil && *r.GroupID != 0 {
			gid = *r.GroupID
		} else {
			gid = personalOf(r.UserID)
		}
		if gid == 0 {
			continue
		}
		if err := db.Model(&WorkflowPlugin{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
			"group_id":    gid,
			"created_by": r.UserID,
		}).Error; err != nil {
			return err
		}
	}
	return db.Migrator().DropColumn(&legacyWPluginRow{}, "UserID")
}

type legacyWPluginInstRow struct {
	ID     uint `gorm:"column:id"`
	UserID uint `gorm:"column:user_id"`
}

func (legacyWPluginInstRow) TableName() string { return "workflow_plugin_installations" }

func migrateWorkflowPluginInstallationsTenancy(db *gorm.DB, personalOf func(uint) uint) error {
	if !db.Migrator().HasTable("workflow_plugin_installations") {
		return nil
	}
	if !db.Migrator().HasColumn(&legacyWPluginInstRow{}, "UserID") {
		return nil
	}
	var rows []legacyWPluginInstRow
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		gid := personalOf(r.UserID)
		if gid == 0 {
			continue
		}
		if err := db.Model(&WorkflowPluginInstallation{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
			"group_id":    gid,
			"created_by": r.UserID,
		}).Error; err != nil {
			return err
		}
	}
	return db.Migrator().DropColumn(&legacyWPluginInstRow{}, "UserID")
}

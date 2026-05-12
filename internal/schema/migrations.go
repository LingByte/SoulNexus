package schema

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Entity lists for GORM AutoMigrate per binary; imported by cmd/* entrypoints via bootstrap.Options.MigrateModels.

import (
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/notification/mail"
	"github.com/LingByte/SoulNexus/pkg/notification/sms"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

// ServerEntities is the full schema for cmd/server (and SIP when sharing the main API database).
func ServerEntities() []any {
	return []any{
		&utils.Config{},
		&models.User{},
		&models.UserProfile{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.UserRole{},
		&models.UserPermission{},
		&models.Group{},
		&models.UserCredential{},
		&models.GroupMember{},
		&models.GroupInvitation{},
		&models.GroupActivityLog{},
		&models.Agent{},
		&models.ChatSession{},
		&models.ChatMessage{},
		&models.LLMUsage{},
		&models.LLMUsageUserDaily{},
		&models.LLMUsageUserModelDaily{},
		&models.LLMChannel{},
		&models.LLMAbility{},
		&models.LLMModelMeta{},
		&models.LLMToken{},
		&models.ASRChannel{},
		&models.TTSChannel{},
		&models.SpeechUsage{},
		&models.TwoFA{},
		&models.TwoFABackupCode{},
		&models.Passkey{},
		&models.PasskeyChallenge{},
		&models.InternalNotification{},
		&models.NotificationChannel{},
		&models.MailTemplate{},
		&mail.MailLog{},
		&sms.SMSLog{},
		&models.VoiceTrainingTask{},
		&models.VoiceClone{},
		&models.Voiceprint{},
		&models.VoiceSynthesis{},
		&models.VoiceTrainingText{},
		&models.VoiceTrainingTextSegment{},
		&middleware.OperationLog{},
		&models.JSTemplate{},
		&models.JSTemplateVersion{},
		&models.Device{},
		&models.OTA{},
		&models.UsageRecord{},
		&models.Bill{},
		&models.Announcement{},
		&models.WorkflowDefinition{},
		&models.WorkflowInstance{},
		&models.WorkflowVersion{},
		&models.WorkflowPlugin{},
		&models.WorkflowPluginVersion{},
		&models.WorkflowPluginReview{},
		&models.WorkflowPluginInstallation{},
		&models.NodePlugin{},
		&models.NodePluginVersion{},
		&models.NodePluginReview{},
		&models.NodePluginInstallation{},
		&models.UserDevice{},
		&models.LoginHistory{},
		&models.AccountLock{},
		&models.DeviceErrorLog{},
		&models.CallRecording{},
		&models.OAuthClient{},
	}
}

// AuthEntities is the subset migrated by cmd/auth when running as a standalone user service.
func AuthEntities() []any {
	return []any{
		&utils.Config{},
		&models.User{},
		&models.UserProfile{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.UserRole{},
		&models.UserPermission{},
		&models.UserCredential{},
		&models.LoginHistory{},
		&models.AccountLock{},
		&models.InternalNotification{},
		&models.NotificationChannel{},
		&models.MailTemplate{},
		&mail.MailLog{},
		&sms.SMSLog{},
		&models.OAuthClient{},
		&models.UserDevice{},
		&models.Device{},
	}
}

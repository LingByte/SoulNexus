package schema

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Entity lists for GORM AutoMigrate per binary; imported by cmd/* entrypoints via bootstrap.Options.MigrateModels.

import (
	"github.com/LingByte/SoulNexus/internal/models/auth"
	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/notification/mail"
	"github.com/LingByte/SoulNexus/pkg/notification/sms"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

// ServerEntities is the unified schema for cmd/server (business API + user auth/RBAC).
func ServerEntities() []any {
	entities := []any{
		&utils.Config{},
		&auth.User{},
		&auth.UserProfile{},
		&auth.Role{},
		&auth.Permission{},
		&auth.RolePermission{},
		&auth.UserRole{},
		&auth.UserPermission{},
		&auth.UserCredential{},
		&auth.TwoFA{},
		&auth.TwoFABackupCode{},
		&auth.Passkey{},
		&auth.PasskeyChallenge{},
		&auth.LoginHistory{},
		&auth.AccountLock{},
		&auth.UserDevice{},
		&svcmodels.Group{},
		&svcmodels.GroupMember{},
		&svcmodels.GroupInvitation{},
		&svcmodels.GroupActivityLog{},
		&svcmodels.Agent{},
		&svcmodels.ChatSession{},
		&svcmodels.ChatMessage{},
		&svcmodels.LLMUsage{},
		&svcmodels.LLMUsageUserDaily{},
		&svcmodels.LLMUsageUserModelDaily{},
		&svcmodels.LLMChannel{},
		&svcmodels.LLMAbility{},
		&svcmodels.LLMModelMeta{},
		&svcmodels.LLMToken{},
		&svcmodels.ASRChannel{},
		&svcmodels.TTSChannel{},
		&svcmodels.SpeechUsage{},
		&svcmodels.InternalNotification{},
		&svcmodels.NotificationChannel{},
		&svcmodels.MailTemplate{},
		&mail.MailLog{},
		&sms.SMSLog{},
		&svcmodels.VoiceTrainingTask{},
		&svcmodels.VoiceClone{},
		&svcmodels.Voiceprint{},
		&svcmodels.VoiceSynthesis{},
		&middleware.OperationLog{},
		&svcmodels.JSTemplate{},
		&svcmodels.JSTemplateVersion{},
		&svcmodels.Device{},
		&svcmodels.OTA{},
		&svcmodels.UsageRecord{},
		&svcmodels.Bill{},
		&svcmodels.Announcement{},
		&svcmodels.WorkflowDefinition{},
		&svcmodels.WorkflowInstance{},
		&svcmodels.WorkflowVersion{},
		&svcmodels.WorkflowPlugin{},
		&svcmodels.WorkflowPluginVersion{},
		&svcmodels.WorkflowPluginReview{},
		&svcmodels.WorkflowPluginInstallation{},
		&svcmodels.NodePlugin{},
		&svcmodels.NodePluginVersion{},
		&svcmodels.NodePluginReview{},
		&svcmodels.NodePluginInstallation{},
		&svcmodels.DeviceErrorLog{},
		&svcmodels.CallRecording{},
		&svcmodels.KnowledgeNamespace{},
		&svcmodels.KnowledgeDocument{},
	}
	return entities
}

// AuthEntities is the subset migrated by cmd/auth when running as a standalone user service.
func AuthEntities() []any {
	return []any{
		&utils.Config{},
		&auth.User{},
		&auth.UserProfile{},
		&auth.Role{},
		&auth.Permission{},
		&auth.RolePermission{},
		&auth.UserRole{},
		&auth.UserPermission{},
		&auth.UserCredential{},
		&svcmodels.Group{},
		&svcmodels.GroupMember{},
		&auth.TwoFA{},
		&auth.TwoFABackupCode{},
		&auth.Passkey{},
		&auth.PasskeyChallenge{},
		&auth.LoginHistory{},
		&auth.AccountLock{},
		&middleware.OperationLog{},
		&svcmodels.InternalNotification{},
		&svcmodels.NotificationChannel{},
		&svcmodels.MailTemplate{},
		&mail.MailLog{},
		&sms.SMSLog{},
		&auth.UserDevice{},
		&svcmodels.Device{},
	}
}

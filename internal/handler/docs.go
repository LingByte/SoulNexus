package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"

	"github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/internal/apidocs"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/search"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) GetObjs() []LingEcho.WebObject {
	return []LingEcho.WebObject{
		{
			Group:       "lingEcho",
			Desc:        "User",
			Model:       models.User{},
			Name:        "user",
			Filterables: []string{"UpdatedAt", "CreatedAt"},
			Editables:   []string{"Email", "Phone", "FirstName", "LastName", "DisplayName", "Role", "Permissions", "Enabled"},
			Searchables: []string{},
			Orderables:  []string{"UpdatedAt"},
			GetDB: func(c *gin.Context, isCreate bool) *gorm.DB {
				if isCreate {
					return h.db
				}
				return h.db.Where("deleted_at", nil)
			},
			BeforeCreate: func(db *gorm.DB, ctx *gin.Context, vptr any) error {
				return nil
			},
		},
	}
}

func (h *Handlers) RegisterAdmin(router *gin.RouterGroup) {
	adminObjs := models.GetLingEchoAdminObjects()
	iconInternalNotification, _ := LingEcho.EmbedStaticAssets.ReadFile("static/img/icon_internal_notification.svg")
	iconOperatorLog, _ := LingEcho.EmbedStaticAssets.ReadFile("static/img/icon_user.svg")
	iconChatSessionLog, _ := LingEcho.EmbedStaticAssets.ReadFile("static/img/icon_chat_log.svg")
	admins := []models.AdminObject{
		{
			Group:       "System",
			Desc:        "SysUser",
			Model:       models.User{},
			Name:        "SysUser",
			Filterables: []string{"UpdatedAt", "CreatedAt"},
			Editables:   []string{"Email", "Phone", "FirstName", "LastName", "DisplayName", "Role", "Permissions", "Enabled"},
			Searchables: []string{},
			Orderables:  []string{"UpdatedAt"},
			GetDB: func(c *gin.Context, isCreate bool) *gorm.DB {
				if isCreate {
					return h.db
				}
				return h.db.Where("deleted_at", nil)
			},
			BeforeCreate: func(db *gorm.DB, ctx *gin.Context, vptr any) error {
				return nil
			},
		},
		{
			Model:       &notification.InternalNotification{},
			Group:       "System",
			Name:        "InternalNotification",
			Desc:        "This is a notification used to notify the user of the system.",
			Shows:       []string{"ID", "UserID", "Title", "Read", "CreatedAt"},
			Editables:   []string{"UserID", "Title", "Content", "Read"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"Title", "Content"},
			Icon:        &models.AdminIcon{SVG: string(iconInternalNotification)},
		},
		{
			Model:       &middleware.OperationLog{},                                                                          // Related model OperationLog
			Group:       "System",                                                                                            // Business group
			Name:        "Operation Log",                                                                                     // Display name in admin panel
			Desc:        "Logs the operations performed by users in the system.",                                             // Description
			Shows:       []string{"ID", "UserID", "Username", "Action", "Target", "IPAddress", "RequestMethod", "CreatedAt"}, // Displayed fields
			Editables:   []string{"Details"},                                                                                 // Editable fields (usually logs should not be edited, but details can be updated)
			Orderables:  []string{"CreatedAt"},                                                                               // Sortable fields
			Searchables: []string{"Username", "Action", "Target", "IPAddress"},                                               // Searchable fields
			Icon:        &models.AdminIcon{SVG: string(iconOperatorLog)},                                                     // Icon
		},
		{
			Model:       &models.ChatSessionLog{},                                                                                          // Related model ChatSessionLog
			Group:       "Chat",                                                                                                            // Business group
			Name:        "Chat Session Logs",                                                                                               // Display name in admin panel
			Desc:        "Logs the chat sessions between users and AI assistants.",                                                         // Description
			Shows:       []string{"ID", "SessionID", "UserID", "AssistantID", "ChatType", "CreatedAt"},                                     // Displayed fields
			Editables:   []string{"SessionID", "UserID", "AssistantID", "ChatType", "UserMessage", "AgentMessage", "AudioURL", "Duration"}, // Editable fields
			Orderables:  []string{"CreatedAt"},                                                                                             // Sortable fields
			Searchables: []string{"SessionID", "UserID", "ChatType"},                                                                       // Searchable fields
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},                                                                // Icon
		},
		{
			Model:       &models.UserCredential{},
			Group:       "Settings",
			Name:        "User Credentials",
			Desc:        "User authentication credentials and tokens.",
			Shows:       []string{"ID", "UserID", "Type", "CreatedAt"},
			Editables:   []string{"UserID", "Type", "Value"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"UserID", "Type"},
			Icon:        &models.AdminIcon{SVG: string(iconOperatorLog)},
		},
		// AI Assistant management
		{
			Model: &models.Assistant{},
			Group: "AI Assistant",
			Name:  "Assistants",
			Desc:  "AI assistants and their configurations.",
			Shows: []string{"ID", "Name", "Description", "UserID", "LLMModel", "TtsProvider", "EnableGraphMemory", "CreatedAt"},
			Editables: []string{
				"Name",
				"Description",
				"SystemPrompt",
				"Temperature",
				"MaxTokens",
				"LLMModel",
				"TtsProvider",
				"Language",
				"Speaker",
				"VoiceCloneID",
				"JsSourceID",
				"EnableGraphMemory",
			},
			Orderables:  []string{"CreatedAt", "Name"},
			Searchables: []string{"Name", "Description"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		{
			Model:       &models.JSTemplate{},
			Group:       "AI Assistant",
			Name:        "JS Templates",
			Desc:        "JavaScript templates for customizing AI assistant client interface.",
			Shows:       []string{"ID", "JsSourceID", "Name", "Type", "UserID", "GroupID", "CreatedAt"},
			Editables:   []string{"JsSourceID", "Name", "Type", "Content", "Usage", "UserID", "GroupID"},
			Orderables:  []string{"CreatedAt", "Name"},
			Searchables: []string{"Name", "Type", "JsSourceID"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// Voice training management
		{
			Model:       &models.VoiceTrainingTask{},
			Group:       "Voice Training",
			Name:        "Voice Training Tasks",
			Desc:        "Voice training tasks and their status.",
			Shows:       []string{"ID", "UserID", "TaskName", "Status", "CreatedAt"},
			Editables:   []string{"TaskName", "Status", "TaskID"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"TaskName", "Status"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		{
			Model:       &models.VoiceClone{},
			Group:       "Voice Training",
			Name:        "Voice Clones",
			Desc:        "Trained voice clones and their configurations.",
			Shows:       []string{"ID", "UserID", "VoiceName", "IsActive", "CreatedAt"},
			Editables:   []string{"VoiceName", "VoiceDescription", "IsActive"},
			Orderables:  []string{"CreatedAt", "VoiceName"},
			Searchables: []string{"VoiceName", "VoiceDescription"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		{
			Model:       &models.VoiceSynthesis{},
			Group:       "Voice Training",
			Name:        "Voice Synthesis",
			Desc:        "Voice synthesis records and history.",
			Shows:       []string{"ID", "UserID", "VoiceCloneID", "Status", "CreatedAt"},
			Editables:   []string{"Text", "Language", "Status"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"Text", "Status"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		{
			Model:       &models.VoiceTrainingText{},
			Group:       "Voice Training",
			Name:        "Voice Training Texts",
			Desc:        "Text materials for voice training.",
			Shows:       []string{"ID", "TextID", "TextName", "Language", "IsActive", "CreatedAt"},
			Editables:   []string{"TextID", "TextName", "Language", "IsActive"},
			Orderables:  []string{"CreatedAt", "TextName"},
			Searchables: []string{"TextName"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		{
			Model:       &models.VoiceTrainingTextSegment{},
			Group:       "Voice Training",
			Name:        "Voice Training Text Segments",
			Desc:        "Text segments for voice training.",
			Shows:       []string{"ID", "TextID", "SegID", "SegText", "CreatedAt"},
			Editables:   []string{"TextID", "SegID", "SegText"},
			Orderables:  []string{"CreatedAt", "ID"},
			Searchables: []string{"SegText"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// Workflow management
		{
			Model:       &models.WorkflowDefinition{},
			Group:       "Workflow",
			Name:        "Workflow Definitions",
			Desc:        "Workflow definitions and templates.",
			Shows:       []string{"ID", "Name", "Slug", "Status", "Version", "UserID", "CreatedAt"},
			Editables:   []string{"Name", "Slug", "Description", "Status", "Definition", "Settings", "Triggers", "Tags"},
			Orderables:  []string{"CreatedAt", "UpdatedAt", "Name"},
			Searchables: []string{"Name", "Slug", "Description"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		{
			Model:       &models.WorkflowInstance{},
			Group:       "Workflow",
			Name:        "Workflow Instances",
			Desc:        "Workflow execution instances and their status.",
			Shows:       []string{"ID", "DefinitionID", "DefinitionName", "Status", "CurrentNodeID", "StartedAt", "CompletedAt", "CreatedAt"},
			Editables:   []string{"Status", "ContextData", "ResultData"},
			Orderables:  []string{"CreatedAt", "StartedAt", "CompletedAt"},
			Searchables: []string{"DefinitionName", "Status"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// Billing management
		{
			Model:       &models.UsageRecord{},
			Group:       "Billing",
			Name:        "Usage Records",
			Desc:        "User usage records and statistics.",
			Shows:       []string{"ID", "UserID", "UsageType", "Model", "PromptTokens", "CompletionTokens", "CreatedAt"},
			Editables:   []string{"UserID", "UsageType", "Model", "PromptTokens", "CompletionTokens"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"UserID", "UsageType", "Model"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		{
			Model:       &models.Bill{},
			Group:       "Billing",
			Name:        "Bills",
			Desc:        "Billing records and invoices.",
			Shows:       []string{"ID", "BillNo", "Title", "Status", "UserID", "StartTime", "EndTime", "CreatedAt"},
			Editables:   []string{"Title", "Status", "Notes"},
			Orderables:  []string{"CreatedAt", "StartTime", "EndTime"},
			Searchables: []string{"BillNo", "Title"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// Quota management
		{
			Model:       &models.UserQuota{},
			Group:       "Quota",
			Name:        "User Quotas",
			Desc:        "User quota allocations and usage.",
			Shows:       []string{"ID", "UserID", "QuotaType", "TotalQuota", "UsedQuota", "Period", "CreatedAt"},
			Editables:   []string{"UserID", "QuotaType", "TotalQuota", "UsedQuota", "Period", "Description"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"UserID", "QuotaType"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		{
			Model:       &models.GroupQuota{},
			Group:       "Quota",
			Name:        "Group Quotas",
			Desc:        "Group quota allocations and usage.",
			Shows:       []string{"ID", "GroupID", "QuotaType", "TotalQuota", "UsedQuota", "Period", "CreatedAt"},
			Editables:   []string{"GroupID", "QuotaType", "TotalQuota", "UsedQuota", "Period", "Description"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"GroupID", "QuotaType"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// Alert management
		{
			Model:       &models.AlertRule{},
			Group:       "Alert",
			Name:        "Alert Rules",
			Desc:        "Alert rules and trigger conditions.",
			Shows:       []string{"ID", "UserID", "Name", "AlertType", "Severity", "Enabled", "CreatedAt"},
			Editables:   []string{"Name", "Description", "AlertType", "Severity", "Conditions", "Enabled"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"Name", "AlertType"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		{
			Model:       &models.Alert{},
			Group:       "Alert",
			Name:        "Alerts",
			Desc:        "Alert records and notifications.",
			Shows:       []string{"ID", "UserID", "RuleID", "AlertType", "Severity", "Title", "Status", "CreatedAt"},
			Editables:   []string{"Status", "ResolvedAt", "ResolvedBy"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"Title", "AlertType", "Status"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// Device management
		{
			Model:       &models.Device{},
			Group:       "Device",
			Name:        "Devices",
			Desc:        "IoT devices and their configurations.",
			Shows:       []string{"ID", "MacAddress", "UserID", "Board", "AppVersion", "AutoUpdate", "AssistantID", "CreatedAt"},
			Editables:   []string{"UserID", "GroupID", "Board", "AppVersion", "AutoUpdate", "AssistantID"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"ID", "MacAddress", "Board"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// OTA management
		{
			Model:       &models.OTA{},
			Group:       "OTA",
			Name:        "OTA Firmware",
			Desc:        "Firmware OTA update management.",
			Shows:       []string{"ID", "FirmwareName", "Type", "Version", "Size", "Sort", "CreatedAt"},
			Editables:   []string{"FirmwareName", "Type", "Version", "Size", "Remark", "FirmwarePath", "Sort"},
			Orderables:  []string{"CreatedAt", "Sort", "Version"},
			Searchables: []string{"FirmwareName", "Type", "Version"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// Overview config
		{
			Model:       &models.OverviewConfig{},
			Group:       "System",
			Name:        "Overview Config",
			Desc:        "Overview page configurations for organizations.",
			Shows:       []string{"ID", "OrganizationID", "Name", "Description", "CreatedAt"},
			Editables:   []string{"OrganizationID", "Name", "Description", "Config"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"Name", "Description"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// Group management
		{
			Model:       &models.Group{},
			Group:       "System",
			Name:        "Groups",
			Desc:        "User groups and organizations.",
			Shows:       []string{"ID", "Name", "Type", "CreatorID", "CreatedAt"},
			Editables:   []string{"Name", "Type", "Extra", "Avatar", "Permission"},
			Orderables:  []string{"CreatedAt", "Name"},
			Searchables: []string{"Name", "Type"},
			Icon:        &models.AdminIcon{SVG: string(iconOperatorLog)},
		},
		{
			Model:       &models.GroupMember{},
			Group:       "System",
			Name:        "Group Members",
			Desc:        "Group membership management.",
			Shows:       []string{"ID", "UserID", "GroupID", "Role", "CreatedAt"},
			Editables:   []string{"UserID", "GroupID", "Role"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"Role"},
			Icon:        &models.AdminIcon{SVG: string(iconOperatorLog)},
		},
		// Voiceprint management
		{
			Model:       &models.Voiceprint{},
			Group:       "Voice Training",
			Name:        "Voiceprints",
			Desc:        "Voice biometric data and speaker identification.",
			Shows:       []string{"ID", "SpeakerID", "AssistantID", "SpeakerName", "CreatedAt"},
			Editables:   []string{"SpeakerID", "AssistantID", "SpeakerName"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"SpeakerID", "SpeakerName"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// Node plugins
		{
			Model:       &models.NodePlugin{},
			Group:       "Workflow",
			Name:        "Node Plugins",
			Desc:        "Workflow node plugins and extensions.",
			Shows:       []string{"ID", "Name", "Slug", "DisplayName", "Category", "Version", "Status", "CreatedAt"},
			Editables:   []string{"Name", "Slug", "DisplayName", "Description", "Category", "Version", "Status", "Icon", "Color", "Tags"},
			Orderables:  []string{"CreatedAt", "Name"},
			Searchables: []string{"Name", "DisplayName", "Description", "Category"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// Workflow plugins
		{
			Model:       &models.WorkflowPlugin{},
			Group:       "Workflow",
			Name:        "Workflow Plugins",
			Desc:        "Published workflow plugins for reuse.",
			Shows:       []string{"ID", "Name", "Slug", "DisplayName", "Category", "Version", "Status", "DownloadCount", "StarCount", "CreatedAt"},
			Editables:   []string{"Name", "Slug", "DisplayName", "Description", "Category", "Version", "Status", "Icon", "Color", "Tags", "Author", "License"},
			Orderables:  []string{"CreatedAt", "DownloadCount", "StarCount", "Rating"},
			Searchables: []string{"Name", "DisplayName", "Description", "Category", "Author"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// User devices
		{
			Model:       &models.UserDevice{},
			Group:       "System",
			Name:        "User Devices",
			Desc:        "User device registrations and trust management.",
			Shows:       []string{"ID", "UserID", "DeviceID", "DeviceName", "DeviceType", "OS", "Browser", "IsTrusted", "IsActive", "LastUsedAt", "CreatedAt"},
			Editables:   []string{"DeviceName", "IsTrusted", "IsActive"},
			Orderables:  []string{"CreatedAt", "LastUsedAt"},
			Searchables: []string{"DeviceID", "DeviceName", "DeviceType", "OS", "Browser", "IPAddress"},
			Icon:        &models.AdminIcon{SVG: string(iconOperatorLog)},
		},
		// Login history
		{
			Model:       &models.LoginHistory{},
			Group:       "System",
			Name:        "Login History",
			Desc:        "User login history and security monitoring.",
			Shows:       []string{"ID", "UserID", "Email", "IPAddress", "Location", "Country", "City", "LoginType", "Success", "IsSuspicious", "CreatedAt"},
			Editables:   []string{"IsSuspicious", "FailureReason"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"Email", "IPAddress", "Location", "Country", "City", "LoginType"},
			Icon:        &models.AdminIcon{SVG: string(iconOperatorLog)},
		},
				// Alert Notifications
		{
			Model:       &models.AlertNotification{},
			Group:       "Alert",
			Name:        "Alert Notifications",
			Desc:        "Alert notification delivery records.",
			Shows:       []string{"ID", "AlertID", "Channel", "Status", "CreatedAt"},
			Editables:   []string{"Status", "ErrorMessage"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"Channel", "Status"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// Workflow Versions
		{
			Model:       &models.WorkflowVersion{},
			Group:       "Workflow",
			Name:        "Workflow Versions",
			Desc:        "Historical versions of workflow definitions.",
			Shows:       []string{"ID", "DefinitionID", "Version", "ChangeLog", "CreatedAt"},
			Editables:   []string{"ChangeLog", "IsActive"},
			Orderables:  []string{"CreatedAt", "Version"},
			Searchables: []string{"ChangeLog"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// Node Plugin Versions
		{
			Model:       &models.NodePluginVersion{},
			Group:       "Workflow",
			Name:        "Node Plugin Versions",
			Desc:        "Version history of node plugins.",
			Shows:       []string{"ID", "PluginID", "Version", "ChangeLog", "CreatedAt"},
			Editables:   []string{"ChangeLog", "IsActive"},
			Orderables:  []string{"CreatedAt", "Version"},
			Searchables: []string{"ChangeLog"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// Node Plugin Reviews
		{
			Model:       &models.NodePluginReview{},
			Group:       "Workflow",
			Name:        "Node Plugin Reviews",
			Desc:        "User reviews and ratings for node plugins.",
			Shows:       []string{"ID", "PluginID", "UserID", "Rating", "Comment", "CreatedAt"},
			Editables:   []string{"Rating", "Comment", "IsApproved"},
			Orderables:  []string{"CreatedAt", "Rating"},
			Searchables: []string{"Comment"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
		// Node Plugin Installations
		{
			Model:       &models.NodePluginInstallation{},
			Group:       "Workflow",
			Name:        "Node Plugin Installations",
			Desc:        "Plugin installation records by users.",
			Shows:       []string{"ID", "UserID", "PluginID", "Version", "Status", "CreatedAt"},
			Editables:   []string{"Status"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"Status"},
			Icon:        &models.AdminIcon{SVG: string(iconChatSessionLog)},
		},
	}
	models.RegisterAdmins(router, h.db, append(adminObjs, admins...))
}

func (h *Handlers) GetDocs() []apidocs.UriDoc {
	// Define the API documentation
	uriDocs := []apidocs.UriDoc{
		// ==================== User Authorization ====================
		{
			Group:   "User Authorization",
			Path:    config.GlobalConfig.Server.APIPrefix + "/auth/login",
			Method:  http.MethodPost,
			Desc:    "User login with email and password",
			Request: apidocs.GetDocDefine(models.LoginForm{}),
			Response: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "email", Type: apidocs.TYPE_STRING},
					{Name: "activation", Type: apidocs.TYPE_BOOLEAN, CanNull: true},
				},
			},
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/logout",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "User logout, if `?next={NEXT_URL}`is not empty, redirect to {NEXT_URL}",
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/register",
			Method:       http.MethodPost,
			AuthRequired: false,
			Desc:         "User register with email and password",
			Request:      apidocs.GetDocDefine(models.RegisterUserForm{}),
			Response: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "email", Type: apidocs.TYPE_STRING, Desc: "The email address"},
					{Name: "activation", Type: apidocs.TYPE_BOOLEAN, Desc: "Is the account activated"},
					{Name: "expired", Type: apidocs.TYPE_STRING, Default: "180d", CanNull: true, Desc: "If email verification is required, it will be verified within the valid time"},
				},
			},
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/register/email",
			Method:       http.MethodPost,
			AuthRequired: false,
			Desc:         "User register with email verification code",
			Request:      apidocs.GetDocDefine(models.EmailOperatorForm{}),
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/login/password",
			Method:       http.MethodPost,
			AuthRequired: false,
			Desc:         "User login with password",
			Request:      apidocs.GetDocDefine(models.LoginForm{}),
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/login/email",
			Method:       http.MethodPost,
			AuthRequired: false,
			Desc:         "User login with email verification code",
			Request:      apidocs.GetDocDefine(models.EmailOperatorForm{}),
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/info",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get current user information",
			Response: &apidocs.DocField{
				Type:   "object",
				Fields: apidocs.GetDocDefine(models.User{}).Fields,
			},
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/reset-password",
			Method:       http.MethodPost,
			AuthRequired: false,
			Desc:         "Send a verification code to the email address to reset password",
			Request:      apidocs.GetDocDefine(models.ResetPasswordForm{}),
			Response: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "expired", Type: apidocs.TYPE_STRING, Default: "30m", Desc: "Must be verified within the valid time"},
				},
			},
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/reset-password/confirm",
			Method:       http.MethodPost,
			AuthRequired: false,
			Desc:         "Confirm password reset with token",
			Request:      apidocs.GetDocDefine(models.ResetPasswordDoneForm{}),
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/change-password",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Change password when user is logged in",
			Request:      apidocs.GetDocDefine(models.ChangePasswordForm{}),
			Response: &apidocs.DocField{
				Type: apidocs.TYPE_BOOLEAN,
				Desc: "true if success",
			},
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/send/email",
			Method:       http.MethodPost,
			AuthRequired: false,
			Desc:         "Send email verification code",
			Request:      apidocs.GetDocDefine(models.SendEmailVerifyEmail{}),
			Response: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "expired", Type: apidocs.TYPE_STRING, Default: "30m", Desc: "Must be verified within the valid time"},
				},
			},
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/verify-email",
			Method:       http.MethodGet,
			AuthRequired: false,
			Desc:         "Verify email address with token",
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/send-email-verification",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Send email verification code to current user",
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/verify-phone",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Verify phone number",
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/send-phone-verification",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Send phone verification code",
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/update",
			Method:       http.MethodPut,
			AuthRequired: true,
			Desc:         "Update user profile",
			Request:      apidocs.GetDocDefine(models.UpdateUserRequest{}),
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/update/preferences",
			Method:       http.MethodPut,
			AuthRequired: true,
			Desc:         "Update user preferences",
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/update/basic/info",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Update user basic information",
			Request:      apidocs.GetDocDefine(models.UserBasicInfoUpdate{}),
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/notification-settings",
			Method:       http.MethodPut,
			AuthRequired: true,
			Desc:         "Update notification settings",
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/user-preferences",
			Method:       http.MethodPut,
			AuthRequired: true,
			Desc:         "Update user preferences",
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/stats",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get user statistics",
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/avatar/upload",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Upload user avatar",
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/two-factor/setup",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Setup two-factor authentication",
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/two-factor/enable",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Enable two-factor authentication",
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/two-factor/disable",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Disable two-factor authentication",
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/two-factor/status",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get two-factor authentication status",
		},
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.Server.APIPrefix + "/auth/activity",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get user activity logs",
		},

		// ==================== System Module ====================
		{
			Group:        "System Module",
			Path:         config.GlobalConfig.Server.APIPrefix + "/system/health",
			Method:       http.MethodGet,
			Summary:      "数据库健康状态",
			AuthRequired: false,
			Desc:         "检查数据库健康状态",
			Response: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "status", Type: apidocs.TYPE_STRING, Desc: "healthy or unhealthy"},
				},
			},
		},
		{
			Group:        "System Module",
			Path:         config.GlobalConfig.Server.APIPrefix + "/system/init",
			Method:       http.MethodGet,
			AuthRequired: false,
			Desc:         "System initialization endpoint, returns basic configuration information",
			Response: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{
						Name: "database",
						Type: "object",
						Fields: []apidocs.DocField{
							{Name: "driver", Type: apidocs.TYPE_STRING, Desc: "Database driver (e.g., sqlite, mysql, postgres)"},
							{Name: "isMemoryDB", Type: apidocs.TYPE_BOOLEAN, Desc: "Whether the database is a memory database (SQLite)"},
						},
					},
					{
						Name: "email",
						Type: "object",
						Fields: []apidocs.DocField{
							{Name: "configured", Type: apidocs.TYPE_BOOLEAN, Desc: "Whether email is configured"},
						},
					},
				},
			},
		},
		{
			Group:        "System Module",
			Path:         config.GlobalConfig.Server.APIPrefix + "/system/rate-limiter/config",
			Method:       http.MethodPost,
			AuthRequired: false,
			Desc:         "Update rate limiter configuration",
		},
		{
			Group:        "System Module",
			Path:         config.GlobalConfig.Server.APIPrefix + "/system/search/status",
			Method:       http.MethodGet,
			AuthRequired: false,
			Desc:         "Get search feature status and configuration",
			Response: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "enabled", Type: apidocs.TYPE_BOOLEAN, Desc: "Whether search is enabled"},
					{Name: "searchPath", Type: apidocs.TYPE_STRING, Desc: "Search index path"},
					{Name: "batchSize", Type: apidocs.TYPE_INT, Desc: "Batch size for indexing"},
					{Name: "schedule", Type: apidocs.TYPE_STRING, Desc: "Cron schedule for indexing"},
				},
			},
		},
		{
			Group:        "System Module",
			Path:         config.GlobalConfig.Server.APIPrefix + "/system/search/config",
			Method:       http.MethodPut,
			AuthRequired: true,
			Desc:         "Update search configuration (admin only)",
			Request: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "enabled", Type: apidocs.TYPE_BOOLEAN, CanNull: true, Desc: "Enable or disable search"},
					{Name: "path", Type: apidocs.TYPE_STRING, CanNull: true, Desc: "Search index path"},
					{Name: "batchSize", Type: apidocs.TYPE_INT, CanNull: true, Desc: "Batch size for indexing"},
					{Name: "schedule", Type: apidocs.TYPE_STRING, CanNull: true, Desc: "Cron schedule for indexing"},
				},
			},
		},
		{
			Group:        "System Module",
			Path:         config.GlobalConfig.Server.APIPrefix + "/system/search/enable",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Enable search feature (admin only)",
		},
		{
			Group:        "System Module",
			Path:         config.GlobalConfig.Server.APIPrefix + "/system/search/disable",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Disable search feature (admin only)",
		},

		// ==================== Notifications ====================
		{
			Group:        "Notifications",
			Path:         config.GlobalConfig.Server.APIPrefix + "/notification/unread-count",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get unread notification count",
			Response: &apidocs.DocField{
				Type: apidocs.TYPE_INT,
				Desc: "Number of unread notifications",
			},
		},
		{
			Group:        "Notifications",
			Path:         config.GlobalConfig.Server.APIPrefix + "/notification",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "List user notifications with pagination",
			Response: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "list", Type: "array"},
					{Name: "total", Type: apidocs.TYPE_INT},
					{Name: "totalUnread", Type: apidocs.TYPE_INT},
					{Name: "totalRead", Type: apidocs.TYPE_INT},
					{Name: "page", Type: apidocs.TYPE_INT},
					{Name: "size", Type: apidocs.TYPE_INT},
				},
			},
		},
		{
			Group:        "Notifications",
			Path:         config.GlobalConfig.Server.APIPrefix + "/notification/readAll",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Mark all notifications as read",
		},
		{
			Group:        "Notifications",
			Path:         config.GlobalConfig.Server.APIPrefix + "/notification/read/:id",
			Method:       http.MethodPut,
			AuthRequired: true,
			Desc:         "Mark a notification as read",
		},
		{
			Group:        "Notifications",
			Path:         config.GlobalConfig.Server.APIPrefix + "/notification/:id",
			Method:       http.MethodDelete,
			AuthRequired: true,
			Desc:         "Delete a notification",
		},
		{
			Group:        "Notifications",
			Path:         config.GlobalConfig.Server.APIPrefix + "/notification/batch-delete",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Batch delete notifications",
		},

		// ==================== Groups ====================
		{
			Group:        "Groups",
			Path:         config.GlobalConfig.Server.APIPrefix + "/group",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Create a new group",
			Request: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "name", Type: apidocs.TYPE_STRING, Required: true},
					{Name: "type", Type: apidocs.TYPE_STRING},
					{Name: "extra", Type: apidocs.TYPE_STRING},
					{Name: "permission", Type: "object"},
				},
			},
		},
		{
			Group:        "Groups",
			Path:         config.GlobalConfig.Server.APIPrefix + "/group",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "List all groups",
			Response: &apidocs.DocField{
				Type:   "array",
				Fields: apidocs.GetDocDefine(models.Group{}).Fields,
			},
		},
		{
			Group:        "Groups",
			Path:         config.GlobalConfig.Server.APIPrefix + "/group/:id",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get a group by ID",
			Response: &apidocs.DocField{
				Type:   "object",
				Fields: apidocs.GetDocDefine(models.Group{}).Fields,
			},
		},
		{
			Group:        "Groups",
			Path:         config.GlobalConfig.Server.APIPrefix + "/group/:id",
			Method:       http.MethodPut,
			AuthRequired: true,
			Desc:         "Update a group",
		},
		{
			Group:        "Groups",
			Path:         config.GlobalConfig.Server.APIPrefix + "/group/:id",
			Method:       http.MethodDelete,
			AuthRequired: true,
			Desc:         "Delete a group",
		},

		// ==================== Assistants ====================
		{
			Group:        "Assistants",
			Path:         config.GlobalConfig.Server.APIPrefix + "/assistant/add",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Create a new AI assistant",
			Request: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "name", Type: apidocs.TYPE_STRING, Required: true},
					{Name: "description", Type: apidocs.TYPE_STRING},
					{Name: "icon", Type: apidocs.TYPE_STRING},
				},
			},
			Response: &apidocs.DocField{
				Type:   "object",
				Fields: apidocs.GetDocDefine(models.Assistant{}).Fields,
			},
		},
		{
			Group:        "Assistants",
			Path:         config.GlobalConfig.Server.APIPrefix + "/assistant",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "List user's assistants",
			Response: &apidocs.DocField{
				Type:   "array",
				Fields: apidocs.GetDocDefine(models.Assistant{}).Fields,
			},
		},
		{
			Group:        "Assistants",
			Path:         config.GlobalConfig.Server.APIPrefix + "/assistant/:id",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get an assistant by ID",
			Response: &apidocs.DocField{
				Type:   "object",
				Fields: apidocs.GetDocDefine(models.Assistant{}).Fields,
			},
		},
		{
			Group:        "Assistants",
			Path:         config.GlobalConfig.Server.APIPrefix + "/assistant/:id",
			Method:       http.MethodPut,
			AuthRequired: true,
			Desc:         "Update an assistant",
			Request: &apidocs.DocField{
				Type:   "object",
				Fields: apidocs.GetDocDefine(models.Assistant{}).Fields,
			},
		},
		{
			Group:        "Assistants",
			Path:         config.GlobalConfig.Server.APIPrefix + "/assistant/:id",
			Method:       http.MethodDelete,
			AuthRequired: true,
			Desc:         "Delete an assistant",
		},
		{
			Group:        "Assistants",
			Path:         config.GlobalConfig.Server.APIPrefix + "/assistant/:id/js",
			Method:       http.MethodPut,
			AuthRequired: true,
			Desc:         "Update assistant's JavaScript template",
		},
		{
			Group:        "Assistants",
			Path:         config.GlobalConfig.Server.APIPrefix + "/assistant/lingecho/client/:id/loader.js",
			Method:       http.MethodGet,
			AuthRequired: false,
			Desc:         "Get Voice Sculptor loader JavaScript for assistant",
		},
		
		// ==================== JS Templates ====================
		{
			Group:        "JS Templates",
			Path:         config.GlobalConfig.Server.APIPrefix + "/js-templates",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Create a new JS template",
			Request:      apidocs.GetDocDefine(models.JSTemplate{}),
		},
		{
			Group:        "JS Templates",
			Path:         config.GlobalConfig.Server.APIPrefix + "/js-templates/:id",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get a JS template by ID",
			Response: &apidocs.DocField{
				Type:   "object",
				Fields: apidocs.GetDocDefine(models.JSTemplate{}).Fields,
			},
		},
		{
			Group:        "JS Templates",
			Path:         config.GlobalConfig.Server.APIPrefix + "/js-templates/name/:name",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get JS templates by name",
			Response: &apidocs.DocField{
				Type:   "array",
				Fields: apidocs.GetDocDefine(models.JSTemplate{}).Fields,
			},
		},
		{
			Group:        "JS Templates",
			Path:         config.GlobalConfig.Server.APIPrefix + "/js-templates",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "List JS templates",
		},
		{
			Group:        "JS Templates",
			Path:         config.GlobalConfig.Server.APIPrefix + "/js-templates/:id",
			Method:       http.MethodPut,
			AuthRequired: true,
			Desc:         "Update a JS template",
		},
		{
			Group:        "JS Templates",
			Path:         config.GlobalConfig.Server.APIPrefix + "/js-templates/:id",
			Method:       http.MethodDelete,
			AuthRequired: true,
			Desc:         "Delete a JS template",
		},
		{
			Group:        "JS Templates",
			Path:         config.GlobalConfig.Server.APIPrefix + "/js-templates/default",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "List default JS templates",
		},
		{
			Group:        "JS Templates",
			Path:         config.GlobalConfig.Server.APIPrefix + "/js-templates/custom",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "List custom JS templates",
		},
		{
			Group:        "JS Templates",
			Path:         config.GlobalConfig.Server.APIPrefix + "/js-templates/search",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Search JS templates",
		},

		// ==================== Chat ====================
		{
			Group:        "Chat",
			Path:         config.GlobalConfig.Server.APIPrefix + "/chat/chat-session-log",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get chat session logs",
		},
		{
			Group:        "Chat",
			Path:         config.GlobalConfig.Server.APIPrefix + "/chat/chat-session-log/:id",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get chat session log detail",
			Response: &apidocs.DocField{
				Type:   "object",
				Fields: apidocs.GetDocDefine(models.ChatSessionLogDetail{}).Fields,
			},
		},
		{
			Group:        "Chat",
			Path:         config.GlobalConfig.Server.APIPrefix + "/chat/chat-session-log/by-session/:sessionId",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get all chat logs for a specific session",
			Response: &apidocs.DocField{
				Type:   "array",
				Fields: apidocs.GetDocDefine(models.ChatSessionLogDetail{}).Fields,
			},
		},
		{
			Group:        "Chat",
			Path:         config.GlobalConfig.Server.APIPrefix + "/chat/chat-session-log/by-assistant/:assistantId",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get chat session logs by assistant ID",
		},
		{
			Group:        "Chat",
			Path:         config.GlobalConfig.Server.APIPrefix + "/chat/call",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Handle WebRTC connection for real-time voice chat",
		},

		// ==================== Credentials ====================
		{
			Group:        "Credentials",
			Path:         config.GlobalConfig.Server.APIPrefix + "/credentials",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Create a new user credential",
			Request:      apidocs.GetDocDefine(models.UserCredentialRequest{}),
			Response: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "apiKey", Type: apidocs.TYPE_STRING},
					{Name: "apiSecret", Type: apidocs.TYPE_STRING},
					{Name: "name", Type: apidocs.TYPE_STRING},
				},
			},
		},
		{
			Group:        "Credentials",
			Path:         config.GlobalConfig.Server.APIPrefix + "/credentials",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get user credentials",
			Response: &apidocs.DocField{
				Type:   "array",
				Fields: apidocs.GetDocDefine(models.UserCredential{}).Fields,
			},
		},
		{
			Group:        "Credentials",
			Path:         config.GlobalConfig.Server.APIPrefix + "/credentials/:id",
			Method:       http.MethodDelete,
			AuthRequired: true,
			Desc:         "Delete a credential",
		},

		// ==================== Knowledge Base ====================
		{
			Group:        "Knowledge Base",
			Path:         config.GlobalConfig.Server.APIPrefix + "/knowledge-base",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "List current user's knowledge bases",
		},
		{
			Group:        "Knowledge Base",
			Path:         config.GlobalConfig.Server.APIPrefix + "/knowledge-base/supported-document-types",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "List supported knowledge document upload formats",
		},
		{
			Group:        "Knowledge Base",
			Path:         config.GlobalConfig.Server.APIPrefix + "/knowledge-base",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Create a knowledge base",
		},
		{
			Group:        "Knowledge Base",
			Path:         config.GlobalConfig.Server.APIPrefix + "/knowledge-base/:id",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get knowledge base detail",
		},
		{
			Group:        "Knowledge Base",
			Path:         config.GlobalConfig.Server.APIPrefix + "/knowledge-base/:id",
			Method:       http.MethodPut,
			AuthRequired: true,
			Desc:         "Update a knowledge base",
		},
		{
			Group:        "Knowledge Base",
			Path:         config.GlobalConfig.Server.APIPrefix + "/knowledge-base/:id",
			Method:       http.MethodDelete,
			AuthRequired: true,
			Desc:         "Delete a knowledge base",
		},
		{
			Group:        "Knowledge Base",
			Path:         config.GlobalConfig.Server.APIPrefix + "/knowledge-base/:id/documents",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "List documents in knowledge base (aggregated from vector store)",
		},
		{
			Group:        "Knowledge Base",
			Path:         config.GlobalConfig.Server.APIPrefix + "/knowledge-base/:id/documents/upload",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Upload document to knowledge base (optional Accept: application/x-ndjson for progress stream)",
		},
		{
			Group:        "Knowledge Base",
			Path:         config.GlobalConfig.Server.APIPrefix + "/knowledge-base/:id/documents",
			Method:       http.MethodDelete,
			AuthRequired: true,
			Desc:         "Delete document from knowledge base",
		},
		{
			Group:        "Knowledge Base",
			Path:         config.GlobalConfig.Server.APIPrefix + "/knowledge-base/:id/recall-test",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Knowledge base recall experiment",
		},

		// ==================== Xunfei TTS ====================
		{
			Group:        "Xunfei TTS",
			Path:         config.GlobalConfig.Server.APIPrefix + "/xunfei/synthesize",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Xunfei text-to-speech synthesis",
			Request: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "assetId", Type: apidocs.TYPE_STRING, Required: true},
					{Name: "text", Type: apidocs.TYPE_STRING, Required: true},
					{Name: "language", Type: apidocs.TYPE_STRING, Required: true},
					{Name: "key", Type: apidocs.TYPE_STRING},
				},
			},
			Response: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "url", Type: apidocs.TYPE_STRING},
				},
			},
		},
		{
			Group:        "Xunfei TTS",
			Path:         config.GlobalConfig.Server.APIPrefix + "/xunfei/task/create",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Create Xunfei voice training task",
			Request: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "taskName", Type: apidocs.TYPE_STRING, Required: true},
					{Name: "sex", Type: apidocs.TYPE_INT},
					{Name: "ageGroup", Type: apidocs.TYPE_INT},
					{Name: "language", Type: apidocs.TYPE_STRING},
				},
			},
			Response: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "taskId", Type: apidocs.TYPE_STRING},
				},
			},
		},
		{
			Group:        "Xunfei TTS",
			Path:         config.GlobalConfig.Server.APIPrefix + "/xunfei/task/submit-audio",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Submit audio file for Xunfei training",
		},
		{
			Group:        "Xunfei TTS",
			Path:         config.GlobalConfig.Server.APIPrefix + "/xunfei/task/query",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Query Xunfei training task status",
			Request: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "taskId", Type: apidocs.TYPE_STRING, Required: true},
				},
			},
		},
		{
			Group:        "Xunfei TTS",
			Path:         config.GlobalConfig.Server.APIPrefix + "/xunfei/training-texts",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get Xunfei training texts",
		},

		// ==================== Voice Training ====================
		{
			Group:        "Voice Training",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/training/create",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Create voice training task",
			Request: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "taskName", Type: apidocs.TYPE_STRING, Required: true},
					{Name: "sex", Type: apidocs.TYPE_INT},
					{Name: "ageGroup", Type: apidocs.TYPE_INT},
					{Name: "language", Type: apidocs.TYPE_STRING},
				},
			},
		},
		{
			Group:        "Voice Training",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/training/submit-audio",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Submit audio file for voice training",
		},
		{
			Group:        "Voice Training",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/training/query",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Query voice training task status",
			Request: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "taskId", Type: apidocs.TYPE_STRING, Required: true},
				},
			},
		},
		{
			Group:        "Voice Training",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/clones",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get user's voice clones",
			Response: &apidocs.DocField{
				Type: "array",
			},
		},
		{
			Group:        "Voice Training",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/clones/:id",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get a voice clone by ID",
		},
		{
			Group:        "Voice Training",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/clones/update",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Update a voice clone",
			Request: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "id", Type: apidocs.TYPE_INT, Required: true},
					{Name: "voiceName", Type: apidocs.TYPE_STRING, Required: true},
					{Name: "voiceDescription", Type: apidocs.TYPE_STRING},
				},
			},
		},
		{
			Group:        "Voice Training",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/clones/delete",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Delete a voice clone",
		},
		{
			Group:        "Voice Training",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/synthesize",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Synthesize speech with trained voice",
			Request: &apidocs.DocField{
				Type: "object",
				Fields: []apidocs.DocField{
					{Name: "voiceCloneId", Type: apidocs.TYPE_INT, Required: true},
					{Name: "text", Type: apidocs.TYPE_STRING, Required: true},
					{Name: "language", Type: apidocs.TYPE_STRING},
					{Name: "storageKey", Type: apidocs.TYPE_STRING},
				},
			},
		},
		{
			Group:        "Voice Training",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/synthesis/history",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get synthesis history",
		},
		{
			Group:        "Voice Training",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/synthesis/delete",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Delete a synthesis record",
		},
		{
			Group:        "Voice Training",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/training-texts",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get training texts",
		},
		{
			Group:        "Voice Training",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/oneshot_text",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "One-shot text synthesis",
		},
		{
			Group:        "Voice Training",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/audio_status",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get audio processing status",
		},
		{
			Group:        "Voice Training",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/options",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get voice options list based on TTS provider",
		},

		// ==================== WebSocket ====================
		{
			Group:        "WebSocket",
			Path:         config.GlobalConfig.Server.APIPrefix + "/ws",
			Method:       "GET",
			AuthRequired: true,
			Desc:         "WebSocket connection endpoint",
		},
		{
			Group:        "WebSocket",
			Path:         config.GlobalConfig.Server.APIPrefix + "/voice/websocket",
			Method:       "GET",
			AuthRequired: true,
			Desc:         "WebSocket voice connection endpoint (supports multiple providers)",
		},
		{
			Group:        "WebSocket",
			Path:         config.GlobalConfig.Server.APIPrefix + "/ws/stats",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get WebSocket statistics",
		},
		{
			Group:        "WebSocket",
			Path:         config.GlobalConfig.Server.APIPrefix + "/ws/health",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "WebSocket health check",
		},
		{
			Group:        "WebSocket",
			Path:         config.GlobalConfig.Server.APIPrefix + "/ws/user/:user_id",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get user WebSocket statistics",
		},
		{
			Group:        "WebSocket",
			Path:         config.GlobalConfig.Server.APIPrefix + "/ws/group/:group",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "Get group WebSocket statistics",
		},
		{
			Group:        "WebSocket",
			Path:         config.GlobalConfig.Server.APIPrefix + "/ws/message",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Send WebSocket message",
		},
		{
			Group:        "WebSocket",
			Path:         config.GlobalConfig.Server.APIPrefix + "/ws/broadcast",
			Method:       http.MethodPost,
			AuthRequired: true,
			Desc:         "Broadcast WebSocket message",
		},
		{
			Group:        "WebSocket",
			Path:         config.GlobalConfig.Server.APIPrefix + "/ws/user/:user_id",
			Method:       http.MethodDelete,
			AuthRequired: true,
			Desc:         "Disconnect user from WebSocket",
		},
		{
			Group:        "WebSocket",
			Path:         config.GlobalConfig.Server.APIPrefix + "/ws/group/:group",
			Method:       http.MethodDelete,
			AuthRequired: true,
			Desc:         "Disconnect group from WebSocket",
		},
	}

	// 从数据库读取搜索配置，如果数据库中没有则使用配置文件
	searchEnabled := utils.GetBoolValue(h.db, constants.KEY_SEARCH_ENABLED)
	if !searchEnabled && config.GlobalConfig != nil {
		searchEnabled = config.GlobalConfig.Features.SearchEnabled
	}

	if searchEnabled {
		uriDocs = append(uriDocs, []apidocs.UriDoc{
			{
				Group:   "Search",
				Path:    config.GlobalConfig.Server.APIPrefix + "/search",
				Method:  http.MethodPost,
				Desc:    "Execute a search query",
				Request: apidocs.GetDocDefine(search.SearchRequest{}),
				Response: &apidocs.DocField{
					Type: "object",
					Fields: []apidocs.DocField{
						{Name: "Total", Type: apidocs.TYPE_INT},
						{Name: "Took", Type: apidocs.TYPE_INT},
						{Name: "Hits", Type: "array", Fields: []apidocs.DocField{
							{Name: "ID", Type: apidocs.TYPE_STRING},
							{Name: "Score", Type: apidocs.TYPE_FLOAT},
							{Name: "Fields", Type: "object"},
						}},
					},
				},
			},
			{
				Group:        "Search",
				Path:         config.GlobalConfig.Server.APIPrefix + "/search/index",
				Method:       http.MethodPost,
				AuthRequired: true,
				Desc:         "Index a new document",
				Request:      apidocs.GetDocDefine(search.Doc{}),
				Response: &apidocs.DocField{
					Type: apidocs.TYPE_BOOLEAN,
					Desc: "true if document is indexed successfully",
				},
			},
			{
				Group:        "Search",
				Path:         config.GlobalConfig.Server.APIPrefix + "/search/delete",
				Method:       http.MethodPost,
				AuthRequired: true,
				Desc:         "Delete a document by its ID",
				Request: &apidocs.DocField{
					Type: "object",
					Fields: []apidocs.DocField{
						{Name: "ID", Type: apidocs.TYPE_STRING},
					},
				},
				Response: &apidocs.DocField{
					Type: apidocs.TYPE_BOOLEAN,
					Desc: "true if document is deleted successfully",
				},
			},
			{
				Group:        "Search",
				Path:         config.GlobalConfig.Server.APIPrefix + "/search/auto-complete",
				Method:       http.MethodPost,
				AuthRequired: false,
				Desc:         "Get search query auto-completion suggestions",
				Request: &apidocs.DocField{
					Type: "object",
					Fields: []apidocs.DocField{
						{Name: "Keyword", Type: apidocs.TYPE_STRING},
					},
				},
				Response: &apidocs.DocField{
					Type: "object",
					Fields: []apidocs.DocField{
						{Name: "suggestions", Type: "array", Fields: []apidocs.DocField{
							{Name: "suggestion", Type: apidocs.TYPE_STRING},
						}},
					},
				},
			},
			{
				Group:        "Search",
				Path:         config.GlobalConfig.Server.APIPrefix + "/search/suggest",
				Method:       http.MethodPost,
				AuthRequired: false,
				Desc:         "Get search suggestions based on the keyword",
				Request: &apidocs.DocField{
					Type: "object",
					Fields: []apidocs.DocField{
						{Name: "Keyword", Type: apidocs.TYPE_STRING},
					},
				},
				Response: &apidocs.DocField{
					Type: "object",
					Fields: []apidocs.DocField{
						{Name: "suggestions", Type: "array", Fields: []apidocs.DocField{
							{Name: "suggestion", Type: apidocs.TYPE_STRING},
						}},
					},
				},
			},
		}...)
	}
	return uriDocs
}

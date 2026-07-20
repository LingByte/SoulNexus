package listeners

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/notification/inbox"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// InitNotifyListeners wires inbox and platform-admin notification signals.
func InitNotifyListeners(db *gorm.DB) {
	if db == nil {
		return
	}
	connectAsync := func(event string, fn func(*gorm.DB, ...any)) {
		utils.Sig().Connect(event, func(_ any, params ...any) {
			workDB := db
			workParams := params
			if n := len(params); n > 0 {
				if passed, ok := params[n-1].(*gorm.DB); ok {
					workDB = passed
					workParams = params[:n-1]
				}
			}
			go fn(workDB, workParams...)
		})
	}
	connectAsync(constants.SigNotifyOpLog, deliverOpLogNotify)
	connectAsync(constants.SigNotifyTenantProvisioned, deliverTenantProvisionedNotify)
}

func deliverOpLogNotify(db *gorm.DB, params ...any) {
	if len(params) == 0 {
		return
	}
	p, ok := params[0].(constants.NotifyOpLogPayload)
	if !ok || !p.Success {
		return
	}
	summary := strings.TrimSpace(p.Summary)
	name := strings.TrimSpace(p.ResourceName)
	if name == "" {
		name = "—"
	}

	switch p.Resource {
	case constants.OpResourceTenantUser:
		if p.Action == constants.OpActionUpdate && strings.Contains(strings.ToLower(summary), "changed password") {
			notifyActor(db, p.OperatorKind, p.OperatorID, "",
				"密码已更改",
				"您的登录密码已成功修改。如非本人操作，请立即联系管理员。")
			break
		}
		title, content := tenantUserInboxMessage(p.Action, name, summary)
		if title != "" {
			sendInbox(db, p.ResourceID, title, content)
		}
	case constants.OpResourceTenantGroup:
		title, content := tenantGroupInboxMessage(p.Action, name, summary)
		if title == "" {
			break
		}
		var ids []uint
		_ = db.Model(&models.TenantUserGroup{}).Where("group_id = ?", p.ResourceID).Pluck("tenant_user_id", &ids).Error
		sendInboxMany(db, ids, title, content)
	case constants.OpResourceTenantRole:
		title, content := tenantRoleInboxMessage(p.Action, name, summary)
		if title == "" {
			break
		}
		ids, err := models.ListTenantUserIDsByRoleID(db, p.ResourceID)
		if err != nil {
			break
		}
		sendInboxMany(db, ids, title, content)
	case constants.OpResourceAssistant:
		if p.Action == constants.OpActionCreate {
			notifyActorWith(db, p.OperatorKind, p.OperatorID, "",
				"智能体已创建",
				fmt.Sprintf("您已成功创建智能体「%s」。\n\n接下来可以：完善提示词与欢迎语、绑定知识库 / NLU、配置 ASR·LLM·TTS，然后在会话流程中选用该智能体。", name),
				inbox.SendOptions{ActionURL: "/assistant-manager", ActionLabel: "查看智能体"})
		}
	case constants.OpResourceCredential:
		switch {
		case p.Action == constants.OpActionCreate:
			notifyActorWith(db, p.OperatorKind, p.OperatorID, "",
				"API Key 已创建",
				fmt.Sprintf("您已创建 API Key「%s」。\n\n完整密钥仅在创建时展示一次，请立即复制并妥善保管。勿将密钥提交到代码仓库或公开页面。", name),
				inbox.SendOptions{ActionURL: "/profile/access-keys", ActionLabel: "管理密钥"})
		case p.Action == constants.OpActionRegenerate || strings.Contains(strings.ToLower(summary), "regenerated"):
			notifyActorWith(db, p.OperatorKind, p.OperatorID, "",
				"API Key 已轮转",
				fmt.Sprintf("API Key「%s」已重新生成，旧密钥立即失效。\n\n请更新所有使用该密钥的客户端与挂件配置。", name),
				inbox.SendOptions{ActionURL: "/profile/access-keys", ActionLabel: "管理密钥"})
		}
	case constants.OpResourceVoiceClone:
		if p.Action == constants.OpActionCreate {
			notifyActorWith(db, p.OperatorKind, p.OperatorID, "",
				"音色克隆已创建",
				fmt.Sprintf("您已创建音色克隆任务「%s」。\n\n训练完成后，可在智能体语音配置中选用该音色。", name),
				inbox.SendOptions{ActionURL: "/voice-clone-manager", ActionLabel: "查看音色"})
		}
	case constants.OpResourceVoiceprint:
		if p.Action == constants.OpActionCreate {
			notifyActorWith(db, p.OperatorKind, p.OperatorID, "",
				"声纹已创建",
				fmt.Sprintf("您已成功录入声纹「%s」。\n\n可在身份核验场景中启用该声纹。", name),
				inbox.SendOptions{ActionURL: "/voiceprint-manager", ActionLabel: "查看声纹"})
		}
	case constants.OpResourceWorkflow:
		if p.Action == constants.OpActionPublish || p.Action == constants.OpActionCreate {
			notifyActorWith(db, p.OperatorKind, p.OperatorID, "",
				"工作流已发布",
				fmt.Sprintf("工作流「%s」已发布。\n\n%s\n\n可在工作流管理中查看定义、触发与运行记录。", name, summary),
				inbox.SendOptions{ActionURL: "/workflows", ActionLabel: "查看工作流"})
		}
	case constants.OpResourceMCPMarket:
		if p.Action == constants.OpActionPublish || p.Action == constants.OpActionCreate {
			notifyActorWith(db, p.OperatorKind, p.OperatorID, "",
				"MCP 工具已发布",
				fmt.Sprintf("MCP 工具「%s」已发布到市场。\n\n租户可在智能体工具配置中订阅并绑定该能力。", name),
				inbox.SendOptions{ActionURL: "/mcp-market", ActionLabel: "打开市场"})
		}
	case constants.OpResourcePlatformAdmin:
		if p.Action == constants.OpActionUpdate && strings.Contains(strings.ToLower(summary), "changed password") {
			notifyActor(db, p.OperatorKind, p.OperatorID, "",
				"密码已更改",
				"您的登录密码已成功修改。如非本人操作，请立即联系管理员。")
		}
	}

	if p.TenantID > 0 {
		if p.Resource == constants.OpResourceTenant && p.Action == constants.OpActionCreate {
			return
		}
		title := "租户操作通知"
		content := summary
		if content == "" {
			content = fmt.Sprintf("租户 #%d 发生 %s 操作（%s）", p.TenantID, p.Action, p.Resource)
		}
		notifyPlatformAdmins(db, "", title, content)
	}
}

func deliverTenantProvisionedNotify(db *gorm.DB, params ...any) {
	if len(params) == 0 {
		return
	}
	p, ok := params[0].(constants.NotifyTenantProvisionedPayload)
	if !ok || p.TenantID == 0 {
		return
	}
	adminName := strings.TrimSpace(p.AdminDisplayName)
	if adminName == "" {
		adminName = p.AdminEmail
	}
	sourceLabel := "自助注册"
	if strings.TrimSpace(p.Source) == "platform" {
		sourceLabel = "平台创建"
	}
	title := "新租户已开通"
	content := fmt.Sprintf("租户「%s」已通过%s完成开通，管理员：%s。", p.TenantName, sourceLabel, adminName)
	notifyPlatformAdmins(db, p.ClientIP, title, content)
}

func sendInbox(db *gorm.DB, userID uint, title, content string) {
	sendInboxWith(db, userID, title, content, inbox.SendOptions{})
}

func sendInboxWith(db *gorm.DB, userID uint, title, content string, opt inbox.SendOptions) {
	if db == nil || userID == 0 {
		return
	}
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || content == "" {
		return
	}
	if err := inbox.NewService(db).SendWith(userID, title, content, opt); err != nil {
		logger.Warn("notify inbox failed", zap.Uint("userId", userID), zap.String("title", title), zap.Error(err))
	}
}

func sendInboxMany(db *gorm.DB, userIDs []uint, title, content string) {
	sendInboxManyWith(db, userIDs, title, content, inbox.SendOptions{})
}

func sendInboxManyWith(db *gorm.DB, userIDs []uint, title, content string, opt inbox.SendOptions) {
	seen := make(map[uint]struct{}, len(userIDs))
	for _, uid := range userIDs {
		if uid == 0 {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		sendInboxWith(db, uid, title, content, opt)
	}
}

func notifyActor(db *gorm.DB, operatorKind string, operatorID uint, clientIP, title, content string) {
	notifyActorWith(db, operatorKind, operatorID, clientIP, title, content, inbox.SendOptions{})
}

func notifyActorWith(db *gorm.DB, operatorKind string, operatorID uint, clientIP, title, content string, opt inbox.SendOptions) {
	if operatorID == 0 {
		return
	}
	sendInboxWith(db, operatorID, title, content, opt)
	email, receive := resolveActorEmailNotify(db, operatorKind, operatorID)
	if !receive || email == "" {
		return
	}
	html := fmt.Sprintf("<p>%s</p>", strings.ReplaceAll(content, "\n", "<br>"))
	if u := strings.TrimSpace(opt.ActionURL); u != "" {
		label := strings.TrimSpace(opt.ActionLabel)
		if label == "" {
			label = "查看详情"
		}
		html += fmt.Sprintf(`<p><a href="%s">%s</a></p>`, u, label)
	}
	mailer := notification.NewMailer(db, operatorID, clientIP)
	if err := mailer.SendRaw(context.Background(), email, title, html); err != nil {
		logger.Warn("notify actor email failed",
			zap.String("operatorKind", operatorKind),
			zap.Uint("operatorId", operatorID),
			zap.String("email", email),
			zap.Error(err))
	}
}

func resolveActorEmailNotify(db *gorm.DB, operatorKind string, operatorID uint) (email string, receive bool) {
	switch operatorKind {
	case constants.OpOperatorPlatformAdmin:
		adm, err := models.GetPlatformAdminByID(db, operatorID)
		if err != nil {
			return "", false
		}
		return strings.TrimSpace(adm.Email), adm.ReceiveEmailNotify
	case constants.OpOperatorTenantUser:
		var user models.TenantUser
		if err := db.Select("email", "receive_email_notify").First(&user, operatorID).Error; err != nil {
			return "", false
		}
		return strings.TrimSpace(user.Email), user.ReceiveEmailNotify
	default:
		return "", false
	}
}

func notifyPlatformAdmins(db *gorm.DB, clientIP, title, content string) {
	admins, err := models.ListActivePlatformAdmins(db)
	if err != nil || len(admins) == 0 {
		return
	}
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || content == "" {
		return
	}
	html := fmt.Sprintf("<p>%s</p>", strings.ReplaceAll(content, "\n", "<br>"))
	for _, adm := range admins {
		sendInbox(db, adm.ID, title, content)
		if !adm.ReceiveEmailNotify || strings.TrimSpace(adm.Email) == "" {
			continue
		}
		mailer := notification.NewMailer(db, adm.ID, clientIP)
		if err := mailer.SendRaw(context.Background(), adm.Email, title, html); err != nil {
			logger.Warn("notify platform admin email failed",
				zap.Uint("adminId", adm.ID), zap.String("email", adm.Email), zap.Error(err))
		}
	}
}

func tenantUserInboxMessage(action, name, summary string) (string, string) {
	switch action {
	case constants.OpActionCreate:
		return "成员账号已创建", fmt.Sprintf("您的组织成员账号已开通（%s）。如有疑问请联系管理员。", name)
	case constants.OpActionUpdate:
		lower := strings.ToLower(summary)
		if strings.Contains(lower, "status") {
			return "账号状态已变更", fmt.Sprintf("您的账号状态已更新：%s", summary)
		}
		if strings.Contains(lower, "roles") {
			return "角色已变更", fmt.Sprintf("您的角色权限已更新：%s", summary)
		}
		if strings.Contains(lower, "departments") {
			return "部门已变更", fmt.Sprintf("您的所属部门已更新：%s", summary)
		}
		return "账号信息已更新", fmt.Sprintf("您的账号信息已更新：%s", summary)
	case constants.OpActionDelete:
		return "成员账号已停用", fmt.Sprintf("您的组织成员账号（%s）已被停用。", name)
	case constants.OpActionRestore:
		return "成员账号已恢复", fmt.Sprintf("您的组织成员账号（%s）已恢复可用。", name)
	default:
		return "", ""
	}
}

func tenantGroupInboxMessage(action, name, summary string) (string, string) {
	switch action {
	case constants.OpActionCreate:
		return "部门已创建", fmt.Sprintf("组织内新增部门「%s」。", name)
	case constants.OpActionUpdate:
		return "部门已更新", fmt.Sprintf("部门「%s」信息已更新。", name)
	case constants.OpActionDelete:
		return "部门已删除", fmt.Sprintf("您所在的部门「%s」已被删除或调整，请关注最新组织信息。", name)
	default:
		if summary != "" {
			return "组织变更通知", summary
		}
		return "", ""
	}
}

func tenantRoleInboxMessage(action, name, summary string) (string, string) {
	switch action {
	case constants.OpActionCreate:
		return "角色已创建", fmt.Sprintf("组织内新增角色「%s」。", name)
	case constants.OpActionUpdate:
		return "角色权限已变更", fmt.Sprintf("您所属角色「%s」的权限已更新，请留意功能变化。", name)
	case constants.OpActionDelete:
		return "角色已删除", fmt.Sprintf("角色「%s」已被删除，您的权限可能已调整。", name)
	default:
		return "", ""
	}
}

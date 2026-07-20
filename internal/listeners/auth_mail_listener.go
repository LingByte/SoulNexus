package listeners

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/notification/inbox"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// InitAuthMailListeners wires auth-related mail signals to async delivery handlers.
func InitAuthMailListeners(db *gorm.DB) {
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

	connectAsync(constants.SigMailWelcome, deliverWelcomeMail)
	connectAsync(constants.SigMailVerificationCode, deliverVerificationCodeMail)
	connectAsync(constants.SigMailDeviceVerifyCode, deliverDeviceVerifyCodeMail)
	connectAsync(constants.SigMailNewDeviceLogin, deliverNewDeviceLoginMail)
}

func deliverWelcomeMail(db *gorm.DB, params ...any) {
	if len(params) == 0 {
		return
	}
	p, ok := params[0].(constants.MailWelcomePayload)
	if !ok || p.UserID == 0 {
		return
	}

	if p.PrincipalType == models.UserDevicePrincipalPlatformAdmin {
		return
	}
	var row models.TenantUser
	if err := db.Select("welcome_notified_at").First(&row, p.UserID).Error; err != nil || row.WelcomeNotifiedAt != nil {
		return
	}

	username := strings.TrimSpace(p.DisplayName)
	if username == "" {
		username = p.Email
	}
	now := time.Now()
	welcomeBody := fmt.Sprintf(
		`您好，%s！

欢迎加入 SoulNexus 智能语音对话平台。账号已就绪，您可以：

· 配置 AI 智能体与提示词，绑定知识库与 NLU 意图模型
· 在智能体与语音会话中接入实时对话能力
· 在个人中心查看站内信、安全设置与用量

如有疑问，可随时通过站内信与系统公告获取最新进展。祝您使用愉快！`,
		username,
	)
	_ = inbox.NewService(db).SendWith(p.UserID, "欢迎加入 SoulNexus", welcomeBody, inbox.SendOptions{
		ActionURL:   "/assistant-manager",
		ActionLabel: "前往智能体",
	})

	if p.ReceiveEmail {
		mailer := notification.NewMailer(db, p.UserID, p.ClientIP)
		if err := mailer.Send(context.Background(), p.Email, notification.TmplWelcome, map[string]any{
			"Username":  username,
			"VerifyURL": "",
		}); err != nil {
			logger.Warn("auth mail: welcome email failed",
				zap.Uint("userId", p.UserID), zap.String("email", p.Email), zap.Error(err))
		}
	}

	_ = db.Model(&models.TenantUser{}).Where("id = ?", p.UserID).Updates(map[string]any{"welcome_notified_at": now}).Error
}

func deliverVerificationCodeMail(db *gorm.DB, params ...any) {
	if len(params) == 0 {
		return
	}
	p, ok := params[0].(constants.MailVerificationCodePayload)
	if !ok || strings.TrimSpace(p.Email) == "" || strings.TrimSpace(p.Code) == "" {
		return
	}
	mailer := notification.NewMailer(db, p.UserID, p.ClientIP)
	if err := mailer.SendVerificationCode(p.Email, p.Code); err != nil {
		logger.Warn("auth mail: verification code failed",
			zap.String("purpose", p.Purpose),
			zap.String("email", p.Email),
			zap.Error(err))
	}
}

func deliverDeviceVerifyCodeMail(db *gorm.DB, params ...any) {
	if len(params) == 0 {
		return
	}
	p, ok := params[0].(constants.MailDeviceVerifyCodePayload)
	if !ok || strings.TrimSpace(p.Email) == "" || strings.TrimSpace(p.Code) == "" {
		return
	}
	mailer := notification.NewMailer(db, p.UserID, p.ClientIP)
	if err := mailer.SendDeviceVerificationCode(p.Email, p.Username, p.Code, p.DeviceKey); err != nil {
		logger.Warn("auth mail: device verify code failed",
			zap.String("email", p.Email),
			zap.Error(err))
	}
}

func deliverNewDeviceLoginMail(db *gorm.DB, params ...any) {
	if len(params) == 0 {
		return
	}
	p, ok := params[0].(constants.MailNewDeviceLoginPayload)
	if !ok || strings.TrimSpace(p.Email) == "" {
		return
	}
	mailer := notification.NewMailer(db, p.UserID, p.ClientIP)
	_ = mailer.SendNewDeviceLoginAlert(
		p.Email, p.Username, p.LoginTime, p.ClientIP, p.Location,
		p.DeviceType, "", "",
		p.IsSuspicious, "", "",
	)
}

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package notification 邮件通知门面：DB 模板 + DB 渠道，唯一入口。
//
// 设计：
//   - 渠道：必须由 internal/listeners 注入 ChannelLoader，从 NotificationChannel 表加载。
//   - 模板：必须由 internal/listeners 注入 TemplateLoader，从 MailTemplate 表加载。
//   - 不再支持任何 embed 兜底 / 静态 MailConfig / Deprecated 兼容入口。
//   - 首次启动时由 bootstrap/seeds.go 把 templates/email/*.html 写入 MailTemplate 表。
package notification

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification/mail"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 模板编码常量；与 MailTemplate.Code、seeds 中的 code 严格一致。
const (
	TmplWelcome            = "welcome"
	TmplVerification       = "verification"
	TmplEmailVerification  = "email_verification"
	TmplPasswordReset      = "password_reset"
	TmplDeviceVerification = "device_verification"
	TmplGroupInvitation    = "group_invitation"
	TmplNewDeviceLogin     = "new_device_login"
)

// ChannelLoader 解析指定 orgID 启用的邮件渠道列表。
type ChannelLoader func(db *gorm.DB, orgID uint) ([]mail.MailConfig, error)

// TemplateLoader 按 (orgID, code, locale) 取 (subject, htmlBody)。
type TemplateLoader func(db *gorm.DB, orgID uint, code, locale string) (subject, htmlBody string, err error)

var (
	channelLoader  ChannelLoader
	templateLoader TemplateLoader
)

// RegisterChannelLoader 由 internal/listeners 注入。
func RegisterChannelLoader(fn ChannelLoader) { channelLoader = fn }

// RegisterTemplateLoader 由 internal/listeners 注入。
func RegisterTemplateLoader(fn TemplateLoader) { templateLoader = fn }

// Mailer 业务层使用的统一邮件发送门面。
type Mailer struct {
	db     *gorm.DB
	orgID  uint
	userID uint
	ip     string
}

// NewMailer 构造 Mailer。db 必传；orgID/userID/ip 任意，仅用于渠道解析和日志归属。
func NewMailer(db *gorm.DB, orgID, userID uint, ip string) *Mailer {
	return &Mailer{db: db, orgID: orgID, userID: userID, ip: ip}
}

// SendRaw 直接以已渲染好的 subject/html 发送（不查模板）。
func (m *Mailer) SendRaw(ctx context.Context, to, subject, htmlBody string) error {
	if m == nil || m.db == nil {
		return errors.New("notification: mailer not initialized with db")
	}
	if channelLoader == nil {
		err := errors.New("notification: channel loader not registered")
		m.recordPreflightFailure(to, subject, htmlBody, "no_channel", err.Error())
		return err
	}
	cfgs, err := channelLoader(m.db, m.orgID)
	if err != nil {
		werr := fmt.Errorf("notification: load channels: %w", err)
		m.recordPreflightFailure(to, subject, htmlBody, "load_channel", werr.Error())
		return werr
	}
	if len(cfgs) == 0 {
		err := errors.New("notification: no enabled mail channels")
		m.recordPreflightFailure(to, subject, htmlBody, "no_channel", err.Error())
		return err
	}
	opts := []mail.MailerOption{}
	if m.userID > 0 {
		opts = append(opts, mail.WithMailLogUserID(m.userID))
	}
	if m.orgID > 0 {
		opts = append(opts, mail.WithMailLogOrgID(m.orgID))
	}
	mailer, err := mail.NewMailer(cfgs, m.db, m.ip, opts...)
	if err != nil {
		m.recordPreflightFailure(to, subject, htmlBody, "init_mailer", err.Error())
		return err
	}
	if err := mailer.SendHTML(ctx, to, subject, htmlBody); err != nil {
		logger.Error("notification: send failed",
			zap.String("to", to), zap.String("subject", subject),
			zap.Uint("userId", m.userID), zap.Error(err))
		return err
	}
	logger.Info("notification: send ok",
		zap.String("to", to), zap.String("subject", subject), zap.Uint("userId", m.userID))
	return nil
}

// recordPreflightFailure 在尚未进入 mail.SendHTML（即未走 mail 包内置 mail_log）之前的失败路径
// 兜底写入一条 mail_logs 记录，便于运营在“邮件日志”中看到所有发送尝试。
func (m *Mailer) recordPreflightFailure(to, subject, htmlBody, channelLabel, errMsg string) {
	if m == nil || m.db == nil {
		return
	}
	if _, dbErr := mail.CreateFailedMailLog(m.db, m.orgID, m.userID, "none", channelLabel, to, subject, htmlBody, errMsg, 0, m.ip); dbErr != nil {
		logger.Error("notification: preflight failed mail_log create failed",
			zap.String("to", to), zap.String("subject", subject), zap.Error(dbErr))
	}
}

// Send 根据模板 code + data 渲染并发送。
func (m *Mailer) Send(ctx context.Context, to, code string, data map[string]any) error {
	if m == nil || m.db == nil {
		return errors.New("notification: mailer not initialized with db")
	}
	if templateLoader == nil {
		err := errors.New("notification: template loader not registered")
		m.recordPreflightFailure(to, "[template:"+code+"]", "", "no_template", err.Error())
		return err
	}
	subj, html, err := templateLoader(m.db, m.orgID, code, "")
	if err != nil {
		werr := fmt.Errorf("notification: load template %q: %w", code, err)
		m.recordPreflightFailure(to, "[template:"+code+"]", "", "load_template", werr.Error())
		return werr
	}
	subjOut, err := renderTemplate(subj, data)
	if err != nil {
		werr := fmt.Errorf("notification: render subject %q: %w", code, err)
		m.recordPreflightFailure(to, subj, html, "render_subject", werr.Error())
		return werr
	}
	htmlOut, err := renderTemplate(html, data)
	if err != nil {
		werr := fmt.Errorf("notification: render html %q: %w", code, err)
		m.recordPreflightFailure(to, subjOut, html, "render_html", werr.Error())
		return werr
	}
	return m.SendRaw(ctx, to, subjOut, htmlOut)
}

func renderTemplate(src string, data any) (string, error) {
	tmpl, err := template.New("email").Parse(src)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ----------------- 业务便捷方法 -----------------

func (m *Mailer) SendWelcomeEmail(to, username, verifyURL string) error {
	return m.Send(context.Background(), to, TmplWelcome, map[string]any{
		"Username": username, "VerifyURL": verifyURL,
	})
}

func (m *Mailer) SendVerificationCode(to, code string) error {
	return m.Send(context.Background(), to, TmplVerification, map[string]any{"Code": code})
}

func (m *Mailer) SendVerificationEmail(to, username, verifyURL string) error {
	return m.Send(context.Background(), to, TmplEmailVerification, map[string]any{
		"Username": username, "VerifyURL": verifyURL,
	})
}

func (m *Mailer) SendPasswordResetEmail(to, username, resetURL string) error {
	return m.Send(context.Background(), to, TmplPasswordReset, map[string]any{
		"Username": username, "ResetURL": resetURL,
	})
}

func (m *Mailer) SendDeviceVerificationCode(to, username, code, deviceID string) error {
	return m.Send(context.Background(), to, TmplDeviceVerification, map[string]any{
		"Username": username, "Code": code, "DeviceID": deviceID,
	})
}

func (m *Mailer) SendGroupInvitationEmail(to, inviteeName, inviterName, groupName, groupType, groupDescription, acceptURL string) error {
	return m.Send(context.Background(), to, TmplGroupInvitation, map[string]any{
		"InviteeName":      inviteeName,
		"InviterName":      inviterName,
		"GroupName":        groupName,
		"GroupType":        groupType,
		"GroupDescription": groupDescription,
		"AcceptURL":        acceptURL,
	})
}

func (m *Mailer) SendNewDeviceLoginAlert(to, username, loginTime, ipAddress, location, deviceType, os, browser string, isSuspicious bool, securityURL, changePasswordURL string) error {
	return m.Send(context.Background(), to, TmplNewDeviceLogin, map[string]any{
		"Username":          username,
		"LoginTime":         loginTime,
		"IPAddress":         ipAddress,
		"Location":          location,
		"DeviceType":        deviceType,
		"OS":                os,
		"Browser":           browser,
		"IsSuspicious":      isSuspicious,
		"SecurityURL":       securityURL,
		"ChangePasswordURL": changePasswordURL,
	})
}

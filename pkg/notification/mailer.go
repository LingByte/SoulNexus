// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package notification 出站通知门面：邮件 / 短信 / Webhook / 站内信。
//
// 设计：
//   - 邮件渠道：internal/listeners 注入 ChannelLoader，从 NotificationChannel 表加载。
//   - 邮件模板：internal/listeners 注入 TemplateLoader，从 MailTemplate 表加载。
//   - Webhook：租户级 HTTP 回调，见 notification/webhook 子包。
package notification

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification/inbox"
	"github.com/LingByte/SoulNexus/pkg/notification/mail"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	TmplWelcome            = "welcome"
	TmplVerification       = "verification"
	TmplEmailVerification  = "email_verification"
	TmplPasswordReset      = "password_reset"
	TmplDeviceVerification = "device_verification"
	TmplGroupInvitation    = "group_invitation"
	TmplNewDeviceLogin     = "new_device_login"
	TmplAIReportDaily  = "ai_report_daily"
	TmplAIReportWeekly = "ai_report_weekly"
)

// ChannelLoader loads enabled system email channels.
type ChannelLoader func(db *gorm.DB) ([]mail.MailConfig, error)

// TemplateLoader loads mail template by code.
type TemplateLoader func(db *gorm.DB, code, locale string) (subject, htmlBody string, err error)

var (
	channelLoader  ChannelLoader
	templateLoader TemplateLoader
)

// RegisterChannelLoader 由 internal/listeners 注入。
func RegisterChannelLoader(fn ChannelLoader) { channelLoader = fn }

// RegisterTemplateLoader 由 internal/listeners 注入。
func RegisterTemplateLoader(fn TemplateLoader) { templateLoader = fn }

// MailerOption configures optional Mailer behaviour.
type MailerOption func(*Mailer)

// WithRetry sets per-send retry policy for multi-channel failover.
func WithRetry(p mail.RetryPolicy) MailerOption {
	return func(m *Mailer) {
		m.retry = p.Normalized()
	}
}

// Mailer 业务层使用的统一邮件发送门面。
type Mailer struct {
	db     *gorm.DB
	userID uint
	ip     string
	retry  mail.RetryPolicy
}

// NewMailer constructs a Mailer. db is required; userID/ip are optional and used for mail log attribution only.
func NewMailer(db *gorm.DB, userID uint, ip string, opts ...MailerOption) *Mailer {
	m := &Mailer{
		db:     db,
		userID: userID,
		ip:     ip,
		retry:  mail.DefaultRetryPolicy(),
	}
	for _, fn := range opts {
		fn(m)
	}
	return m
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
	cfgs, err := channelLoader(m.db)
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
	opts := []mail.MailerOption{mail.WithRetry(m.retry)}
	if m.userID > 0 {
		opts = append(opts, mail.WithMailLogUserID(m.userID))
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

func (m *Mailer) mirrorInbox(title, content string) {
	if m == nil || m.db == nil || m.userID == 0 || title == "" || content == "" {
		return
	}
	if err := inbox.NewService(m.db).Send(m.userID, title, content); err != nil {
		logger.Warn("notification: inbox mirror failed",
			zap.Uint("userId", m.userID), zap.String("title", title), zap.Error(err))
	}
}

func (m *Mailer) recordPreflightFailure(to, subject, htmlBody, channelLabel, errMsg string) {
	if m == nil || m.db == nil {
		return
	}
	if _, dbErr := mail.CreateFailedMailLog(m.db, m.userID, "none", channelLabel, to, subject, htmlBody, errMsg, 0, m.ip); dbErr != nil {
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
	subj, html, err := templateLoader(m.db, code, "")
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

func (m *Mailer) SendWelcomeEmail(to, username, verifyURL string) error {
	err := m.Send(context.Background(), to, TmplWelcome, map[string]any{
		"Username": username, "VerifyURL": verifyURL,
	})
	if err == nil {
		m.mirrorInbox("欢迎注册", fmt.Sprintf("欢迎加入 SoulNexus，%s！请查收邮件完成邮箱验证。", username))
	}
	return err
}

func (m *Mailer) SendVerificationCode(to, code string) error {
	return m.Send(context.Background(), to, TmplVerification, map[string]any{"Code": code})
}

func (m *Mailer) SendVerificationEmail(to, username, verifyURL string) error {
	err := m.Send(context.Background(), to, TmplEmailVerification, map[string]any{
		"Username": username, "VerifyURL": verifyURL,
	})
	if err == nil {
		m.mirrorInbox("邮箱验证", fmt.Sprintf("%s，请查收邮件完成邮箱验证。", username))
	}
	return err
}

func (m *Mailer) SendPasswordResetEmail(to, username, resetURL string) error {
	err := m.Send(context.Background(), to, TmplPasswordReset, map[string]any{
		"Username": username, "ResetURL": resetURL,
	})
	if err == nil {
		m.mirrorInbox("密码重置", fmt.Sprintf("%s，您已申请重置密码，请查收邮件完成操作。如非本人操作请忽略。", username))
	}
	return err
}

func (m *Mailer) SendDeviceVerificationCode(to, username, code, deviceID string) error {
	return m.Send(context.Background(), to, TmplDeviceVerification, map[string]any{
		"Username": username, "Code": code, "DeviceID": deviceID,
	})
}

func (m *Mailer) SendGroupInvitationEmail(to, inviteeName, inviterName, groupName, groupType, groupDescription, acceptURL string) error {
	err := m.Send(context.Background(), to, TmplGroupInvitation, map[string]any{
		"InviteeName":      inviteeName,
		"InviterName":      inviterName,
		"GroupName":        groupName,
		"GroupType":        groupType,
		"GroupDescription": groupDescription,
		"AcceptURL":        acceptURL,
	})
	if err == nil {
		m.mirrorInbox("团队邀请", fmt.Sprintf("%s 邀请您加入「%s」，请查收邮件接受邀请。", inviterName, groupName))
	}
	return err
}

func (m *Mailer) SendNewDeviceLoginAlert(to, username, loginTime, ipAddress, location, deviceType, os, browser string, isSuspicious bool, securityURL, changePasswordURL string) error {
	err := m.Send(context.Background(), to, TmplNewDeviceLogin, map[string]any{
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
	if err == nil {
		title := "新设备登录提醒"
		if isSuspicious {
			title = "可疑登录提醒"
		}
		deviceInfo := formatLoginDeviceLabel(deviceType, os, browser)
		content := fmt.Sprintf("您的账号于 %s 通过 %s 登录（IP：%s）。如非本人操作，请立即修改密码并检查设备管理。", loginTime, deviceInfo, ipAddress)
		m.mirrorInbox(title, content)
	}
	return err
}

func formatLoginDeviceLabel(deviceType, os, browser string) string {
	label := strings.TrimSpace(deviceType)
	switch strings.ToLower(label) {
	case "desktop":
		label = "桌面端"
	case "mobile":
		label = "移动端"
	case "":
		label = "未知设备"
	}
	if os = strings.TrimSpace(os); os != "" {
		label += " · " + os
	}
	if browser = strings.TrimSpace(browser); browser != "" {
		label += " · " + browser
	}
	return label
}

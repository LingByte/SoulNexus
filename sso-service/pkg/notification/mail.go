package notification

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MailProvider defines the interface for email providers
type MailProvider interface {
	SendHTML(to, subject, htmlBody string) (string, error) // Returns messageID
}

// MailConfig email configuration (supports both SMTP and SendCloud)
type MailConfig struct {
	// Provider type: "smtp" or "sendcloud"
	Provider string `json:"provider"`

	// SMTP configuration
	Host     string `json:"host"`     // SMTP server address
	Port     int64  `json:"port"`     // SMTP server port
	Username string `json:"username"` // SMTP username
	Password string `json:"password"` // SMTP password

	// SendCloud configuration
	APIUser string `json:"api_user"` // SendCloud API User
	APIKey  string `json:"api_key"`  // SendCloud API Key

	// Common
	From string `json:"from"` // Sender email address
}

// MailNotification email notification service (supports SMTP and SendCloud)
type MailNotification struct {
	provider  MailProvider
	DB        *gorm.DB
	UserID    uint
	IPAddress string // For tracking emails sent without user context
}

// NewMailNotification creates email notification instance without database
func NewMailNotification(config MailConfig) *MailNotification {
	provider := createProvider(config)
	return &MailNotification{
		provider: provider,
	}
}

// NewMailNotificationWithDB creates email notification instance with database
func NewMailNotificationWithDB(config MailConfig, db *gorm.DB, userID uint) *MailNotification {
	provider := createProvider(config)
	return &MailNotification{
		provider: provider,
		DB:       db,
		UserID:   userID,
	}
}

// NewMailNotificationWithIP creates email notification instance with IP address (for anonymous sends)
func NewMailNotificationWithIP(config MailConfig, db *gorm.DB, ipAddress string) *MailNotification {
	provider := createProvider(config)
	return &MailNotification{
		provider:  provider,
		DB:        db,
		IPAddress: ipAddress,
	}
}

// createProvider creates the appropriate mail provider based on config
func createProvider(config MailConfig) MailProvider {
	if config.Provider == "sendcloud" {
		return NewSendCloudClient(SendCloudConfig{
			APIUser: config.APIUser,
			APIKey:  config.APIKey,
			From:    config.From,
		})
	}
	// Default to SMTP
	return NewSMTPClient(SMTPConfig{
		Host:     config.Host,
		Port:     config.Port,
		Username: config.Username,
		Password: config.Password,
		From:     config.From,
	})
}

// Send sends email
func (m *MailNotification) Send(to, subject, body string) error {
	messageID, err := m.provider.SendHTML(to, subject, body)

	// Only log if we have a messageID
	if messageID != "" && m.DB != nil {
		if m.UserID > 0 {
			CreateMailLog(m.DB, m.UserID, to, subject, messageID)
		} else if m.IPAddress != "" {
			CreateMailLogWithIP(m.DB, 0, to, subject, messageID, m.IPAddress)
		}
	}

	return err
}

// SendHTML sends HTML email
func (m *MailNotification) SendHTML(to, subject, htmlBody string) error {
	messageID, err := m.provider.SendHTML(to, subject, htmlBody)

	logger.Info("Email sent via provider",
		zap.String("to", to),
		zap.String("subject", subject),
		zap.String("messageId", messageID),
		zap.Error(err),
		zap.Uint("userId", m.UserID))

	// Only log if we have a messageID
	if messageID != "" && m.DB != nil {
		if m.UserID > 0 {
			CreateMailLog(m.DB, m.UserID, to, subject, messageID)
		} else if m.IPAddress != "" {
			CreateMailLogWithIP(m.DB, 0, to, subject, messageID, m.IPAddress)
		}
	}

	return err
}

// renderTemplate renders an embedded template with data
func renderTemplate(templateStr string, data interface{}) (string, error) {
	tmpl, err := template.New("email").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

// SendWelcomeEmail sends welcome email using embedded template
func (m *MailNotification) SendWelcomeEmail(to string, username string, verifyURL string) error {
	data := map[string]string{
		"Username":  username,
		"VerifyURL": verifyURL,
	}

	htmlBody, err := renderTemplate(LingEcho.WelcomeHTML, data)
	if err != nil {
		return err
	}

	return m.SendHTML(to, "欢迎加入 LingEcho", htmlBody)
}

// SendVerificationCode sends verification code email using embedded template
func (m *MailNotification) SendVerificationCode(to, code string) error {
	data := map[string]string{
		"Code": code,
	}

	htmlBody, err := renderTemplate(LingEcho.VerificationHTML, data)
	if err != nil {
		return err
	}

	return m.SendHTML(to, "您的 LingEcho 验证码", htmlBody)
}

// SendVerificationEmail sends email verification email using embedded template
func (m *MailNotification) SendVerificationEmail(to, username, verifyURL string) error {
	data := map[string]string{
		"Username":  username,
		"VerifyURL": verifyURL,
	}

	htmlBody, err := renderTemplate(LingEcho.EmailVerificationHTML, data)
	if err != nil {
		return err
	}

	return m.SendHTML(to, "请验证您的邮箱地址", htmlBody)
}

// SendPasswordResetEmail sends password reset email using embedded template
func (m *MailNotification) SendPasswordResetEmail(to, username, resetURL string) error {
	data := map[string]string{
		"Username": username,
		"ResetURL": resetURL,
	}

	htmlBody, err := renderTemplate(LingEcho.PasswordResetHTML, data)
	if err != nil {
		return err
	}

	return m.SendHTML(to, "密码重置请求", htmlBody)
}

// SendDeviceVerificationCode sends device verification code email using embedded template
func (m *MailNotification) SendDeviceVerificationCode(to, username, code, deviceID string) error {
	data := map[string]string{
		"Username": username,
		"Code":     code,
		"DeviceID": deviceID,
	}

	htmlBody, err := renderTemplate(LingEcho.DeviceVerificationHTML, data)
	if err != nil {
		return err
	}

	return m.SendHTML(to, "设备验证码", htmlBody)
}

// SendGroupInvitationEmail sends organization invitation email using embedded template
func (m *MailNotification) SendGroupInvitationEmail(to, inviteeName, inviterName, groupName, groupType, groupDescription, acceptURL string) error {
	data := map[string]string{
		"InviteeName":      inviteeName,
		"InviterName":      inviterName,
		"GroupName":        groupName,
		"GroupType":        groupType,
		"GroupDescription": groupDescription,
		"AcceptURL":        acceptURL,
	}

	htmlBody, err := renderTemplate(LingEcho.GroupInvitationHTML, data)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("您收到了来自 %s 的组织邀请", inviterName)
	return m.SendHTML(to, subject, htmlBody)
}

// SendNewDeviceLoginAlert sends new device login alert email using embedded template
func (m *MailNotification) SendNewDeviceLoginAlert(to, username, loginTime, ipAddress, location, deviceType, os, browser string, isSuspicious bool, securityURL, changePasswordURL string) error {
	data := map[string]interface{}{
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
	}

	htmlBody, err := renderTemplate(LingEcho.NewDeviceLoginHTML, data)
	if err != nil {
		return err
	}

	subject := "新设备登录提醒"
	if isSuspicious {
		subject = "⚠️ 可疑登录警告"
	}

	return m.SendHTML(to, subject, htmlBody)
}

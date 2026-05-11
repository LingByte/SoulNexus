package mail

import (
	"context"
	"fmt"
	"mime"
	"net/mail"
	"strings"
	"time"
	"unicode"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

// SMTP: after a successful handoff to the server we only record StatusSent (no callbacks).
// SendCloud: starts as StatusSent after API success; webhooks may refine to delivered, opened, etc.
const (
	StatusSent         = "sent"
	StatusDelivered    = "delivered"
	StatusFailed       = "failed"
	StatusSoftBounce   = "soft_bounce"
	StatusInvalid      = "invalid"
	StatusSpam         = "spam"
	StatusClicked      = "clicked"
	StatusOpened       = "opened"
	StatusUnsubscribed = "unsubscribed"
	StatusUnknown      = "unknown"
)

const (
	ProviderSMTP      = "smtp"
	ProviderSendCloud = "sendcloud"
)

type MailProvider interface {
	// Kind Mail Provider
	Kind() string

	// SendHTMLWith Send HTML Email with variable substitution (supports {{VarName}} placeholders)
	SendHTMLWith(to, subject, htmlBody string, vars map[string]any) (messageID string, err error)

	// SendTextWith Send Text Email with variable substitution (supports {{VarName}} placeholders)
	SendTextWith(to, subject, textBody string, vars map[string]any) (messageID string, err error)
}

// MailConfig selects provider and credentials (JSON tags for config files).
// Name is optional and used in mail_logs and ops logs to identify the channel when multiple are configured.
type MailConfig struct {
	Provider string `json:"provider"` // "smtp" | "sendcloud"
	Name     string `json:"name"`     // channel label, e.g. "primary-smtp", "backup-sendcloud"
	Host     string `json:"host"`
	Port     int64  `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	APIUser  string `json:"api_user"`
	APIKey   string `json:"api_key"`
	From     string `json:"from"`                // 发件邮箱；也可写 RFC 格式「显示名 <email>」
	FromName string `json:"from_name,omitempty"` // 可选显示名；与 From 中 Name 二选一，此项在纯邮箱 From 时生效
}

// MultiChannelMailConfig is a convenience wrapper for JSON/YAML config: ordered list of channels
// passed to NewMailerMulti / NewMailerMultiWithDB (first channel is default primary; order is preserved for failover).
type MultiChannelMailConfig struct {
	Channels []MailConfig `json:"channels"`
}

// RetryPolicy controls send retries (exponential backoff between attempts).
type RetryPolicy struct {
	MaxAttempts    int           // total tries including the first; default 1 = no retry
	InitialBackoff time.Duration // delay before 2nd attempt; default 200ms
	MaxBackoff     time.Duration // cap; default 5s
}

func defaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 200 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
	}
}

func (p RetryPolicy) normalized() RetryPolicy {
	if p.MaxAttempts < 1 {
		p.MaxAttempts = 1
	}
	if p.InitialBackoff <= 0 {
		p.InitialBackoff = 200 * time.Millisecond
	}
	if p.MaxBackoff <= 0 {
		p.MaxBackoff = 5 * time.Second
	}
	if p.MaxBackoff < p.InitialBackoff {
		p.MaxBackoff = p.InitialBackoff
	}
	return p
}

// MailerOption configures optional behaviour for Mailer.
type MailerOption func(*mailerOptions)

type mailerOptions struct {
	retry         RetryPolicy
	mailLogUserID *uint // 写入 mail_logs.user_id；与 NewMailerMultiWithDB 的显式 userID 同时存在时以后者为准
	mailLogOrgID  *uint // 写入 mail_logs.org_id；默认 0（系统/未绑定）
}

// WithRetry sets retry policy (merged with defaults for zero fields if you only set MaxAttempts).
func WithRetry(p RetryPolicy) MailerOption {
	return func(o *mailerOptions) {
		o.retry = p
	}
}

// WithMailLogUserID sets mail_logs.user_id for sends that use NewMailerMultiWithIP (no session user on mailer).
func WithMailLogUserID(uid uint) MailerOption {
	return func(o *mailerOptions) {
		if uid == 0 {
			o.mailLogUserID = nil
			return
		}
		u := uid
		o.mailLogUserID = &u
	}
}

// WithMailLogOrgID sets mail_logs.org_id for DB logging.
func WithMailLogOrgID(orgID uint) MailerOption {
	return func(o *mailerOptions) {
		if orgID == 0 {
			o.mailLogOrgID = nil
			return
		}
		v := orgID
		o.mailLogOrgID = &v
	}
}

// InitialMailStatus returns the DB status right after a successful provider send.
func InitialMailStatus(kind string) string {
	switch kind {
	case ProviderSMTP:
		return StatusSent
	case ProviderSendCloud:
		return StatusSent
	default:
		return StatusSent
	}
}

// SendCloudEventToStatus maps SendCloud webhook event codes (numeric or common names) to MailLog status.
func SendCloudEventToStatus(event string) string {
	e := strings.TrimSpace(strings.ToLower(event))
	switch e {
	case "1", "deliver", "delivered":
		return StatusDelivered
	case "3", "spam":
		return StatusSpam
	case "4", "invalid":
		return StatusInvalid
	case "5", "soft_bounce", "softbounce":
		return StatusSoftBounce
	case "10", "click", "clicked":
		return StatusClicked
	case "11", "open", "opened":
		return StatusOpened
	case "12", "unsubscribe", "unsubscribed":
		return StatusUnsubscribed
	case "18", "request":
		return StatusSent
	default:
		return StatusUnknown
	}
}

// sleepCtx waits up to d or returns ctx.Err().
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// ReplacePlaceholders replaces {{VarName}} placeholders in template with values from vars map.
func ReplacePlaceholders(template string, vars map[string]any) string {
	result := template
	for key, value := range vars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// ParsedSender is the envelope address (SMTP MAIL FROM / SendCloud `from`) plus display name and full RFC From header line.
type ParsedSender struct {
	Envelope   string // bare email only
	Display    string // UTF-8 display name for SendCloud fromName; may be empty
	HeaderFrom string // value for MIME "From:" header (includes encoded-word if needed)
}

// ParseMailSender parses MailConfig.from (optional "Name <email>") and optional from_name fallback.
func ParseMailSender(fromField, nameFallback string) (ParsedSender, error) {
	fromField = strings.TrimSpace(fromField)
	nameFallback = strings.TrimSpace(nameFallback)
	if fromField == "" {
		return ParsedSender{}, fmt.Errorf("empty from address")
	}
	if a, err := mail.ParseAddress(fromField); err == nil && a.Address != "" {
		envelope := strings.TrimSpace(a.Address)
		disp := strings.TrimSpace(a.Name)
		if disp == "" {
			disp = nameFallback
		}
		return ParsedSender{
			Envelope:   envelope,
			Display:    disp,
			HeaderFrom: formatMailFromHeader(disp, envelope),
		}, nil
	}
	if strings.Contains(fromField, "@") && !strings.ContainsAny(fromField, "<>") {
		envelope := strings.TrimSpace(fromField)
		return ParsedSender{
			Envelope:   envelope,
			Display:    nameFallback,
			HeaderFrom: formatMailFromHeader(nameFallback, envelope),
		}, nil
	}
	return ParsedSender{}, fmt.Errorf("invalid from address %q", fromField)
}

func formatMailFromHeader(displayName, email string) string {
	dn := strings.TrimSpace(displayName)
	em := strings.TrimSpace(email)
	if em == "" {
		return ""
	}
	if dn == "" {
		return em
	}
	if isASCIIString(dn) {
		esc := strings.ReplaceAll(dn, `\`, `\\`)
		esc = strings.ReplaceAll(esc, `"`, `\"`)
		return fmt.Sprintf("\"%s\" <%s>", esc, em)
	}
	return fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("UTF-8", dn), em)
}

func isASCIIString(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

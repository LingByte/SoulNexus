package mail

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"gorm.io/gorm"
)

// providerSlot is one send channel (SMTP or SendCloud) with an optional label.
type providerSlot struct {
	label    string
	provider MailProvider
}

type mailerRuntime struct {
	db        *gorm.DB
	orgID     uint
	userID    uint
	ipAddress string
}

// Mailer sends HTML mail over one or more channels: each send round-robins the starting channel,
// then fails over to the rest until one succeeds or all are exhausted. Per-channel retries use RetryPolicy.
type Mailer struct {
	channels  []providerSlot
	retry     RetryPolicy
	rrCounter uint32
	rt        mailerRuntime
}

// channelLabel returns MailConfig.Name or a short derived label for logs.
func channelLabel(cfg MailConfig) string {
	if strings.TrimSpace(cfg.Name) != "" {
		return strings.TrimSpace(cfg.Name)
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case ProviderSendCloud:
		if cfg.APIUser != "" {
			return "sendcloud:" + cfg.APIUser
		}
		return ProviderSendCloud
	default:
		if cfg.Host != "" {
			return fmt.Sprintf("smtp:%s:%d", cfg.Host, cfg.Port)
		}
		return ProviderSendCloud
	}
}

// NewMailer builds a Mailer for one or more channels.
func NewMailer(channels []MailConfig, db *gorm.DB, ip string, opts ...MailerOption) (*Mailer, error) {
	if len(channels) == 0 {
		return nil, errors.New("notification: at least one mail channel is required")
	}
	slots := make([]providerSlot, 0, len(channels))
	for i, cfg := range channels {
		provider, err := NewProviderFromConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("notification: channel %d (%s): %w", i, channelLabel(cfg), err)
		}
		slots = append(slots, providerSlot{label: channelLabel(cfg), provider: provider})
	}
	o := mailerOptions{retry: defaultRetryPolicy()}
	for _, fn := range opts {
		fn(&o)
	}
	rt := mailerRuntime{db: db, ipAddress: ip}
	if rt.userID == 0 && o.mailLogUserID != nil {
		rt.userID = *o.mailLogUserID
	}
	if rt.orgID == 0 && o.mailLogOrgID != nil {
		rt.orgID = *o.mailLogOrgID
	}
	return &Mailer{
		channels: slots,
		retry:    o.retry.normalized(),
		rt:       rt,
	}, nil
}

// SendHTML sends HTML mail with per-channel retries and cross-channel failover.
func (m *Mailer) SendHTML(ctx context.Context, to, subject, htmlBody string) error {
	if strings.TrimSpace(to) == "" {
		return errors.New("notification: empty recipient")
	}
	if len(m.channels) == 0 {
		return errors.New("notification: no channels configured")
	}
	policy := m.retry
	n := len(m.channels)
	start := int(atomic.AddUint32(&m.rrCounter, 1)-1) % n
	order := make([]providerSlot, n)
	for i := 0; i < n; i++ {
		order[i] = m.channels[(start+i)%n]
	}
	var lastErr error
	var failParts []string
	totalAttempts := 0
	for chIdx, slot := range order {
		backoff := policy.InitialBackoff
		for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			totalAttempts++
			messageID, err := slot.provider.SendHTMLWith(to, subject, htmlBody, nil)
			if err == nil {
				logger.Infof("notification: send ok - to=%s subject=%s messageId=%s channel=%s userId=%d provider=%s",
					to, subject, messageID, slot.label, m.rt.userID, string(slot.provider.Kind()))
				if m.rt.db != nil {
					status := InitialMailStatus(slot.provider.Kind())
					_, dbErr := CreateMailLog(m.rt.db, m.rt.orgID, m.rt.userID, string(slot.provider.Kind()), slot.label, to, subject, htmlBody, messageID, status, m.rt.ipAddress)
					if dbErr != nil {
						logger.Errorf("notification: mail log create failed - to=%s messageId=%s channel=%s err=%v",
							to, messageID, slot.label, dbErr)
					}
				}
				return nil
			}
			lastErr = err
			failParts = append(failParts, fmt.Sprintf("[%s] %v", slot.label, err))
			logger.Warnf("notification: send attempt failed - channelIndex=%d channel=%s attempt=%d maxAttempts=%d to=%s provider=%s err=%v",
				chIdx, slot.label, attempt, policy.MaxAttempts, to, string(slot.provider.Kind()), err)
			if attempt >= policy.MaxAttempts {
				break
			}
			if err := sleepCtx(ctx, backoff); err != nil {
				return err
			}
			if next := backoff * 2; next > policy.MaxBackoff {
				backoff = policy.MaxBackoff
			} else {
				backoff = next
			}
		}
	}
	errMsg := strings.Join(failParts, "; ")
	if len(errMsg) > 4000 {
		errMsg = errMsg[:4000] + "…"
	}
	logger.Errorf("notification: all channels failed - to=%s subject=%s channel=multi userId=%d err=%v",
		to, subject, m.rt.userID, lastErr)
	if m.rt.db != nil {
		_, dbErr := CreateFailedMailLog(m.rt.db, m.rt.orgID, m.rt.userID, "multi", "", to, subject, htmlBody, errMsg, totalAttempts, m.rt.ipAddress)
		if dbErr != nil {
			logger.Errorf("notification: failed mail log create failed - to=%s err=%v", to, dbErr)
		}
	}
	if lastErr == nil {
		lastErr = errors.New("notification: all channels failed")
	}
	return lastErr
}

// NewProviderFromConfig returns the provider implementation for a single MailConfig.
func NewProviderFromConfig(cfg MailConfig) (MailProvider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case ProviderSendCloud:
		if cfg.APIUser == "" || cfg.APIKey == "" || cfg.From == "" {
			return nil, fmt.Errorf("notification: sendcloud requires api_user, api_key, from")
		}
		return NewSendCloudClient(SendCloudConfig{
			APIUser:  cfg.APIUser,
			APIKey:   cfg.APIKey,
			From:     cfg.From,
			FromName: cfg.FromName,
		})
	default:
		if cfg.Host == "" || cfg.Port == 0 || cfg.From == "" {
			return nil, fmt.Errorf("notification: smtp requires host, port, from")
		}
		return NewSMTPClient(SMTPConfig{
			Host:     cfg.Host,
			Port:     cfg.Port,
			Username: cfg.Username,
			Password: cfg.Password,
			From:     cfg.From,
			FromName: cfg.FromName,
		})
	}
}

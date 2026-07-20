// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	deliveryTimeout   = 8 * time.Second
	defaultMaxAttempt = 5
)

// Payload is the JSON body POSTed to tenant webhook URLs.
type Payload struct {
	Event     string         `json:"event"`
	Timestamp string         `json:"timestamp"`
	TenantID  uint           `json:"tenantId"`
	CallID    string         `json:"callId,omitempty"`
	From      string         `json:"from,omitempty"`
	To        string         `json:"to,omitempty"`
	Direction string         `json:"direction,omitempty"`
	Extra     map[string]any `json:"extra,omitempty"`
}

// Dispatch emits one event to all matching enabled webhooks for the tenant (async).
func Dispatch(db *gorm.DB, lg *zap.Logger, tenantID uint, event, callID, from, to, direction string, extra map[string]any) {
	if db == nil || tenantID == 0 || strings.TrimSpace(event) == "" {
		return
	}
	var hooks []models.TenantWebhook
	if err := db.Where("tenant_id = ? AND enabled = ?", tenantID, true).Find(&hooks).Error; err != nil {
		if lg != nil {
			lg.Warn("webhook list failed", zap.Uint("tenant_id", tenantID), zap.Error(err))
		}
		return
	}
	if len(hooks) == 0 {
		return
	}
	body := Payload{
		Event:     event,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		TenantID:  tenantID,
		CallID:    strings.TrimSpace(callID),
		From:      strings.TrimSpace(from),
		To:        strings.TrimSpace(to),
		Direction: strings.TrimSpace(direction),
		Extra:     extra,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return
	}
	for _, hook := range hooks {
		if !hookSubscribed(hook, event) {
			continue
		}
		h := hook
		go deliverAttempt(db, lg, h, event, callID, raw, 1, defaultMaxAttempt, 0)
	}
}

func hookSubscribed(h models.TenantWebhook, event string) bool {
	if len(h.Events) == 0 {
		return true
	}
	var events []string
	if err := json.Unmarshal(h.Events, &events); err != nil {
		return false
	}
	for _, e := range events {
		if strings.TrimSpace(e) == event {
			return true
		}
	}
	return false
}

func deliverAttempt(db *gorm.DB, lg *zap.Logger, hook models.TenantWebhook, event, callID string, raw []byte, attempt, maxAttempts int, existingID uint) {
	url := strings.TrimSpace(hook.URL)
	if url == "" {
		return
	}
	if err := utils.ValidateURLForSSRF(url); err != nil {
		recordOrUpdateDelivery(db, existingID, hook, event, callID, raw, attempt, maxAttempts, 0, "ssrf blocked: "+err.Error(), "dlq", nil)
		if lg != nil {
			lg.Warn("webhook URL blocked by SSRF guard",
				zap.Uint("webhook_id", hook.ID),
				zap.String("event", event),
				zap.Error(err),
			)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), deliveryTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		scheduleRetryOrDLQ(db, lg, existingID, hook, event, callID, raw, attempt, maxAttempts, 0, "build request: "+err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SoulNexus-Webhook/1.0")
	req.Header.Set("X-Webhook-Event", event)
	if sec := strings.TrimSpace(hook.Secret); sec != "" {
		mac := hmac.New(sha256.New, []byte(sec))
		_, _ = mac.Write(raw)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Webhook-Signature", "sha256="+sig)
	}
	resp, err := http.DefaultClient.Do(req)
	code := 0
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	} else {
		code = resp.StatusCode
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if code < 200 || code >= 300 {
			errMsg = "unexpected status"
		}
	}
	if errMsg == "" {
		recordOrUpdateDelivery(db, existingID, hook, event, callID, raw, attempt, maxAttempts, code, "", "success", nil)
		return
	}
	scheduleRetryOrDLQ(db, lg, existingID, hook, event, callID, raw, attempt, maxAttempts, code, errMsg)
}

func scheduleRetryOrDLQ(db *gorm.DB, lg *zap.Logger, existingID uint, hook models.TenantWebhook, event, callID string, raw []byte, attempt, maxAttempts, code int, errMsg string) {
	if attempt >= maxAttempts {
		recordOrUpdateDelivery(db, existingID, hook, event, callID, raw, attempt, maxAttempts, code, errMsg, "dlq", nil)
		if lg != nil {
			lg.Warn("webhook delivery moved to DLQ",
				zap.Uint("webhook_id", hook.ID),
				zap.String("event", event),
				zap.Int("attempt", attempt),
				zap.String("error", errMsg),
			)
		}
		return
	}
	next := time.Now().Add(retryBackoff(attempt))
	recordOrUpdateDelivery(db, existingID, hook, event, callID, raw, attempt, maxAttempts, code, errMsg, "pending", &next)
	if lg != nil {
		lg.Warn("webhook delivery failed; scheduled retry",
			zap.Uint("webhook_id", hook.ID),
			zap.String("event", event),
			zap.Int("attempt", attempt),
			zap.Time("next_retry_at", next),
			zap.String("error", errMsg),
		)
	}
}

func retryBackoff(attempt int) time.Duration {
	// 2s, 4s, 8s, 16s, 32s (capped)
	d := time.Duration(1<<attempt) * time.Second
	if d > 60*time.Second {
		d = 60 * time.Second
	}
	return d
}

func recordOrUpdateDelivery(db *gorm.DB, existingID uint, hook models.TenantWebhook, event, callID string, raw []byte, attempt, maxAttempts, httpCode int, errMsg, status string, next *time.Time) {
	if db == nil {
		return
	}
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempt
	}
	if existingID > 0 {
		updates := map[string]any{
			"status":        status,
			"http_code":     httpCode,
			"error":         errMsg,
			"attempt":       attempt,
			"max_attempts":  maxAttempts,
			"next_retry_at": next,
		}
		_ = db.Model(&models.TenantWebhookDelivery{}).Where("id = ?", existingID).Updates(updates).Error
		return
	}
	row := models.TenantWebhookDelivery{
		TenantID:    hook.TenantID,
		WebhookID:   hook.ID,
		Event:       event,
		CallID:      callID,
		Status:      status,
		HTTPCode:    httpCode,
		Error:       errMsg,
		Payload:     string(raw),
		Attempt:     attempt,
		MaxAttempts: maxAttempts,
		NextRetryAt: next,
	}
	_ = db.Create(&row).Error
}

// ProcessPendingRetries delivers due pending webhook rows (single-node worker).
func ProcessPendingRetries(db *gorm.DB, lg *zap.Logger, limit int) int {
	if db == nil || limit <= 0 {
		return 0
	}
	now := time.Now()
	var rows []models.TenantWebhookDelivery
	if err := db.Where("status = ? AND next_retry_at IS NOT NULL AND next_retry_at <= ?", "pending", now).
		Order("next_retry_at ASC").Limit(limit).Find(&rows).Error; err != nil {
		if lg != nil {
			lg.Warn("webhook retry poll failed", zap.Error(err))
		}
		return 0
	}
	n := 0
	for _, row := range rows {
		var hook models.TenantWebhook
		if err := db.First(&hook, row.WebhookID).Error; err != nil {
			_ = db.Model(&row).Updates(map[string]any{"status": "dlq", "error": "webhook missing"}).Error
			continue
		}
		nextAttempt := row.Attempt + 1
		maxAttempts := row.MaxAttempts
		if maxAttempts <= 0 {
			maxAttempts = defaultMaxAttempt
		}
		deliverAttempt(db, lg, hook, row.Event, row.CallID, []byte(row.Payload), nextAttempt, maxAttempts, row.ID)
		n++
	}
	return n
}

// ValidEvent returns true when event is in the supported catalog.
func ValidEvent(event string) bool {
	for _, e := range constants.AllWebhookEvents {
		if e == event {
			return true
		}
	}
	return false
}

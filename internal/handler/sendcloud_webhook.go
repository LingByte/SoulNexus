// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification/mail"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// handleSendCloudWebhook 处理 SendCloud 单条 webhook（JSON 或 x-www-form-urlencoded 均可）。
func (h *Handlers) handleSendCloudWebhook(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusBadRequest, errors.New("failed to read request body"))
		return
	}
	bodyBytes = []byte(strings.TrimSpace(string(bodyBytes)))
	logger.Info("sendcloud webhook received",
		zap.Int("length", len(bodyBytes)),
		zap.String("content-type", c.Request.Header.Get("Content-Type")))

	if len(bodyBytes) == 0 {
		response.Success(c, "webhook received", nil)
		return
	}

	if err := mail.ApplySendCloudWebhookToMailLog(h.db, bodyBytes); err != nil {
		logger.Warn("sendcloud webhook apply failed", zap.Error(err), zap.String("body", string(bodyBytes)))
		response.AbortWithStatusJSON(c, http.StatusBadRequest, err)
		return
	}
	response.Success(c, "webhook processed", nil)
}

// handleSendCloudWebhookBatch 处理批量 webhook 数组。
func (h *Handlers) handleSendCloudWebhookBatch(c *gin.Context) {
	var events []mail.SendCloudWebhookEvent
	if err := c.ShouldBindJSON(&events); err != nil {
		response.AbortWithStatusJSON(c, http.StatusBadRequest, errors.New("failed to parse webhook events"))
		return
	}
	for _, ev := range events {
		if ev.MessageID == "" {
			continue
		}
		status := mail.SendCloudEventToStatus(ev.Event)
		errorMsg := ""
		if ev.SmtpError != "" {
			errorMsg = ev.SmtpError
		} else if ev.SmtpStatus != "" {
			errorMsg = ev.SmtpStatus
		}
		if err := mail.UpdateMailLogStatusByMessageID(h.db, ev.MessageID, mail.ProviderSendCloud, status, errorMsg); err != nil {
			logger.Error("sendcloud batch update failed",
				zap.String("messageId", ev.MessageID),
				zap.Error(err))
		}
	}
	response.Success(c, "webhooks processed", gin.H{"count": len(events)})
}

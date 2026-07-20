package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/notification/mail"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// registerSendCloudWebhookRoutes mounts inbound SendCloud delivery webhooks on the public API
// (no JWT — SendCloud servers cannot authenticate as tenant users).
//
// Configure in SendCloud console:
//
//	POST {API_PREFIX}/webhooks/sendcloud
func (h *Handlers) registerSendCloudWebhookRoutes(r *humax.Group) {
	limit := middleware.AuthRateLimiter(120, time.Minute, 40)
	g := r.Group("webhooks/sendcloud")
	{
		g.POST("", limit, h.handleSendCloudWebhook)
		g.POST("/batch", limit, h.handleSendCloudWebhookBatch)
	}
}

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
		response.SuccessI18n(c, i18n.KeyWebhookReceived, nil)
		return
	}

	if err := mail.ApplySendCloudWebhookToMailLog(h.db, bodyBytes); err != nil {
		logger.Warn("sendcloud webhook apply failed", zap.Error(err), zap.String("body", string(bodyBytes)))
		response.AbortWithStatusJSON(c, http.StatusBadRequest, err)
		return
	}
	response.SuccessI18n(c, i18n.KeyWebhookProcessed, nil)
}

func (h *Handlers) handleSendCloudWebhookBatch(c *gin.Context) {
	var events []mail.SendCloudWebhookEvent
	if err := c.ShouldBindJSON(&events); err != nil {
		response.AbortWithStatusJSON(c, http.StatusBadRequest, errors.New("failed to parse webhook events"))
		return
	}
	applied := 0
	for _, ev := range events {
		if err := mail.ApplySendCloudWebhookEvent(h.db, &ev); err != nil {
			logger.Error("sendcloud batch update failed",
				zap.String("messageId", ev.MessageID),
				zap.Error(err))
			continue
		}
		if ev.MessageID != "" {
			applied++
		}
	}
	response.SuccessI18n(c, i18n.KeyWebhookProcessed, gin.H{"count": len(events), "applied": applied})
}

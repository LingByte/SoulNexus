package handlers

import (
	"errors"
	"io"
	"net/http"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// handleSendCloudWebhook handles SendCloud webhook events
func (h *Handlers) handleSendCloudWebhook(c *gin.Context) {
	logger.Info("=== SendCloud Webhook Request Received ===")
	logger.Info("Request Method", zap.String("method", c.Request.Method))
	logger.Info("Request URL", zap.String("url", c.Request.URL.String()))
	logger.Info("Request Headers", zap.Any("headers", c.Request.Header))
	logger.Info("Request Content-Type", zap.String("content-type", c.Request.Header.Get("Content-Type")))

	// Read request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("Failed to read webhook body", zap.Error(err))
		response.AbortWithStatusJSON(c, http.StatusBadRequest, errors.New("failed to read request body"))
		return
	}

	logger.Info("Received SendCloud webhook body", zap.String("body", string(bodyBytes)))
	logger.Info("Webhook body length", zap.Int("length", len(bodyBytes)))

	// Parse webhook event
	event, err := notification.ParseSendCloudWebhookEvent(bodyBytes)
	if err != nil {
		logger.Error("Failed to parse webhook event", zap.Error(err), zap.String("body", string(bodyBytes)))
		response.AbortWithStatusJSON(c, http.StatusBadRequest, errors.New("failed to parse webhook event"))
		return
	}

	logger.Info("Parsed webhook event successfully",
		zap.String("event", event.Event),
		zap.String("messageId", event.MessageID),
		zap.String("email", event.Email),
		zap.String("smtpStatus", event.SmtpStatus),
		zap.String("smtpError", event.SmtpError))

	// Skip "request" events as they don't have useful tracking information
	if event.Event == "request" {
		logger.Info("Skipping request event - no tracking data", zap.String("messageId", event.MessageID))
		response.Success(c, "webhook received", nil)
		return
	}

	// Skip if no messageId
	if event.MessageID == "" {
		logger.Warn("Webhook event has no messageId, skipping", zap.String("event", event.Event))
		response.Success(c, "webhook received", nil)
		return
	}

	// Get SendCloud config from environment or database
	// For now, we'll just update the email log status
	// In production, you should validate the webhook signature

	// Update email log status based on webhook event
	if err := h.db.Model(&notification.MailLog{}).
		Where("message_id = ?", event.MessageID).
		First(&notification.MailLog{}).Error; err != nil {
		// Log not found, but still return 200 to acknowledge receipt
		logger.Warn("Email log not found for message", zap.String("messageId", event.MessageID))
		response.Success(c, "webhook received", nil)
		return
	}

	// Convert event type to status
	status := notification.EventTypeToStatus(event.Event)

	// Build error message if applicable
	errorMsg := ""
	if event.SmtpError != "" {
		errorMsg = event.SmtpError
	}

	// Update email log status
	if err := h.db.Model(&notification.MailLog{}).
		Where("message_id = ?", event.MessageID).
		Updates(map[string]interface{}{
			"status":    status,
			"error_msg": errorMsg,
		}).Error; err != nil {
		logger.Error("Failed to update email log", zap.Error(err), zap.String("messageId", event.MessageID))
	}

	logger.Info("Webhook processed successfully", zap.String("messageId", event.MessageID), zap.String("status", status))
	response.Success(c, "webhook processed", nil)
}

// handleSendCloudWebhookBatch handles batch SendCloud webhook events
func (h *Handlers) handleSendCloudWebhookBatch(c *gin.Context) {
	// Parse multiple webhook events (array)
	var events []notification.SendCloudWebhookEvent
	if err := c.ShouldBindJSON(&events); err != nil {
		response.AbortWithStatusJSON(c, http.StatusBadRequest, errors.New("failed to parse webhook events"))
		return
	}

	// Process each event
	for _, event := range events {
		status := notification.EventTypeToStatus(event.Event)
		errorMsg := ""
		if event.SmtpError != "" {
			errorMsg = event.SmtpError
		}

		// Update email log status
		if err := h.db.Model(&notification.MailLog{}).
			Where("message_id = ?", event.MessageID).
			Updates(map[string]interface{}{
				"status":    status,
				"error_msg": errorMsg,
			}).Error; err != nil {
			logger.Error("Failed to update email log", zap.Error(err), zap.String("messageId", event.MessageID))
		}
	}

	response.Success(c, "webhooks processed", gin.H{
		"count": len(events),
	})
}

package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateAlertRuleRequest Create alert rule request
type CreateAlertRuleRequest struct {
	Name          string                       `json:"name" binding:"required"`
	Description   string                       `json:"description"`
	AlertType     models.AlertType             `json:"alertType" binding:"required"`
	Severity      models.AlertSeverity         `json:"severity" binding:"required"`
	Conditions    *models.AlertCondition       `json:"conditions" binding:"required"`
	Channels      []models.NotificationChannel `json:"channels" binding:"required"`
	WebhookURL    string                       `json:"webhookUrl"`
	WebhookMethod string                       `json:"webhookMethod"`
	Cooldown      int                          `json:"cooldown"`
	Enabled       bool                         `json:"enabled"`
}

// UpdateAlertRuleRequest Update alert rule request
type UpdateAlertRuleRequest struct {
	Name          *string                       `json:"name"`
	Description   *string                       `json:"description"`
	Severity      *models.AlertSeverity         `json:"severity"`
	Conditions    *models.AlertCondition        `json:"conditions"`
	Channels      *[]models.NotificationChannel `json:"channels"`
	WebhookURL    *string                       `json:"webhookUrl"`
	WebhookMethod *string                       `json:"webhookMethod"`
	Cooldown      *int                          `json:"cooldown"`
	Enabled       *bool                         `json:"enabled"`
}

// CreateAlertRule Create alert rule
func (h *Handlers) CreateAlertRule(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	var req CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Parameter error", err.Error())
		return
	}

	// Validate alert type
	switch req.AlertType {
	case models.AlertTypeSystemError, models.AlertTypeQuotaExceeded, models.AlertTypeServiceError, models.AlertTypeCustom:
		// Valid type
	default:
		response.Fail(c, "Parameter error", "Invalid alert type")
		return
	}

	// Validate severity
	switch req.Severity {
	case models.AlertSeverityCritical, models.AlertSeverityHigh, models.AlertSeverityMedium, models.AlertSeverityLow:
		// Valid severity
	default:
		response.Fail(c, "Parameter error", "Invalid severity")
		return
	}

	// Validate notification channels
	if len(req.Channels) == 0 {
		response.Fail(c, "Parameter error", "At least one notification channel is required")
		return
	}

	for _, channel := range req.Channels {
		switch channel {
		case models.NotificationChannelEmail, models.NotificationChannelInternal, models.NotificationChannelWebhook, models.NotificationChannelSMS:
			// Valid channel
		default:
			response.Fail(c, "Parameter error", fmt.Sprintf("Invalid notification channel: %s", channel))
			return
		}
	}

	// If using webhook, URL must be provided
	hasWebhook := false
	for _, ch := range req.Channels {
		if ch == models.NotificationChannelWebhook {
			hasWebhook = true
			break
		}
	}
	if hasWebhook && req.WebhookURL == "" {
		response.Fail(c, "Parameter error", "Webhook URL is required when using Webhook notification")
		return
	}

	// Set default values
	if req.Cooldown <= 0 {
		req.Cooldown = 300 // Default 5 minutes
	}
	if req.WebhookMethod == "" {
		req.WebhookMethod = "POST"
	}

	// Create alert rule
	rule := models.AlertRule{
		UserID:        user.ID,
		Name:          req.Name,
		Description:   req.Description,
		AlertType:     req.AlertType,
		Severity:      req.Severity,
		WebhookURL:    req.WebhookURL,
		WebhookMethod: req.WebhookMethod,
		Cooldown:      req.Cooldown,
		Enabled:       req.Enabled,
	}

	// Set conditions
	if err := rule.SetConditions(req.Conditions); err != nil {
		response.Fail(c, "Parameter error", "Condition format error: "+err.Error())
		return
	}

	// Set notification channels
	if err := rule.SetChannels(req.Channels); err != nil {
		response.Fail(c, "Parameter error", "Notification channel format error: "+err.Error())
		return
	}

	if err := h.db.Create(&rule).Error; err != nil {
		response.Fail(c, "Failed to create alert rule", err.Error())
		return
	}

	response.Success(c, "Alert rule created successfully", rule)
}

// ListAlertRules List alert rules
func (h *Handlers) ListAlertRules(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	var rules []models.AlertRule
	query := h.db.Where("user_id = ?", user.ID)

	// Optional: Filter by type
	if alertType := c.Query("alertType"); alertType != "" {
		query = query.Where("alert_type = ?", alertType)
	}

	// Optional: Filter by enabled status
	if enabled := c.Query("enabled"); enabled != "" {
		enabledBool := enabled == "true"
		query = query.Where("enabled = ?", enabledBool)
	}

	if err := query.Order("created_at DESC").Find(&rules).Error; err != nil {
		response.Fail(c, "Query failed", err.Error())
		return
	}

	response.Success(c, "Query successful", rules)
}

// GetAlertRule Get alert rule details
func (h *Handlers) GetAlertRule(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "Parameter error", "Invalid rule ID")
		return
	}

	var rule models.AlertRule
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "Rule not found", nil)
		} else {
			response.Fail(c, "Query failed", err.Error())
		}
		return
	}

	response.Success(c, "Query successful", rule)
}

// UpdateAlertRule Update alert rule
func (h *Handlers) UpdateAlertRule(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "Parameter error", "Invalid rule ID")
		return
	}

	var rule models.AlertRule
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "Rule not found", nil)
		} else {
			response.Fail(c, "Query failed", err.Error())
		}
		return
	}

	var req UpdateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Parameter error", err.Error())
		return
	}

	// Update fields
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.Severity != nil {
		// Validate severity
		switch *req.Severity {
		case models.AlertSeverityCritical, models.AlertSeverityHigh, models.AlertSeverityMedium, models.AlertSeverityLow:
			rule.Severity = *req.Severity
		default:
			response.Fail(c, "Parameter error", "Invalid severity")
			return
		}
	}
	if req.Conditions != nil {
		if err := rule.SetConditions(req.Conditions); err != nil {
			response.Fail(c, "Parameter error", "Condition format error: "+err.Error())
			return
		}
	}
	if req.Channels != nil {
		// Validate notification channels
		if len(*req.Channels) == 0 {
			response.Fail(c, "Parameter error", "At least one notification channel is required")
			return
		}
		hasWebhook := false
		for _, ch := range *req.Channels {
			if ch == models.NotificationChannelWebhook {
				hasWebhook = true
				break
			}
		}
		if hasWebhook {
			webhookURL := req.WebhookURL
			if webhookURL == nil {
				webhookURL = &rule.WebhookURL
			}
			if webhookURL == nil || *webhookURL == "" {
				response.Fail(c, "Parameter error", "Webhook URL is required when using Webhook notification")
				return
			}
		}
		if err := rule.SetChannels(*req.Channels); err != nil {
			response.Fail(c, "Parameter error", "Notification channel format error: "+err.Error())
			return
		}
	}
	if req.WebhookURL != nil {
		rule.WebhookURL = *req.WebhookURL
	}
	if req.WebhookMethod != nil {
		rule.WebhookMethod = *req.WebhookMethod
	}
	if req.Cooldown != nil {
		if *req.Cooldown <= 0 {
			response.Fail(c, "Parameter error", "Cooldown time must be greater than 0")
			return
		}
		rule.Cooldown = *req.Cooldown
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := h.db.Save(&rule).Error; err != nil {
		response.Fail(c, "Update failed", err.Error())
		return
	}

	response.Success(c, "Update successful", rule)
}

// DeleteAlertRule Delete alert rule
func (h *Handlers) DeleteAlertRule(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "Parameter error", "Invalid rule ID")
		return
	}

	// Check if rule exists and belongs to current user
	var rule models.AlertRule
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "Rule not found", nil)
		} else {
			response.Fail(c, "Query failed", err.Error())
		}
		return
	}

	// Delete rule
	if err := h.db.Delete(&rule).Error; err != nil {
		response.Fail(c, "Delete failed", err.Error())
		return
	}

	response.Success(c, "Delete successful", nil)
}

// ListAlerts List alerts
func (h *Handlers) ListAlerts(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	var alerts []models.Alert
	query := h.db.Where("user_id = ?", user.ID).Preload("Rule")

	// Optional: Filter by status
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// Optional: Filter by type
	if alertType := c.Query("alertType"); alertType != "" {
		query = query.Where("alert_type = ?", alertType)
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query.Model(&models.Alert{}).Count(&total)

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&alerts).Error; err != nil {
		response.Fail(c, "Query failed", err.Error())
		return
	}

	response.Success(c, "Query successful", gin.H{
		"list":     alerts,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetAlert Get alert details
func (h *Handlers) GetAlert(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "Parameter error", "Invalid alert ID")
		return
	}

	var alert models.Alert
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).Preload("Rule").First(&alert).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "Alert not found", nil)
		} else {
			response.Fail(c, "Query failed", err.Error())
		}
		return
	}

	// Load notification records
	var notifications []models.AlertNotification
	h.db.Where("alert_id = ?", alert.ID).Order("created_at DESC").Find(&notifications)
	alertData := gin.H{
		"alert":         alert,
		"notifications": notifications,
	}

	response.Success(c, "Query successful", alertData)
}

// ResolveAlert Resolve alert
func (h *Handlers) ResolveAlert(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "Parameter error", "Invalid alert ID")
		return
	}

	var alert models.Alert
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&alert).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "Alert not found", nil)
		} else {
			response.Fail(c, "Query failed", err.Error())
		}
		return
	}

	now := time.Now()
	alert.Status = models.AlertStatusResolved
	alert.ResolvedAt = &now
	alert.ResolvedBy = &user.ID

	if err := h.db.Save(&alert).Error; err != nil {
		response.Fail(c, "Update failed", err.Error())
		return
	}

	response.Success(c, "Alert resolved", alert)
}

// MuteAlert Mute alert
func (h *Handlers) MuteAlert(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "Parameter error", "Invalid alert ID")
		return
	}

	var alert models.Alert
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&alert).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, "Alert not found", nil)
		} else {
			response.Fail(c, "Query failed", err.Error())
		}
		return
	}

	alert.Status = models.AlertStatusMuted
	if err := h.db.Save(&alert).Error; err != nil {
		response.Fail(c, "Update failed", err.Error())
		return
	}

	response.Success(c, "Alert muted", alert)
}

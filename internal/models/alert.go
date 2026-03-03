package models

import (
	"encoding/json"
	"time"
)

// AlertType defines the type of alert
type AlertType string

const (
	AlertTypeSystemError   AlertType = "system_error"   // System error alert
	AlertTypeQuotaExceeded AlertType = "quota_exceeded" // Quota exceeded alert
	AlertTypeServiceError  AlertType = "service_error"  // Service error alert
	AlertTypeCustom        AlertType = "custom"         // Custom alert
)

// AlertSeverity defines the severity level of alert
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical" // Critical
	AlertSeverityHigh     AlertSeverity = "high"     // High
	AlertSeverityMedium   AlertSeverity = "medium"   // Medium
	AlertSeverityLow      AlertSeverity = "low"      // Low
)

// AlertStatus defines the status of alert
type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "active"   // Active
	AlertStatusResolved AlertStatus = "resolved" // Resolved
	AlertStatusMuted    AlertStatus = "muted"    // Muted
)

// NotificationChannel defines the notification channel type
type NotificationChannel string

const (
	NotificationChannelEmail    NotificationChannel = "email"    // Email
	NotificationChannelInternal NotificationChannel = "internal" // Internal notification
	NotificationChannelWebhook  NotificationChannel = "webhook"  // Webhook
	NotificationChannelSMS      NotificationChannel = "sms"      // SMS (reserved)
)

// AlertRule defines alert rule configuration
type AlertRule struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`

	UserID      uint   `json:"userId" gorm:"index"`                    // User ID, 0 means system-level rule
	Name        string `json:"name" gorm:"size:200"`                   // Rule name
	Description string `json:"description,omitempty" gorm:"type:text"` // Rule description

	// Alert type and conditions
	AlertType AlertType     `json:"alertType" gorm:"size:50;index"` // Alert type
	Severity  AlertSeverity `json:"severity" gorm:"size:20"`        // Severity level

	// Trigger conditions (JSON format)
	Conditions string `json:"conditions" gorm:"type:text"` // Trigger conditions in JSON format

	// Notification configuration
	Channels      string `json:"channels" gorm:"type:text"`                             // Notification channels in JSON array format: ["email", "internal", "webhook"]
	WebhookURL    string `json:"webhookUrl,omitempty" gorm:"size:500"`                  // Webhook URL
	WebhookMethod string `json:"webhookMethod,omitempty" gorm:"size:10;default:'POST'"` // Webhook request method

	// Status
	Enabled  bool `json:"enabled" gorm:"default:true"` // Whether enabled
	Cooldown int  `json:"cooldown" gorm:"default:300"` // Cooldown time in seconds to prevent duplicate alerts

	// Statistics
	TriggerCount  int64      `json:"triggerCount" gorm:"default:0"` // Trigger count
	LastTriggerAt *time.Time `json:"lastTriggerAt,omitempty"`       // Last trigger time
}

func (AlertRule) TableName() string {
	return "alert_rules"
}

// AlertCondition defines alert condition structure (for parsing Conditions JSON)
type AlertCondition struct {
	// Quota-related conditions
	QuotaType      string  `json:"quotaType,omitempty"`      // Quota type: storage, llm_tokens, api_calls, etc.
	QuotaThreshold float64 `json:"quotaThreshold,omitempty"` // Quota threshold (percentage, 0-100)

	// System error-related conditions
	ErrorCount  int `json:"errorCount,omitempty"`  // Error count threshold
	ErrorWindow int `json:"errorWindow,omitempty"` // Time window in seconds

	// Service error-related conditions
	ServiceName  string  `json:"serviceName,omitempty"`  // Service name
	FailureRate  float64 `json:"failureRate,omitempty"`  // Failure rate threshold (percentage)
	ResponseTime int     `json:"responseTime,omitempty"` // Response time threshold in milliseconds

	// Custom conditions
	CustomExpression string `json:"customExpression,omitempty"` // Custom expression (reserved)
}

// Alert defines alert record
type Alert struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`

	UserID uint      `json:"userId" gorm:"index"` // User ID
	RuleID uint      `json:"ruleId" gorm:"index"` // Rule ID
	Rule   AlertRule `json:"rule,omitempty" gorm:"foreignKey:RuleID"`

	// Alert information
	AlertType AlertType     `json:"alertType" gorm:"size:50;index"`
	Severity  AlertSeverity `json:"severity" gorm:"size:20"`
	Title     string        `json:"title" gorm:"size:200"`    // Alert title
	Message   string        `json:"message" gorm:"type:text"` // Alert message

	// Alert data (JSON format)
	Data string `json:"data,omitempty" gorm:"type:text"` // Alert-related data in JSON format

	// Status
	Status     AlertStatus `json:"status" gorm:"size:20;index;default:'active'"`
	ResolvedAt *time.Time  `json:"resolvedAt,omitempty"` // Resolution time
	ResolvedBy *uint       `json:"resolvedBy,omitempty"` // Resolver user ID

	// Notification status
	Notified   bool       `json:"notified" gorm:"default:false"` // Whether notified
	NotifiedAt *time.Time `json:"notifiedAt,omitempty"`          // Notification time
}

func (Alert) TableName() string {
	return "alerts"
}

// AlertNotification defines alert notification record
type AlertNotification struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`

	AlertID uint  `json:"alertId" gorm:"index"` // Alert ID
	Alert   Alert `json:"alert,omitempty" gorm:"foreignKey:AlertID"`

	Channel NotificationChannel `json:"channel" gorm:"size:20"`             // Notification channel
	Status  string              `json:"status" gorm:"size:20"`              // Notification status: success, failed
	Message string              `json:"message,omitempty" gorm:"type:text"` // Notification message or error message

	SentAt *time.Time `json:"sentAt,omitempty"` // Send time
}

func (AlertNotification) TableName() string {
	return "alert_notifications"
}

// ParseConditions parses condition JSON
func (r *AlertRule) ParseConditions() (*AlertCondition, error) {
	if r.Conditions == "" {
		return &AlertCondition{}, nil
	}
	var cond AlertCondition
	err := json.Unmarshal([]byte(r.Conditions), &cond)
	return &cond, err
}

// SetConditions sets condition JSON
func (r *AlertRule) SetConditions(cond *AlertCondition) error {
	data, err := json.Marshal(cond)
	if err != nil {
		return err
	}
	r.Conditions = string(data)
	return nil
}

// GetChannels gets notification channel list
func (r *AlertRule) GetChannels() []NotificationChannel {
	if r.Channels == "" {
		return []NotificationChannel{}
	}
	var channels []NotificationChannel
	json.Unmarshal([]byte(r.Channels), &channels)
	return channels
}

// SetChannels sets notification channel list
func (r *AlertRule) SetChannels(channels []NotificationChannel) error {
	data, err := json.Marshal(channels)
	if err != nil {
		return err
	}
	r.Channels = string(data)
	return nil
}

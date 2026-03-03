package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAlertDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&AlertRule{}, &Alert{}, &AlertNotification{})
	require.NoError(t, err)

	return db
}

func TestAlertRule_TableName(t *testing.T) {
	rule := AlertRule{}
	assert.Equal(t, "alert_rules", rule.TableName())
}

func TestAlert_TableName(t *testing.T) {
	alert := Alert{}
	assert.Equal(t, "alerts", alert.TableName())
}

func TestAlertNotification_TableName(t *testing.T) {
	notification := AlertNotification{}
	assert.Equal(t, "alert_notifications", notification.TableName())
}

func TestAlertRule_ParseConditions(t *testing.T) {
	rule := &AlertRule{}

	// Test empty conditions
	cond, err := rule.ParseConditions()
	assert.NoError(t, err)
	assert.NotNil(t, cond)
	assert.Equal(t, "", cond.QuotaType)

	// Test valid JSON conditions
	rule.Conditions = `{"quotaType":"storage","quotaThreshold":80.5,"errorCount":10}`
	cond, err = rule.ParseConditions()
	assert.NoError(t, err)
	assert.Equal(t, "storage", cond.QuotaType)
	assert.Equal(t, 80.5, cond.QuotaThreshold)
	assert.Equal(t, 10, cond.ErrorCount)

	// Test invalid JSON
	rule.Conditions = `{"invalid json`
	cond, err = rule.ParseConditions()
	assert.Error(t, err)
	assert.NotNil(t, cond) // Returns empty struct, not nil
}

func TestAlertRule_SetConditions(t *testing.T) {
	rule := &AlertRule{}

	cond := &AlertCondition{
		QuotaType:      "llm_tokens",
		QuotaThreshold: 90.0,
		ErrorCount:     5,
		ErrorWindow:    300,
	}

	err := rule.SetConditions(cond)
	assert.NoError(t, err)
	assert.NotEmpty(t, rule.Conditions)

	// Verify by parsing back
	parsedCond, err := rule.ParseConditions()
	assert.NoError(t, err)
	assert.Equal(t, "llm_tokens", parsedCond.QuotaType)
	assert.Equal(t, 90.0, parsedCond.QuotaThreshold)
	assert.Equal(t, 5, parsedCond.ErrorCount)
	assert.Equal(t, 300, parsedCond.ErrorWindow)
}

func TestAlertRule_GetChannels(t *testing.T) {
	rule := &AlertRule{}

	// Test empty channels
	channels := rule.GetChannels()
	assert.Empty(t, channels)

	// Test with channels
	rule.Channels = `["email","internal","webhook"]`
	channels = rule.GetChannels()
	assert.Len(t, channels, 3)
	assert.Contains(t, channels, NotificationChannelEmail)
	assert.Contains(t, channels, NotificationChannelInternal)
	assert.Contains(t, channels, NotificationChannelWebhook)

	// Test invalid JSON (should return empty slice)
	rule.Channels = `["invalid json`
	channels = rule.GetChannels()
	assert.Empty(t, channels)
}

func TestAlertRule_SetChannels(t *testing.T) {
	rule := &AlertRule{}

	channels := []NotificationChannel{
		NotificationChannelEmail,
		NotificationChannelInternal,
		NotificationChannelWebhook,
	}

	err := rule.SetChannels(channels)
	assert.NoError(t, err)
	assert.NotEmpty(t, rule.Channels)

	// Verify by parsing back
	parsedChannels := rule.GetChannels()
	assert.Len(t, parsedChannels, 3)
	assert.Contains(t, parsedChannels, NotificationChannelEmail)
	assert.Contains(t, parsedChannels, NotificationChannelInternal)
	assert.Contains(t, parsedChannels, NotificationChannelWebhook)
}

func TestAlertRule_CRUD(t *testing.T) {
	db := setupAlertDB(t)

	// Create alert rule
	rule := &AlertRule{
		UserID:      1,
		Name:        "Test Alert Rule",
		Description: "Test description",
		AlertType:   AlertTypeQuotaExceeded,
		Severity:    AlertSeverityHigh,
		Conditions:  `{"quotaType":"storage","quotaThreshold":85.0}`,
		Channels:    `["email","internal"]`,
		WebhookURL:  "https://example.com/webhook",
		Enabled:     true,
		Cooldown:    300,
	}

	err := db.Create(rule).Error
	assert.NoError(t, err)
	assert.NotZero(t, rule.ID)
	assert.NotZero(t, rule.CreatedAt)
	assert.NotZero(t, rule.UpdatedAt)

	// Read
	var retrieved AlertRule
	err = db.First(&retrieved, rule.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Test Alert Rule", retrieved.Name)
	assert.Equal(t, AlertTypeQuotaExceeded, retrieved.AlertType)
	assert.Equal(t, AlertSeverityHigh, retrieved.Severity)
	assert.True(t, retrieved.Enabled)

	// Update
	retrieved.Name = "Updated Alert Rule"
	retrieved.Severity = AlertSeverityCritical
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	// Verify update
	var updated AlertRule
	err = db.First(&updated, rule.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Updated Alert Rule", updated.Name)
	assert.Equal(t, AlertSeverityCritical, updated.Severity)

	// Delete
	err = db.Delete(&updated).Error
	assert.NoError(t, err)

	// Verify deletion
	var deleted AlertRule
	err = db.First(&deleted, rule.ID).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestAlert_CRUD(t *testing.T) {
	db := setupAlertDB(t)

	// Create alert rule first
	rule := &AlertRule{
		UserID:    1,
		Name:      "Test Rule",
		AlertType: AlertTypeSystemError,
		Severity:  AlertSeverityMedium,
		Enabled:   true,
	}
	err := db.Create(rule).Error
	require.NoError(t, err)

	// Create alert
	alert := &Alert{
		UserID:    1,
		RuleID:    rule.ID,
		AlertType: AlertTypeSystemError,
		Severity:  AlertSeverityMedium,
		Title:     "Test Alert",
		Message:   "This is a test alert message",
		Data:      `{"error":"test error","timestamp":"2023-01-01T10:00:00Z"}`,
		Status:    AlertStatusActive,
		Notified:  false,
	}

	err = db.Create(alert).Error
	assert.NoError(t, err)
	assert.NotZero(t, alert.ID)
	assert.NotZero(t, alert.CreatedAt)
	assert.NotZero(t, alert.UpdatedAt)

	// Read with association
	var retrieved Alert
	err = db.Preload("Rule").First(&retrieved, alert.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Test Alert", retrieved.Title)
	assert.Equal(t, AlertStatusActive, retrieved.Status)
	assert.Equal(t, "Test Rule", retrieved.Rule.Name)

	// Update status
	now := time.Now()
	retrieved.Status = AlertStatusResolved
	retrieved.ResolvedAt = &now
	resolvedBy := uint(2)
	retrieved.ResolvedBy = &resolvedBy
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	// Verify update
	var updated Alert
	err = db.First(&updated, alert.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, AlertStatusResolved, updated.Status)
	assert.NotNil(t, updated.ResolvedAt)
	assert.NotNil(t, updated.ResolvedBy)
	assert.Equal(t, uint(2), *updated.ResolvedBy)
}

func TestAlertNotification_CRUD(t *testing.T) {
	db := setupAlertDB(t)

	// Create alert first
	rule := &AlertRule{
		UserID:    1,
		Name:      "Test Rule",
		AlertType: AlertTypeServiceError,
		Severity:  AlertSeverityLow,
		Enabled:   true,
	}
	err := db.Create(rule).Error
	require.NoError(t, err)

	alert := &Alert{
		UserID:    1,
		RuleID:    rule.ID,
		AlertType: AlertTypeServiceError,
		Severity:  AlertSeverityLow,
		Title:     "Service Error Alert",
		Message:   "Service is experiencing issues",
		Status:    AlertStatusActive,
	}
	err = db.Create(alert).Error
	require.NoError(t, err)

	// Create notification
	notification := &AlertNotification{
		AlertID: alert.ID,
		Channel: NotificationChannelEmail,
		Status:  "success",
		Message: "Email sent successfully",
		SentAt:  &alert.CreatedAt,
	}

	err = db.Create(notification).Error
	assert.NoError(t, err)
	assert.NotZero(t, notification.ID)
	assert.NotZero(t, notification.CreatedAt)

	// Read with association
	var retrieved AlertNotification
	err = db.Preload("Alert").First(&retrieved, notification.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, NotificationChannelEmail, retrieved.Channel)
	assert.Equal(t, "success", retrieved.Status)
	assert.Equal(t, "Service Error Alert", retrieved.Alert.Title)
}

func TestAlertTypes_Constants(t *testing.T) {
	// Test AlertType constants
	assert.Equal(t, AlertType("system_error"), AlertTypeSystemError)
	assert.Equal(t, AlertType("quota_exceeded"), AlertTypeQuotaExceeded)
	assert.Equal(t, AlertType("service_error"), AlertTypeServiceError)
	assert.Equal(t, AlertType("custom"), AlertTypeCustom)

	// Test AlertSeverity constants
	assert.Equal(t, AlertSeverity("critical"), AlertSeverityCritical)
	assert.Equal(t, AlertSeverity("high"), AlertSeverityHigh)
	assert.Equal(t, AlertSeverity("medium"), AlertSeverityMedium)
	assert.Equal(t, AlertSeverity("low"), AlertSeverityLow)

	// Test AlertStatus constants
	assert.Equal(t, AlertStatus("active"), AlertStatusActive)
	assert.Equal(t, AlertStatus("resolved"), AlertStatusResolved)
	assert.Equal(t, AlertStatus("muted"), AlertStatusMuted)

	// Test NotificationChannel constants
	assert.Equal(t, NotificationChannel("email"), NotificationChannelEmail)
	assert.Equal(t, NotificationChannel("internal"), NotificationChannelInternal)
	assert.Equal(t, NotificationChannel("webhook"), NotificationChannelWebhook)
	assert.Equal(t, NotificationChannel("sms"), NotificationChannelSMS)
}

func TestAlertCondition_ComplexStructure(t *testing.T) {
	rule := &AlertRule{}

	// Test complex condition structure
	cond := &AlertCondition{
		QuotaType:        "api_calls",
		QuotaThreshold:   95.0,
		ErrorCount:       20,
		ErrorWindow:      600,
		ServiceName:      "llm-service",
		FailureRate:      15.5,
		ResponseTime:     5000,
		CustomExpression: "cpu_usage > 80 AND memory_usage > 90",
	}

	err := rule.SetConditions(cond)
	assert.NoError(t, err)

	parsedCond, err := rule.ParseConditions()
	assert.NoError(t, err)
	assert.Equal(t, "api_calls", parsedCond.QuotaType)
	assert.Equal(t, 95.0, parsedCond.QuotaThreshold)
	assert.Equal(t, 20, parsedCond.ErrorCount)
	assert.Equal(t, 600, parsedCond.ErrorWindow)
	assert.Equal(t, "llm-service", parsedCond.ServiceName)
	assert.Equal(t, 15.5, parsedCond.FailureRate)
	assert.Equal(t, 5000, parsedCond.ResponseTime)
	assert.Equal(t, "cpu_usage > 80 AND memory_usage > 90", parsedCond.CustomExpression)
}

func TestAlertRule_DefaultValues(t *testing.T) {
	db := setupAlertDB(t)

	rule := &AlertRule{
		UserID:    1,
		Name:      "Default Values Test",
		AlertType: AlertTypeCustom,
		Severity:  AlertSeverityMedium,
		// Don't set Enabled and Cooldown to test defaults
	}

	err := db.Create(rule).Error
	assert.NoError(t, err)

	var retrieved AlertRule
	err = db.First(&retrieved, rule.ID).Error
	assert.NoError(t, err)

	// Check default values
	assert.True(t, retrieved.Enabled)                // default:true
	assert.Equal(t, 300, retrieved.Cooldown)         // default:300
	assert.Equal(t, "POST", retrieved.WebhookMethod) // default:'POST'
}

func TestAlert_DefaultStatus(t *testing.T) {
	db := setupAlertDB(t)

	alert := &Alert{
		UserID:    1,
		RuleID:    1,
		AlertType: AlertTypeSystemError,
		Severity:  AlertSeverityHigh,
		Title:     "Default Status Test",
		Message:   "Test message",
		// Don't set Status to test default
	}

	err := db.Create(alert).Error
	assert.NoError(t, err)

	var retrieved Alert
	err = db.First(&retrieved, alert.ID).Error
	assert.NoError(t, err)

	assert.Equal(t, AlertStatusActive, retrieved.Status) // default:'active'
	assert.False(t, retrieved.Notified)                  // default:false
}

func TestAlertRule_TriggerStatistics(t *testing.T) {
	db := setupAlertDB(t)

	now := time.Now()
	rule := &AlertRule{
		UserID:        1,
		Name:          "Trigger Stats Test",
		AlertType:     AlertTypeQuotaExceeded,
		Severity:      AlertSeverityHigh,
		Enabled:       true,
		TriggerCount:  5,
		LastTriggerAt: &now,
	}

	err := db.Create(rule).Error
	assert.NoError(t, err)

	var retrieved AlertRule
	err = db.First(&retrieved, rule.ID).Error
	assert.NoError(t, err)

	assert.Equal(t, int64(5), retrieved.TriggerCount)
	assert.NotNil(t, retrieved.LastTriggerAt)
	assert.True(t, retrieved.LastTriggerAt.Equal(now) || retrieved.LastTriggerAt.After(now.Add(-time.Second)))
}

func TestAlertNotification_FailedStatus(t *testing.T) {
	db := setupAlertDB(t)

	// Create alert first
	alert := &Alert{
		UserID:    1,
		RuleID:    1,
		AlertType: AlertTypeSystemError,
		Severity:  AlertSeverityMedium,
		Title:     "Failed Notification Test",
		Message:   "Test message",
		Status:    AlertStatusActive,
	}
	err := db.Create(alert).Error
	require.NoError(t, err)

	// Create failed notification
	notification := &AlertNotification{
		AlertID: alert.ID,
		Channel: NotificationChannelWebhook,
		Status:  "failed",
		Message: "Webhook endpoint returned 500 error",
		SentAt:  nil, // No sent time for failed notification
	}

	err = db.Create(notification).Error
	assert.NoError(t, err)

	var retrieved AlertNotification
	err = db.First(&retrieved, notification.ID).Error
	assert.NoError(t, err)

	assert.Equal(t, "failed", retrieved.Status)
	assert.Contains(t, retrieved.Message, "500 error")
	assert.Nil(t, retrieved.SentAt)
}

// Benchmark tests
func BenchmarkAlertRule_ParseConditions(b *testing.B) {
	rule := &AlertRule{
		Conditions: `{"quotaType":"storage","quotaThreshold":80.5,"errorCount":10,"errorWindow":300}`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := rule.ParseConditions()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAlertRule_SetConditions(b *testing.B) {
	rule := &AlertRule{}
	cond := &AlertCondition{
		QuotaType:      "llm_tokens",
		QuotaThreshold: 90.0,
		ErrorCount:     5,
		ErrorWindow:    300,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := rule.SetConditions(cond)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestAlertRule_JSONSerialization(t *testing.T) {
	rule := &AlertRule{
		UserID:      1,
		Name:        "JSON Test",
		Description: "Test JSON serialization",
		AlertType:   AlertTypeQuotaExceeded,
		Severity:    AlertSeverityHigh,
		Enabled:     true,
		Cooldown:    300,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(rule)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "JSON Test")
	assert.Contains(t, string(jsonData), "quota_exceeded")

	// Test JSON unmarshaling
	var unmarshaled AlertRule
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, rule.Name, unmarshaled.Name)
	assert.Equal(t, rule.AlertType, unmarshaled.AlertType)
	assert.Equal(t, rule.Severity, unmarshaled.Severity)
}

func TestAlert_ComplexData(t *testing.T) {
	db := setupAlertDB(t)

	complexData := map[string]interface{}{
		"error_code":    500,
		"error_message": "Internal server error",
		"stack_trace":   []string{"line1", "line2", "line3"},
		"metadata": map[string]interface{}{
			"user_id":    123,
			"timestamp":  "2023-01-01T10:00:00Z",
			"request_id": "req-12345",
		},
	}

	dataJSON, err := json.Marshal(complexData)
	require.NoError(t, err)

	alert := &Alert{
		UserID:    1,
		RuleID:    1,
		AlertType: AlertTypeSystemError,
		Severity:  AlertSeverityCritical,
		Title:     "Complex Data Test",
		Message:   "Alert with complex data structure",
		Data:      string(dataJSON),
		Status:    AlertStatusActive,
	}

	err = db.Create(alert).Error
	assert.NoError(t, err)

	var retrieved Alert
	err = db.First(&retrieved, alert.ID).Error
	assert.NoError(t, err)

	// Parse the data back
	var parsedData map[string]interface{}
	err = json.Unmarshal([]byte(retrieved.Data), &parsedData)
	assert.NoError(t, err)
	assert.Equal(t, float64(500), parsedData["error_code"]) // JSON numbers are float64
	assert.Equal(t, "Internal server error", parsedData["error_message"])
}

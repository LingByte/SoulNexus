package models

import (
	"context"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	SIPCampaignStatusDraft       = "draft"
	SIPCampaignStatusRunning     = "running"
	SIPCampaignStatusPaused      = "paused"
	SIPCampaignStatusDone        = "done"
	SIPCampaignContactReady      = "ready"
	SIPCampaignContactDialing    = "dialing"
	SIPCampaignContactAnswered   = "answered"
	SIPCampaignContactFailed     = "failed"
	SIPCampaignContactRetrying   = "retrying"
	SIPCampaignContactExhausted  = "exhausted"
	SIPCampaignContactSuppressed = "suppressed"
)

// SIPCampaign stores one outbound campaign configuration.
type SIPCampaign struct {
	BaseModel

	Name string `json:"name" gorm:"size:128;index;not null"`

	Status          string         `json:"status" gorm:"size:24;index;not null;default:draft"`
	Scenario        string         `json:"scenario" gorm:"size:24;index;not null;default:campaign"`
	MediaProfile    string         `json:"mediaProfile" gorm:"size:24;index;not null;default:ai_voice"`
	ScriptID        string         `json:"scriptId" gorm:"size:128;index"`
	ScriptVersion   string         `json:"scriptVersion" gorm:"size:64"`
	ScriptSpec      datatypes.JSON `json:"scriptSpec" gorm:"type:json"`
	SystemPrompt    string         `json:"systemPrompt" gorm:"type:text"`
	OpeningMessage  string         `json:"openingMessage" gorm:"type:text"`
	ClosingMessage  string         `json:"closingMessage" gorm:"type:text"`
	RetrySchedule   string         `json:"retrySchedule" gorm:"size:256;default:5m,30m,2h"`
	MaxAttempts     int            `json:"maxAttempts" gorm:"default:3"`
	MaxCallDuration int            `json:"maxCallDuration" gorm:"default:180"` // seconds
	OutboundHost    string         `json:"outboundHost" gorm:"size:128"`
	OutboundPort    int            `json:"outboundPort" gorm:"default:5060"`
	SignalingAddr   string         `json:"signalingAddr" gorm:"size:128"`
	RequestURIFmt   string         `json:"requestUriFmt" gorm:"size:256"` // e.g. sip:%s@gw.local:5060

	TaskConcurrency   int            `json:"taskConcurrency" gorm:"default:5"`
	GlobalConcurrency int            `json:"globalConcurrency" gorm:"default:20"`
	AllowedTimeStart  string         `json:"allowedTimeStart" gorm:"size:8;default:09:00"`
	AllowedTimeEnd    string         `json:"allowedTimeEnd" gorm:"size:8;default:21:00"`
	Timezone          string         `json:"timezone" gorm:"size:64;default:Asia/Shanghai"`
	Metadata          datatypes.JSON `json:"metadata" gorm:"type:json"`

	StartedAt *time.Time `json:"startedAt" gorm:"index"`
	EndedAt   *time.Time `json:"endedAt" gorm:"index"`
}

func (SIPCampaign) TableName() string {
	return constants.SIP_CAMPAIGN_TABLE_NAME
}

// SIPCampaignContact is one callee in a campaign queue.
type SIPCampaignContact struct {
	BaseModel

	CampaignID uint   `json:"campaignId" gorm:"index;not null"`
	Phone      string `json:"phone" gorm:"size:64;index;not null"`
	RequestURI string `json:"requestUri" gorm:"size:256"`
	Display    string `json:"display" gorm:"size:128"`
	CallerUser string `json:"callerUser" gorm:"size:128"`
	CallerName string `json:"callerName" gorm:"size:128"`

	Status        string         `json:"status" gorm:"size:24;index;not null;default:ready"`
	Priority      int            `json:"priority" gorm:"index;default:0"`
	AttemptCount  int            `json:"attemptCount" gorm:"default:0"`
	MaxAttempts   int            `json:"maxAttempts" gorm:"default:3"`
	NextRunAt     *time.Time     `json:"nextRunAt" gorm:"index"`
	LastDialAt    *time.Time     `json:"lastDialAt" gorm:"index"`
	LastCallID    string         `json:"lastCallId" gorm:"size:128;index"`
	CorrelationID string         `json:"correlationId" gorm:"size:128;index"`
	FailureReason string         `json:"failureReason" gorm:"type:text"`
	Variables     datatypes.JSON `json:"variables" gorm:"type:json"`
}

func (SIPCampaignContact) TableName() string {
	return constants.SIP_CAMPAIGN_CONTACT_TABLE_NAME
}

// SIPCallAttempt keeps attempt-level dial lifecycle and retry decisions.
type SIPCallAttempt struct {
	BaseModel

	CampaignID uint `json:"campaignId" gorm:"index;not null"`
	ContactID  uint `json:"contactId" gorm:"index;not null"`

	AttemptNo     int        `json:"attemptNo" gorm:"index;not null"`
	CallID        string     `json:"callId" gorm:"size:128;index"`
	CorrelationID string     `json:"correlationId" gorm:"size:128;index"`
	State         string     `json:"state" gorm:"size:24;index;not null;default:created"` // created|dialing|answered|failed|retry_pending
	SIPStatusCode int        `json:"sipStatusCode" gorm:"index"`
	FailureReason string     `json:"failureReason" gorm:"type:text"`
	DialedAt      *time.Time `json:"dialedAt" gorm:"index"`
	AnsweredAt    *time.Time `json:"answeredAt" gorm:"index"`
	EndedAt       *time.Time `json:"endedAt" gorm:"index"`
	NextRetryAt   *time.Time `json:"nextRetryAt" gorm:"index"`
}

func (SIPCallAttempt) TableName() string {
	return constants.SIP_CALL_ATTEMPT_TABLE_NAME
}

// SIPScriptRun keeps per-call script step traces for audit.
type SIPScriptRun struct {
	BaseModel

	CampaignID    uint           `json:"campaignId" gorm:"index;not null"`
	ContactID     uint           `json:"contactId" gorm:"index"`
	AttemptID     uint           `json:"attemptId" gorm:"index"`
	CallID        string         `json:"callId" gorm:"size:128;index"`
	CorrelationID string         `json:"correlationId" gorm:"size:128;index"`
	ScriptID      string         `json:"scriptId" gorm:"size:128;index"`
	ScriptVersion string         `json:"scriptVersion" gorm:"size:64"`
	StepID        string         `json:"stepId" gorm:"size:128;index"`
	StepType      string         `json:"stepType" gorm:"size:32"`
	Result        string         `json:"result" gorm:"size:32"` // ok|timeout|retry|skipped|failed
	InputText     string         `json:"inputText" gorm:"type:text"`
	OutputText    string         `json:"outputText" gorm:"type:text"`
	// DurationMs is wall time from step entry to this audit row (ms); Started rows are ~0.
	DurationMs    int            `json:"durationMs" gorm:"column:duration_ms;default:0"`
	Variables     datatypes.JSON `json:"variables" gorm:"type:json"`
}

func (SIPScriptRun) TableName() string {
	return constants.SIP_SCRIPT_RUN_TABLE_NAME
}

// SIPCampaignEvent keeps campaign execution and operation logs for UI terminal.
type SIPCampaignEvent struct {
	BaseModel

	CampaignID    uint           `json:"campaignId" gorm:"index;not null"`
	ContactID     uint           `json:"contactId" gorm:"index"`
	AttemptID     uint           `json:"attemptId" gorm:"index"`
	CallID        string         `json:"callId" gorm:"size:128;index"`
	CorrelationID string         `json:"correlationId" gorm:"size:128;index"`
	Type          string         `json:"type" gorm:"size:32;index;not null"`  // campaign|dispatch|dial|retry|contact|script
	Level         string         `json:"level" gorm:"size:16;index;not null"` // info|warn|error
	Message       string         `json:"message" gorm:"type:text;not null"`
	Meta          datatypes.JSON `json:"meta" gorm:"type:json"`
}

func (SIPCampaignEvent) TableName() string {
	return constants.SIP_CAMPAIGN_EVENT_TABLE_NAME
}

// --- Queries & helpers (handlers / sipserver call these instead of duplicating SQL) ---

// ActiveSIPCampaigns is the non-deleted campaign scope.
func ActiveSIPCampaigns(db *gorm.DB) *gorm.DB {
	return db.Model(&SIPCampaign{}).Where("is_deleted = ?", SoftDeleteStatusActive)
}

// ListSIPCampaignsPage lists active campaigns with optional status / name filters.
func ListSIPCampaignsPage(db *gorm.DB, page, size int, status, nameContains string) ([]SIPCampaign, int64, error) {
	q := ActiveSIPCampaigns(db)
	if s := strings.TrimSpace(status); s != "" {
		q = q.Where("status = ?", s)
	}
	if name := strings.TrimSpace(nameContains); name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	var list []SIPCampaign
	if err := q.Order("id DESC").Offset(offset).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetActiveSIPCampaignByID returns one active campaign by primary key.
func GetActiveSIPCampaignByID(db *gorm.DB, id uint) (SIPCampaign, error) {
	var row SIPCampaign
	err := ActiveSIPCampaigns(db).Where("id = ?", id).First(&row).Error
	return row, err
}

// GetSIPCampaignByID loads a campaign row by id (ignores soft-delete; used by worker/service enqueue).
func GetSIPCampaignByID(ctx context.Context, db *gorm.DB, id uint) (SIPCampaign, error) {
	var row SIPCampaign
	err := db.WithContext(ctx).First(&row, id).Error
	return row, err
}

// SIPCampaignStatusUpdates builds the map for status transitions (started_at / ended_at).
func SIPCampaignStatusUpdates(status string, now time.Time) map[string]any {
	u := map[string]any{"status": status}
	if status == SIPCampaignStatusRunning {
		u["started_at"] = &now
	}
	if status == SIPCampaignStatusDone {
		u["ended_at"] = &now
	}
	return u
}

// UpdateSIPCampaignStatusByID updates status (no is_deleted filter; API / internal worker).
func UpdateSIPCampaignStatusByID(ctx context.Context, db *gorm.DB, id uint, status string) error {
	now := time.Now()
	return db.WithContext(ctx).Model(&SIPCampaign{}).Where("id = ?", id).Updates(SIPCampaignStatusUpdates(status, now)).Error
}

// UpdateActiveSIPCampaignStatus sets status for a non-deleted campaign; returns rows affected.
func UpdateActiveSIPCampaignStatus(db *gorm.DB, id uint, status, updateBy string) (int64, error) {
	now := time.Now()
	u := SIPCampaignStatusUpdates(status, now)
	if updateBy != "" {
		u["update_by"] = updateBy
	}
	res := db.Model(&SIPCampaign{}).Where("id = ? AND is_deleted = ?", id, SoftDeleteStatusActive).Updates(u)
	return res.RowsAffected, res.Error
}

// SoftDeleteSIPCampaignByID soft-deletes a campaign row by id (caller should enforce not-running).
func SoftDeleteSIPCampaignByID(db *gorm.DB, id uint, updateBy string) (int64, error) {
	u := map[string]any{"is_deleted": SoftDeleteStatusDeleted}
	if updateBy != "" {
		u["update_by"] = updateBy
	}
	res := db.Model(&SIPCampaign{}).Where("id = ?", id).Updates(u)
	return res.RowsAffected, res.Error
}

// ActiveSIPCampaignContacts scopes contacts for one campaign.
func ActiveSIPCampaignContacts(db *gorm.DB, campaignID uint) *gorm.DB {
	return db.Model(&SIPCampaignContact{}).Where("campaign_id = ? AND is_deleted = ?", campaignID, SoftDeleteStatusActive)
}

// ListSIPCampaignContactsPage lists active contacts for a campaign.
func ListSIPCampaignContactsPage(db *gorm.DB, campaignID uint, page, size int) ([]SIPCampaignContact, int64, error) {
	q := ActiveSIPCampaignContacts(db, campaignID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	var list []SIPCampaignContact
	if err := q.Order("id DESC").Offset(offset).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ResetSuppressedSIPCampaignContacts moves suppressed rows back to ready for a campaign.
func ResetSuppressedSIPCampaignContacts(db *gorm.DB, campaignID uint, now time.Time) (int64, error) {
	updates := map[string]any{
		"status":         SIPCampaignContactReady,
		"failure_reason": "",
		"next_run_at":    &now,
		"last_dial_at":   nil,
	}
	res := db.Model(&SIPCampaignContact{}).
		Where("campaign_id = ? AND status = ? AND is_deleted = ?", campaignID, SIPCampaignContactSuppressed, SoftDeleteStatusActive).
		Updates(updates)
	return res.RowsAffected, res.Error
}

// SIPCampaignContactBatchItem is one imported contact (VariablesJSON may be empty).
type SIPCampaignContactBatchItem struct {
	Phone, Display, CallerUser, CallerName, RequestURI string
	Priority                                           int
	VariablesJSON                                      datatypes.JSON
}

// BuildSIPCampaignContactsBatch builds rows for bulk Create (skips empty phone).
func BuildSIPCampaignContactsBatch(campaignID uint, maxAttempts int, items []SIPCampaignContactBatchItem, now time.Time) []SIPCampaignContact {
	rows := make([]SIPCampaignContact, 0, len(items))
	for _, it := range items {
		phone := strings.TrimSpace(it.Phone)
		if phone == "" {
			continue
		}
		ma := maxAttempts
		if ma <= 0 {
			ma = 3
		}
		rows = append(rows, SIPCampaignContact{
			CampaignID:  campaignID,
			Phone:       phone,
			Display:     strings.TrimSpace(it.Display),
			CallerUser:  strings.TrimSpace(it.CallerUser),
			CallerName:  strings.TrimSpace(it.CallerName),
			RequestURI:  strings.TrimSpace(it.RequestURI),
			Priority:    it.Priority,
			Status:      SIPCampaignContactReady,
			MaxAttempts: ma,
			NextRunAt:   &now,
			Variables:   it.VariablesJSON,
		})
	}
	return rows
}

// Campaign metrics (global aggregates for admin dashboard).
func CountAllSIPCallAttempts(db *gorm.DB) (int64, error) {
	var n int64
	err := db.Model(&SIPCallAttempt{}).Count(&n).Error
	return n, err
}

func CountSIPCampaignContactsWithStatus(db *gorm.DB, status string) (int64, error) {
	var n int64
	err := db.Model(&SIPCampaignContact{}).Where("status = ?", status).Count(&n).Error
	return n, err
}

func CountSIPCampaignContactsWithStatuses(db *gorm.DB, statuses []string) (int64, error) {
	var n int64
	err := db.Model(&SIPCampaignContact{}).Where("status IN ?", statuses).Count(&n).Error
	return n, err
}

// Log / audit reads
func ListSIPCampaignEventsDesc(db *gorm.DB, campaignID uint, limit int) ([]SIPCampaignEvent, error) {
	var list []SIPCampaignEvent
	err := db.Where("campaign_id = ?", campaignID).Order("id DESC").Limit(limit).Find(&list).Error
	return list, err
}

func ListSIPCallAttemptsDesc(db *gorm.DB, campaignID uint, limit int) ([]SIPCallAttempt, error) {
	var list []SIPCallAttempt
	err := db.Where("campaign_id = ?", campaignID).Order("id DESC").Limit(limit).Find(&list).Error
	return list, err
}

func ListSIPScriptRunsDesc(db *gorm.DB, campaignID uint, limit int) ([]SIPScriptRun, error) {
	var list []SIPScriptRun
	err := db.Where("campaign_id = ?", campaignID).Order("id DESC").Limit(limit).Find(&list).Error
	return list, err
}

// GetSIPCampaignContactPhone returns phone for a contact id (empty if not found).
func GetSIPCampaignContactPhone(db *gorm.DB, contactID uint) (string, error) {
	var c SIPCampaignContact
	err := db.Select("phone").Where("id = ?", contactID).First(&c).Error
	if err != nil {
		return "", err
	}
	return c.Phone, nil
}

// InsertSIPCampaignEvent persists one operator / worker log line.
func InsertSIPCampaignEvent(ctx context.Context, db *gorm.DB, e *SIPCampaignEvent) error {
	if db == nil || e == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return db.WithContext(ctx).Create(e).Error
}

// ListRunningSIPCampaigns returns campaigns with status=running (worker; includes soft-deleted filter off).
func ListRunningSIPCampaigns(ctx context.Context, db *gorm.DB) ([]SIPCampaign, error) {
	var list []SIPCampaign
	err := db.WithContext(ctx).Where("status = ?", SIPCampaignStatusRunning).Find(&list).Error
	return list, err
}

// ListCampaignContactsReadyToDial picks contacts eligible for this poll tick.
func ListCampaignContactsReadyToDial(ctx context.Context, db *gorm.DB, campaignID uint, limit int, now time.Time) ([]SIPCampaignContact, error) {
	if limit <= 0 {
		limit = 1
	}
	var list []SIPCampaignContact
	err := db.WithContext(ctx).
		Where("campaign_id = ? AND status IN ? AND (next_run_at IS NULL OR next_run_at <= ?)", campaignID,
			[]string{SIPCampaignContactReady, SIPCampaignContactRetrying}, now).
		Order("priority desc, id asc").
		Limit(limit).
		Find(&list).Error
	return list, err
}

// TryClaimSIPCampaignContactDialing CAS-updates ready/retrying → dialing.
func TryClaimSIPCampaignContactDialing(ctx context.Context, db *gorm.DB, contactID uint) bool {
	tx := db.WithContext(ctx).Model(&SIPCampaignContact{}).
		Where("id = ? AND status IN ?", contactID, []string{SIPCampaignContactReady, SIPCampaignContactRetrying}).
		Updates(map[string]any{"status": SIPCampaignContactDialing})
	return tx.Error == nil && tx.RowsAffected == 1
}

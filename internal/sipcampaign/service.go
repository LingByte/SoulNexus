package sipcampaign

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/sip/conversation"
	"github.com/LingByte/SoulNexus/pkg/sip/outbound"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup

	pollInterval      time.Duration
	globalConcurrency int
	dedupeWindow      time.Duration

	metrics CampaignMetrics
}

type CampaignMetrics struct {
	Invited    atomic.Int64
	Answered   atomic.Int64
	Failed     atomic.Int64
	Retrying   atomic.Int64
	Suppressed atomic.Int64
}

type CreateCampaignInput struct {
	Name              string `json:"name"`
	Scenario          string `json:"scenario"`
	MediaProfile      string `json:"media_profile"`
	ScriptID          string `json:"script_id"`
	ScriptVersion     string `json:"script_version"`
	ScriptSpec        string `json:"script_spec"`
	SystemPrompt      string `json:"system_prompt"`
	OpeningMessage    string `json:"opening_message"`
	ClosingMessage    string `json:"closing_message"`
	RetrySchedule     string `json:"retry_schedule"`
	MaxAttempts       int    `json:"max_attempts"`
	TaskConcurrency   int    `json:"task_concurrency"`
	GlobalConcurrency int    `json:"global_concurrency"`
	OutboundHost      string `json:"outbound_host"`
	OutboundPort      int    `json:"outbound_port"`
	SignalingAddr     string `json:"signaling_addr"`
	RequestURIFmt     string `json:"request_uri_fmt"`
}

type ContactInput struct {
	Phone      string         `json:"phone"`
	Display    string         `json:"display"`
	CallerUser string         `json:"caller_user"`
	CallerName string         `json:"caller_name"`
	RequestURI string         `json:"request_uri"`
	Priority   int            `json:"priority"`
	Variables  map[string]any `json:"variables"`
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		db:                db,
		pollInterval:      1500 * time.Millisecond,
		globalConcurrency: 20,
		dedupeWindow:      24 * time.Hour,
	}
}

func (s *Service) AutoMigrate() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.AutoMigrate(
		&models.SIPCampaign{},
		&models.SIPCampaignContact{},
		&models.SIPCallAttempt{},
		&models.SIPScriptRun{},
	)
}

func (s *Service) CreateCampaign(ctx context.Context, in CreateCampaignInput) (*models.SIPCampaign, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("campaign service unavailable")
	}
	var scriptSpec datatypes.JSON
	if raw := strings.TrimSpace(in.ScriptSpec); raw != "" {
		scriptSpec = datatypes.JSON([]byte(raw))
	}
	c := &models.SIPCampaign{
		Name:              strings.TrimSpace(in.Name),
		Status:            models.SIPCampaignStatusDraft,
		Scenario:          emptyOr(in.Scenario, string(outbound.ScenarioCampaign)),
		MediaProfile:      emptyOr(in.MediaProfile, string(outbound.MediaProfileAI)),
		ScriptID:          strings.TrimSpace(in.ScriptID),
		ScriptVersion:     strings.TrimSpace(in.ScriptVersion),
		ScriptSpec:        scriptSpec,
		SystemPrompt:      strings.TrimSpace(in.SystemPrompt),
		OpeningMessage:    strings.TrimSpace(in.OpeningMessage),
		ClosingMessage:    strings.TrimSpace(in.ClosingMessage),
		RetrySchedule:     emptyOr(in.RetrySchedule, "5m,30m,2h"),
		MaxAttempts:       maxInt(in.MaxAttempts, 3),
		TaskConcurrency:   maxInt(in.TaskConcurrency, 5),
		GlobalConcurrency: maxInt(in.GlobalConcurrency, 20),
		OutboundHost:      strings.TrimSpace(in.OutboundHost),
		OutboundPort:      maxInt(in.OutboundPort, 5060),
		SignalingAddr:     strings.TrimSpace(in.SignalingAddr),
		RequestURIFmt:     strings.TrimSpace(in.RequestURIFmt),
	}
	if c.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if err := s.db.WithContext(ctx).Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) EnqueueContacts(ctx context.Context, campaignID uint, contacts []ContactInput) (int, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("campaign service unavailable")
	}
	if campaignID == 0 {
		return 0, fmt.Errorf("campaign_id is required")
	}
	var campaign models.SIPCampaign
	if err := s.db.WithContext(ctx).First(&campaign, campaignID).Error; err != nil {
		return 0, err
	}
	now := time.Now()
	items := make([]models.SIPCampaignContact, 0, len(contacts))
	for _, c := range contacts {
		phone := strings.TrimSpace(c.Phone)
		if phone == "" {
			continue
		}
		b, _ := json.Marshal(c.Variables)
		items = append(items, models.SIPCampaignContact{
			CampaignID:  campaignID,
			Phone:       phone,
			Display:     strings.TrimSpace(c.Display),
			CallerUser:  strings.TrimSpace(c.CallerUser),
			CallerName:  strings.TrimSpace(c.CallerName),
			RequestURI:  strings.TrimSpace(c.RequestURI),
			Priority:    c.Priority,
			Status:      models.SIPCampaignContactReady,
			MaxAttempts: maxInt(campaign.MaxAttempts, 3),
			NextRunAt:   &now,
			Variables:   datatypes.JSON(b),
		})
	}
	if len(items) == 0 {
		return 0, nil
	}
	return len(items), s.db.WithContext(ctx).Create(&items).Error
}

func (s *Service) StartCampaign(ctx context.Context, campaignID uint) error {
	return s.setCampaignStatus(ctx, campaignID, models.SIPCampaignStatusRunning)
}

func (s *Service) PauseCampaign(ctx context.Context, campaignID uint) error {
	return s.setCampaignStatus(ctx, campaignID, models.SIPCampaignStatusPaused)
}

func (s *Service) ResumeCampaign(ctx context.Context, campaignID uint) error {
	return s.setCampaignStatus(ctx, campaignID, models.SIPCampaignStatusRunning)
}

func (s *Service) setCampaignStatus(ctx context.Context, campaignID uint, status string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("campaign service unavailable")
	}
	updates := map[string]any{"status": status}
	now := time.Now()
	if status == models.SIPCampaignStatusRunning {
		updates["started_at"] = &now
	}
	if status == models.SIPCampaignStatusDone {
		updates["ended_at"] = &now
	}
	return s.db.WithContext(ctx).Model(&models.SIPCampaign{}).Where("id = ?", campaignID).Updates(updates).Error
}

func (s *Service) HandleDialEvent(ctx context.Context, evt outbound.DialEvent) {
	if s == nil || s.db == nil {
		return
	}
	campaignID, contactID, attemptNo, ok := parseCorrelation(evt.CorrelationID)
	if !ok {
		return
	}
	switch evt.State {
	case outbound.DialEventInvited:
		s.metrics.Invited.Add(1)
	case outbound.DialEventEstablished:
		s.metrics.Answered.Add(1)
		now := time.Now()
		_ = s.db.WithContext(ctx).Model(&models.SIPCallAttempt{}).
			Where("campaign_id = ? AND contact_id = ? AND attempt_no = ?", campaignID, contactID, attemptNo).
			Updates(map[string]any{"state": "answered", "answered_at": &now, "call_id": evt.CallID, "sip_status_code": evt.StatusCode}).Error
		_ = s.db.WithContext(ctx).Model(&models.SIPCampaignContact{}).
			Where("id = ?", contactID).
			Updates(map[string]any{"status": models.SIPCampaignContactAnswered, "last_call_id": evt.CallID}).Error
	case outbound.DialEventFailed:
		s.metrics.Failed.Add(1)
		s.markAttemptFailed(ctx, campaignID, contactID, attemptNo, evt)
	}
}

func (s *Service) markAttemptFailed(ctx context.Context, campaignID, contactID uint, attemptNo int, evt outbound.DialEvent) {
	var contact models.SIPCampaignContact
	if err := s.db.WithContext(ctx).First(&contact, contactID).Error; err != nil {
		return
	}
	now := time.Now()
	retryAt := s.computeNextRetry(contact.AttemptCount)
	state := models.SIPCampaignContactFailed
	if retryAt != nil && contact.AttemptCount < contact.MaxAttempts {
		state = models.SIPCampaignContactRetrying
		s.metrics.Retrying.Add(1)
	}
	_ = s.db.WithContext(ctx).Model(&models.SIPCallAttempt{}).
		Where("campaign_id = ? AND contact_id = ? AND attempt_no = ?", campaignID, contactID, attemptNo).
		Updates(map[string]any{
			"state":          "failed",
			"sip_status_code": evt.StatusCode,
			"failure_reason": emptyOr(evt.Reason, "failed"),
			"ended_at":       &now,
			"next_retry_at":  retryAt,
		}).Error
	updates := map[string]any{
		"status":         state,
		"failure_reason": emptyOr(evt.Reason, "failed"),
	}
	if retryAt != nil {
		updates["next_run_at"] = retryAt
	}
	if state == models.SIPCampaignContactFailed && contact.AttemptCount >= contact.MaxAttempts {
		updates["status"] = models.SIPCampaignContactExhausted
	}
	_ = s.db.WithContext(ctx).Model(&models.SIPCampaignContact{}).Where("id = ?", contactID).Updates(updates).Error
}

func (s *Service) computeNextRetry(attemptCount int) *time.Time {
	delays := []time.Duration{5 * time.Minute, 30 * time.Minute, 2 * time.Hour}
	if attemptCount <= 0 || attemptCount > len(delays) {
		return nil
	}
	t := time.Now().Add(delays[attemptCount-1])
	return &t
}

func emptyOr(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func maxInt(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}

func parseCorrelation(v string) (campaignID uint, contactID uint, attemptNo int, ok bool) {
	// camp:<campaignID>:contact:<contactID>:attempt:<n>
	parts := strings.Split(strings.TrimSpace(v), ":")
	if len(parts) != 6 || parts[0] != "camp" || parts[2] != "contact" || parts[4] != "attempt" {
		return 0, 0, 0, false
	}
	var cID, ctID uint64
	var aNo int
	_, err := fmt.Sscanf(v, "camp:%d:contact:%d:attempt:%d", &cID, &ctID, &aNo)
	if err != nil || cID == 0 || ctID == 0 || aNo < 1 {
		return 0, 0, 0, false
	}
	return uint(cID), uint(ctID), aNo, true
}

// RecordScriptStep writes one script trace row.
func (s *Service) RecordScriptStep(ctx context.Context, evt outbound.ScriptRunEvent) error {
	if s == nil || s.db == nil {
		return nil
	}
	campaignID, contactID, _, _ := parseCorrelation(evt.CorrelationID)
	row := models.SIPScriptRun{
		CampaignID:    campaignID,
		ContactID:     contactID,
		CallID:        evt.CallID,
		CorrelationID: evt.CorrelationID,
		ScriptID:      evt.ScriptID,
		ScriptVersion: evt.ScriptVersion,
		StepID:        evt.StepID,
		StepType:      evt.StepType,
		Result:        evt.Result,
		InputText:     evt.InputText,
		OutputText:    evt.OutputText,
	}
	return s.db.WithContext(ctx).Create(&row).Error
}

// PrepareCallPrompt binds campaign script text to this call id.
func (s *Service) PrepareCallPrompt(callID, correlationID string) {
	if s == nil || s.db == nil {
		return
	}
	campaignID, _, _, ok := parseCorrelation(correlationID)
	if !ok {
		return
	}
	var c models.SIPCampaign
	if err := s.db.First(&c, campaignID).Error; err != nil {
		return
	}
	prompt := strings.TrimSpace(c.SystemPrompt)
	if prompt == "" {
		return
	}
	if c.OpeningMessage != "" {
		prompt += "\n开场必须先说：" + strings.TrimSpace(c.OpeningMessage)
	}
	if c.ClosingMessage != "" {
		prompt += "\n结束前必须说：" + strings.TrimSpace(c.ClosingMessage)
	}
	conversation.SetSIPCallSystemPrompt(callID, prompt)
}

// RunScriptIfConfigured executes hybrid script trace flow when media profile is "script".
func (s *Service) RunScriptIfConfigured(ctx context.Context, leg outbound.EstablishedLeg, scriptID string) {
	if s == nil || s.db == nil {
		return
	}
	campaignID, _, _, ok := parseCorrelation(leg.CorrelationID)
	if !ok {
		return
	}
	var c models.SIPCampaign
	if err := s.db.First(&c, campaignID).Error; err != nil {
		return
	}
	raw := strings.TrimSpace(string(c.ScriptSpec))
	if raw == "" {
		return
	}
	script, err := outbound.ParseHybridScript(raw)
	if err != nil {
		if logger.Lg != nil {
			logger.Lg.Warn("campaign script parse failed", zap.Uint("campaign_id", campaignID), zap.Error(err))
		}
		return
	}
	if scriptID != "" {
		script.ID = scriptID
	}
	runner := outbound.NewHybridScriptRunner(script, scriptRecorder{s: s})
	go func() {
		_ = runner.Run(ctx, leg)
	}()
}

type scriptRecorder struct {
	s *Service
}

func (r scriptRecorder) Record(ctx context.Context, event outbound.ScriptRunEvent) error {
	if r.s == nil {
		return nil
	}
	return r.s.RecordScriptStep(ctx, event)
}

func (s *Service) logInfo(msg string, fields ...zap.Field) {
	if logger.Lg != nil {
		logger.Lg.Info(msg, fields...)
	}
}

func (s *Service) SnapshotMetrics() map[string]int64 {
	if s == nil {
		return map[string]int64{}
	}
	return map[string]int64{
		"invited_total":    s.metrics.Invited.Load(),
		"answered_total":   s.metrics.Answered.Load(),
		"failed_total":     s.metrics.Failed.Load(),
		"retrying_total":   s.metrics.Retrying.Load(),
		"suppressed_total": s.metrics.Suppressed.Load(),
	}
}


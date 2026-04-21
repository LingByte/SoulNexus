package sipserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/sip/conversation"
	"github.com/LingByte/SoulNexus/pkg/sip/outbound"
	"github.com/LingByte/SoulNexus/pkg/sip/scriptlisten"
	"github.com/LingByte/SoulNexus/pkg/sip/task"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CampaignService struct {
	db                  *gorm.DB
	dialTargetResolver  func(ctx context.Context, phone string) (outbound.DialTarget, bool)
	dialer              Dialer
	mu                  sync.Mutex
	running             bool
	stopCh              chan struct{}
	wg                  sync.WaitGroup
	pollInterval        time.Duration
	globalConcurrency   int
	dedupeWindow        time.Duration
	metrics             CampaignMetrics
	dispatcher          *task.Scheduler[campaignDispatchTask, struct{}]
	dispatchMu          sync.Mutex
	dispatchMeta        map[string]campaignDispatchTask
	dispatchOutstanding map[uint]int
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
	RetrySchedule     string `json:"retry_schedule"`
	MaxAttempts       int    `json:"max_attempts"`
	TaskConcurrency   int    `json:"task_concurrency"`
	GlobalConcurrency int    `json:"global_concurrency"`
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

type campaignDispatchTask struct {
	Campaign models.SIPCampaign
	Contact  models.SIPCampaignContact
}

func NewCampaignService(db *gorm.DB) *CampaignService {
	return &CampaignService{
		db:                  db,
		pollInterval:        1500 * time.Millisecond,
		globalConcurrency:   20,
		dedupeWindow:        24 * time.Hour,
		dispatchMeta:        map[string]campaignDispatchTask{},
		dispatchOutstanding: map[uint]int{},
	}
}

// SetDialTargetResolver injects dynamic dial target lookup (e.g. SIP REGISTER bindings).
func (s *CampaignService) SetDialTargetResolver(fn func(ctx context.Context, phone string) (outbound.DialTarget, bool)) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.dialTargetResolver = fn
	s.mu.Unlock()
}

func (s *CampaignService) CreateCampaign(ctx context.Context, in CreateCampaignInput) (*models.SIPCampaign, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("campaign service unavailable")
	}
	var scriptSpec datatypes.JSON
	if raw := strings.TrimSpace(in.ScriptSpec); raw != "" {
		scriptSpec = datatypes.JSON(raw)
	}
	c := &models.SIPCampaign{
		Name:              strings.TrimSpace(in.Name),
		Status:            models.SIPCampaignStatusDraft,
		Scenario:          emptyOr(in.Scenario, string(outbound.ScenarioCampaign)),
		MediaProfile:      emptyOr(in.MediaProfile, string(outbound.MediaProfileAI)),
		ScriptID:          strings.TrimSpace(in.ScriptID),
		ScriptVersion:     strings.TrimSpace(in.ScriptVersion),
		ScriptSpec:        scriptSpec,
		RetrySchedule:     emptyOr(in.RetrySchedule, "5m,30m,2h"),
		MaxAttempts:       maxInt(in.MaxAttempts, 3),
		TaskConcurrency:   maxInt(in.TaskConcurrency, 5),
		GlobalConcurrency: maxInt(in.GlobalConcurrency, 20),
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

func (s *CampaignService) EnqueueContacts(ctx context.Context, campaignID uint, contacts []ContactInput) (int, error) {
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

func (s *CampaignService) StartCampaign(ctx context.Context, campaignID uint) error {
	return s.setCampaignStatus(ctx, campaignID, models.SIPCampaignStatusRunning)
}

func (s *CampaignService) PauseCampaign(ctx context.Context, campaignID uint) error {
	return s.setCampaignStatus(ctx, campaignID, models.SIPCampaignStatusPaused)
}

func (s *CampaignService) ResumeCampaign(ctx context.Context, campaignID uint) error {
	return s.setCampaignStatus(ctx, campaignID, models.SIPCampaignStatusRunning)
}

func (s *CampaignService) setCampaignStatus(ctx context.Context, campaignID uint, status string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("campaign service unavailable")
	}
	return models.UpdateSIPCampaignStatusByID(ctx, s.db, campaignID, status)
}

func (s *CampaignService) HandleDialEvent(ctx context.Context, evt outbound.DialEvent) {
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
		s.appendEvent(ctx, models.SIPCampaignEvent{
			CampaignID:    campaignID,
			ContactID:     contactID,
			CallID:        evt.CallID,
			CorrelationID: evt.CorrelationID,
			Type:          "dial",
			Level:         "info",
			Message:       "INVITE sent to target",
		})
	case outbound.DialEventProvisional:
		s.appendEvent(ctx, models.SIPCampaignEvent{
			CampaignID:    campaignID,
			ContactID:     contactID,
			CallID:        evt.CallID,
			CorrelationID: evt.CorrelationID,
			Type:          "dial",
			Level:         "info",
			Message:       fmt.Sprintf("provisional response: sip=%d", evt.StatusCode),
		})
	case outbound.DialEventEstablished:
		s.metrics.Answered.Add(1)
		now := time.Now()
		attemptUpdates := map[string]any{
			"state":       "answered",
			"answered_at": &now,
			"call_id":     evt.CallID,
		}
		if evt.StatusCode > 0 && s.hasAttemptSIPStatusCodeColumn() {
			attemptUpdates["sip_status_code"] = evt.StatusCode
		}
		s.updateAttemptRow(ctx, campaignID, contactID, attemptNo, attemptUpdates)
		_ = s.db.WithContext(ctx).Model(&models.SIPCampaignContact{}).
			Where("id = ?", contactID).
			Updates(map[string]any{"status": models.SIPCampaignContactAnswered, "last_call_id": evt.CallID}).Error
		s.appendEvent(ctx, models.SIPCampaignEvent{
			CampaignID:    campaignID,
			ContactID:     contactID,
			CallID:        evt.CallID,
			CorrelationID: evt.CorrelationID,
			Type:          "dial",
			Level:         "info",
			Message:       "call established",
		})
	case outbound.DialEventFailed:
		s.metrics.Failed.Add(1)
		s.markAttemptFailed(ctx, campaignID, contactID, attemptNo, evt)
		s.appendEvent(ctx, models.SIPCampaignEvent{
			CampaignID:    campaignID,
			ContactID:     contactID,
			CallID:        evt.CallID,
			CorrelationID: evt.CorrelationID,
			Type:          "dial",
			Level:         "error",
			Message:       fmt.Sprintf("dial failed: sip=%d reason=%s", evt.StatusCode, emptyOr(evt.Reason, "unknown")),
		})
	}
}

func (s *CampaignService) markAttemptFailed(ctx context.Context, campaignID, contactID uint, attemptNo int, evt outbound.DialEvent) {
	var contact models.SIPCampaignContact
	if err := s.db.WithContext(ctx).First(&contact, contactID).Error; err != nil {
		return
	}
	now := time.Now()
	retryAt := s.computeNextRetry(attemptNo)
	state := models.SIPCampaignContactFailed
	if retryAt != nil && attemptNo < contact.MaxAttempts {
		state = models.SIPCampaignContactRetrying
		s.metrics.Retrying.Add(1)
	}
	attemptUpdates := map[string]any{
		"state":          "failed",
		"failure_reason": emptyOr(evt.Reason, "failed"),
		"ended_at":       &now,
		"next_retry_at":  retryAt,
	}
	if evt.StatusCode > 0 && s.hasAttemptSIPStatusCodeColumn() {
		attemptUpdates["sip_status_code"] = evt.StatusCode
	}
	s.updateAttemptRow(ctx, campaignID, contactID, attemptNo, attemptUpdates)
	updates := map[string]any{
		"status":         state,
		"failure_reason": emptyOr(evt.Reason, "failed"),
	}
	if retryAt != nil {
		updates["next_run_at"] = retryAt
		s.appendEvent(ctx, models.SIPCampaignEvent{
			CampaignID:    campaignID,
			ContactID:     contactID,
			CorrelationID: evt.CorrelationID,
			Type:          "retry",
			Level:         "warn",
			Message:       "scheduled retry at " + retryAt.Format(time.RFC3339),
		})
	}
	if state == models.SIPCampaignContactFailed && attemptNo >= contact.MaxAttempts {
		updates["status"] = models.SIPCampaignContactExhausted
	}
	_ = s.db.WithContext(ctx).Model(&models.SIPCampaignContact{}).Where("id = ?", contactID).Updates(updates).Error
}

func (s *CampaignService) computeNextRetry(attemptCount int) *time.Time {
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
func (s *CampaignService) RecordScriptStep(ctx context.Context, evt outbound.ScriptRunEvent) error {
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
		DurationMs:    int(evt.DurationMS),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	msg := fmt.Sprintf("step=%s type=%s result=%s input=%s output=%s", evt.StepID, evt.StepType, evt.Result, nonEmptyOr(evt.InputText, "-"), nonEmptyOr(evt.OutputText, "-"))
	if evt.StepType == constants.SIPScriptStepListen && evt.Result == constants.SIPScriptRunStarted {
		msg = "listen wait (asr pending): " + msg
	}
	if evt.StepType == constants.SIPScriptStepListen && evt.Result == constants.SIPScriptRunMatched {
		msg = "listen got user text: " + msg
	}
	s.appendEvent(ctx, models.SIPCampaignEvent{
		CampaignID: campaignID,
		ContactID:  contactID,
		CallID:     strings.TrimSpace(evt.CallID),
		Type:       "script",
		Level:      "info",
		Message:    msg,
		Meta:       datatypes.JSON([]byte(`{}`)),
	})
	return nil
}

func (s *CampaignService) appendEvent(ctx context.Context, evt models.SIPCampaignEvent) {
	if s == nil || s.db == nil || evt.CampaignID == 0 {
		return
	}
	_ = models.InsertSIPCampaignEvent(ctx, s.db, &evt)
}

func (s *CampaignService) updateAttemptRow(ctx context.Context, campaignID, contactID uint, attemptNo int, updates map[string]any) {
	if s == nil || s.db == nil {
		return
	}
	err := s.db.WithContext(ctx).Model(&models.SIPCallAttempt{}).
		Where("campaign_id = ? AND contact_id = ? AND attempt_no = ?", campaignID, contactID, attemptNo).
		Updates(updates).Error
	if err == nil {
		return
	}
	// Compatibility fallback for old schemas that miss sip_status_code.
	if strings.Contains(strings.ToLower(err.Error()), "unknown column") && strings.Contains(strings.ToLower(err.Error()), "sip_status_code") {
		delete(updates, "sip_status_code")
		_ = s.db.WithContext(ctx).Model(&models.SIPCallAttempt{}).
			Where("campaign_id = ? AND contact_id = ? AND attempt_no = ?", campaignID, contactID, attemptNo).
			Updates(updates).Error
	}
}

func (s *CampaignService) hasAttemptSIPStatusCodeColumn() bool {
	if s == nil || s.db == nil {
		return false
	}
	return s.db.Migrator().HasColumn(&models.SIPCallAttempt{}, "sip_status_code")
}

// PrepareCallPrompt binds campaign script text to this call id.
func (s *CampaignService) PrepareCallPrompt(callID, correlationID string) {
	_ = callID
	_ = correlationID
}

// RunScriptIfConfigured executes hybrid script trace flow when media profile is "script".
func (s *CampaignService) RunScriptIfConfigured(ctx context.Context, leg outbound.EstablishedLeg, scriptID string) {
	releaseScriptMode := true
	defer func() {
		if releaseScriptMode {
			conversation.ClearSIPScriptMode(leg.CallID)
		}
	}()
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
	lastTurnIndex := 0
	lastTurnReply := ""
	runner := outbound.NewHybridScriptRunner(script, scriptRecorder{s: s}).WithHooks(outbound.RuntimeHooks{
		OnSay: func(runCtx context.Context, runLeg outbound.EstablishedLeg, prompt string) error {
			if runLeg.Session == nil {
				return fmt.Errorf("script say: session not ready")
			}
			return conversation.SpeakTextOnce(runCtx, runLeg.Session, prompt, logger.Lg)
		},
		OnListen: func(runCtx context.Context, runLeg outbound.EstablishedLeg, timeout time.Duration, notBefore time.Time, step outbound.HybridStep) (outbound.ListenResult, error) {
			res, err := s.waitNextTurn(runCtx, runLeg.CallID, lastTurnIndex, timeout, notBefore, outbound.DTMFDigitToNextMap(step))
			if err != nil {
				if logger.Lg != nil && runLeg.Session != nil && strings.Contains(strings.ToLower(err.Error()), "timeout") {
					if rtpSess := runLeg.Session.RTPSession(); rtpSess != nil {
						st := rtpSess.StatsSnapshot()
						logger.Lg.Warn("script listen timeout with rtp diagnostics",
							zap.String("call_id", runLeg.CallID),
							zap.String("correlation_id", runLeg.CorrelationID),
							zap.Duration("listen_timeout", timeout),
							zap.String("local_socket", st.LocalSocket),
							zap.String("remote_sdp", st.RemoteSDP),
							zap.String("remote_now", st.RemoteNow),
							zap.Uint64("tx_packets", st.TXPackets),
							zap.Uint64("tx_bytes", st.TXBytes),
							zap.Uint64("rx_packets", st.RXPackets),
							zap.Uint64("rx_bytes", st.RXBytes),
							zap.Int64("first_tx_ms_ago", st.FirstTXAgo),
							zap.Int64("first_rx_ms_ago", st.FirstRXAgo),
						)
					}
				}
				return outbound.ListenResult{}, err
			}
			lastTurnIndex = res.Index
			lastTurnReply = res.Turn.LLMText
			if strings.TrimSpace(res.DTMFDigit) != "" {
				return outbound.ListenResult{
					DTMFDigit: strings.TrimSpace(res.DTMFDigit),
				}, nil
			}
			return outbound.ListenResult{
				InputText: strings.TrimSpace(res.Turn.ASRText),
				ReplyText: strings.TrimSpace(res.Turn.LLMText),
			}, nil
		},
		OnLLMReply: func(_ context.Context, _ outbound.EstablishedLeg, _ string, _ string) (string, error) {
			if strings.TrimSpace(lastTurnReply) == "" {
				return "", fmt.Errorf("script llm reply unavailable")
			}
			return strings.TrimSpace(lastTurnReply), nil
		},
		IsEndIntent: func(input string, sc outbound.HybridScript) bool {
			in := strings.ToLower(strings.TrimSpace(input))
			if in == "" {
				return false
			}
			for _, it := range sc.EndIntents {
				v := strings.ToLower(strings.TrimSpace(it))
				if v != "" && strings.Contains(in, v) {
					return true
				}
			}
			return false
		},
	})
	releaseScriptMode = false
	go func() {
		defer conversation.ClearSIPScriptMode(leg.CallID)
		if err := runner.Run(ctx, leg); err != nil {
			if logger.Lg != nil {
				logger.Lg.Warn("campaign script run failed", zap.String("call_id", leg.CallID), zap.Error(err))
			}
			return
		}
		// Script ended normally: actively send BYE so call does not linger after farewell.
		time.Sleep(300 * time.Millisecond)
		conversation.RequestSIPHangup(leg.CallID)
		cID, ctID, _, ok := parseCorrelation(leg.CorrelationID)
		if ok {
			s.appendEvent(context.Background(), models.SIPCampaignEvent{
				CampaignID:    cID,
				ContactID:     ctID,
				CallID:        leg.CallID,
				CorrelationID: leg.CorrelationID,
				Type:          "script",
				Level:         "info",
				Message:       "script finished, hangup requested",
				Meta:          datatypes.JSON([]byte(`{}`)),
			})
		}
	}()
}

type turnFetchResult struct {
	Index     int
	Turn      models.SIPCallDialogTurn
	DTMFDigit string // non-empty when resolved via keypad (matched dtmf_transitions)
}

// scriptListenPollInterval is how often we poll DB for a new user turn during script listen (default 120ms).
func scriptListenPollInterval() time.Duration {
	ms := 120
	if s := strings.TrimSpace(utils.GetEnv(constants.EnvSIPScriptListenPollMS)); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 50 && n <= 500 {
			ms = n
		}
	}
	return time.Duration(ms) * time.Millisecond
}

func (s *CampaignService) waitNextTurn(ctx context.Context, callID string, afterIndex int, timeout time.Duration, notBefore time.Time, dtmfNext map[string]string) (turnFetchResult, error) {
	if s == nil || s.db == nil {
		return turnFetchResult{}, fmt.Errorf("script listen: db unavailable")
	}
	if strings.TrimSpace(callID) == "" {
		return turnFetchResult{}, fmt.Errorf("script listen: empty call id")
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(scriptListenPollInterval())
	defer ticker.Stop()
	wake, wakeCancel := scriptlisten.Subscribe(callID)
	defer wakeCancel()

	for {
		res, ok := s.fetchTurn(callID, afterIndex, notBefore)
		if ok {
			scriptlisten.ClearDTMF(callID)
			return res, nil
		}
		if len(dtmfNext) > 0 {
			if _, d, ok := scriptlisten.TryConsumeDTMF(callID, notBefore, dtmfNext); ok {
				return turnFetchResult{Index: afterIndex, DTMFDigit: d}, nil
			}
		}
		if time.Now().After(deadline) {
			return turnFetchResult{}, fmt.Errorf("script listen timeout")
		}
		select {
		case <-ctx.Done():
			return turnFetchResult{}, ctx.Err()
		case <-wake:
		case <-ticker.C:
		}
	}
}

func (s *CampaignService) fetchTurn(callID string, afterIndex int, notBefore time.Time) (turnFetchResult, bool) {
	if s == nil || s.db == nil {
		return turnFetchResult{}, false
	}
	row, err := models.SelectSIPCallTurnsByCallID(s.db, callID)
	if err != nil {
		return turnFetchResult{}, false
	}
	turns, err := models.UnmarshalSIPCallTurns(row.Turns)
	if err != nil || len(turns) == 0 {
		return turnFetchResult{}, false
	}
	if len(turns) <= afterIndex {
		return turnFetchResult{}, false
	}
	// Skip assistant-only rows (empty ASRText) so listen steps always consume user utterances.
	for i := afterIndex; i < len(turns); i++ {
		t := turns[i]
		if strings.TrimSpace(t.ASRText) == "" {
			continue
		}
		if !notBefore.IsZero() && !t.At.IsZero() && t.At.Before(notBefore) {
			continue
		}
		return turnFetchResult{
			Index: i + 1,
			Turn:  t,
		}, true
	}
	return turnFetchResult{}, false
}

type scriptRecorder struct {
	s *CampaignService
}

func (r scriptRecorder) Record(ctx context.Context, event outbound.ScriptRunEvent) error {
	if r.s == nil {
		return nil
	}
	return r.s.RecordScriptStep(ctx, event)
}

func nonEmptyOr(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func (s *CampaignService) tryAcquireCampaignSlot(campaignID uint, limit int) bool {
	if s == nil {
		return false
	}
	if limit <= 0 {
		limit = 1
	}
	s.dispatchMu.Lock()
	defer s.dispatchMu.Unlock()
	cur := s.dispatchOutstanding[campaignID]
	if cur >= limit {
		return false
	}
	s.dispatchOutstanding[campaignID] = cur + 1
	return true
}

func (s *CampaignService) releaseCampaignSlot(campaignID uint) {
	if s == nil {
		return
	}
	s.dispatchMu.Lock()
	defer s.dispatchMu.Unlock()
	cur := s.dispatchOutstanding[campaignID]
	if cur <= 1 {
		delete(s.dispatchOutstanding, campaignID)
		return
	}
	s.dispatchOutstanding[campaignID] = cur - 1
}

// CancelCampaignQueuedTasks removes pending (not running) dispatch tasks for one campaign.
func (s *CampaignService) CancelCampaignQueuedTasks(ctx context.Context, campaignID uint) (int, error) {
	if s == nil || campaignID == 0 || s.dispatcher == nil {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	pending := s.dispatcher.PendingSnapshot()
	canceledContactIDs := make([]uint, 0)
	canceled := 0
	for _, p := range pending {
		s.dispatchMu.Lock()
		meta, ok := s.dispatchMeta[p.TaskID]
		s.dispatchMu.Unlock()
		if !ok || meta.Campaign.ID != campaignID {
			continue
		}
		if s.dispatcher.CancelTaskByID(p.TaskID) {
			s.dispatchMu.Lock()
			delete(s.dispatchMeta, p.TaskID)
			s.dispatchMu.Unlock()
			s.releaseCampaignSlot(campaignID)
			canceled++
			if meta.Contact.ID > 0 {
				canceledContactIDs = append(canceledContactIDs, meta.Contact.ID)
			}
		}
	}
	if len(canceledContactIDs) > 0 && s.db != nil {
		now := time.Now()
		_ = s.db.WithContext(ctx).Model(&models.SIPCampaignContact{}).
			Where("campaign_id = ? AND id IN ? AND status = ?", campaignID, canceledContactIDs, models.SIPCampaignContactDialing).
			Updates(map[string]any{
				"status":         models.SIPCampaignContactReady,
				"failure_reason": "",
				"next_run_at":    &now,
			}).Error
	}
	if canceled > 0 {
		s.appendEvent(ctx, models.SIPCampaignEvent{
			CampaignID: campaignID,
			Type:       "dispatch",
			Level:      "warn",
			Message:    fmt.Sprintf("cancel queued tasks on campaign status change: %d", canceled),
		})
	}
	return canceled, nil
}

func (s *CampaignService) SnapshotMetrics() map[string]any {
	if s == nil {
		return map[string]any{}
	}
	out := map[string]any{
		"invited_total":    s.metrics.Invited.Load(),
		"answered_total":   s.metrics.Answered.Load(),
		"failed_total":     s.metrics.Failed.Load(),
		"retrying_total":   s.metrics.Retrying.Load(),
		"suppressed_total": s.metrics.Suppressed.Load(),
	}
	if s.dispatcher == nil {
		return out
	}
	st := s.dispatcher.Stats()
	out["task_queued"] = st.Queued
	out["task_channel_len"] = st.ChannelLen
	out["task_running"] = st.Running
	out["task_unfinished"] = st.Unfinished
	pending := s.dispatcher.PendingSnapshot()
	s.dispatchMu.Lock()
	preview := make([]map[string]any, 0, len(pending))
	perCampaignQueued := map[uint]int{}
	for i, p := range pending {
		meta, ok := s.dispatchMeta[p.TaskID]
		if !ok {
			continue
		}
		perCampaignQueued[meta.Campaign.ID]++
		preview = append(preview, map[string]any{
			"queue_index": i,
			"task_id":     p.TaskID,
			"priority":    p.Priority,
			"campaign_id": meta.Campaign.ID,
			"contact_id":  meta.Contact.ID,
			"phone":       strings.TrimSpace(meta.Contact.Phone),
		})
		if len(preview) >= 50 {
			break
		}
	}
	perCampaignRunning := map[uint]int{}
	for cid, n := range s.dispatchOutstanding {
		if n > 0 {
			perCampaignRunning[cid] = n
		}
	}
	s.dispatchMu.Unlock()
	out["pending_preview"] = preview
	out["per_campaign_queued"] = perCampaignQueued
	out["per_campaign_running"] = perCampaignRunning
	return out
}

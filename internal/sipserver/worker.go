package sipserver

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/sip/conversation"
	"github.com/LingByte/SoulNexus/pkg/sip/outbound"
	"github.com/LingByte/SoulNexus/pkg/sip/task"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
)

type Dialer interface {
	Dial(ctx context.Context, req outbound.DialRequest) (callID string, err error)
}

func (s *CampaignService) StartWorker(dialer Dialer) {
	if s == nil || s.db == nil || dialer == nil {
		return
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.dialer = dialer
	s.dispatcher = task.NewScheduler[campaignDispatchTask, struct{}](s.globalConcurrency, logger.Lg)
	s.dispatchMeta = map[string]campaignDispatchTask{}
	s.dispatchOutstanding = map[uint]int{}
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.tick()
			}
		}
	}()
}

func (s *CampaignService) StopWorker() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	close(s.stopCh)
	s.running = false
	dispatcher := s.dispatcher
	s.dispatcher = nil
	s.dispatchMeta = map[string]campaignDispatchTask{}
	s.dispatchOutstanding = map[uint]int{}
	s.mu.Unlock()
	if dispatcher != nil {
		_ = dispatcher.Stop()
	}
	s.wg.Wait()
}

func (s *CampaignService) tick() {
	ctx := context.Background()
	now := time.Now()
	campaigns, err := models.ListRunningSIPCampaigns(ctx, s.db)
	if err != nil {
		return
	}
	for _, c := range campaigns {
		limit := c.TaskConcurrency
		if limit <= 0 {
			limit = 1
		}
		contacts, err := models.ListCampaignContactsReadyToDial(ctx, s.db, c.ID, limit, now)
		if err != nil {
			continue
		}
		for _, contact := range contacts {
			if !s.tryClaim(ctx, contact.ID) {
				continue
			}
			_ = s.enqueueDispatchTask(c, contact)
		}
	}
}

func (s *CampaignService) tryClaim(ctx context.Context, contactID uint) bool {
	return models.TryClaimSIPCampaignContactDialing(ctx, s.db, contactID)
}

func (s *CampaignService) enqueueDispatchTask(campaign models.SIPCampaign, contact models.SIPCampaignContact) bool {
	if s == nil || s.dispatcher == nil {
		return false
	}
	campaignLimit := campaign.TaskConcurrency
	if campaignLimit <= 0 {
		campaignLimit = 1
	}
	if !s.tryAcquireCampaignSlot(campaign.ID, campaignLimit) {
		now := time.Now()
		_ = s.db.WithContext(context.Background()).Model(&models.SIPCampaignContact{}).
			Where("id = ? AND status = ?", contact.ID, models.SIPCampaignContactDialing).
			Updates(map[string]any{
				"status":         models.SIPCampaignContactReady,
				"failure_reason": "",
				"next_run_at":    &now,
			}).Error
		s.appendEvent(context.Background(), models.SIPCampaignEvent{
			CampaignID: campaign.ID,
			ContactID:  contact.ID,
			Type:       "dispatch",
			Level:      "info",
			Message:    fmt.Sprintf("skip enqueue: campaign concurrency full (%d)", campaignLimit),
		})
		return false
	}
	priority := contact.Priority
	taskBody := campaignDispatchTask{Campaign: campaign, Contact: contact}
	t := s.dispatcher.SubmitTask(context.Background(), priority, taskBody, func(ctx context.Context, p campaignDispatchTask) (struct{}, error) {
		s.processContact(ctx, s.dialer, p.Campaign, p.Contact)
		return struct{}{}, nil
	})
	s.dispatchMu.Lock()
	s.dispatchMeta[t.ID] = taskBody
	s.dispatchMu.Unlock()
	pos := s.dispatcher.GetTaskPosition(t.ID)
	s.appendEvent(context.Background(), models.SIPCampaignEvent{
		CampaignID: campaign.ID,
		ContactID:  contact.ID,
		Type:       "dispatch",
		Level:      "info",
		Message:    fmt.Sprintf("task queued id=%s priority=%d queue_pos=%d", t.ID, priority, pos),
	})
	go func(taskID string, cID, ctID uint) {
		err := <-t.Err
		s.dispatchMu.Lock()
		delete(s.dispatchMeta, taskID)
		s.dispatchMu.Unlock()
		s.releaseCampaignSlot(cID)
		level := "info"
		msg := fmt.Sprintf("task done id=%s", taskID)
		if err != nil {
			level = "warn"
			msg = fmt.Sprintf("task done id=%s err=%v", taskID, err)
		}
		s.appendEvent(context.Background(), models.SIPCampaignEvent{
			CampaignID: cID,
			ContactID:  ctID,
			Type:       "dispatch",
			Level:      level,
			Message:    msg,
		})
	}(t.ID, campaign.ID, contact.ID)
	return true
}

func (s *CampaignService) processContact(ctx context.Context, dialer Dialer, campaign models.SIPCampaign, contact models.SIPCampaignContact) {
	if s.isDuplicateWithinWindow(ctx, contact.ID, contact.Phone, campaign.ID) {
		s.metrics.Suppressed.Add(1)
		_ = s.db.WithContext(ctx).Model(&models.SIPCampaignContact{}).
			Where("id = ?", contact.ID).
			Updates(map[string]any{"status": models.SIPCampaignContactSuppressed, "failure_reason": "dedupe_24h"}).Error
		s.appendEvent(ctx, models.SIPCampaignEvent{
			CampaignID: campaign.ID,
			ContactID:  contact.ID,
			Type:       "dispatch",
			Level:      "warn",
			Message:    "suppressed by dedupe window",
		})
		return
	}
	attemptNo := contact.AttemptCount + 1
	correlationID := fmt.Sprintf("camp:%d:contact:%d:attempt:%d", campaign.ID, contact.ID, attemptNo)
	now := time.Now()
	attempt := models.SIPCallAttempt{
		CampaignID:    campaign.ID,
		ContactID:     contact.ID,
		AttemptNo:     attemptNo,
		State:         "dialing",
		CorrelationID: correlationID,
		DialedAt:      &now,
	}
	if err := s.db.WithContext(ctx).Create(&attempt).Error; err != nil {
		// Release contact from "dialing" state to avoid stuck records.
		_ = s.db.WithContext(ctx).Model(&models.SIPCampaignContact{}).
			Where("id = ?", contact.ID).
			Updates(map[string]any{
				"status":         models.SIPCampaignContactReady,
				"failure_reason": "attempt_create_failed",
			}).Error
		s.appendEvent(ctx, models.SIPCampaignEvent{
			CampaignID: campaign.ID,
			ContactID:  contact.ID,
			Type:       "dispatch",
			Level:      "error",
			Message:    "failed to create attempt row: " + err.Error(),
		})
		return
	}
	_ = s.db.WithContext(ctx).Model(&models.SIPCampaignContact{}).Where("id = ?", contact.ID).Updates(map[string]any{
		"attempt_count": attemptNo,
		"last_dial_at":  &now,
	}).Error
	target, err := buildDialTarget(campaign, contact)
	if err != nil {
		_ = s.db.WithContext(ctx).Model(&models.SIPCampaignContact{}).Where("id = ?", contact.ID).
			Updates(map[string]any{"status": models.SIPCampaignContactFailed, "failure_reason": err.Error()}).Error
		s.updateAttemptRow(ctx, campaign.ID, contact.ID, attemptNo, map[string]any{
			"state":          "failed",
			"failure_reason": "build_target_failed:" + err.Error(),
			"ended_at":       &now,
		})
		s.appendEvent(ctx, models.SIPCampaignEvent{
			CampaignID:    campaign.ID,
			ContactID:     contact.ID,
			AttemptID:     attempt.ID,
			CorrelationID: correlationID,
			Type:          "dispatch",
			Level:         "error",
			Message:       "failed to build dial target: " + err.Error(),
		})
		return
	}
	targetSource := "campaign_static"
	targetResolveNote := "build_dial_target"
	resolveFromRegister := shouldResolveFromRegister(contact.Phone)
	// Prefer latest REGISTER binding for extension-like phone values (e.g. "tide"),
	// so campaign dialing follows current Contact IP:port instead of stale static signaling_addr.
	// For extension calls, do NOT fallback to static signaling when no registration is found.
	if resolveFromRegister {
		if rt, ok := s.resolveRegisteredDialTarget(ctx, contact.Phone); ok {
			target = rt
			targetSource = "register_resolved"
			targetResolveNote = "resolver_or_db_hit"
			if logger.Lg != nil {
				logger.Lg.Info("campaign dial target resolved from register",
					zap.Uint("campaign_id", campaign.ID),
					zap.Uint("contact_id", contact.ID),
					zap.String("phone", strings.TrimSpace(contact.Phone)),
					zap.String("request_uri", target.RequestURI),
					zap.String("signaling", target.SignalingAddr),
				)
			}
			s.appendEvent(ctx, models.SIPCampaignEvent{
				CampaignID:    campaign.ID,
				ContactID:     contact.ID,
				AttemptID:     attempt.ID,
				CorrelationID: correlationID,
				Type:          "dispatch",
				Level:         "info",
				Message:       fmt.Sprintf("dial target resolved from register uri=%s signaling=%s", target.RequestURI, target.SignalingAddr),
			})
		} else {
			targetSource = "register_miss"
			targetResolveNote = "register_missing_or_stale"
			if logger.Lg != nil {
				logger.Lg.Warn("campaign dial target register miss, fail fast for extension",
					zap.Uint("campaign_id", campaign.ID),
					zap.Uint("contact_id", contact.ID),
					zap.String("phone", strings.TrimSpace(contact.Phone)),
				)
			}
			s.appendEvent(ctx, models.SIPCampaignEvent{
				CampaignID:    campaign.ID,
				ContactID:     contact.ID,
				AttemptID:     attempt.ID,
				CorrelationID: correlationID,
				Type:          "dispatch",
				Level:         "warn",
				Message:       "register target not found/stale; extension call aborted (no static fallback)",
			})
			s.HandleDialEvent(context.Background(), outbound.DialEvent{
				CallID:        "",
				CorrelationID: correlationID,
				Scenario:      outbound.Scenario(strings.TrimSpace(campaign.Scenario)),
				MediaProfile:  outbound.MediaProfile(strings.TrimSpace(campaign.MediaProfile)),
				State:         outbound.DialEventFailed,
				StatusCode:    480,
				Reason:        "register_missing_or_stale",
				At:            time.Now(),
			})
			return
		}
	}
	req := outbound.DialRequest{
		Scenario:          outbound.Scenario(strings.TrimSpace(campaign.Scenario)),
		Target:            target,
		ScriptID:          strings.TrimSpace(campaign.ScriptID),
		CorrelationID:     correlationID,
		MediaProfile:      outbound.MediaProfile(strings.TrimSpace(campaign.MediaProfile)),
		CallerUser:        strings.TrimSpace(contact.CallerUser),
		CallerDisplayName: strings.TrimSpace(contact.CallerName),
	}
	callID, err := dialer.Dial(ctx, req)
	if err != nil {
		if logger.Lg != nil {
			logger.Lg.Error("campaign dial failed before invite",
				zap.Uint("campaign_id", campaign.ID),
				zap.Uint("contact_id", contact.ID),
				zap.String("correlation_id", correlationID),
				zap.String("request_uri", target.RequestURI),
				zap.String("signaling", target.SignalingAddr),
				zap.String("target_source", targetSource),
				zap.String("target_note", targetResolveNote),
				zap.Error(err),
			)
		}
		evt := outbound.DialEvent{
			CallID:        "",
			CorrelationID: correlationID,
			Scenario:      req.Scenario,
			MediaProfile:  req.MediaProfile,
			State:         outbound.DialEventFailed,
			StatusCode:    0,
			Reason:        "dial_error:" + err.Error(),
			At:            time.Now(),
		}
		s.HandleDialEvent(context.Background(), evt)
		return
	}
	if req.MediaProfile == outbound.MediaProfileScript {
		conversation.MarkSIPScriptMode(callID)
	}
	s.appendEvent(ctx, models.SIPCampaignEvent{
		CampaignID:    campaign.ID,
		ContactID:     contact.ID,
		AttemptID:     attempt.ID,
		CallID:        callID,
		CorrelationID: correlationID,
		Type:          "dispatch",
		Level:         "info",
		Message:       fmt.Sprintf("dial request dispatched uri=%s signaling=%s source=%s note=%s", target.RequestURI, target.SignalingAddr, targetSource, targetResolveNote),
	})
	_ = s.db.WithContext(ctx).Model(&models.SIPCampaignContact{}).Where("id = ?", contact.ID).Updates(map[string]any{
		"correlation_id": correlationID,
		"last_call_id":   callID,
	}).Error
	go s.watchDialAttemptTimeout(campaign, contact.ID, attempt.ID, attemptNo, callID, correlationID)
	s.PrepareCallPrompt(callID, correlationID)
	if logger.Lg != nil {
		logger.Lg.Info("campaign dial dispatched",
			zap.Uint("campaign_id", campaign.ID),
			zap.Uint("contact_id", contact.ID),
			zap.String("call_id", callID),
			zap.String("correlation_id", correlationID),
			zap.String("request_uri", target.RequestURI),
			zap.String("signaling", target.SignalingAddr),
			zap.String("target_source", targetSource),
			zap.String("target_note", targetResolveNote),
		)
	}
}

func (s *CampaignService) resolveRegisteredDialTarget(ctx context.Context, phone string) (outbound.DialTarget, bool) {
	if s == nil {
		return outbound.DialTarget{}, false
	}
	s.mu.Lock()
	resolver := s.dialTargetResolver
	s.mu.Unlock()
	if resolver == nil {
		return outbound.DialTarget{}, false
	}
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return outbound.DialTarget{}, false
	}
	// Keep campaign behavior for PSTN numbers; resolver is for SIP extension/AOR mapping.
	if !shouldResolveFromRegister(phone) {
		return outbound.DialTarget{}, false
	}
	username := normalizeDialUsername(phone)
	if dt, ok := resolver(ctx, username); ok {
		return dt, true
	}
	// Fallback: direct DB lookup in case injected resolver misses due to domain filtering/runtime mismatch.
	if s.db == nil {
		return outbound.DialTarget{}, false
	}
	row, err := models.FindLatestOnlineSIPUserByUsername(ctx, s.db, username)
	if err != nil || row.RemoteIP == "" || row.RemotePort <= 0 {
		return outbound.DialTarget{}, false
	}
	if !isSIPRegisterFresh(row.LastSeenAt) {
		return outbound.DialTarget{}, false
	}
	domain := effectiveDialDomain(row.Domain, row.RemoteIP)
	reqURI := fmt.Sprintf("sip:%s@%s:5060", row.Username, domain)
	sig := row.RemoteIP + ":" + strconv.Itoa(row.RemotePort)
	return outbound.DialTarget{RequestURI: reqURI, SignalingAddr: sig}, true
}

func shouldResolveFromRegister(phone string) bool {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return false
	}
	if strings.HasPrefix(phone, "+") || isDigits(phone) {
		return false
	}
	return true
}

func normalizeDialUsername(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return phone
	}
	low := strings.ToLower(phone)
	if strings.HasPrefix(low, "sip:") {
		if u, err := url.Parse(phone); err == nil {
			if u.User != nil && strings.TrimSpace(u.User.Username()) != "" {
				return strings.TrimSpace(u.User.Username())
			}
			// fallback for unusual sip URI parsing cases
			phone = strings.TrimPrefix(phone, "sip:")
		}
	}
	// handle "user@domain" form
	if at := strings.Index(phone, "@"); at > 0 {
		phone = phone[:at]
	}
	// strip optional display quote leftovers
	phone = strings.Trim(phone, "\"' <>")
	return strings.TrimSpace(phone)
}

func isDigits(v string) bool {
	if v == "" {
		return false
	}
	for _, r := range v {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (s *CampaignService) watchDialAttemptTimeout(campaign models.SIPCampaign, contactID, attemptID uint, attemptNo int, callID, correlationID string) {
	const timeout = 45 * time.Second
	<-time.After(timeout)
	ctx := context.Background()
	var attempt models.SIPCallAttempt
	if err := s.db.WithContext(ctx).
		Where("id = ? AND campaign_id = ? AND contact_id = ? AND attempt_no = ?", attemptID, campaign.ID, contactID, attemptNo).
		First(&attempt).Error; err != nil {
		return
	}
	if strings.TrimSpace(attempt.State) != "dialing" {
		return
	}
	s.appendEvent(ctx, models.SIPCampaignEvent{
		CampaignID:    campaign.ID,
		ContactID:     contactID,
		AttemptID:     attemptID,
		CallID:        callID,
		CorrelationID: correlationID,
		Type:          "dial",
		Level:         "error",
		Message:       "dial timeout: no final SIP response within 45s",
	})
	s.HandleDialEvent(ctx, outbound.DialEvent{
		CallID:        callID,
		CorrelationID: correlationID,
		Scenario:      outbound.Scenario(strings.TrimSpace(campaign.Scenario)),
		MediaProfile:  outbound.MediaProfile(strings.TrimSpace(campaign.MediaProfile)),
		State:         outbound.DialEventFailed,
		StatusCode:    408,
		Reason:        "timeout_no_final_response",
		At:            time.Now(),
	})
}

func (s *CampaignService) isDuplicateWithinWindow(ctx context.Context, contactID uint, phone string, campaignID uint) bool {
	if strings.TrimSpace(phone) == "" {
		return false
	}
	windowStart := time.Now().Add(-s.dedupeWindow)
	var count int64
	_ = s.db.WithContext(ctx).Model(&models.SIPCampaignContact{}).
		Where("id <> ? AND phone = ? AND campaign_id = ? AND attempt_count > 0 AND last_dial_at >= ?", contactID, phone, campaignID, windowStart).
		Count(&count).Error
	return count > 0
}

func buildDialTarget(c models.SIPCampaign, ct models.SIPCampaignContact) (outbound.DialTarget, error) {
	host := strings.TrimSpace(utils.GetEnv(constants.EnvSIPOutboundHost))
	if host == "" {
		return outbound.DialTarget{}, fmt.Errorf("SIP_OUTBOUND_HOST is required")
	}
	port := 5060
	if s := strings.TrimSpace(utils.GetEnv(constants.EnvSIPOutboundPort)); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 65535 {
			port = n
		}
	}
	defaultSig := strings.TrimSpace(utils.GetEnv(constants.EnvSIPSignalingAddr))
	if defaultSig == "" {
		defaultSig = fmt.Sprintf("%s:%d", host, port)
	}

	if u := strings.TrimSpace(ct.RequestURI); u != "" {
		return outbound.DialTarget{RequestURI: u, SignalingAddr: defaultSig}, nil
	}
	if tmpl := strings.TrimSpace(c.RequestURIFmt); tmpl != "" {
		return outbound.DialTarget{
			RequestURI:    fmt.Sprintf(tmpl, strings.TrimSpace(ct.Phone)),
			SignalingAddr: defaultSig,
		}, nil
	}
	return outbound.DialTarget{
		RequestURI:    fmt.Sprintf("sip:%s@%s:%d", strings.TrimSpace(ct.Phone), host, port),
		SignalingAddr: defaultSig,
	}, nil
}

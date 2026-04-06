package sipcampaign

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/sip/outbound"
	"go.uber.org/zap"
)

type Dialer interface {
	Dial(ctx context.Context, req outbound.DialRequest) (callID string, err error)
}

func (s *Service) StartWorker(dialer Dialer) {
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
	s.mu.Unlock()

	sem := make(chan struct{}, s.globalConcurrency)
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
				s.tick(dialer, sem)
			}
		}
	}()
}

func (s *Service) StopWorker() {
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
	s.mu.Unlock()
	s.wg.Wait()
}

func (s *Service) tick(dialer Dialer, sem chan struct{}) {
	ctx := context.Background()
	now := time.Now()
	var campaigns []models.SIPCampaign
	if err := s.db.WithContext(ctx).
		Where("status = ?", models.SIPCampaignStatusRunning).
		Find(&campaigns).Error; err != nil {
		return
	}
	for _, c := range campaigns {
		limit := c.TaskConcurrency
		if limit <= 0 {
			limit = 1
		}
		var contacts []models.SIPCampaignContact
		if err := s.db.WithContext(ctx).
			Where("campaign_id = ? AND status IN ? AND (next_run_at IS NULL OR next_run_at <= ?)", c.ID, []string{models.SIPCampaignContactReady, models.SIPCampaignContactRetrying}, now).
			Order("priority desc, id asc").
			Limit(limit).
			Find(&contacts).Error; err != nil {
			continue
		}
		for _, contact := range contacts {
			if !s.tryClaim(ctx, contact.ID) {
				continue
			}
			sem <- struct{}{}
			s.wg.Add(1)
			go func(campaign models.SIPCampaign, ct models.SIPCampaignContact) {
				defer s.wg.Done()
				defer func() { <-sem }()
				s.processContact(context.Background(), dialer, campaign, ct)
			}(c, contact)
		}
	}
}

func (s *Service) tryClaim(ctx context.Context, contactID uint) bool {
	tx := s.db.WithContext(ctx).Model(&models.SIPCampaignContact{}).
		Where("id = ? AND status IN ?", contactID, []string{models.SIPCampaignContactReady, models.SIPCampaignContactRetrying}).
		Updates(map[string]any{"status": models.SIPCampaignContactDialing})
	return tx.Error == nil && tx.RowsAffected == 1
}

func (s *Service) processContact(ctx context.Context, dialer Dialer, campaign models.SIPCampaign, contact models.SIPCampaignContact) {
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
		"last_dial_at": &now,
	}).Error
	target, err := buildDialTarget(campaign, contact)
	if err != nil {
		_ = s.db.WithContext(ctx).Model(&models.SIPCampaignContact{}).Where("id = ?", contact.ID).
			Updates(map[string]any{"status": models.SIPCampaignContactFailed, "failure_reason": err.Error()}).Error
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
	s.appendEvent(ctx, models.SIPCampaignEvent{
		CampaignID:    campaign.ID,
		ContactID:     contact.ID,
		AttemptID:     attempt.ID,
		CallID:        callID,
		CorrelationID: correlationID,
		Type:          "dispatch",
		Level:         "info",
		Message:       fmt.Sprintf("dial request dispatched uri=%s signaling=%s", target.RequestURI, target.SignalingAddr),
	})
	_ = s.db.WithContext(ctx).Model(&models.SIPCampaignContact{}).Where("id = ?", contact.ID).Updates(map[string]any{
		"attempt_count":  attemptNo,
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
			zap.String("correlation_id", correlationID))
	}
}

func (s *Service) watchDialAttemptTimeout(campaign models.SIPCampaign, contactID, attemptID uint, attemptNo int, callID, correlationID string) {
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

func (s *Service) isDuplicateWithinWindow(ctx context.Context, contactID uint, phone string, campaignID uint) bool {
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
	if u := strings.TrimSpace(ct.RequestURI); u != "" {
		sig := strings.TrimSpace(c.SignalingAddr)
		if sig == "" {
			return outbound.DialTarget{}, fmt.Errorf("signaling_addr required when contact request_uri is set")
		}
		return outbound.DialTarget{RequestURI: u, SignalingAddr: sig}, nil
	}
	if tmpl := strings.TrimSpace(c.RequestURIFmt); tmpl != "" {
		sig := strings.TrimSpace(c.SignalingAddr)
		if sig == "" {
			return outbound.DialTarget{}, fmt.Errorf("signaling_addr required with request_uri_fmt")
		}
		return outbound.DialTarget{
			RequestURI:    fmt.Sprintf(tmpl, strings.TrimSpace(ct.Phone)),
			SignalingAddr: sig,
		}, nil
	}
	host := strings.TrimSpace(c.OutboundHost)
	if host == "" {
		return outbound.DialTarget{}, fmt.Errorf("outbound_host is required")
	}
	port := c.OutboundPort
	if port <= 0 {
		port = 5060
	}
	sig := strings.TrimSpace(c.SignalingAddr)
	if sig == "" {
		sig = fmt.Sprintf("%s:%d", host, port)
	}
	return outbound.DialTarget{
		RequestURI:    fmt.Sprintf("sip:%s@%s:%d", strings.TrimSpace(ct.Phone), host, port),
		SignalingAddr: sig,
	}, nil
}


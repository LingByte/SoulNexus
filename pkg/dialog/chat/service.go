// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package chat is the channel-agnostic text dialog core (history, KB, NLU, tools).
package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	"github.com/LingByte/SoulNexus/pkg/dialog/session"
	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
	stagenlu "github.com/LingByte/SoulNexus/pkg/dialog/stages/nlu"
	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const defaultHistoryTurns = 20 // user+assistant pairs ≈ 40 messages

// EnsureParams opens or reuses a dialog conversation.
type EnsureParams struct {
	TenantID         uint
	AssistantID      uint
	Channel          string
	ChannelAccountID string
	ExternalUserID   string
	Metadata         map[string]any
}

// TurnResult is one completed user→assistant exchange.
type TurnResult struct {
	ConversationID uint
	Reply          string
	LatencyMs      int64
	UserMessageID  uint
	AssistantMsgID uint
	Confidence     *float64
	ToolsJSON      string
}

// StreamHandlers receives SSE-style turn events.
type StreamHandlers struct {
	OnDelta     func(delta string)
	OnStage     func(stage string) // thinking | retrieving | tools | generating
	OnCompleted func(res TurnResult)
}

// Service runs text dialog turns against the shared cascaded LLM stack.
type Service struct {
	db *gorm.DB
	lg *zap.Logger
}

// New returns a chat service.
func New(db *gorm.DB, lg *zap.Logger) *Service {
	if lg == nil {
		lg = zap.NewNop()
	}
	return &Service{db: db, lg: lg}
}

type historySeeder interface {
	SeedHistory(msgs []providers.LLMMessage)
}

type systemAugmenter interface {
	AppendSystemAppendix(appendix string)
}

type toolTracer interface {
	LastToolTrace() []providers.LLMToolCall
}

// EnsureConversation finds an open conversation or creates one.
func (s *Service) EnsureConversation(ctx context.Context, p EnsureParams) (*models.DialogConversation, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("dialog/chat: nil service")
	}
	if p.TenantID == 0 || p.AssistantID == 0 {
		return nil, errors.New("dialog/chat: tenant_id and assistant_id required")
	}
	channel := strings.TrimSpace(p.Channel)
	if channel == "" {
		channel = models.DialogChannelAPI
	}
	ext := strings.TrimSpace(p.ExternalUserID)
	if ext == "" {
		ext = "anonymous"
	}
	acct := strings.TrimSpace(p.ChannelAccountID)

	// Debug sessions should not reuse prior turns (avoids stale hello/history).
	if channel != models.DialogChannelDebug {
		var existing models.DialogConversation
		q := s.db.WithContext(ctx).Where(
			"tenant_id = ? AND channel = ? AND channel_account_id = ? AND external_user_id = ? AND status = ?",
			p.TenantID, channel, acct, ext, models.DialogConvStatusOpen,
		)
		if p.AssistantID > 0 {
			q = q.Where("assistant_id = ?", p.AssistantID)
		}
		err := q.Order("id DESC").First(&existing).Error
		if err == nil {
			return &existing, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	meta := ""
	if len(p.Metadata) > 0 {
		if raw, mErr := json.Marshal(p.Metadata); mErr == nil {
			meta = string(raw)
		}
	}
	row := models.DialogConversation{
		TenantID:         p.TenantID,
		AssistantID:      p.AssistantID,
		Channel:          channel,
		ChannelAccountID: acct,
		ExternalUserID:   ext,
		Status:           models.DialogConvStatusOpen,
		MetadataJSON:     meta,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// GetConversation loads by id scoped to tenant.
func (s *Service) GetConversation(ctx context.Context, tenantID, id uint) (*models.DialogConversation, error) {
	var row models.DialogConversation
	err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ListMessages returns messages oldest-first.
func (s *Service) ListMessages(ctx context.Context, conversationID uint, limit int) ([]models.DialogMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows []models.DialogMessage
	err := s.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// EndConversation marks the session closed and emits dialog.session_ended.
func (s *Service) EndConversation(ctx context.Context, tenantID, id uint) error {
	conv, err := s.GetConversation(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if conv.Status == models.DialogConvStatusClosed {
		return nil
	}
	if err := s.db.WithContext(ctx).Model(conv).Update("status", models.DialogConvStatusClosed).Error; err != nil {
		return err
	}
	callbinding.ClearAssistantID(conv.CallKey())
	callbinding.ClearTenantID(conv.CallKey())
	stageknow.ClearBindingCache(conv.CallKey())
	stagenlu.ClearBinding(conv.CallKey())
	notification.DispatchWebhook(s.db, s.lg, tenantID, constants.WebhookEventDialogEnded, conv.CallKey(),
		conv.ExternalUserID, "", conv.Channel, map[string]any{
			"conversationId": conv.ID,
			"assistantId":    conv.AssistantID,
			"channel":        conv.Channel,
		})
	return nil
}

// HandleUserText runs one LLM turn with persisted history.
func (s *Service) HandleUserText(ctx context.Context, tenantID, conversationID uint, text string) (TurnResult, error) {
	return s.handleUserText(ctx, tenantID, conversationID, text, nil)
}

// HandleUserTextStream runs a turn and emits deltas via handlers (tool path may flush once).
func (s *Service) HandleUserTextStream(ctx context.Context, tenantID, conversationID uint, text string, h StreamHandlers) (TurnResult, error) {
	return s.handleUserText(ctx, tenantID, conversationID, text, &h)
}

func (s *Service) handleUserText(ctx context.Context, tenantID, conversationID uint, text string, stream *StreamHandlers) (TurnResult, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return TurnResult{}, errors.New("dialog/chat: empty message")
	}
	conv, err := s.GetConversation(ctx, tenantID, conversationID)
	if err != nil {
		return TurnResult{}, err
	}
	if conv.Status != models.DialogConvStatusOpen {
		return TurnResult{}, errors.New("dialog/chat: conversation closed")
	}

	callID := conv.CallKey()
	callbinding.SetAssistantID(callID, conv.AssistantID)
	callbinding.SetTenantID(callID, conv.TenantID)
	callbinding.SetAISource(callID, "dialog_"+conv.Channel)

	env, ok, err := tenantcfg.Resolve(ctx, conv.TenantID, callID)
	if err != nil {
		return TurnResult{}, err
	}
	if !ok {
		return TurnResult{}, errors.New("dialog/chat: tenant voice config not found")
	}
	if !tenantcfg.LLMReady(env) {
		return TurnResult{}, fmt.Errorf("dialog/chat: %s", tenantcfg.LLMReadinessReason(env))
	}

	stageknow.PrepareCallKnowledgeBinding(callID, env, conv.TenantID, stageknow.SearchConfigFromVoiceEnv(env), s.lg)
	stagenlu.PrepareCallNLUBinding(callID, env, s.lg)

	hist, _ := s.loadHistoryForSeed(ctx, conversationID)

	llm, err := session.BuildLLMForChat(cascaded.WithTextDialog(ctx), env, callID)
	if err != nil {
		return TurnResult{}, err
	}
	if seeder, ok := llm.(historySeeder); ok {
		seeder.SeedHistory(hist)
	}
	s.applySurfaceAndSkills(llm, conv.TenantID, conv.Channel, env)

	userMsg := models.DialogMessage{
		ConversationID: conversationID,
		Role:           models.DialogMsgRoleUser,
		Content:        text,
	}
	if err := s.db.WithContext(ctx).Create(&userMsg).Error; err != nil {
		return TurnResult{}, err
	}

	replyCtx := cascaded.WithPreferTools(
		cascaded.WithTextDialog(
			cascaded.WithMaxToolRounds(ctx, cascaded.DefaultTextToolRounds),
		),
	)

	started := time.Now()
	emitStage := func(stage string) {
		if stream != nil && stream.OnStage != nil && stage != "" {
			stream.OnStage(stage)
		}
	}
	emitStage("thinking")
	onDelta := func(seg string, complete bool) error {
		if stream != nil && stream.OnDelta != nil && seg != "" && !complete {
			stream.OnDelta(seg)
		}
		return nil
	}
	emitStage("tools")
	reply, err := llm.StreamReply(replyCtx, text, onDelta)
	if err != nil {
		return TurnResult{}, err
	}
	reply = strings.TrimSpace(reply)

	var knowledgeJSON string
	var kbRows []any
	if takeKB := session.TakeKnowledgeRetrievalsFn(); takeKB != nil {
		if rows := takeKB(callID); len(rows) > 0 {
			kbRows = make([]any, len(rows))
			for i := range rows {
				kbRows[i] = rows[i]
			}
			if raw, mErr := json.Marshal(rows); mErr == nil {
				knowledgeJSON = string(raw)
			}
		}
	}

	// KB remediation: bound + no retrieval this turn + not pure chitchat.
	if len(kbRows) == 0 && stageknow.ResolveBinding(callID).Enabled && needsKBRemediation(text) {
		emitStage("retrieving")
		block := stageknow.ForceSearchBlockForQuery(context.Background(), callID, text, s.lg)
		if strings.TrimSpace(block) != "" {
			emitStage("generating")
			remCtx := cascaded.WithForcedKnowledgeBlock(replyCtx, block)
			reply2, remErr := llm.StreamReply(remCtx, text, onDelta)
			if remErr == nil && strings.TrimSpace(reply2) != "" {
				reply = strings.TrimSpace(reply2)
			}
			if takeKB := session.TakeKnowledgeRetrievalsFn(); takeKB != nil {
				if rows := takeKB(callID); len(rows) > 0 {
					if raw, mErr := json.Marshal(rows); mErr == nil {
						knowledgeJSON = string(raw)
					}
				}
			}
		}
	} else {
		emitStage("generating")
	}

	latency := time.Since(started).Milliseconds()
	toolsJSON := captureToolsJSON(llm)

	asstMsg := models.DialogMessage{
		ConversationID: conversationID,
		Role:           models.DialogMsgRoleAssistant,
		Content:        reply,
		KnowledgeJSON:  knowledgeJSON,
		ToolsJSON:      toolsJSON,
		LatencyMs:      latency,
	}
	if err := s.db.WithContext(ctx).Create(&asstMsg).Error; err != nil {
		return TurnResult{}, err
	}

	res := TurnResult{
		ConversationID: conversationID,
		Reply:          reply,
		LatencyMs:      latency,
		UserMessageID:  userMsg.ID,
		AssistantMsgID: asstMsg.ID,
		ToolsJSON:      toolsJSON,
	}

	webhookPayload := map[string]any{
		"conversationId": conversationID,
		"assistantId":    conv.AssistantID,
		"channel":        conv.Channel,
		"userText":       truncateRunes(text, 200),
		"replyText":      truncateRunes(reply, 200),
		"latencyMs":      latency,
	}

	go s.finishTurnAsync(env, tenantID, callID, conv.ExternalUserID, conv.Channel, asstMsg.ID, text, reply, webhookPayload)

	if stream != nil && stream.OnCompleted != nil {
		stream.OnCompleted(res)
	}
	return res, nil
}

func (s *Service) applySurfaceAndSkills(llm cascaded.LLMService, tenantID uint, channel string, env tenantcfg.VoiceEnv) {
	names := skillNamesFromEnv(env)
	if aug, ok := llm.(systemAugmenter); ok {
		surface := SurfaceForChannel(channel)
		var parts []string
		if surface.SystemAppendix != "" {
			parts = append(parts, surface.SystemAppendix)
		}
		if skills := LoadSkillsAppendix(s.db, tenantID, names); skills != "" {
			parts = append(parts, skills)
		}
		if len(parts) > 0 {
			aug.AppendSystemAppendix(strings.Join(parts, "\n\n"))
		}
	}
	RegisterBoundSkillTools(s.db, tenantID, names, llm, s.lg)
}

func (s *Service) finishTurnAsync(
	env tenantcfg.VoiceEnv,
	tenantID uint,
	callID, externalUserID, channel string,
	msgID uint,
	userText, reply string,
	webhookPayload map[string]any,
) {
	if cr, ok := scoreTurnConfidence(context.Background(), env, userText, reply, s.lg); ok {
		_ = s.db.Model(&models.DialogMessage{}).Where("id = ?", msgID).Updates(map[string]any{
			"confidence":      cr.Score,
			"confidence_json": confidenceJSON(cr),
		}).Error
		if webhookPayload != nil {
			webhookPayload["confidence"] = cr.Score
		}
	}
	notification.DispatchWebhook(s.db, s.lg, tenantID, constants.WebhookEventDialogTurn, callID,
		externalUserID, "", channel, webhookPayload)
}

func captureToolsJSON(llm cascaded.LLMService) string {
	tr, ok := llm.(toolTracer)
	if !ok {
		return ""
	}
	calls := tr.LastToolTrace()
	if len(calls) == 0 {
		return ""
	}
	type toolRow struct {
		ID        string `json:"id,omitempty"`
		Name      string `json:"name"`
		Arguments string `json:"arguments,omitempty"`
	}
	rows := make([]toolRow, 0, len(calls))
	for _, c := range calls {
		rows = append(rows, toolRow{ID: c.ID, Name: c.Function.Name, Arguments: c.Function.Arguments})
	}
	raw, err := json.Marshal(rows)
	if err != nil {
		return ""
	}
	return string(raw)
}

func needsKBRemediation(text string) bool {
	return len([]rune(strings.TrimSpace(text))) > 2
}

func (s *Service) loadHistoryForSeed(ctx context.Context, conversationID uint) ([]providers.LLMMessage, error) {
	limit := defaultHistoryTurns * 2
	var rows []models.DialogMessage
	err := s.db.WithContext(ctx).
		Where("conversation_id = ? AND role IN ?", conversationID, []string{models.DialogMsgRoleUser, models.DialogMsgRoleAssistant}).
		Order("id DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	out := make([]providers.LLMMessage, 0, len(rows))
	for _, r := range rows {
		content := r.Content
		// Replay tool use as a compact appendix so the next turn remembers prior tool calls
		// without requiring full tool_result protocol messages (phase-1 compatible).
		if r.Role == models.DialogMsgRoleAssistant {
			if summary := FormatToolsMemoryAppendix(r.ToolsJSON); summary != "" {
				content = strings.TrimSpace(content)
				if content == "" {
					content = summary
				} else {
					content = content + "\n\n" + summary
				}
			}
		}
		out = append(out, providers.LLMMessage{Role: r.Role, Content: content})
	}
	return out, nil
}

// FormatToolsMemoryAppendix builds a short Chinese tool-memory block from ToolsJSON.
func FormatToolsMemoryAppendix(toolsJSON string) string {
	toolsJSON = strings.TrimSpace(toolsJSON)
	if toolsJSON == "" {
		return ""
	}
	var rows []struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(toolsJSON), &rows); err != nil || len(rows) == 0 {
		return ""
	}
	parts := make([]string, 0, len(rows))
	for _, r := range rows {
		name := strings.TrimSpace(r.Name)
		if name == "" {
			continue
		}
		args := strings.TrimSpace(r.Arguments)
		if args == "" {
			parts = append(parts, name)
			continue
		}
		if len([]rune(args)) > 120 {
			args = string([]rune(args)[:120]) + "…"
		}
		parts = append(parts, fmt.Sprintf("%s(%s)", name, args))
	}
	if len(parts) == 0 {
		return ""
	}
	return "【工具记忆·供后续回合参考】本助手上轮已调用：" + strings.Join(parts, "；")
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if max <= 0 || len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

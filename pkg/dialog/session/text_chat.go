package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
	stagenlu "github.com/LingByte/SoulNexus/pkg/dialog/stages/nlu"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
)

// TextMessageResult is the reply plus latency metrics for a text turn.
type TextMessageResult struct {
	Reply   string
	Metrics TurnMetricsWire
}

// TextChatBridge delegates persisted text turns to pkg/dialog/chat when wired.
type TextChatBridge struct {
	EnsureConversation func(ctx context.Context, tenantID, assistantID uint, channel, externalUserID string) (conversationID uint, welcome string, err error)
	HandleUserText     func(ctx context.Context, tenantID, conversationID uint, text string) (reply string, latencyMs int64, err error)
	EndConversation    func(ctx context.Context, tenantID, conversationID uint) error
}

var textChatBridge TextChatBridge

// SetTextChatBridge installs the dialog/chat persistence bridge (once at bootstrap).
func SetTextChatBridge(b TextChatBridge) {
	textChatBridge = b
}

// InitTextChat wires tenant LLM for an internal text-only session
// (e.g. internal text-chat bridge). Public H5/IM text uses /lingecho/dialog/v1.
// When TextChatBridge is set, opens a persisted dialog conversation; otherwise
// falls back to an in-memory LLM (tools enabled via PreferTools).
func (s *Session) InitTextChat(ctx context.Context) (welcome string, err error) {
	if s == nil {
		return "", errors.New("dialog/session: nil session")
	}
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return "", errors.New("dialog/session: session ended")
	}
	if s.dialogConvID > 0 {
		w := s.textWelcome
		s.mu.Unlock()
		return w, nil
	}
	if s.textLLM != nil {
		w := s.textWelcome
		s.mu.Unlock()
		return w, nil
	}
	s.mu.Unlock()

	resolveCtx := ctx
	if resolveCtx == nil {
		resolveCtx = context.Background()
	}

	if textChatBridge.EnsureConversation != nil {
		ext := "session-" + s.ID
		channel := "api"
		if s.skipCallPersist {
			channel = "debug"
		}
		convID, wel, err := textChatBridge.EnsureConversation(resolveCtx, s.TenantID, s.AssistantID, channel, ext)
		if err != nil {
			return "", err
		}
		s.mu.Lock()
		s.dialogConvID = convID
		s.textWelcome = wel
		s.mu.Unlock()
		return wel, nil
	}

	env, ok, err := tenantcfg.Resolve(resolveCtx, s.TenantID, s.CallID)
	if err != nil {
		return "", fmt.Errorf("dialog/session: tenant config: %w", err)
	}
	if !ok {
		return "", errors.New("dialog/session: tenant voice config not found")
	}
	if !tenantcfg.LLMReady(env) {
		reason := tenantcfg.LLMReadinessReason(env)
		if reason == "" {
			reason = "tenant LLM config incomplete"
		}
		return "", fmt.Errorf("dialog/session: %s (assistant_id=%d voice_mode=%s)", reason, s.AssistantID, env.VoiceMode)
	}
	stageknow.PrepareCallKnowledgeBinding(s.CallID, env, s.TenantID, stageknow.SearchConfigFromVoiceEnv(env), nil)
	stagenlu.PrepareCallNLUBinding(s.CallID, env, nil)
	if providerHooks.BuildLLM == nil {
		return "", errors.New("dialog/session: LLM provider not wired")
	}

	llmCtx, llmCancel := context.WithCancel(context.Background())
	llm, err := providerHooks.BuildLLM(llmCtx, env, s.CallID)
	if err != nil {
		llmCancel()
		return "", err
	}
	welcome = tenantcfg.ResolvedAssistantWelcome(env)
	s.mu.Lock()
	s.textLLM = llm
	s.textWelcome = welcome
	s.textLLMCancel = llmCancel
	s.mu.Unlock()
	return welcome, nil
}

// SendTextMessage runs one LLM turn for a text session.
func (s *Session) SendTextMessage(ctx context.Context, text string) (TextMessageResult, error) {
	if s == nil {
		return TextMessageResult{}, errors.New("dialog/session: nil session")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return TextMessageResult{}, errors.New("dialog/session: empty message")
	}
	s.mu.Lock()
	ended := s.ended
	convID := s.dialogConvID
	llm := s.textLLM
	s.mu.Unlock()
	if ended {
		return TextMessageResult{}, errors.New("dialog/session: session ended")
	}

	if convID > 0 && textChatBridge.HandleUserText != nil {
		started := time.Now()
		reply, latencyMs, err := textChatBridge.HandleUserText(ctx, s.TenantID, convID, text)
		if err != nil {
			return TextMessageResult{}, err
		}
		turnID := s.nextTurnID()
		firstToken := started
		if latencyMs > 0 {
			firstToken = started.Add(time.Duration(latencyMs/2) * time.Millisecond)
		}
		metrics := MetricsFromTextTurn(s.ID, turnID, text, reply, started, firstToken, "text")
		if latencyMs > 0 {
			metrics.LLMWallMs = int(latencyMs)
		}
		if takeKB := takeKnowledgeRetrievals; takeKB != nil {
			metrics.KnowledgeRetrievals = takeKB(fmt.Sprintf("dialog-%d", convID))
		}
		return TextMessageResult{Reply: reply, Metrics: metrics}, nil
	}

	if llm == nil {
		return TextMessageResult{}, errors.New("dialog/session: text chat not initialized")
	}
	started := time.Now()
	var firstToken time.Time
	reply, err := llm.StreamReply(cascaded.WithPreferTools(ctx), text, func(delta string, complete bool) error {
		if !complete && strings.TrimSpace(delta) != "" && firstToken.IsZero() {
			firstToken = time.Now()
		}
		return nil
	})
	if err != nil {
		return TextMessageResult{}, err
	}
	reply = strings.TrimSpace(reply)
	turnID := s.nextTurnID()
	metrics := MetricsFromTextTurn(s.ID, turnID, text, reply, started, firstToken, "text")
	if takeKB := takeKnowledgeRetrievals; takeKB != nil {
		metrics.KnowledgeRetrievals = takeKB(s.CallID)
	}
	return TextMessageResult{
		Reply:   reply,
		Metrics: metrics,
	}, nil
}

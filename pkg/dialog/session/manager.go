package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
	stagenlu "github.com/LingByte/SoulNexus/pkg/dialog/stages/nlu"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/google/uuid"
	"strings"
)

const defaultSampleRate = 16000

// Config configures the global session manager.
type Config struct {
	WebSocketPath   string
	WebRTCOfferPath string
}

// Manager tracks active voice sessions.
type Manager struct {
	cfg Config
	mu  sync.RWMutex
	m   map[string]*Session
}

var (
	defaultMgr *Manager
	mgrOnce    sync.Once
)

// InitDefault configures the process-wide session manager.
func InitDefault(cfg Config) {
	mgrOnce.Do(func() {
		if cfg.WebSocketPath == "" {
			cfg.WebSocketPath = "lingecho/voice-session/v1/ws"
		}
		if cfg.WebRTCOfferPath == "" {
			cfg.WebRTCOfferPath = "lingecho/voice-session/v1/webrtc/offer"
		}
		defaultMgr = &Manager{
			cfg: cfg,
			m:   make(map[string]*Session),
		}
	})
}

// Default returns the process-wide manager (nil if InitDefault not called).
func Default() *Manager { return defaultMgr }

// Create registers a new session and returns connection metadata.
func (m *Manager) Create(p CreateParams) (Info, error) {
	if m == nil {
		return Info{}, errors.New("dialog/session: manager not initialized")
	}
	if p.TenantID == 0 {
		return Info{}, errors.New("dialog/session: tenant_id required")
	}
	sr := p.SampleRate
	if sr <= 0 {
		sr = defaultSampleRate
	}
	id := uuid.NewString()
	callID := p.CallID
	if callID == "" {
		callID = id
	}
	if p.AssistantID > 0 {
		callbinding.SetAssistantID(callID, p.AssistantID)
	}
	callbinding.SetTenantID(callID, p.TenantID)
	callbinding.SetUserID(callID, p.UserID)
	callbinding.SetCredentialID(callID, p.CredentialID)
	if p.SkipCallPersist {
		RegisterDebugCall(callID)
	}
	s := &Session{
		ID:              id,
		TenantID:        p.TenantID,
		AssistantID:     p.AssistantID,
		Transport:       p.Transport,
		SampleRate:      sr,
		CallID:          callID,
		CreatedAt:       time.Now().UTC(),
		skipCallPersist: p.SkipCallPersist,
		dialogMode:      strings.TrimSpace(p.DialogMode),
		realtimeTemp:    p.RealtimeTemperature,
	}
	m.mu.Lock()
	m.m[id] = s
	m.mu.Unlock()

	// Overlap welcome TTS dial with SDP/ICE / WS upgrade so playout is warm.
	go kickWelcomePrewarm(s)

	return Info{
		SessionID:       id,
		TenantID:        p.TenantID,
		AssistantID:     p.AssistantID,
		Transport:       p.Transport,
		SampleRate:      sr,
		WebSocketPath:   m.cfg.WebSocketPath,
		WebRTCOfferPath: m.cfg.WebRTCOfferPath,
		DialogMode:      s.dialogMode,
		CreatedAt:       s.CreatedAt,
	}, nil
}

func kickWelcomePrewarm(s *Session) {
	if s == nil {
		return
	}
	env, ok, _ := tenantcfg.Resolve(context.Background(), s.TenantID, s.CallID)
	if !ok {
		return
	}
	env = EffectiveVoiceEnv(s, env)
	StartAssistantWelcomePrewarm(s.CallID, env, s.SampleRate)
}

// SetTurnNotify installs a callback for completed dialog turns (web debug UI).
func (s *Session) SetTurnNotify(fn func(cascaded.TurnRecord)) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.onTurn = fn
	s.mu.Unlock()
}

// Get returns a session by ID.
func (m *Manager) Get(id string) (*Session, bool) {
	if m == nil {
		return nil, false
	}
	m.mu.RLock()
	s, ok := m.m[id]
	m.mu.RUnlock()
	return s, ok
}

// End removes and tears down a session.
func (m *Manager) End(id string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	s := m.m[id]
	delete(m.m, id)
	m.mu.Unlock()
	if s != nil {
		s.close(context.Background())
	}
}

// BindPort stores the transport MediaPort on the session (pre-attach).
func (s *Session) BindPort(port engine.MediaPort) error {
	if s == nil || port == nil {
		return errors.New("dialog/session: invalid bind")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return errors.New("dialog/session: session ended")
	}
	s.port = port
	return nil
}

// StartEngine loads tenant config and attaches the dialog engine to the bound port.
func (s *Session) StartEngine(ctx context.Context, lg engine.Logger) error {
	if s == nil {
		return errors.New("dialog/session: nil session")
	}
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return errors.New("dialog/session: session ended")
	}
	if s.attached {
		s.mu.Unlock()
		return nil
	}
	port := s.port
	s.mu.Unlock()
	if port == nil {
		return errors.New("dialog/session: port not bound")
	}
	env, ok, err := tenantcfg.Resolve(ctx, s.TenantID, s.CallID)
	if err != nil {
		return fmt.Errorf("dialog/session: tenant config: %w", err)
	}
	if !ok {
		return errors.New("dialog/session: tenant voice config not found")
	}
	env = EffectiveVoiceEnv(s, env)
	if !tenantcfg.VoiceReady(env) {
		reason := tenantcfg.VoiceReadinessReason(env)
		if reason == "" {
			reason = "tenant voice config incomplete"
		}
		return errors.New("dialog/session: " + reason)
	}
	s.mu.Lock()
	rtTemp := s.realtimeTemp
	s.mu.Unlock()
	if rtTemp > 0 {
		env = cloneVoiceEnvWithRealtimeTemperature(env, rtTemp)
	}
	stageknow.PrepareCallKnowledgeBinding(s.CallID, env, s.TenantID, stageknow.SearchConfigFromVoiceEnv(env), nil)
	stagenlu.PrepareCallNLUBinding(s.CallID, env, nil)
	runCtx, cancel := context.WithCancel(ctx)
	mode := ResolveMode(env)
	s.mu.Lock()
	onTurn := s.onTurn
	s.mu.Unlock()
	liveObs := s.pipelineLiveTranscriptObserver()
	detach, err := AttachEngine(runCtx, AttachParams{
		Port:             port,
		Env:              env,
		Mode:             mode,
		Logger:           lg,
		OnTurn:           onTurn,
		OnLiveTranscript: liveObs,
		SkipCallPersist:  s.skipCallPersist,
	})
	if err != nil {
		cancel()
		return err
	}
	s.mu.Lock()
	s.cancel = cancel
	s.detach = detach
	s.attached = true
	s.mu.Unlock()
	return nil
}

func (s *Session) close(ctx context.Context) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return
	}
	s.ended = true
	cancel := s.cancel
	textCancel := s.textLLMCancel
	dialogConvID := s.dialogConvID
	tenantID := s.TenantID
	s.textLLMCancel = nil
	s.textLLM = nil
	s.dialogConvID = 0
	detach := s.detach
	port := s.port
	s.mu.Unlock()
	if dialogConvID > 0 && textChatBridge.EndConversation != nil {
		_ = textChatBridge.EndConversation(ctx, tenantID, dialogConvID)
	}
	if cancel != nil {
		cancel()
	}
	if textCancel != nil {
		textCancel()
	}
	if detach != nil {
		_ = detach(ctx)
	}
	callID := s.CallID
	if c, ok := port.(interface{ Close() error }); ok {
		_ = c.Close()
	}
	if callID != "" {
		stageknow.ClearBindingCache(callID)
		stageknow.ClearSearchCache(callID)
		stagenlu.ClearBinding(callID)
		callbinding.ClearAssistantID(callID)
		callbinding.ClearTenantID(callID)
		callbinding.ClearUserID(callID)
		callbinding.ClearCredentialID(callID)
		callbinding.ClearTransitPoolIDs(callID)
		callbinding.ClearAISource(callID)
		callbinding.ClearJSSourceID(callID)
		callbinding.ClearSpeakerContext(callID)
		callbinding.ClearSpeakerRoster(callID)
		callbinding.ClearUserUtterancePCM(callID)
		UnregisterDebugCall(callID)
	}
}

// Close ends the session.
func (s *Session) Close() {
	if defaultMgr != nil {
		defaultMgr.End(s.ID)
	} else {
		s.close(context.Background())
	}
}

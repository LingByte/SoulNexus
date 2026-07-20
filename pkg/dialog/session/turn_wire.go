package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	"github.com/LingByte/SoulNexus/pkg/dialog/turn"
)

const wireProtocolVersion = 1

const (
	WireTypeTranscriptUser      = "transcript.user"
	WireTypeTranscriptAssistant = "transcript.assistant"
	WireTypeTurnMetrics         = "turn.metrics"
)

// TurnMetricsWire is latency data for one completed dialog turn (debug UI).
type TurnMetricsWire struct {
	Type          string `json:"type"`
	V             int    `json:"v,omitempty"`
	SessionID     string `json:"sessionId,omitempty"`
	TurnID        string `json:"turnId,omitempty"`
	UserText      string `json:"userText,omitempty"`
	AssistantText string `json:"assistantText,omitempty"`
	// LLMFirstMs: ASR-final (or user send) → first LLM token/text.
	LLMFirstMs int `json:"llmFirstMs,omitempty"`
	// LLMWallMs: ASR-final → assistant turn end.
	LLMWallMs int `json:"llmWallMs,omitempty"`
	// TTSFirstMs: ASR-final → first audible AI PCM (pipeline TTS TTFB).
	TTSFirstMs int `json:"ttsFirstMs,omitempty"`
	// PipelineMs: end-to-end turn wall time (same as LLMWallMs in current topology).
	PipelineMs int `json:"pipelineMs,omitempty"`
	// E2EFirstMs: user stopped speaking → first AI PCM toward client
	// (includes VAD/ASR hangover after speech; falls back to TTSFirstMs).
	E2EFirstMs  int    `json:"e2eFirstMs,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Transport   string `json:"transport,omitempty"`
	CompletedAt string `json:"completedAt,omitempty"`
	// KnowledgeRetrievals is populated when search_knowledge_base ran this turn (debug UI).
	KnowledgeRetrievals []turn.KnowledgeRetrievalRecord `json:"knowledgeRetrievals,omitempty"`
}

// BindTurnNotify emits live + completed transcripts for the debug UI.
func (s *Session) BindTurnNotify(write func([]byte) error, mode, transport string) {
	if s == nil || write == nil {
		return
	}
	s.mu.Lock()
	s.wireWrite = write
	s.mu.Unlock()
	s.SetTurnNotify(func(rec cascaded.TurnRecord) {
		u := strings.TrimSpace(rec.UserText)
		a := strings.TrimSpace(rec.AIText)
		if a == "" {
			return
		}
		turnID := s.nextTurnID()
		if u != "" {
			if b, err := encodeTranscript(s.ID, WireTypeTranscriptUser, turnID, u, true); err == nil {
				_ = write(b)
			}
			// Xiaozhi dialect caption.
			if b, err := json.Marshal(map[string]any{
				"type": "stt", "text": u, "session_id": s.ID, "sessionId": s.ID,
			}); err == nil {
				_ = write(b)
			}
		}
		if b, err := encodeTranscript(s.ID, WireTypeTranscriptAssistant, turnID, a, true); err == nil {
			_ = write(b)
		}
		if b, err := json.Marshal(map[string]any{"type": "llm_response", "text": a}); err == nil {
			_ = write(b)
		}
		m := metricsFromTurnRecord(s.ID, turnID, rec, mode, transport)
		if takeKB := takeKnowledgeRetrievals; takeKB != nil {
			// Peek so persist can still Take the same buffer.
			if peek := peekKnowledgeRetrievals; peek != nil {
				m.KnowledgeRetrievals = peek(s.CallID)
			} else {
				m.KnowledgeRetrievals = takeKB(s.CallID)
			}
		}
		if b, err := json.Marshal(m); err == nil {
			_ = write(b)
		}
	})
}

func (s *Session) nextTurnID() string {
	n := atomic.AddUint64(&s.turnSeq, 1)
	return fmt.Sprintf("%s-%d", s.ID, n)
}

func metricsFromTurnRecord(sessionID, turnID string, rec cascaded.TurnRecord, mode, transport string) TurnMetricsWire {
	e2e := rec.E2EFirstByteMs
	if e2e <= 0 {
		e2e = rec.TTSFirstByteMs
	}
	if e2e <= 0 {
		e2e = rec.LLMFirstMs
	}
	completed := ""
	if !rec.CompletedAt.IsZero() {
		completed = rec.CompletedAt.UTC().Format(time.RFC3339Nano)
	}
	return TurnMetricsWire{
		Type:          WireTypeTurnMetrics,
		V:             wireProtocolVersion,
		SessionID:     sessionID,
		TurnID:        turnID,
		UserText:      strings.TrimSpace(rec.UserText),
		AssistantText: strings.TrimSpace(rec.AIText),
		LLMFirstMs:    rec.LLMFirstMs,
		LLMWallMs:     rec.LLMWallMs,
		TTSFirstMs:    rec.TTSFirstByteMs,
		PipelineMs:    rec.PipelineMs,
		E2EFirstMs:    e2e,
		Mode:          mode,
		Transport:     transport,
		CompletedAt:   completed,
	}
}

type wireTranscript struct {
	Type      string `json:"type"`
	V         int    `json:"v,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	TurnID    string `json:"turnId,omitempty"`
	Text      string `json:"text,omitempty"`
	Final     bool   `json:"final,omitempty"`
}

func encodeTranscript(sessionID, typ, turnID, text string, final bool) ([]byte, error) {
	return json.Marshal(wireTranscript{
		Type:      typ,
		V:         wireProtocolVersion,
		SessionID: sessionID,
		TurnID:    turnID,
		Text:      text,
		Final:     final,
	})
}

// MetricsFromTextTurn builds metrics for pure-text LLM turns.
func MetricsFromTextTurn(sessionID, turnID, userText, reply string, started, firstToken time.Time, mode string) TurnMetricsWire {
	wall := int(time.Since(started).Milliseconds())
	first := 0
	if !firstToken.IsZero() {
		first = int(firstToken.Sub(started).Milliseconds())
	}
	return TurnMetricsWire{
		Type:          WireTypeTurnMetrics,
		V:             wireProtocolVersion,
		SessionID:     sessionID,
		TurnID:        turnID,
		UserText:      strings.TrimSpace(userText),
		AssistantText: strings.TrimSpace(reply),
		LLMFirstMs:    first,
		LLMWallMs:     wall,
		PipelineMs:    wall,
		E2EFirstMs:    first,
		Mode:          mode,
		Transport:     string(TransportText),
		CompletedAt:   time.Now().UTC().Format(time.RFC3339Nano),
	}
}

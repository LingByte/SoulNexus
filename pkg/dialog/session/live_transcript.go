package session

import (
	"encoding/json"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
)

func (s *Session) pipelineLiveTranscriptObserver() func(pipeline.Frame) {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	write := s.wireWrite
	s.mu.Unlock()
	if write == nil {
		return nil
	}
	return func(f pipeline.Frame) {
		switch f.Kind {
		case pipeline.KindTextInterim:
			text := strings.TrimSpace(f.Text)
			if text == "" {
				return
			}
			if b, err := encodeTranscript(s.ID, WireTypeTranscriptUser, s.liveTurnID, text, false); err == nil {
				_ = write(b)
			}
		case pipeline.KindTextFinal:
			text := strings.TrimSpace(f.Text)
			if text == "" {
				return
			}
			s.liveTurnID = s.nextTurnID()
			if b, err := encodeTranscript(s.ID, WireTypeTranscriptUser, s.liveTurnID, text, true); err == nil {
				_ = write(b)
			}
			if b, err := json.Marshal(map[string]any{
				"type": "stt", "text": text, "session_id": s.ID, "sessionId": s.ID,
			}); err == nil {
				_ = write(b)
			}
		case pipeline.KindAIText:
			text := strings.TrimSpace(f.Text)
			if text == "" {
				return
			}
			if s.liveTurnID == "" {
				s.liveTurnID = s.nextTurnID()
			}
			if b, err := encodeTranscript(s.ID, WireTypeTranscriptAssistant, s.liveTurnID, text, false); err == nil {
				_ = write(b)
			}
		case pipeline.KindAITextDone:
			s.liveTurnID = ""
		}
	}
}

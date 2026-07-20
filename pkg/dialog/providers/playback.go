package providers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/turn"
	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
	siptts "github.com/LingByte/SoulNexus/pkg/voice/tts"
	"go.uber.org/zap"
)

// StreamTurnTimings captures per-turn latency breakdown.
type StreamTurnTimings = turn.StreamTimings

// StreamPlainTextToTTS speaks fixed text without calling an LLM.
func StreamPlainTextToTTS(ctx context.Context, text string, ttsPipe *siptts.Pipeline, lg *zap.Logger, ttsPrep func(string) string) (string, StreamTurnTimings, error) {
	var meta StreamTurnTimings
	if ttsPipe == nil {
		return "", meta, fmt.Errorf("nil tts pipe")
	}
	text = prepareSpeechText(text, ttsPrep)
	if text == "" {
		return "", meta, nil
	}
	t0 := time.Now()
	if err := ttsPipe.Speak(text); err != nil {
		meta.TTSMs = int(time.Since(t0).Milliseconds())
		if errors.Is(err, context.Canceled) {
			return text, meta, nil
		}
		return "", meta, err
	}
	_ = ttsPipe.Finalize()
	meta.TTSMs = int(time.Since(t0).Milliseconds())
	return text, meta, nil
}

// StreamLLMToTTS runs one LLM turn with streaming TTS output.
// onDelta is optional; invoked with accumulated assistant text during streaming.
func StreamLLMToTTS(ctx context.Context, llmProvider ChatLLM, callID, model, userText string, ttsPipe *siptts.Pipeline, lg *zap.Logger, ttsPrep func(string) string, onDelta func(accumulated string)) (string, StreamTurnTimings, error) {
	var meta StreamTurnTimings
	if llmProvider == nil {
		return "", meta, fmt.Errorf("nil llm provider")
	}
	if ttsPipe == nil {
		return "", meta, fmt.Errorf("nil tts pipe")
	}
	ttsMs := 0
	speak := func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		t0 := time.Now()
		err := ttsPipe.Speak(s)
		ttsMs += int(time.Since(t0).Milliseconds())
		return err
	}
	var full strings.Builder
	var speakErr error
	seg := siptts.NewSegmenter(siptts.PipelineSegmenterConfigFromEnv(), func(s string, _ bool) {
		if speakErr != nil {
			return
		}
		s = prepareSpeechText(stageknow.StripQuoteTags(s), ttsPrep)
		if s == "" {
			return
		}
		if err := speak(s); err != nil {
			speakErr = err
		}
	})
	streamStart := time.Now()
	gotFirst := false
	toolNames := llmProvider.ListFunctionTools()
	useTools := NeedsNonStreamToolRound(toolNames, userText)
	if useTools {
		temp := float32(0.7)
		reply, err := llmProvider.QueryWithOptions(userText, LLMQueryOptions{
			Model:           model,
			Temperature:     &temp,
			KnowledgeCallID: callID,
		})
		meta.LLMWallMs = int(time.Since(streamStart).Milliseconds())
		meta.LLMFirstMs = meta.LLMWallMs
		if err != nil {
			return "", meta, err
		}
		reply = stageknow.PrepareAssistantSpeech(callID, reply)
		reply = prepareSpeechText(reply, ttsPrep)
		if reply == "" {
			return "", meta, nil
		}
		seg.Push(reply)
		seg.Complete()
		if speakErr != nil {
			meta.TTSMs = ttsMs
			if errors.Is(speakErr, context.Canceled) {
				return reply, meta, nil
			}
			return "", meta, speakErr
		}
		_ = ttsPipe.Finalize()
		meta.TTSMs = ttsMs
		return strings.TrimSpace(reply), meta, nil
	}
	options := LLMQueryOptions{Model: model, Stream: true, KnowledgeCallID: callID}
	reply, err := llmProvider.QueryStream(userText, options, func(piece string, _ bool) error {
		if piece == "" {
			return nil
		}
		if !gotFirst {
			meta.LLMFirstMs = int(time.Since(streamStart).Milliseconds())
			gotFirst = true
		}
		full.WriteString(piece)
		if onDelta != nil {
			onDelta(full.String())
		}
		seg.Push(piece)
		return nil
	})
	meta.LLMWallMs = int(time.Since(streamStart).Milliseconds())
	if err != nil {
		ttsMs = 0
		t0 := time.Now()
		reply, err = llmProvider.QueryWithOptions(userText, LLMQueryOptions{Model: model, KnowledgeCallID: callID})
		meta.LLMWallMs = int(time.Since(t0).Milliseconds())
		meta.LLMFirstMs = meta.LLMWallMs
		if err != nil {
			return "", meta, err
		}
		reply = stageknow.PrepareAssistantSpeech(callID, reply)
		reply = prepareSpeechText(reply, ttsPrep)
		if reply == "" {
			return "", meta, nil
		}
		if err := speak(reply); err != nil {
			meta.TTSMs = ttsMs
			if errors.Is(err, context.Canceled) {
				return reply, meta, nil
			}
			return "", meta, err
		}
		_ = ttsPipe.Finalize()
		meta.TTSMs = ttsMs
		return strings.TrimSpace(reply), meta, nil
	}
	if strings.TrimSpace(reply) == "" {
		reply = full.String()
	}
	stageknow.ValidateQuotes(callID, reply, nil)
	reply = stageknow.StripQuoteTags(reply)
	seg.Complete()
	if speakErr != nil {
		meta.TTSMs = ttsMs
		if errors.Is(speakErr, context.Canceled) {
			return strings.TrimSpace(reply), meta, nil
		}
		return "", meta, speakErr
	}
	_ = ttsPipe.Finalize()
	meta.TTSMs = ttsMs
	return strings.TrimSpace(reply), meta, nil
}

func prepareSpeechText(s string, ttsPrep func(string) string) string {
	s = NormalizeTTSText(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	if ttsPrep != nil {
		s = strings.TrimSpace(ttsPrep(s))
	}
	return NormalizeTTSText(s)
}

// NormalizeTTSText removes segments TTS providers commonly reject.
func NormalizeTTSText(s string) string {
	s = siptts.SanitizeForSpeech(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 {
			continue
		}
		b.WriteRune(r)
	}
	s = strings.TrimSpace(b.String())
	if s == "" {
		return ""
	}
	onlyPunct := true
	for _, r := range s {
		if (r >= '0' && r <= '9') ||
			(r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= 0x4e00 && r <= 0x9fff) {
			onlyPunct = false
			break
		}
		switch r {
		case ' ', '\t', '\n', '\r', '，', ',', '。', '.', '！', '!', '？', '?',
			'；', ';', '：', ':', '、', '…', '-', '—', '~', '～', '“', '”', '"', '\'', '`', '(', ')', '（', '）', '[', ']', '【', '】', '<', '>', '《', '》':
		default:
			onlyPunct = false
			break
		}
	}
	if onlyPunct {
		return ""
	}
	return s
}

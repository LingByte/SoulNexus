// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package app

import (
	"fmt"
	"encoding/json"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/audio/rnnoise"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/asr"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/vad"
	voicertc "github.com/LingByte/SoulNexus/pkg/voiceserver/voice/webrtc"
)

// ============================================================================
// Process-wide runtime flags. Each cmd/voice startup calls SetX() once after
// loading config; transports read the same values via NewBargeInDetector,
// NewDenoiserOrNil, ApplyDialogReconnect, etc. so every transport gets
// identical behaviour from a single config knob.
// ============================================================================

var (
	bargeInFlag          bool
	bargeInThresholdFlag float64
	bargeInFramesFlag    int

	denoiseFlag         bool
	denoiseFallbackOnce sync.Once

	asrSentenceFilterFlag       bool
	asrSentenceFilterSimilarity float64

	recordChunkFlag time.Duration

	dialogReconnectFlag        int
	dialogReconnectBackoffFlag time.Duration

	ttsPrewarmFlag bool
)

// SetBargeIn configures the per-call VAD-based barge-in detector. Every
// transport (SIP / xiaozhi / WebRTC) consults these values through
// NewBargeInDetector — one knob, uniform behaviour.
func SetBargeIn(enabled bool, threshold float64, frames int) {
	bargeInFlag = enabled
	bargeInThresholdFlag = threshold
	bargeInFramesFlag = frames
}

// SetDenoise toggles per-call RNNoise wiring on the WebRTC ASR feed
// path. The real implementation needs librnnoise available at link-time
// (-tags rnnoise + CGO_ENABLED=1); the stub build returns ErrUnavailable
// from rnnoise.New() and we silently fall back to passthrough.
func SetDenoise(b bool) {
	denoiseFlag = b
}

// SetASRSentenceFilter enables the per-call asr.SentenceFilter installed
// by ApplyDialogReconnect. similarity is the threshold passed to
// NewSentenceFilter (0 disables the similarity check, leaving only the
// sentence-boundary buffering).
func SetASRSentenceFilter(enabled bool, similarity float64) {
	asrSentenceFilterFlag = enabled
	asrSentenceFilterSimilarity = similarity
}

// SetRecordChunk configures the rolling-recording cadence. Read by
// MakeRecorderFactory through RecordingChunkInterval(). 0 disables
// rolling uploads.
func SetRecordChunk(d time.Duration) {
	recordChunkFlag = d
}

// RecordingChunkInterval returns the rolling-chunk upload cadence
// configured via SetRecordChunk.
func RecordingChunkInterval() time.Duration { return recordChunkFlag }

// SetDialogReconnect configures the dialog-plane reconnect behaviour
// shared by every transport.
func SetDialogReconnect(retries int, initialBackoff time.Duration) {
	dialogReconnectFlag = retries
	dialogReconnectBackoffFlag = initialBackoff
}

// SetTTSPrewarm toggles the boot-time prewarm of common short replies.
// Every TTS factory consults TTSPrewarmEnabled() so a single toggle
// uniformly affects all transports.
func SetTTSPrewarm(b bool) {
	ttsPrewarmFlag = b
}

// TTSPrewarmEnabled is the read accessor consumed by NewXiaozhiFactory
// (and any other factory that wants to opt in).
func TTSPrewarmEnabled() bool { return ttsPrewarmFlag }

// PrewarmTexts returns the small set of phrases whose first-byte
// latency matters most for a smoke test. Real deployments should
// replace this with a tenant- or scenario-specific list.
func PrewarmTexts() []string {
	return []string{
		"您好，已收到",
		"听得到，您请讲。",
		"请稍等。",
		"您好，我现在没法回答这个问题，请稍后再试。",
	}
}

// ============================================================================
// Constructors used by ServerConfig fields on the xiaozhi/webrtc adapters.
// ============================================================================

// NewBargeInDetector returns a configured *vad.Detector, or nil if
// barge-in is disabled globally. Callers pass the returned value into
// gateway.ClientConfig.BargeIn — a nil detector is the documented
// "disabled" signal there, so no extra branching is needed at the call
// site.
func NewBargeInDetector() *vad.Detector {
	if !bargeInFlag {
		return nil
	}
	d := vad.NewDetector()
	if bargeInThresholdFlag > 0 {
		d.SetThreshold(bargeInThresholdFlag)
	}
	if bargeInFramesFlag > 0 {
		d.SetConsecutiveFrames(bargeInFramesFlag)
	}
	return d
}

// NewDenoiserOrNil constructs one denoiser per call, ready to wire into
// pkg/voiceserver/voice/asr.Pipeline.SetDenoiser. Returns nil when:
//   - VOICE_DENOISE is off (operator opt-out), or
//   - the rnnoise package is the stub build, so New() returned
//     ErrUnavailable. Logging the fallback once on the first call is
//     enough — repeated identical lines per call would just be noise.
//
// The returned value satisfies the voicertc.Denoiser interface
// (Process + Close) which pkg/voiceserver/audio/rnnoise.Denoiser
// implements directly.
func NewDenoiserOrNil() voicertc.Denoiser {
	if !denoiseFlag {
		return nil
	}
	d, err := rnnoise.New()
	if err != nil {
		denoiseFallbackOnce.Do(func() {
			logger.Info(fmt.Sprintf("[denoise] disabled: %v (build with -tags rnnoise to enable)", err))
		})
		return nil
	}
	return d
}

// ApplyDialogReconnect copies the per-process reconnect knobs into a
// gateway.ClientConfig. Transports call this right before NewClient so
// every transport gets identical reconnect behaviour from one set of
// configs. Idempotent — safe to call on already-populated configs.
func ApplyDialogReconnect(cfg *gateway.ClientConfig) {
	if cfg == nil {
		return
	}
	cfg.ReconnectAttempts = dialogReconnectFlag
	cfg.ReconnectInitialBackoff = dialogReconnectBackoffFlag
	cfg.HoldTextFirst = HoldMessages.First
	cfg.HoldTextRetry = HoldMessages.Retry
	cfg.HoldTextGiveUp = HoldMessages.GiveUp
	// Per-call SentenceFilter when the operator opted in. Constructed
	// fresh per-call (no shared state across calls), or nil when the
	// config toggle is off — preserving legacy per-token forwarding
	// semantics.
	if asrSentenceFilterFlag {
		cfg.ASRSentenceFilter = asr.NewSentenceFilter(asrSentenceFilterSimilarity)
	}
}

// ============================================================================
// Hold messages. JSON-loaded prompt set used during dialog reconnect.
// ============================================================================

// HoldMessageSet mirrors the JSON file under scripts/hold/. Loaded once
// at startup; falls back to hardcoded Chinese defaults if the file is
// missing or malformed (logged at warn).
type HoldMessageSet struct {
	First  string `json:"first_attempt"`
	Retry  string `json:"retry"`
	GiveUp string `json:"give_up"`
}

// HoldMessages is the process-singleton reconnect prompt set, consulted
// by ApplyDialogReconnect on every transport.
var HoldMessages = HoldMessageSet{
	First:  "请稍等。",
	Retry:  "正在重新连接，请稍候。",
	GiveUp: "抱歉，当前服务暂时不可用，请稍后再试。",
}

// LoadHoldMessages reads scripts/hold/hold_messages.json (or the path
// passed via VOICE_HOLD_MESSAGES) and overlays its values onto the
// process-singleton HoldMessages. Missing keys keep the defaults;
// missing file = no-op + log line.
func LoadHoldMessages(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Info(fmt.Sprintf("[hold] using built-in defaults (read %s: %v)", path, err))
		return
	}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		logger.Info(fmt.Sprintf("[hold] using built-in defaults (parse %s: %v)", path, err))
		return
	}
	if v, ok := raw["first_attempt"]; ok && strings.TrimSpace(v) != "" {
		HoldMessages.First = v
	}
	if v, ok := raw["retry"]; ok && strings.TrimSpace(v) != "" {
		HoldMessages.Retry = v
	}
	if v, ok := raw["give_up"]; ok && strings.TrimSpace(v) != "" {
		HoldMessages.GiveUp = v
	}
	logger.Info(fmt.Sprintf("[hold] loaded reconnect prompts from %s", path))
}

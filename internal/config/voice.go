package config

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"log"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

// VoiceServerConfig is the env-backed configuration for cmd/voice.
// It is loaded from .env-voice plus optional .env-voice.${APP_ENV|MODE}.
type VoiceServerConfig struct {
	Log logger.LogConfig

	SIPAddr      string
	LocalIP      string
	RTPStart     int
	RTPEnd       int
	OutboundURI  string
	OutboundHold time.Duration

	HTTPAddr        string
	DialogWS        string
	EnableXiaozhi   bool
	XiaozhiPath     string
	EnableWebRTC    bool
	WebRTCOfferPath string

	SoulnexusHardwarePath          string
	SoulnexusHardwareBindingURL    string
	SoulnexusHardwareBindingSecret string

	Record       bool
	RecordBucket string
	RecordChunk  time.Duration

	ASREcho   bool
	ReplyText string

	TTSPrewarm                  bool
	BargeIn                     bool
	BargeInThreshold            float64
	BargeInFrames               int
	Denoise                     bool
	ASRSentenceFilter           bool
	ASRSentenceFilterSimilarity float64

	HoldMessages           string
	DialogReconnect        int
	DialogReconnectBackoff time.Duration

	EnableSFU           bool
	SFUPath             string
	SFUSecret           string
	SFUAllowAnon        bool
	SFUMaxParticipants  int
	SFUMaxRooms         int
	SFUAllowedOrigins   string
	SFUTokenAdminSecret string
	SFURecord           bool
	SFURecordBucket     string
	SFUWebhookURL       string
}

var VoiceGlobalConfig *VoiceServerConfig

// LoadVoice loads `.env-voice` then `.env-voice.{APP_ENV|MODE}` and builds
// both GlobalConfig (for shared helpers) and VoiceGlobalConfig (cmd/voice).
func LoadVoice() error {
	if err := utils.LoadModuleEnvs("voice", resolveAppEnv()); err != nil {
		log.Printf("Note: voice module env load: %v", err)
	}
	if err := buildGlobalConfig(); err != nil {
		return err
	}
	VoiceGlobalConfig = buildVoiceConfig()
	return VoiceGlobalConfig.Validate()
}

func buildVoiceConfig() *VoiceServerConfig {
	return &VoiceServerConfig{
		Log: logger.LogConfig{
			Level:      voiceString("VOICE_LOG_LEVEL", voiceString("LOG_LEVEL", "info")),
			Filename:   voiceString("VOICE_LOG_FILENAME", "./logs/voice.log"),
			MaxSize:    voiceInt("VOICE_LOG_MAX_SIZE", 100),
			MaxAge:     voiceInt("VOICE_LOG_MAX_AGE", 30),
			MaxBackups: voiceInt("VOICE_LOG_MAX_BACKUPS", 5),
			Daily:      voiceBool("VOICE_LOG_DAILY", true),
		},

		SIPAddr:      voiceString("VOICE_SIP_ADDR", "127.0.0.1:5060"),
		LocalIP:      voiceString("VOICE_LOCAL_IP", ""),
		RTPStart:     voiceInt("VOICE_RTP_START", 30000),
		RTPEnd:       voiceInt("VOICE_RTP_END", 30100),
		OutboundURI:  voiceString("VOICE_OUTBOUND_URI", ""),
		OutboundHold: voiceDuration("VOICE_OUTBOUND_HOLD", 15*time.Second),

		HTTPAddr:        voiceString("VOICE_HTTP_ADDR", "127.0.0.1:7080"),
		DialogWS:        voiceString("VOICE_DIALOG_WS", "ws://localhost:7072/ws/call"),
		EnableXiaozhi:   voiceBool("VOICE_ENABLE_XIAOZHI", true),
		XiaozhiPath:     voiceString("VOICE_XIAOZHI_WS_PATH", "/xiaozhi/v1/"),
		EnableWebRTC:    voiceBool("VOICE_ENABLE_WEBRTC", true),
		WebRTCOfferPath: voiceString("VOICE_WEBRTC_HTTP_PATH", "/webrtc/v1/offer"),

		SoulnexusHardwarePath:          voiceString("VOICE_SOULNEXUS_HW_WS_PATH", "/voice/lingecho/v1/"),
		SoulnexusHardwareBindingURL:    voiceString("VOICE_SOULNEXUS_HW_BINDING_URL", "http://localhost:7072/api/voice/lingecho/binding"),
		SoulnexusHardwareBindingSecret: voiceString("VOICE_SOULNEXUS_HW_BINDING_SECRET", voiceString("LINGECHO_HARDWARE_BINDING_SECRET", "")),

		Record:       voiceBool("VOICE_RECORD", false),
		RecordBucket: voiceString("VOICE_RECORD_BUCKET", "voiceserver-recordings"),
		RecordChunk:  voiceDuration("VOICE_RECORD_CHUNK", 30*time.Second),

		ASREcho:   voiceBool("VOICE_ASR_ECHO", false),
		ReplyText: voiceString("VOICE_REPLY_TEXT", "您好，已收到"),

		TTSPrewarm:                  voiceBool("VOICE_TTS_PREWARM", false),
		BargeIn:                     voiceBool("VOICE_BARGE_IN", true),
		BargeInThreshold:            voiceFloat("VOICE_BARGE_IN_THRESHOLD", 1500),
		BargeInFrames:               voiceInt("VOICE_BARGE_IN_FRAMES", 1),
		Denoise:                     voiceBool("VOICE_DENOISE", false),
		ASRSentenceFilter:           voiceBool("VOICE_ASR_SENTENCE_FILTER", false),
		ASRSentenceFilterSimilarity: voiceFloat("VOICE_ASR_SENTENCE_FILTER_SIMILARITY", 0.85),

		HoldMessages:           voiceString("VOICE_HOLD_MESSAGES", "scripts/hold/hold_messages.json"),
		DialogReconnect:        voiceInt("VOICE_DIALOG_RECONNECT", 3),
		DialogReconnectBackoff: voiceDuration("VOICE_DIALOG_RECONNECT_BACKOFF", time.Second),

		EnableSFU:           voiceBool("VOICE_ENABLE_SFU", false),
		SFUPath:             voiceString("VOICE_SFU_WS_PATH", "/sfu/v1/ws"),
		SFUSecret:           voiceString("VOICE_SFU_SECRET", ""),
		SFUAllowAnon:        voiceBool("VOICE_SFU_ALLOW_ANON", false),
		SFUMaxParticipants:  voiceInt("VOICE_SFU_MAX_PARTICIPANTS", 16),
		SFUMaxRooms:         voiceInt("VOICE_SFU_MAX_ROOMS", 1024),
		SFUAllowedOrigins:   voiceString("VOICE_SFU_ALLOWED_ORIGINS", ""),
		SFUTokenAdminSecret: voiceString("VOICE_SFU_TOKEN_ADMIN_SECRET", ""),
		SFURecord:           voiceBool("VOICE_SFU_RECORD", false),
		SFURecordBucket:     voiceString("VOICE_SFU_RECORD_BUCKET", "sfu-recordings"),
		SFUWebhookURL:       voiceString("VOICE_SFU_WEBHOOK_URL", ""),
	}
}

// Validate enforces the invariants the voice server depends on at boot.
// Errors here surface as a fatal log line in cmd/voice — the process
// refuses to start rather than expose a half-working listener.
func (c *VoiceServerConfig) Validate() error {
	if c == nil {
		return errors.New("voice config is nil")
	}

	// --- SIP listener ---
	if c.SIPAddr == "" {
		return errors.New("VOICE_SIP_ADDR is required (e.g. 127.0.0.1:5060)")
	}
	if c.RTPStart <= 0 || c.RTPEnd <= 0 || c.RTPStart > c.RTPEnd {
		return errors.New("VOICE_RTP_START / VOICE_RTP_END are invalid: both > 0 and start <= end")
	}
	if c.RTPEnd-c.RTPStart < 1 {
		return errors.New("VOICE_RTP_END - VOICE_RTP_START must be >= 1 (need at least one RTP port pair)")
	}

	// --- HTTP transports gate ---
	// At least one transport must be enabled when the HTTP listener is set,
	// or the HTTP listener should be disabled (empty Addr). A listener that
	// only answers /healthz with no transports usually indicates a config typo.
	httpEnabled := c.HTTPAddr != ""
	anyTransport := c.EnableXiaozhi || c.EnableWebRTC || c.EnableSFU || c.SoulnexusHardwarePath != ""
	if httpEnabled && !anyTransport {
		return errors.New("VOICE_HTTP_ADDR is set but no transport is enabled (set VOICE_ENABLE_XIAOZHI / VOICE_ENABLE_WEBRTC / VOICE_ENABLE_SFU)")
	}

	// --- Dialog plane required for ASR/TTS-bridged transports ---
	if (c.EnableXiaozhi || c.EnableWebRTC) && c.DialogWS == "" {
		return errors.New("VOICE_DIALOG_WS is required when xiaozhi or webrtc transports are enabled")
	}

	// --- SFU consistency ---
	if c.EnableSFU && !c.SFUAllowAnon && c.SFUSecret == "" {
		return errors.New("VOICE_SFU_SECRET is required when EnableSFU=true and SFUAllowAnon=false (set VOICE_SFU_ALLOW_ANON=true for local dev)")
	}
	if c.EnableSFU && c.SFUMaxParticipants <= 0 {
		return errors.New("VOICE_SFU_MAX_PARTICIPANTS must be > 0")
	}
	if c.EnableSFU && c.SFUMaxRooms <= 0 {
		return errors.New("VOICE_SFU_MAX_ROOMS must be > 0")
	}

	// --- Hardware (legacy ESP32) consistency ---
	if c.SoulnexusHardwarePath != "" && c.SoulnexusHardwareBindingURL == "" {
		return errors.New("VOICE_SOULNEXUS_HW_BINDING_URL is required when VOICE_SOULNEXUS_HW_WS_PATH is set")
	}

	// --- Recording consistency ---
	if c.Record && c.RecordBucket == "" {
		return errors.New("VOICE_RECORD_BUCKET must be non-empty when VOICE_RECORD=true")
	}
	if c.RecordChunk < 0 {
		return errors.New("VOICE_RECORD_CHUNK must be >= 0 (use 0 to disable rolling uploads)")
	}

	// --- Dialog reconnect knobs ---
	if c.DialogReconnect < 0 {
		return errors.New("VOICE_DIALOG_RECONNECT must be >= 0")
	}
	if c.DialogReconnectBackoff < 0 {
		return errors.New("VOICE_DIALOG_RECONNECT_BACKOFF must be >= 0")
	}

	// --- Barge-in / ASR sentence filter ---
	if c.BargeIn && c.BargeInThreshold < 0 {
		return errors.New("VOICE_BARGE_IN_THRESHOLD must be >= 0")
	}
	if c.BargeIn && c.BargeInFrames <= 0 {
		return errors.New("VOICE_BARGE_IN_FRAMES must be > 0 when VOICE_BARGE_IN=true")
	}
	if c.ASRSentenceFilter && (c.ASRSentenceFilterSimilarity < 0 || c.ASRSentenceFilterSimilarity > 1) {
		return errors.New("VOICE_ASR_SENTENCE_FILTER_SIMILARITY must be in [0, 1]")
	}

	return nil
}

func voiceString(key, defaultValue string) string {
	if v := utils.GetEnv(key); v != "" {
		return v
	}
	return defaultValue
}

func voiceBool(key string, defaultValue bool) bool {
	if utils.GetEnv(key) == "" {
		return defaultValue
	}
	return utils.GetBoolEnv(key)
}

func voiceInt(key string, defaultValue int) int {
	if utils.GetEnv(key) == "" {
		return defaultValue
	}
	return int(utils.GetIntEnv(key))
}

func voiceFloat(key string, defaultValue float64) float64 {
	if utils.GetEnv(key) == "" {
		return defaultValue
	}
	return utils.GetFloatEnvWithDefault(key, defaultValue)
}

func voiceDuration(key string, defaultValue time.Duration) time.Duration {
	if v := utils.GetEnv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultValue
}

package transport

import (
	"context"
	"fmt"
	"time"

	"github.com/LingByte/lingllm/protocol/voice/asr"
	"github.com/LingByte/lingllm/protocol/voice/dialog"
	"github.com/LingByte/lingllm/protocol/voice/gateway"
	"github.com/LingByte/lingllm/protocol/voice/tts"
)

// SessionFactory mints per-call ASR engine and TTS service instances.
type SessionFactory interface {
	NewASR(ctx context.Context, callID string) (engine asr.Engine, sampleRate int, err error)
	NewTTS(ctx context.Context, callID string) (svc tts.TTSService, sampleRate int, err error)
}

// CallConfig wires dialog.Session + gateway.Client for one media call.
type CallConfig struct {
	CallID     string
	DialogURL  string
	Meta       dialog.StartMeta
	Factory    SessionFactory
	OnAudioOut func([]byte) error

	InputCodec    string
	OutputCodec   string
	PCMSampleRate int
	Denoiser      asr.Denoiser
	TTSCache      *tts.CacheConfig

	Gateway gateway.ClientConfig

	OnHangup     func(reason string)
	OnTurn       func(dialog.TurnEvent)
	OnFirstAudio func(dialog.FirstAudioEvent)

	// PaceRealtime paces TTS frames at wall-clock rate (recommended for WebRTC/RTP).
	PaceRealtime *bool
	// TTSFrameDuration sets downlink PCM frame size (default 20ms for WebRTC).
	TTSFrameDuration time.Duration
}

// NewCall builds a dialog session and dialog-plane gateway client.
func NewCall(ctx context.Context, cfg CallConfig) (*dialog.Session, *gateway.Client, error) {
	if cfg.Factory == nil {
		return nil, nil, fmt.Errorf("transport: nil SessionFactory")
	}
	if cfg.OnAudioOut == nil {
		return nil, nil, fmt.Errorf("transport: nil OnAudioOut")
	}
	if cfg.CallID == "" {
		return nil, nil, fmt.Errorf("transport: empty call_id")
	}

	engine, _, err := cfg.Factory.NewASR(ctx, cfg.CallID)
	if err != nil {
		return nil, nil, fmt.Errorf("transport: asr: %w", err)
	}
	ttsSvc, _, err := cfg.Factory.NewTTS(ctx, cfg.CallID)
	if err != nil {
		return nil, nil, fmt.Errorf("transport: tts: %w", err)
	}

	gwCfg := cfg.Gateway
	gwCfg.URL = cfg.DialogURL
	gwCfg.CallID = cfg.CallID
	if gwCfg.OnHangup == nil {
		gwCfg.OnHangup = cfg.OnHangup
	}
	if gwCfg.OnTurn == nil {
		gwCfg.OnTurn = cfg.OnTurn
	}

	gw, err := gateway.NewClient(gwCfg)
	if err != nil {
		return nil, nil, err
	}

	sr := cfg.PCMSampleRate
	if sr <= 0 {
		sr = 16000
	}

	onTurn := cfg.OnTurn
	if gwCfg.OnTurn != nil {
		onTurn = gwCfg.OnTurn
	}

	frameDur := cfg.TTSFrameDuration
	if frameDur <= 0 {
		frameDur = 20 * time.Millisecond
	}

	sess, err := dialog.NewSession(ctx, dialog.Config{
		CallID:           cfg.CallID,
		Meta:             cfg.Meta,
		Engine:           engine,
		TTSService:       ttsSvc,
		TTSCache:         cfg.TTSCache,
		Denoiser:         cfg.Denoiser,
		InputCodec:       cfg.InputCodec,
		OutputCodec:      cfg.OutputCodec,
		PCMSampleRate:    sr,
		PaceRealtime:     cfg.PaceRealtime,
		TTSFrameDuration: frameDur,
		OnAudioOut:       cfg.OnAudioOut,
		OnEvent: func(ev dialog.Event) {
			_ = gw.SendEvent(ev)
		},
		OnHangup:     cfg.OnHangup,
		OnTurn:       onTurn,
		OnFirstAudio: cfg.OnFirstAudio,
	})
	if err != nil {
		return nil, nil, err
	}
	gw.Bind(sess)
	return sess, gw, nil
}

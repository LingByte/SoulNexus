package voice

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/asr"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"
	"go.uber.org/zap"
)

// AttachConfig wires an ASR pipeline and a TTS pipeline to a SIP MediaLeg in
// one call. Either side can be left nil.
type AttachConfig struct {
	// ASR
	ASR            recognizer.TranscribeService
	ASRSampleRate  int           // e.g. 16000 or 8000
	OnTranscript   asr.TextCallback
	OnASRError     asr.ErrorCallback
	DialogID       string

	// TTS
	TTSService       tts.Service
	TTSInputRate     int  // e.g. 16000 (the rate the service emits)
	TTSPaceRealtime  bool // true for RTP realtime playback

	// Shared
	Logger *zap.Logger
}

// Attached is the live wiring produced by Attach. Call Close to tear down.
type Attached struct {
	ASR *asr.Pipeline
	TTS *tts.Pipeline

	stopTTS func()
	closed  atomic.Bool
}

// Close stops both pipelines. Idempotent.
func (a *Attached) Close() {
	if a == nil || a.closed.Swap(true) {
		return
	}
	if a.ASR != nil {
		_ = a.ASR.Close()
	}
	if a.stopTTS != nil {
		a.stopTTS()
	}
}

// Attach connects cfg.ASR and cfg.TTSService to a MediaLeg. It:
//
//   - installs an input filter that feeds decoded PCM into the ASR Pipeline
//     (inbound RTP → PCM16 → recognizer)
//   - creates a TTS Pipeline whose Sink pushes PCM frames into the MediaLeg
//     output queue (text → PCM16 → outbound RTP)
//
// Callers then use the returned Pipelines:
//
//   att := voice.Attach(ctx, leg, cfg)
//   defer att.Close()
//   leg.Start()
//   _ = att.TTS.Speak("hello")
func Attach(ctx context.Context, leg *session.MediaLeg, cfg AttachConfig) (*Attached, error) {
	if leg == nil {
		return nil, fmt.Errorf("voice.Attach: nil MediaLeg")
	}
	ms := leg.MediaSession()
	if ms == nil {
		return nil, fmt.Errorf("voice.Attach: MediaLeg has no MediaSession")
	}
	bridgeSR := leg.PCMSampleRate()
	log := cfg.Logger
	if log == nil {
		log = zap.NewNop()
	}

	out := &Attached{}

	// ---- ASR side ----
	if cfg.ASR != nil {
		if cfg.ASRSampleRate <= 0 {
			cfg.ASRSampleRate = bridgeSR
		}
		p, err := asr.New(asr.Options{
			ASR:             cfg.ASR,
			InputSampleRate: bridgeSR,
			SampleRate:      cfg.ASRSampleRate,
			Channels:        1,
			DialogID:        cfg.DialogID,
			Logger:          log,
		})
		if err != nil {
			return nil, err
		}
		p.SetTextCallback(cfg.OnTranscript)
		p.SetErrorCallback(cfg.OnASRError)

		proc := media.NewPacketProcessor("voice-asr-feed", media.PriorityHigh,
			func(c context.Context, _ *media.MediaSession, packet media.MediaPacket) error {
				ap, ok := packet.(*media.AudioPacket)
				if !ok || ap == nil || len(ap.Payload) == 0 || ap.IsSynthesized {
					return nil
				}
				return p.ProcessPCM(c, ap.Payload)
			})
		ms.RegisterProcessor(proc)
		out.ASR = p
	}

	// ---- TTS side ----
	if cfg.TTSService != nil {
		if cfg.TTSInputRate <= 0 {
			cfg.TTSInputRate = bridgeSR
		}
		pipe, err := tts.New(tts.Config{
			Service:          cfg.TTSService,
			InputSampleRate:  cfg.TTSInputRate,
			OutputSampleRate: bridgeSR,
			Channels:         1,
			PaceRealtime:     cfg.TTSPaceRealtime,
			Sink: func(frame []byte) error {
				ms.SendToOutput("voice-tts", &media.AudioPacket{
					Payload:       frame,
					IsSynthesized: true,
				})
				return nil
			},
			Logger: log,
		})
		if err != nil {
			if out.ASR != nil {
				_ = out.ASR.Close()
			}
			return nil, err
		}
		pipe.Start(ctx)
		out.TTS = pipe
		out.stopTTS = pipe.Stop
	}

	return out, nil
}

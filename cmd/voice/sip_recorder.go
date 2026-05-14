// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"fmt"
	"context"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/app"
	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/recorder"
)

type pcmRecorder struct {
	callID    string
	codec     string
	persister *app.CallPersister

	mu  sync.Mutex
	rec *recorder.Recorder // late-init in setPCMRate
}

func newRTPRecorder(callID string) *pcmRecorder {
	return &pcmRecorder{callID: callID}
}

// setCodec stamps the negotiated wire codec for the flush log line. The
// recording itself is always PCM16 LE at the bridge rate, so codec is
// purely informational here.
func (r *pcmRecorder) setCodec(name string, _ int) {
	r.mu.Lock()
	r.codec = name
	if r.rec != nil {
		// Recorder.Codec is read at flush time; safe to leave unset
		// when the recorder was already constructed (its log line
		// will simply not include the codec name).
	}
	r.mu.Unlock()
}

// setPCMRate latches the bridge rate and constructs the underlying
// recorder. Idempotent — called once per call from the InviteHandler
// after MediaLeg is built.
func (r *pcmRecorder) setPCMRate(rate int) {
	if rate <= 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rec != nil {
		return
	}
	r.rec = recorder.New(recorder.Config{
		CallID:     r.callID,
		SampleRate: rate,
		Transport:  "sip",
		Codec:      r.codec,
	})
}

// inFilter is a MediaLeg InputFilter: caller-side decoded PCM16 mono.
// Returns (false, nil) — the recorder never drops packets.
func (r *pcmRecorder) inFilter(p media.MediaPacket) (bool, error) {
	if rec := r.recorder(); rec != nil {
		if ap, ok := p.(*media.AudioPacket); ok && ap != nil && len(ap.Payload) > 0 && !ap.IsSynthesized {
			rec.WriteCaller(ap.Payload)
		}
	}
	return false, nil
}

// outFilter is a MediaLeg OutputFilter: AI PCM16 mono before encode.
// Only synthesized (TTS / playback) frames are recorded.
func (r *pcmRecorder) outFilter(p media.MediaPacket) (bool, error) {
	if rec := r.recorder(); rec != nil {
		if ap, ok := p.(*media.AudioPacket); ok && ap != nil && len(ap.Payload) > 0 && ap.IsSynthesized {
			rec.WriteAI(ap.Payload)
		}
	}
	return false, nil
}

// flush builds and writes the stereo WAV once per call. Stamps both the
// legacy SIPCall.recording_url field and a structured call_recording
// row. Idempotent — second call no-ops.
func (r *pcmRecorder) flush() {
	rec := r.recorder()
	if rec == nil {
		return
	}
	info, ok := rec.Flush(context.Background())
	if !ok {
		logger.Info(fmt.Sprintf("[record] call=%s empty (no inbound/outbound PCM) or upload failed, skip", r.callID))
		return
	}
	logger.Info(fmt.Sprintf("[record] call=%s wrote %d bytes stereo WAV → %s",
		r.callID, info.Bytes, info.URL))
	if r.persister == nil {
		return
	}
	r.persister.OnRecording(context.Background(), info.URL, info.Bytes)
	r.persister.AppendRecording(context.Background(), info)
}

func (r *pcmRecorder) recorder() *recorder.Recorder {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rec
}

// ---------- DTMFSink --------------------------------------------------------

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/hraban/opus"
	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

// recordingSession captures one publisher's Opus audio track to a mono
// 48 kHz WAV file. One session per published audio track. Video is not
// recorded today (would require a VP8 → frame container that we don't
// ship; users wanting MP4 should run a downstream Composer subscribing
// like a normal participant).
//
// Lifecycle:
//
//	NewRecording → Push (per RTP packet) … → Close (uploads + emits
//	EventRecordingFinished webhook).
//
// Concurrency: Push is called from one goroutine (the layer pump in
// SimulcastForwarder). Close may be called from any goroutine and is
// idempotent.
type recordingSession struct {
	cfg           *Config
	logger        *zap.Logger
	emitter       *webhookEmitter
	roomName      string
	participantID string
	identity      string
	trackID       string

	dec *opus.Decoder
	// pcm accumulates 16-bit LE mono PCM at 48 kHz. Buffered in memory
	// for the duration of the call; that's fine for typical audio sizes
	// (1 hour mono 48 kHz ≈ 345 MB raw — if that ever becomes a real
	// problem, switch to streamed chunked upload like pkg/voice/recorder).
	mu        sync.Mutex
	pcm       bytes.Buffer
	startedAt time.Time
	closed    bool
}

// const sample rate / channels are fixed by the codec we register in
// engine.go (Opus@48kHz). Channels=1 because we always downmix.
const (
	recSampleRate = 48000
	recChannels   = 1
	// maxOpusFrameSamples — RFC 6716 caps a single Opus frame at 120 ms
	// per channel, so the decode buffer needs that much headroom.
	maxOpusFrameSamples = recSampleRate * 120 / 1000
)

// newRecordingSession allocates the Opus decoder and pre-sizes the PCM
// buffer. Returns nil if recording is disabled or initialisation
// fails — the caller must treat nil as "skip recording, continue
// forwarding".
func newRecordingSession(cfg *Config, logger *zap.Logger, emitter *webhookEmitter, room, participantID, identity, trackID string) *recordingSession {
	if !cfg.EnableRecording {
		return nil
	}
	dec, err := opus.NewDecoder(recSampleRate, recChannels)
	if err != nil {
		logger.Warn("sfu: opus decoder", zap.Error(err))
		return nil
	}
	return &recordingSession{
		cfg:           cfg,
		logger:        logger.With(zap.String("track", trackID)),
		emitter:       emitter,
		roomName:      room,
		participantID: participantID,
		identity:      identity,
		trackID:       trackID,
		dec:           dec,
		startedAt:     time.Now(),
	}
}

// Push decodes one Opus payload and appends the resulting PCM to the
// in-memory buffer. Errors are logged and skipped — a single malformed
// Opus frame should not abort the recording.
func (r *recordingSession) Push(payload []byte) {
	if r == nil || len(payload) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	out := make([]int16, maxOpusFrameSamples)
	n, err := r.dec.Decode(payload, out)
	if err != nil {
		// Decode errors are usually a single corrupt packet (network
		// loss, NACK miss). Logging at debug because a noisy network
		// would flood the warn channel.
		r.logger.Debug("opus decode skip", zap.Error(err))
		return
	}
	// PCM16 LE serialisation — directly appending 2*n bytes is faster
	// than calling binary.Write per sample.
	buf := make([]byte, n*2)
	for i := 0; i < n; i++ {
		v := uint16(out[i])
		buf[i*2] = byte(v)
		buf[i*2+1] = byte(v >> 8)
	}
	r.pcm.Write(buf)
}

// Close finalises the WAV, uploads to the configured store, and emits
// the recording.finished webhook. Idempotent.
func (r *recordingSession) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	pcm := r.pcm.Bytes()
	startedAt := r.startedAt
	r.mu.Unlock()

	if len(pcm) == 0 {
		return
	}

	wav := wrapWAVRecording(pcm, recSampleRate, recChannels)
	store := stores.Default()
	if store == nil {
		r.logger.Warn("sfu: default store unavailable; recording dropped")
		return
	}
	key := fmt.Sprintf("%s/%s-%s-%d.wav",
		sanitiseSegment(r.roomName),
		sanitiseSegment(r.identity),
		sanitiseSegment(r.trackID),
		startedAt.Unix(),
	)
	// store.Store.Write doesn't take a context today; upload time is
	// bounded by whatever HTTP client the backend uses internally.
	if err := store.Write(key, bytes.NewReader(wav)); err != nil {
		r.logger.Warn("sfu: recording upload", zap.Error(err))
		return
	}
	url := store.PublicURL(key)
	duration := time.Duration(len(pcm)/2) * time.Second / time.Duration(recSampleRate)
	r.logger.Info("sfu: recording uploaded",
		zap.String("url", url),
		zap.Duration("duration", duration))
	r.emitter.emit(Event{
		Type:          EventRecordingFinished,
		Room:          r.roomName,
		ParticipantID: r.participantID,
		Identity:      r.identity,
		TrackID:       r.trackID,
		TrackKind:     "audio",
		RecordingURL:  url,
		DurationMs:    duration.Milliseconds(),
		Timestamp:     time.Now().UnixMilli(),
	})
}

// sinkPayload returns a callback the SimulcastForwarder invokes for
// every packet forwarded on the active layer. nil-safe — callers can
// pass the result directly to forwarder.SetPayloadSink without an
// outer nil check.
func (r *recordingSession) sinkPayload() func(payload []byte) {
	if r == nil {
		return nil
	}
	return r.Push
}

// wrapWAVRecording builds a canonical RIFF/WAVE PCM16 container around
// the given little-endian PCM bytes. Duplicates the helper in
// pkg/voice/recorder/recorder.go (which is unexported); kept local to
// avoid creating cross-package coupling for one 30-line function.
func wrapWAVRecording(pcm []byte, sampleRate, channels int) []byte {
	const (
		bitsPerSample = 16
		fmtChunkSize  = 16
	)
	dataLen := uint32(len(pcm))
	byteRate := uint32(sampleRate * channels * bitsPerSample / 8)
	blockAlign := uint16(channels * bitsPerSample / 8)
	totalLen := 36 + dataLen
	buf := make([]byte, 0, 44+len(pcm))
	buf = append(buf, []byte("RIFF")...)
	buf = appendU32LE(buf, totalLen)
	buf = append(buf, []byte("WAVE")...)
	buf = append(buf, []byte("fmt ")...)
	buf = appendU32LE(buf, fmtChunkSize)
	buf = appendU16LE(buf, 1) // PCM format
	buf = appendU16LE(buf, uint16(channels))
	buf = appendU32LE(buf, uint32(sampleRate))
	buf = appendU32LE(buf, byteRate)
	buf = appendU16LE(buf, blockAlign)
	buf = appendU16LE(buf, bitsPerSample)
	buf = append(buf, []byte("data")...)
	buf = appendU32LE(buf, dataLen)
	buf = append(buf, pcm...)
	return buf
}

func appendU16LE(buf []byte, v uint16) []byte {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	return append(buf, b[:]...)
}

func appendU32LE(buf []byte, v uint32) []byte {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	return append(buf, b[:]...)
}

func sanitiseSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}
	out := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, s)
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}

// pionKindAudio is a tiny constant alias to keep callers from importing
// the pion package just to compare against "audio".
var pionKindAudio = webrtc.RTPCodecTypeAudio.String()

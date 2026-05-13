// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package recorder produces a uniform stereo-WAV recording for any
// VoiceServer transport (SIP / xiaozhi / WebRTC). The output format is
// identical across transports so dashboards and offline review tools
// can treat every recording the same:
//
//   - PCM16 LE
//   - 2 channels, L = caller / device / browser, R = AI / TTS
//   - Sample rate = the call's PCM bridge rate (typically 16 kHz)
//   - Wall-clock-aligned: L and R share one absolute time axis so a
//     listener hears events in the order they actually happened, even
//     when frames arrive in bursts due to scheduler / network latency.
//
// Wall-clock alignment is the trick borrowed from LingEchoX's
// `placeWallPCMTrack`: each frame is captured with `time.Now().UnixNano()`
// at write time. At flush time we pick `base = min(wallNs)` across both
// legs as the timeline origin, lay every frame at
// `pos = (wallNs - base) * rate / 1e9`, with a "pen" cursor preventing
// any single leg from overlapping itself during a burst. Gaps are
// zero-padded.
//
// Recorder is goroutine-safe and codec-agnostic — callers feed PCM16 LE
// mono after they have already decoded whatever wire codec they use
// (PCMA/PCMU/G.722/Opus all funnel into the same path).
package recorder

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
	"go.uber.org/zap"
)

// Config configures one Recorder. CallID is required; Bucket defaults
// to "voiceserver-recordings". SampleRate must match the PCM bridge
// rate — every WriteCaller / WriteAI byte stream is treated as PCM16 LE
// mono at this rate.
//
// Store is the destination for the WAV blob at flush time; nil falls
// through to stores.Default() (the process-wide upload backend used by
// the rest of VoiceServer). Tests inject a mock Store here to capture
// the bytes without touching disk or hitting the network.
type Config struct {
	CallID     string // unique call identifier; embedded in the storage key
	SampleRate int    // PCM bridge rate; e.g. 8000 / 16000 / 48000
	Transport  string // free-form label for log lines: "sip" / "xiaozhi" / "webrtc"
	Codec      string // wire codec (informational, used in flush log line)
	Store      stores.Store
	Logger     *zap.Logger

	// ChunkInterval, when > 0, enables rolling partial uploads every
	// interval seconds. Each tick uploads a stereo WAV containing the
	// PCM frames captured since the previous tick to a key like
	// "<callid>-part-<seq>-<ts>.wav". The final Flush still writes a
	// complete <callid>-<ts>.wav containing the entire call, so the
	// chunks are a crash-safety net rather than the canonical
	// recording. 0 = no rolling uploads (original behaviour).
	//
	// Typical setting: 30s. Smaller = more S3 PUTs but better
	// recovery granularity; larger = cheaper but more lost on crash.
	ChunkInterval time.Duration
}

// Recorder captures user and AI PCM streams and produces a stereo WAV.
type Recorder struct {
	cfg Config
	log *zap.Logger

	mu      sync.Mutex
	inSegs  []frame // L = caller / device / browser
	outSegs []frame // R = AI / TTS
	flushed bool

	// Rolling chunk state. inHead / outHead point at the index up to
	// which the chunker has already uploaded; the final Flush still
	// uploads the FULL frame slices, so chunks are a safety net only.
	inHead  int
	outHead int
	partSeq int
	chunkCh chan struct{} // closed by Flush to stop the chunker goroutine

	// chunkKeys remembers every part-*.wav we successfully uploaded so
	// Flush can delete them after the canonical full WAV write
	// succeeds. Crash before Flush → chunks survive in the bucket as
	// recovery data; Flush success → chunks are reclaimed and the
	// bucket carries exactly one row per call.
	chunkKeys []string
}

// frame is one PCM16 LE mono chunk tagged with its arrival wall clock.
// The actual PCM bytes are owned by the recorder (callers MUST NOT
// reuse the slice they pass to WriteCaller / WriteAI).
type frame struct {
	wallNs int64
	pcm    []byte
}

// New validates cfg and returns a ready-to-use Recorder. Returns nil on
// invalid config (callers should treat nil Recorder as "recording
// disabled" and skip without error).
func New(cfg Config) *Recorder {
	if strings.TrimSpace(cfg.CallID) == "" || cfg.SampleRate <= 0 {
		return nil
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	r := &Recorder{
		cfg:     cfg,
		log:     cfg.Logger,
		inSegs:  make([]frame, 0, 256),
		outSegs: make([]frame, 0, 256),
	}
	if cfg.ChunkInterval > 0 {
		r.chunkCh = make(chan struct{})
		go r.chunkLoop()
	}
	return r
}

// chunkLoop is the rolling-upload ticker. It runs until Flush() closes
// chunkCh; each tick uploads the new frames captured since the last
// tick as a partial WAV. We don't truncate the in-memory frames — the
// final Flush still produces the complete recording — so the chunks
// are pure crash-recovery breadcrumbs.
func (r *Recorder) chunkLoop() {
	t := time.NewTicker(r.cfg.ChunkInterval)
	defer t.Stop()
	for {
		select {
		case <-r.chunkCh:
			return
		case <-t.C:
			r.uploadNextChunk()
		}
	}
}

// uploadNextChunk snapshots the new frames since the last tick and
// uploads a partial stereo WAV under "<callid>-part-<seq>-<ts>.wav".
// Best-effort: errors are logged and the head pointers stay where they
// were so the next tick will retry the same range. This means a flaky
// store causes duplicate uploads on the next attempt, not data loss.
func (r *Recorder) uploadNextChunk() {
	r.mu.Lock()
	if r.flushed {
		r.mu.Unlock()
		return
	}
	in := r.inSegs[r.inHead:]
	out := r.outSegs[r.outHead:]
	if len(in) == 0 && len(out) == 0 {
		r.mu.Unlock()
		return
	}
	// Snapshot indices we'll commit if upload succeeds. Stash
	// references rather than copies — frames are immutable once
	// appended (we only ever append, never mutate), so reading them
	// outside the lock is safe.
	inEnd := len(r.inSegs)
	outEnd := len(r.outSegs)
	seq := r.partSeq + 1
	r.mu.Unlock()

	wav := r.buildStereoWAV(in, out)
	if len(wav) == 0 {
		return
	}
	store := r.cfg.Store
	if store == nil {
		store = stores.Default()
	}
	if store == nil {
		// No store available; chunker silently no-ops, the final
		// Flush still has the data in memory.
		return
	}
	ts := time.Now().Unix()
	key := fmt.Sprintf("%s-part-%d-%d.wav", sanitizeFilename(r.cfg.CallID), seq, ts)
	if err := store.Write(key, bytes.NewReader(wav)); err != nil {
		r.log.Warn("recorder: chunk upload failed",
			zap.String("call_id", r.cfg.CallID),
			zap.Int("seq", seq),
			zap.Error(err))
		return
	}
	r.mu.Lock()
	r.inHead = inEnd
	r.outHead = outEnd
	r.partSeq = seq
	r.chunkKeys = append(r.chunkKeys, key)
	r.mu.Unlock()
	r.log.Info("recorder: chunk uploaded",
		zap.String("call_id", r.cfg.CallID),
		zap.Int("seq", seq),
		zap.String("key", key),
		zap.Int("bytes", len(wav)))
}

// buildStereoWAV builds a complete WAV blob from the supplied frames
// using the same wall-clock placement logic as Flush. Used by both the
// final Flush and the rolling chunker so chunk WAVs are byte-shaped
// identical to the final recording.
func (r *Recorder) buildStereoWAV(inSegs, outSegs []frame) []byte {
	if len(inSegs) == 0 && len(outSegs) == 0 {
		return nil
	}
	// Per-WAV base = min wallNs across just THIS WAV's frames so each
	// chunk is independently playable from t=0.
	base := int64(0)
	first := true
	for _, s := range inSegs {
		if first || s.wallNs < base {
			base = s.wallNs
			first = false
		}
	}
	for _, s := range outSegs {
		if first || s.wallNs < base {
			base = s.wallNs
			first = false
		}
	}
	left := placePCMTrackBytes(inSegs, base, r.cfg.SampleRate)
	right := placePCMTrackBytes(outSegs, base, r.cfg.SampleRate)
	stereo := interleaveStereoBytes(left, right)
	return wrapWAV(stereo, r.cfg.SampleRate, 2)
}

// WriteCaller appends a caller-side PCM16 LE mono chunk. nil-safe.
// The recorder copies the slice — callers may reuse the buffer.
func (r *Recorder) WriteCaller(pcm []byte) {
	r.append(true, pcm)
}

// WriteAI appends an AI / TTS-side PCM16 LE mono chunk. nil-safe.
func (r *Recorder) WriteAI(pcm []byte) {
	r.append(false, pcm)
}

func (r *Recorder) append(caller bool, pcm []byte) {
	if r == nil || len(pcm) == 0 {
		return
	}
	buf := append([]byte(nil), pcm...)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.flushed {
		return
	}
	f := frame{wallNs: time.Now().UnixNano(), pcm: buf}
	if caller {
		r.inSegs = append(r.inSegs, f)
	} else {
		r.outSegs = append(r.outSegs, f)
	}
}

// Flush builds the stereo WAV, uploads it via stores.Default(), and
// returns a RecordingInfo suitable for passing to a SessionPersister.
// Returns ok=false when there is nothing to record (no frames captured)
// or when the upload failed; the error is logged either way.
//
// Idempotent: a second Flush returns (zero, false) without re-uploading.
func (r *Recorder) Flush(ctx context.Context) (gateway.RecordingInfo, bool) {
	if r == nil {
		return gateway.RecordingInfo{}, false
	}
	r.mu.Lock()
	if r.flushed {
		r.mu.Unlock()
		return gateway.RecordingInfo{}, false
	}
	r.flushed = true
	// Signal the chunker to stop BEFORE we null the slices, so a
	// concurrent uploadNextChunk that has already entered its critical
	// section just sees flushed=true and returns harmlessly.
	if r.chunkCh != nil {
		close(r.chunkCh)
		r.chunkCh = nil
	}
	inSegs := r.inSegs
	outSegs := r.outSegs
	r.inSegs = nil
	r.outSegs = nil
	r.mu.Unlock()

	rate := r.cfg.SampleRate
	if len(inSegs) == 0 && len(outSegs) == 0 {
		r.log.Info("recorder: no frames captured, skip",
			zap.String("call_id", r.cfg.CallID))
		return gateway.RecordingInfo{}, false
	}

	// Shared base wall-clock = min wallNs across both legs.
	base := int64(0)
	first := true
	for _, s := range inSegs {
		if first || s.wallNs < base {
			base = s.wallNs
			first = false
		}
	}
	for _, s := range outSegs {
		if first || s.wallNs < base {
			base = s.wallNs
			first = false
		}
	}

	left := placePCMTrackBytes(inSegs, base, rate)
	right := placePCMTrackBytes(outSegs, base, rate)
	stereo := interleaveStereoBytes(left, right)
	wav := wrapWAV(stereo, rate, 2)

	ts := time.Now().Unix()
	key := fmt.Sprintf("%s-%d.wav", sanitizeFilename(r.cfg.CallID), ts)
	store := r.cfg.Store
	if store == nil {
		store = stores.Default()
	}
	if store == nil {
		r.log.Warn("recorder: no Store available, skip upload",
			zap.String("call_id", r.cfg.CallID))
		return gateway.RecordingInfo{}, false
	}
	if err := store.Write(key, bytes.NewReader(wav)); err != nil {
		r.log.Warn("recorder: store write failed",
			zap.String("call_id", r.cfg.CallID),
			zap.Error(err))
		return gateway.RecordingInfo{}, false
	}

	// Final WAV is durable; reclaim every part-*.wav we'd uploaded as
	// a crash-safety net so the bucket carries exactly one row per
	// call. Best-effort: delete failures are logged but never demote
	// the recording's success — the canonical full WAV is what
	// dashboards/persistence rows reference.
	r.mu.Lock()
	parts := r.chunkKeys
	r.chunkKeys = nil
	r.mu.Unlock()
	if len(parts) > 0 {
		var failed int
		for _, k := range parts {
			if err := store.Delete(k); err != nil {
				failed++
				r.log.Warn("recorder: chunk delete failed",
					zap.String("call_id", r.cfg.CallID),
					zap.String("key", k),
					zap.Error(err))
			}
		}
		r.log.Info("recorder: chunks reclaimed",
			zap.String("call_id", r.cfg.CallID),
			zap.Int("ok", len(parts)-failed),
			zap.Int("failed", failed))
	}

	durationMs := int64(0)
	if rate > 0 && len(stereo) > 0 {
		// stereo: PCM16 LE 2ch → 4 bytes per sample-frame.
		durationMs = int64(len(stereo)/4) * 1000 / int64(rate)
	}
	r.log.Info("recorder: wav written",
		zap.String("call_id", r.cfg.CallID),
		zap.String("transport", r.cfg.Transport),
		zap.String("codec", r.cfg.Codec),
		zap.Int("rate", rate),
		zap.Int("bytes", len(wav)),
		zap.Int("in_frames", len(inSegs)),
		zap.Int("out_frames", len(outSegs)),
		zap.String("key", key))

	return gateway.RecordingInfo{
		Key:        key,
		URL:        store.PublicURL(key),
		Format:     "wav",
		Layout:     "stereo-l-r", // matches persist.RecordingLayoutStereoLR
		SampleRate: rate,
		Channels:   2,
		Bytes:      len(wav),
		DurationMs: durationMs,
	}, true
}

// ---------- internals: PCM placement / WAV wrapping ----------------------

// placePCMTrackBytes lays PCM frames on a wall-clock timeline anchored
// to baseNs. Each frame is positioned at
// pos = (wallNs - base) * rate / 1e9, with a "pen" guard preventing
// frames from overlapping during a burst. Gaps are zero-padded.
//
// Returns one PCM16 LE mono buffer.
func placePCMTrackBytes(segs []frame, baseNs int64, rate int) []byte {
	if len(segs) == 0 || rate <= 0 {
		return nil
	}
	// Sort by wallNs ASC. n is small (hundreds typically), insertion
	// sort would be fine; stdlib sort is plenty.
	sortFramesByWall(segs)

	// Pen advances forward through the timeline so successive bursts
	// don't overlap. Position is in samples (PCM16 = 2 bytes/sample).
	out := make([]byte, 0, 4096)
	pen := int64(0)
	for _, s := range segs {
		if len(s.pcm) == 0 {
			continue
		}
		samples := int64(len(s.pcm) / 2)
		pos := (s.wallNs - baseNs) * int64(rate) / 1_000_000_000
		if pos < pen {
			pos = pen
		}
		// Pad up to pos with zeros.
		curSamples := int64(len(out) / 2)
		if pos > curSamples {
			out = append(out, make([]byte, (pos-curSamples)*2)...)
		}
		// Append frame.
		out = append(out, s.pcm...)
		pen = pos + samples
	}
	return out
}

// sortFramesByWall is a tiny insertion sort to avoid pulling in the
// sort package for what is almost always near-sorted input (frames
// arrive nearly in time order; only burst-induced reorders shuffle).
func sortFramesByWall(segs []frame) {
	for i := 1; i < len(segs); i++ {
		j := i
		for j > 0 && segs[j-1].wallNs > segs[j].wallNs {
			segs[j-1], segs[j] = segs[j], segs[j-1]
			j--
		}
	}
}

// interleaveStereoBytes merges two PCM16 LE mono buffers into a single
// LR-interleaved stereo PCM16 LE buffer. The shorter side is zero-
// padded so total sample counts match.
func interleaveStereoBytes(left, right []byte) []byte {
	if len(left)%2 != 0 {
		left = left[:len(left)-1]
	}
	if len(right)%2 != 0 {
		right = right[:len(right)-1]
	}
	leftSamples := len(left) / 2
	rightSamples := len(right) / 2
	n := leftSamples
	if rightSamples > n {
		n = rightSamples
	}
	out := make([]byte, n*4) // 2ch × 2 bytes
	for i := 0; i < n; i++ {
		var b0, b1, b2, b3 byte
		if i < leftSamples {
			b0 = left[i*2]
			b1 = left[i*2+1]
		}
		if i < rightSamples {
			b2 = right[i*2]
			b3 = right[i*2+1]
		}
		out[i*4] = b0
		out[i*4+1] = b1
		out[i*4+2] = b2
		out[i*4+3] = b3
	}
	return out
}

// wrapWAV builds a canonical RIFF/WAVE container (PCM16) around the
// little-endian PCM samples. channels=1 (mono) or 2 (LR-interleaved).
func wrapWAV(pcm []byte, sampleRate, channels int) []byte {
	const (
		bitsPerSample = 16
		fmtChunkSize  = 16
		audioFormat   = 1 // PCM
	)
	if channels < 1 {
		channels = 1
	}
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8
	dataSize := len(pcm)
	totalSize := 4 + (8 + fmtChunkSize) + (8 + dataSize)

	buf := make([]byte, 0, 8+totalSize)
	buf = append(buf, "RIFF"...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(totalSize))
	buf = append(buf, "WAVE"...)
	buf = append(buf, "fmt "...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(fmtChunkSize))
	buf = binary.LittleEndian.AppendUint16(buf, audioFormat)
	buf = binary.LittleEndian.AppendUint16(buf, uint16(channels))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(sampleRate))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(byteRate))
	buf = binary.LittleEndian.AppendUint16(buf, uint16(blockAlign))
	buf = binary.LittleEndian.AppendUint16(buf, uint16(bitsPerSample))
	buf = append(buf, "data"...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(dataSize))
	buf = append(buf, pcm...)
	return buf
}

func sanitizeFilename(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_' || r == '.':
			return r
		default:
			return '_'
		}
	}, s)
}

// ErrNotConfigured is returned if Recorder methods are called on a nil
// receiver via reflection. Direct callers won't see this — Go's nil
// receiver pattern means WriteCaller/WriteAI/Flush all no-op on nil.
var ErrNotConfigured = errors.New("recorder: not configured")

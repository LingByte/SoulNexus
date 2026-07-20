// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package recorder produces a uniform stereo-WAV recording for any
// VoiceServer transport (websocket / xiaozhi / WebRTC). The output format is
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
// Wall-clock alignment is the trick borrowed from SoulNexus's
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
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/voice/gateway"
	"go.uber.org/zap"
)

// Config configures one Recorder. CallID is required; SampleRate must
// match the PCM bridge rate — every WriteCaller / WriteAI byte stream is
// treated as PCM16 LE mono at this rate.
//
// Store is the destination for the WAV blob at flush time; nil falls
// through to stores.Default() (the process-wide upload backend used by
// the rest of VoiceServer). Tests inject a mock Store here to capture
// the bytes without touching disk or hitting the network.
type Config struct {
	CallID      string    // unique call identifier; embedded in the storage key
	TenantID    uint      // SaaS tenant scope for object key (0 = unknown)
	RecordingAt time.Time // call recording timeline anchor; zero → flush time UTC date
	SampleRate  int       // PCM bridge rate; e.g. 8000 / 16000 / 48000
	Transport   string    // free-form label for log lines: "websocket" / "webrtc" / "embed"
	Codec       string    // wire codec (informational, used in flush log line)
	Store       stores.Store
	Logger      *zap.Logger

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

// legPlacer tracks one leg's session-wide PCM placement state. Each
// successful place() call advances `pen` (in samples, session-relative)
// by exactly the number of bytes/2 it returned, so concatenating outputs
// across multiple calls produces a continuous per-leg PCM timeline. This
// is the core trick that lets us stream-merge per-leg part files at
// Flush time without rebuilding any per-leg buffer in memory.
type legPlacer struct {
	pen        int64 // session-wide sample position; next byte goes here
	prevTailNs int64 // for jitter snap across place() calls
}

// chunkManifest records one rolling-chunk upload. Each chunk produces
// THREE objects in the store:
//
//   - wavKey: standalone playable stereo WAV for ops triage (afplay)
//   - lKey:   raw L PCM (mono, no header) — used only by Flush to merge
//   - rKey:   raw R PCM (mono, no header) — used only by Flush to merge
//
// lBytes / rBytes are the EXACT byte counts of the raw PCM uploads so
// Flush can compute the final WAV's data-chunk size deterministically
// (without needing to re-Read each part to learn its length).
//
// An empty leg in this chunk is recorded as ""/0 so the Flush merger
// knows to skip the open-and-read for that key.
type chunkManifest struct {
	seq    int
	wavKey string
	lKey   string
	rKey   string
	lBytes int64
	rBytes int64
}

// Recorder captures user and AI PCM streams and produces a stereo WAV.
type Recorder struct {
	cfg Config
	log *zap.Logger

	mu      sync.Mutex
	inSegs  []frame // L = caller / device / browser (evicted after chunk upload)
	outSegs []frame // R = AI / TTS                  (evicted after chunk upload)
	flushed bool

	// Session-wide timeline base. Set lazily when the first frame
	// arrives (across either leg) so a leg that never speaks can still
	// share a t=0 with the other leg.
	sessionBaseNs  int64
	sessionBaseSet bool

	// Per-leg streaming placement state. Cloned (value copy) before
	// each chunk upload; committed back only on success. This is what
	// makes the chunker idempotent on transient store failures: a
	// failed upload leaves r.inPlacer / r.outPlacer untouched, so the
	// next tick re-places the same frames.
	inPlacer  legPlacer
	outPlacer legPlacer

	// Rolling chunk state. partSeq just provides monotonically
	// increasing seq numbers for log + storage keys.
	partSeq int
	chunkCh chan struct{} // closed by Flush to stop the chunker goroutine

	// parts records every chunk's manifest in upload order. Flush
	// iterates this list to stream-merge per-leg PCM into the final
	// WAV, then deletes every key (3 per chunk) at the end.
	parts []chunkManifest

	// 每路（caller / AI）采样率运行时校验状态。每次 append 累计字节数和
	// 时间跨度；一旦观察窗口 ≥ rateCheckMinSeconds 就计算 implied rate =
	// bytes / 2 / elapsed，与 cfg.SampleRate 偏差超过 rateCheckTolerance
	// 立刻 WARN 一次（per-leg 仅 warn 一次，避免日志风暴）。
	// 这能在生产里几秒内捕获到"配错码率"这类潜伏 bug —— 比如把 16k 采样的
	// PCM 写进配置为 8k 的 recorder 会产生 pitch-shift 静电，但一般要等到
	// 会话结束听录音才发现。
	inFirstNs, inLastNs   int64
	inBytes               int64
	inRateWarned          bool
	outFirstNs, outLastNs int64
	outBytes              int64
	outRateWarned         bool

	// Per-leg contiguous placement cursors. TTS/Omni chunks and RTP jitter
	// both arrive with inter-arrival gaps often >80ms (jitter snap).
	// Stamping every Write* with time.Now() inserts silence pads and the
	// recording sounds stuttery / pitch-odd. While successive Writes on
	// the same leg arrive within contiguousGapNs, force a continuous
	// PCM timeline (wallNs advances by sample-duration only).
	inContNextNs, inLastWriteNowNs   int64
	outContNextNs, outLastWriteNowNs int64
}

// 采样率运行时校验阈值。
const (
	// 累计观察窗口达到这么久才开始计算 implied rate（短窗口噪声大）。
	rateCheckMinSeconds = 5
	// implied rate 与 cfg.SampleRate 的相对偏差容忍度。30% 之外才警告 ——
	// 网络抖动 / 突发包到达不均会让短期 implied rate 飘 ±10-15%；30% 留
	// 足余量同时仍能捕获 8k↔16k、16k↔48k 这种量级错配。
	rateCheckTolerance = 0.30
	// Successive WriteCaller / WriteAI gaps under this stay continuous.
	// Covers TTS network jitter and PSTN/RTP bunching (>80ms snap).
	contiguousGapNs = int64(500 * time.Millisecond)
	// When append breaks contiguity it stamps wallNs=now, inserting a
	// real-time hole into the rate window. Reset on that same threshold
	// (not a looser one) so 500ms–1s phrase gaps cannot dilute implied_hz
	// into a false undersample WARN (configured 8k / implied ~5k).
	rateGapResetNs = contiguousGapNs
)

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

// uploadNextChunk drains all frames captured since the previous tick
// and writes THREE objects to the store:
//
//   - <callid>-part-<seq>-<ts>.wav   playable stereo WAV (ops triage)
//   - <callid>-part-<seq>-<ts>-L.pcm raw L PCM (mono, no header)
//   - <callid>-part-<seq>-<ts>-R.pcm raw R PCM (mono, no header)
//
// On success: the per-leg placers advance, the manifest grows, and the
// in-memory frame slices ARE EVICTED (replaced with fresh slices
// containing only frames that arrived during the upload). This keeps
// peak Recorder memory bounded at ~O(ChunkInterval) regardless of call
// duration — the critical reason this exists.
//
// On failure: nothing is committed. The next tick re-snapshots the
// same range and retries. A partially-uploaded chunk (WAV ok, L
// failed, etc.) rolls back its already-uploaded keys best-effort.
//
// Concurrency: uploads run WITHOUT holding r.mu. Flush may run during
// upload — it sets flushed=true, takes whatever frames remain in
// inSegs/outSegs, and ignores parts uploaded after this point. We
// detect that on commit and self-clean-up our orphaned objects.
func (r *Recorder) uploadNextChunk() {
	r.mu.Lock()
	if r.flushed || !r.sessionBaseSet {
		r.mu.Unlock()
		return
	}
	if len(r.inSegs) == 0 && len(r.outSegs) == 0 {
		r.mu.Unlock()
		return
	}
	// Snapshot the current slice headers. Frames are immutable once
	// appended (we only append, never mutate the pcm bytes), so
	// reading them outside the lock is safe even if append() races and
	// reallocates r.inSegs / r.outSegs — our local headers still point
	// at the original backing array.
	inSnap := r.inSegs
	outSnap := r.outSegs
	inLen := len(inSnap)
	outLen := len(outSnap)
	// Clone placers — we'll commit them back only on success.
	inP := r.inPlacer
	outP := r.outPlacer
	baseNs := r.sessionBaseNs
	rate := r.cfg.SampleRate
	seq := r.partSeq + 1
	r.mu.Unlock()

	snapNs := utils.RecordingJitterSnapNs()
	inPenBefore := inP.pen
	outPenBefore := outP.pen
	lBytes := inP.place(inSnap, baseNs, rate, snapNs)
	rBytes := outP.place(outSnap, baseNs, rate, snapNs)
	if len(lBytes) == 0 && len(rBytes) == 0 {
		return
	}
	// Skip near-empty clock ticks (one leg silent / scheduler gap) to avoid
	// half-empty part objects. Require ≥100ms of PCM on at least one leg.
	minChunkPCM := rate * 2 / 10 // 100ms mono PCM16
	if minChunkPCM < 320 {
		minChunkPCM = 320
	}
	if len(lBytes) < minChunkPCM && len(rBytes) < minChunkPCM {
		return
	}

	store := r.cfg.Store
	if store == nil {
		store = stores.Default()
	}
	if store == nil {
		// No store: leave frames in place; final Flush still has them.
		return
	}

	// Build the playable stereo WAV with chunk-local t=0 (= the earlier
	// of the two legs' starting pen). The shorter leg gets a leading
	// zero-pad so both tracks within the part WAV are aligned.
	chunkBase := inPenBefore
	if outPenBefore < chunkBase {
		chunkBase = outPenBefore
	}
	lPrefix := (inPenBefore - chunkBase) * 2 // bytes
	rPrefix := (outPenBefore - chunkBase) * 2
	// In-memory build: this is fine — chunk size is bounded by
	// ChunkInterval, not call duration.
	lForWav := make([]byte, 0, int(lPrefix)+len(lBytes))
	if lPrefix > 0 {
		lForWav = append(lForWav, make([]byte, lPrefix)...)
	}
	lForWav = append(lForWav, lBytes...)
	rForWav := make([]byte, 0, int(rPrefix)+len(rBytes))
	if rPrefix > 0 {
		rForWav = append(rForWav, make([]byte, rPrefix)...)
	}
	rForWav = append(rForWav, rBytes...)
	stereo := interleaveStereoBytes(lForWav, rForWav)
	wav := wrapWAV(stereo, rate, 2)

	ts := time.Now().Unix()
	wavKey, lKey, rKey := r.partObjectKeys(seq, ts)

	if err := store.Write(wavKey, bytes.NewReader(wav)); err != nil {
		r.log.Warn("recorder: chunk wav upload failed",
			zap.String("call_id", r.cfg.CallID), zap.Int("seq", seq), zap.Error(err))
		return
	}
	if len(lBytes) > 0 {
		if err := store.Write(lKey, bytes.NewReader(lBytes)); err != nil {
			r.log.Warn("recorder: chunk L PCM upload failed",
				zap.String("call_id", r.cfg.CallID), zap.Int("seq", seq), zap.Error(err))
			_ = store.Delete(wavKey)
			return
		}
	} else {
		lKey = ""
	}
	if len(rBytes) > 0 {
		if err := store.Write(rKey, bytes.NewReader(rBytes)); err != nil {
			r.log.Warn("recorder: chunk R PCM upload failed",
				zap.String("call_id", r.cfg.CallID), zap.Int("seq", seq), zap.Error(err))
			_ = store.Delete(wavKey)
			if lKey != "" {
				_ = store.Delete(lKey)
			}
			return
		}
	} else {
		rKey = ""
	}

	// Commit. If Flush ran during our upload, abandon our work and
	// best-effort clean up the now-orphaned objects (Flush already
	// re-placed the same frames as part of its tail and streamed them
	// without referencing our manifest).
	r.mu.Lock()
	if r.flushed {
		r.mu.Unlock()
		_ = store.Delete(wavKey)
		if lKey != "" {
			_ = store.Delete(lKey)
		}
		if rKey != "" {
			_ = store.Delete(rKey)
		}
		return
	}
	r.inPlacer = inP
	r.outPlacer = outP
	r.partSeq = seq
	r.parts = append(r.parts, chunkManifest{
		seq: seq, wavKey: wavKey, lKey: lKey, rKey: rKey,
		lBytes: int64(len(lBytes)), rBytes: int64(len(rBytes)),
	})
	// Evict the placed frames. Frames appended during our upload sit
	// at indices [inLen, len(r.inSegs)) and stay in memory for the
	// next tick. Reallocating into a new small slice releases the
	// underlying array (with all the placed pcm bytes) for GC.
	r.inSegs = trimFramesFront(r.inSegs, inLen)
	r.outSegs = trimFramesFront(r.outSegs, outLen)
	r.mu.Unlock()

	r.log.Info("recorder: chunk uploaded",
		zap.String("call_id", r.cfg.CallID),
		zap.Int("seq", seq),
		zap.String("wav_key", wavKey),
		zap.Int("l_bytes", len(lBytes)),
		zap.Int("r_bytes", len(rBytes)))
}

// trimFramesFront drops the first `n` frames from segs and returns a
// fresh slice containing only the remainder. Returning a NEW backing
// array is the whole point — it lets the GC reclaim the consumed
// frames' pcm buffers.
func trimFramesFront(segs []frame, n int) []frame {
	if n >= len(segs) {
		return segs[:0:0] // empty slice, original array drops out of scope
	}
	rest := segs[n:]
	out := make([]frame, len(rest))
	copy(out, rest)
	return out
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
	// Drop a trailing odd byte: a 16-bit PCM sample is two bytes, and a
	// stray single byte would shift every subsequent sample by one byte
	// after placePCMTrackBytes glues frames together — producing a
	// hissing / static "电流音" on the affected channel. Real callers
	// always feed even-length buffers; this is a belt-and-braces guard
	// for resamplers / muxers that occasionally produce an odd tail.
	if len(pcm)&1 != 0 {
		pcm = pcm[:len(pcm)-1]
		if len(pcm) == 0 {
			return
		}
	}
	buf := append([]byte(nil), pcm...)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.flushed {
		return
	}
	now := time.Now().UnixNano()
	wallNs := now
	if r.cfg.SampleRate > 0 {
		// Use inter-arrival time (now - last Write on this leg), not lag vs
		// the continuous audio cursor — otherwise long utterances fall
		// behind wall clock and falsely break contiguity.
		var contNext, lastWrite *int64
		if caller {
			contNext, lastWrite = &r.inContNextNs, &r.inLastWriteNowNs
		} else {
			contNext, lastWrite = &r.outContNextNs, &r.outLastWriteNowNs
		}
		if *contNext > 0 && *lastWrite > 0 {
			gap := now - *lastWrite
			if gap < 0 {
				gap = -gap
			}
			if gap < contiguousGapNs {
				wallNs = *contNext
			}
		}
		samples := int64(len(buf) / 2)
		*lastWrite = now
		*contNext = wallNs + samples*1_000_000_000/int64(r.cfg.SampleRate)
	}
	f := frame{wallNs: wallNs, pcm: buf}
	// Anchor the session timeline on the very first frame across either
	// leg. Both per-leg pens and all subsequent placement math measure
	// sample offset from this point. Doing it lazily (vs. New()) means a
	// recorder that never receives audio doesn't waste a Flush trying
	// to build a zero-length WAV.
	if !r.sessionBaseSet {
		r.sessionBaseNs = f.wallNs
		r.sessionBaseSet = true
	}
	if caller {
		r.inSegs = append(r.inSegs, f)
		r.updateRateStats(true, f.wallNs, int64(len(buf)))
	} else {
		r.outSegs = append(r.outSegs, f)
		r.updateRateStats(false, f.wallNs, int64(len(buf)))
	}
}

// updateRateStats 在锁内被 append 调用，累计单路的字节数与时间跨度，并在
// 累计窗口 ≥ rateCheckMinSeconds 时计算 implied rate（PCM16 = 2 bytes/sample）
// 与 cfg.SampleRate 的相对偏差。超出 rateCheckTolerance 立即 WARN 一次。
//
// 仅警告不修复：自动修复（resample）会掩盖配置错误，而错误一般是上游 bug
// 应当被人看到。warned 标志确保每路每次会话只 warn 一次。
//
// 注意调用位置：必须在 append 已经持有 r.mu 时调用，函数本身不再加锁。
func (r *Recorder) updateRateStats(caller bool, wallNs, byteCount int64) {
	configured := r.cfg.SampleRate
	if configured <= 0 {
		return
	}
	var (
		firstNs *int64
		lastNs  *int64
		bytesP  *int64
		warned  *bool
		legName string
	)
	if caller {
		firstNs = &r.inFirstNs
		lastNs = &r.inLastNs
		bytesP = &r.inBytes
		warned = &r.inRateWarned
		legName = "caller"
	} else {
		firstNs = &r.outFirstNs
		lastNs = &r.outLastNs
		bytesP = &r.outBytes
		warned = &r.outRateWarned
		legName = "ai"
	}
	// Welcome → think → speak leaves multi-second holes. Restart the
	// observation window so implied_hz is not diluted by silence.
	if *lastNs > 0 && wallNs-*lastNs > rateGapResetNs {
		*firstNs = 0
		*bytesP = 0
	}
	if *firstNs == 0 {
		*firstNs = wallNs
	}
	*lastNs = wallNs
	*bytesP += byteCount
	if *warned {
		return
	}
	elapsedNs := *lastNs - *firstNs
	if elapsedNs < int64(rateCheckMinSeconds)*1_000_000_000 {
		return
	}
	// implied rate = (bytes / 2) samples / (elapsedNs / 1e9) seconds
	//             = bytes * 1e9 / (2 * elapsedNs)
	implied := float64(*bytesP) * 1e9 / (2.0 * float64(elapsedNs))
	delta := implied - float64(configured)
	if delta < 0 {
		delta = -delta
	}
	if delta/float64(configured) <= rateCheckTolerance {
		return
	}
	*warned = true
	r.log.Warn("recorder: sample-rate mismatch detected",
		zap.String("call_id", r.cfg.CallID),
		zap.String("leg", legName),
		zap.Int("configured_hz", configured),
		zap.Float64("implied_hz", implied),
		zap.Float64("deviation_pct", delta/float64(configured)*100),
		zap.Int64("observed_bytes", *bytesP),
		zap.Int64("observed_ms", elapsedNs/1_000_000),
		zap.String("hint", "upstream is feeding PCM at a rate that doesn't match cfg.SampleRate; recording will sound pitch-shifted"),
	)
}

// Flush stitches every uploaded chunk-part PCM together with the tail
// (frames that arrived after the last chunker tick), streams the
// combined stereo WAV to the store via io.Pipe (so peak memory is
// O(buffer size), not O(call duration)), then deletes the part files.
//
// Returns ok=false when there is nothing to record or when the upload
// failed; the error is logged either way.
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
	// Stop the chunker BEFORE taking the snapshot. A chunker tick that
	// already passed its `flushed` check just races to upload+commit;
	// our `flushed=true` makes its commit branch into the orphan-delete
	// path so it can't double-place the tail frames.
	if r.chunkCh != nil {
		close(r.chunkCh)
		r.chunkCh = nil
	}
	inSegs := r.inSegs
	outSegs := r.outSegs
	r.inSegs = nil
	r.outSegs = nil
	parts := r.parts
	r.parts = nil
	inP := r.inPlacer
	outP := r.outPlacer
	baseNs := r.sessionBaseNs
	sessionSet := r.sessionBaseSet
	r.mu.Unlock()

	rate := r.cfg.SampleRate
	if !sessionSet {
		r.log.Info("recorder: no frames captured, skip",
			zap.String("call_id", r.cfg.CallID))
		return gateway.RecordingInfo{}, false
	}

	// Place the tail (frames since the last chunker tick) using the
	// SAME placer state the chunker would have used. tailL/tailR
	// concatenate cleanly onto parts[*].lBytes / rBytes streams.
	snapNs := utils.RecordingJitterSnapNs()
	tailL := inP.place(inSegs, baseNs, rate, snapNs)
	tailR := outP.place(outSegs, baseNs, rate, snapNs)

	// Compute total per-leg byte counts up-front. We need the WAV
	// data-chunk size in the header before the body streams, and we
	// don't want to "rewind and patch" on backends that don't
	// re-seek (S3 multipart, COS, …).
	var totalLBytes, totalRBytes int64
	for _, p := range parts {
		totalLBytes += p.lBytes
		totalRBytes += p.rBytes
	}
	totalLBytes += int64(len(tailL))
	totalRBytes += int64(len(tailR))
	legSamples := totalLBytes / 2
	if rs := totalRBytes / 2; rs > legSamples {
		legSamples = rs
	}
	if legSamples == 0 {
		r.log.Info("recorder: zero-length recording, skip",
			zap.String("call_id", r.cfg.CallID))
		return gateway.RecordingInfo{}, false
	}

	store := r.cfg.Store
	if store == nil {
		store = stores.Default()
	}
	if store == nil {
		r.log.Warn("recorder: no Store available, skip upload",
			zap.String("call_id", r.cfg.CallID))
		return gateway.RecordingInfo{}, false
	}

	at := r.cfg.RecordingAt
	if at.IsZero() {
		at = time.Now()
	}
	key := utils.RecordingObjectKey(r.cfg.TenantID, r.cfg.CallID, at, "wav")
	if key == "" {
		r.log.Warn("recorder: empty object key, skip upload",
			zap.String("call_id", r.cfg.CallID))
		return gateway.RecordingInfo{}, false
	}

	// Streaming pipeline:
	//
	//   writer goroutine ──► pw ──► [io.Pipe] ──► pr ──► store.Write
	//                  └─► sha256.Hash
	//                  └─► counterWriter (Bytes for RecordingInfo)
	//
	// Backend Write reads from pr concurrently. If Write fails we cancel
	// the writer via pr.CloseWithError, which makes pw.Write return the
	// same error and lets the goroutine exit cleanly (no leak).
	pr, pw := io.Pipe()
	hasher := sha256.New()
	var totalBytes int64
	cw := &counterWriter{n: &totalBytes}
	mw := io.MultiWriter(pw, hasher, cw)

	var streamErr error
	streamDone := make(chan struct{})
	go func() {
		defer close(streamDone)
		lChain := newPartReaderChain(store, parts, true /*L*/, tailL)
		rChain := newPartReaderChain(store, parts, false /*R*/, tailR)
		defer lChain.Close()
		defer rChain.Close()
		if err := writeWAVHeaderTo(mw, rate, 2, legSamples); err != nil {
			streamErr = err
			_ = pw.CloseWithError(err)
			return
		}
		if err := streamInterleave(mw, lChain, rChain, legSamples); err != nil {
			streamErr = err
			_ = pw.CloseWithError(err)
			return
		}
		_ = pw.Close()
	}()

	if err := store.Write(key, pr); err != nil {
		// Tear down the writer side: pr is the read half, closing it
		// with an error unblocks the goroutine's pw.Write.
		_ = pr.CloseWithError(err)
		<-streamDone
		r.log.Warn("recorder: store write failed",
			zap.String("call_id", r.cfg.CallID), zap.Error(err))
		return gateway.RecordingInfo{}, false
	}
	<-streamDone
	if streamErr != nil {
		r.log.Warn("recorder: stream-build failed",
			zap.String("call_id", r.cfg.CallID), zap.Error(streamErr))
		return gateway.RecordingInfo{}, false
	}

	// Reclaim every part object (3 keys per chunk) now that the
	// canonical final WAV is durable. Best-effort: delete failures are
	// logged but never demote the recording's success.
	deletePartObjects(store, parts, r.log, r.cfg.CallID)

	durationMs := legSamples * 1000 / int64(rate)
	hash := "sha256:" + hex.EncodeToString(hasher.Sum(nil))
	url := strings.TrimSpace(store.PublicURL(key))
	if url == "" {
		url = key
	}

	r.log.Info("recorder: wav written (streamed)",
		zap.String("call_id", r.cfg.CallID),
		zap.String("transport", r.cfg.Transport),
		zap.String("codec", r.cfg.Codec),
		zap.Int("rate", rate),
		zap.Int64("bytes", totalBytes),
		zap.Int64("samples", legSamples),
		zap.Int("parts", len(parts)),
		zap.String("key", key))

	return gateway.RecordingInfo{
		Key:        key,
		URL:        url,
		Format:     "wav",
		Layout:     "stereo-l-r",
		SampleRate: rate,
		Channels:   2,
		Bytes:      int(totalBytes),
		DurationMs: durationMs,
		Hash:       hash,
	}, true
}

// counterWriter tallies bytes written through it. Used to populate
// RecordingInfo.Bytes during the streaming Flush since the final WAV
// never lives in one buffer we could len() on.
type counterWriter struct{ n *int64 }

func (c *counterWriter) Write(p []byte) (int, error) {
	*c.n += int64(len(p))
	return len(p), nil
}

// partReaderChain is an io.Reader that lazily concatenates per-leg PCM
// from a list of chunk parts plus a tail buffer. It opens at most ONE
// part stream at a time, so a 1-hour call with 60-second chunks keeps
// at most ~one HTTP body open instead of 60.
//
// Empty parts (lBytes==0 or rBytes==0 for the requested leg) are
// transparently skipped — they record windows where this leg had no
// audio.
type partReaderChain struct {
	store         stores.Store
	parts         []chunkManifest
	selectL       bool
	tail          []byte
	idx           int
	cur           io.ReadCloser
	tailReader    io.Reader
	tailExhausted bool
}

func newPartReaderChain(store stores.Store, parts []chunkManifest, selectL bool, tail []byte) *partReaderChain {
	return &partReaderChain{
		store:   store,
		parts:   parts,
		selectL: selectL,
		tail:    tail,
	}
}

func (c *partReaderChain) Read(p []byte) (int, error) {
	for {
		if c.cur == nil {
			if c.idx >= len(c.parts) {
				if c.tailReader == nil && !c.tailExhausted {
					if len(c.tail) == 0 {
						c.tailExhausted = true
						return 0, io.EOF
					}
					c.tailReader = bytes.NewReader(c.tail)
				}
				if c.tailReader != nil {
					n, err := c.tailReader.Read(p)
					if errors.Is(err, io.EOF) {
						c.tailReader = nil
						c.tailExhausted = true
						if n > 0 {
							return n, nil
						}
						continue
					}
					return n, err
				}
				return 0, io.EOF
			}
			part := c.parts[c.idx]
			c.idx++
			key := part.lKey
			byteCount := part.lBytes
			if !c.selectL {
				key = part.rKey
				byteCount = part.rBytes
			}
			if key == "" || byteCount == 0 {
				continue
			}
			rc, _, err := c.store.Read(key)
			if err != nil {
				return 0, fmt.Errorf("recorder: open part %s: %w", key, err)
			}
			c.cur = rc
		}
		n, err := c.cur.Read(p)
		if errors.Is(err, io.EOF) {
			_ = c.cur.Close()
			c.cur = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func (c *partReaderChain) Close() error {
	if c.cur != nil {
		_ = c.cur.Close()
		c.cur = nil
	}
	return nil
}

// streamInterleave reads up to legSamples L+R PCM16 sample pairs from
// lr / rr and writes 4-byte interleaved stereo samples to w. EOF on
// either input is treated as zero samples (the corresponding leg ran
// short of the longer leg) — we KNOW the exact total length up-front
// from the manifest so we never under- or over-write the WAV's
// declared data chunk.
//
// Block size: 1024 samples per side = 4096 output bytes per loop. Big
// enough to amortise syscalls, small enough to keep memory tiny.
func streamInterleave(w io.Writer, lr, rr io.Reader, legSamples int64) error {
	const blockSamples = 1024
	lBuf := make([]byte, blockSamples*2)
	rBuf := make([]byte, blockSamples*2)
	out := make([]byte, blockSamples*4)
	var done int64
	for done < legSamples {
		want := legSamples - done
		if want > blockSamples {
			want = blockSamples
		}
		wantBytes := int(want) * 2
		nl, _ := io.ReadFull(lr, lBuf[:wantBytes])
		nr, _ := io.ReadFull(rr, rBuf[:wantBytes])
		// Anything we couldn't read from the leg = zero samples for
		// that side. ReadFull returns whatever bytes it managed to
		// read in `n`, regardless of the error type.
		for i := nl; i < wantBytes; i++ {
			lBuf[i] = 0
		}
		for i := nr; i < wantBytes; i++ {
			rBuf[i] = 0
		}
		for i := 0; i < int(want); i++ {
			out[i*4] = lBuf[i*2]
			out[i*4+1] = lBuf[i*2+1]
			out[i*4+2] = rBuf[i*2]
			out[i*4+3] = rBuf[i*2+1]
		}
		if _, err := w.Write(out[:int(want)*4]); err != nil {
			return err
		}
		done += want
	}
	return nil
}

// writeWAVHeaderTo emits the canonical 44-byte RIFF/WAVE header for
// PCM16 with a known data-chunk size. The caller is responsible for
// streaming exactly `legSamples * channels * 2` bytes of PCM after.
func writeWAVHeaderTo(w io.Writer, sampleRate, channels int, legSamples int64) error {
	const (
		bitsPerSample = 16
		fmtChunkSize  = 16
		audioFormat   = 1
	)
	if channels < 1 {
		channels = 1
	}
	dataSize := uint32(legSamples) * uint32(channels) * uint32(bitsPerSample/8)
	totalSize := uint32(4) + 8 + uint32(fmtChunkSize) + 8 + dataSize
	byteRate := uint32(sampleRate * channels * bitsPerSample / 8)
	blockAlign := uint16(channels * bitsPerSample / 8)

	var hdr [44]byte
	copy(hdr[0:4], "RIFF")
	binary.LittleEndian.PutUint32(hdr[4:8], totalSize)
	copy(hdr[8:12], "WAVE")
	copy(hdr[12:16], "fmt ")
	binary.LittleEndian.PutUint32(hdr[16:20], uint32(fmtChunkSize))
	binary.LittleEndian.PutUint16(hdr[20:22], audioFormat)
	binary.LittleEndian.PutUint16(hdr[22:24], uint16(channels))
	binary.LittleEndian.PutUint32(hdr[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(hdr[28:32], byteRate)
	binary.LittleEndian.PutUint16(hdr[32:34], blockAlign)
	binary.LittleEndian.PutUint16(hdr[34:36], uint16(bitsPerSample))
	copy(hdr[36:40], "data")
	binary.LittleEndian.PutUint32(hdr[40:44], dataSize)
	_, err := w.Write(hdr[:])
	return err
}

// deletePartObjects best-effort removes every key referenced by the
// manifest. Failures are logged but don't propagate — by this point
// the canonical final WAV is durable; orphaned parts are wasted
// storage at worst.
func deletePartObjects(store stores.Store, parts []chunkManifest, log *zap.Logger, callID string) {
	if len(parts) == 0 {
		return
	}
	var failed int
	for _, p := range parts {
		for _, k := range []string{p.wavKey, p.lKey, p.rKey} {
			if k == "" {
				continue
			}
			if err := store.Delete(k); err != nil {
				failed++
				log.Warn("recorder: chunk delete failed",
					zap.String("call_id", callID), zap.String("key", k), zap.Error(err))
			}
		}
	}
	log.Info("recorder: chunks reclaimed",
		zap.String("call_id", callID),
		zap.Int("parts", len(parts)),
		zap.Int("failed", failed))
}

// ---------- internals: PCM placement / WAV wrapping ----------------------

// place is the streaming variant of placePCMTrackBytes used by the
// chunker / Flush. Given the SAME sessionBaseNs across calls, repeated
// invocations on successive frame batches produce a CONTINUOUS per-leg
// PCM stream — each call's output begins at session-sample lp.pen
// (going in) and ends at session-sample lp.pen (going out), so
// concatenating outputs across calls reconstructs the leg's full
// timeline byte-for-byte.
//
// The per-call output length equals (new_pen - old_pen) * 2; concretely,
// it includes any zero-padding for inter-frame gaps WITHIN the batch,
// PLUS any zero-padding for the gap from the prior batch's pen up to
// this batch's first frame's wall position (when the gap exceeds the
// jitter-snap window — sub-snap gaps are absorbed via pen).
//
// Returns nil iff the batch contributes zero new samples (e.g. all
// frames were empty). lp.pen / lp.prevTailNs are mutated in place.
func (lp *legPlacer) place(frames []frame, sessionBaseNs int64, rate int, snapNs int64) []byte {
	if len(frames) == 0 || rate <= 0 {
		return nil
	}
	// Insertion-sort: same rationale as sortFramesByWall — frames are
	// almost always near-sorted by arrival.
	sortFramesByWall(frames)

	startPen := lp.pen
	out := make([]byte, 0, 4096)
	for _, s := range frames {
		if len(s.pcm) == 0 {
			continue
		}
		samples := int64(len(s.pcm) / 2)
		pos := (s.wallNs - sessionBaseNs) * int64(rate) / 1_000_000_000
		// Jitter snap: micro-gaps from scheduling noise collapse to pen.
		// prevTailNs == 0 on the very first call ever — treat as "no
		// prior frame" by allowing snap so the first frame of the
		// session lands at pos without artificial leading silence.
		if pos > lp.pen && lp.prevTailNs > 0 && (s.wallNs-lp.prevTailNs) < snapNs {
			pos = lp.pen
		}
		if pos < lp.pen {
			// Burst arrival: keep frames non-overlapping.
			pos = lp.pen
		}
		// Zero-pad from current pen up to pos (real silence).
		if pos > lp.pen {
			out = append(out, make([]byte, (pos-lp.pen)*2)...)
		}
		out = append(out, s.pcm...)
		lp.pen = pos + samples
		lp.prevTailNs = s.wallNs + samples*1_000_000_000/int64(rate)
	}
	if lp.pen == startPen {
		return nil
	}
	return out
}

// placePCMTrackBytes lays PCM frames on a wall-clock timeline anchored
// to baseNs. Each frame is positioned at
// pos = (wallNs - base) * rate / 1e9, with a "pen" guard preventing
// frames from overlapping during a burst. Gaps are zero-padded.
//
// JITTER SNAP (SoulNexus improvement over VoiceServer):
// Pacing layers (TTS pipeline `time.Sleep(FrameDuration)`, Go scheduler,
// RTP jitter buffer) introduce ±1-3ms of wall-clock noise between
// frames that are LOGICALLY contiguous in the audio stream. The
// upstream VoiceServer recorder placed every frame on the raw
// wall-clock grid, so each such micro-gap got zero-padded — audible as
// periodic clicks / "电流音" during continuous TTS playback. We fix it
// by snapping `pos` to `pen` whenever the wall-clock advance is within
// `jitterSnapNs` of the previous frame's tail. Real silence gaps
// (between turns, barge-in, etc.) are preserved because they exceed
// the snap window.
//
// Returns one PCM16 LE mono buffer.
func placePCMTrackBytes(segs []frame, baseNs int64, rate int) []byte {
	if len(segs) == 0 || rate <= 0 {
		return nil
	}
	// Sort by wallNs ASC. n is small (hundreds typically), insertion
	// sort would be fine; stdlib sort is plenty.
	sortFramesByWall(segs)

	// 80ms snap window: comfortably larger than any single-frame
	// scheduler hiccup (a typical TTS frame is 20ms; even a 60ms stall
	// is still "the same continuous utterance"), small enough that a
	// real inter-turn pause (≥100ms) still produces an audible silence
	// gap in the recording.
	// 通过 RECORDING_JITTER_SNAP_MS 可覆盖（clamp 到 10~500ms）。
	jitterSnapNs := utils.RecordingJitterSnapNs()

	// Pen advances forward through the timeline so successive bursts
	// don't overlap. Position is in samples (PCM16 = 2 bytes/sample).
	out := make([]byte, 0, 4096)
	pen := int64(0)
	prevTailNs := baseNs
	for _, s := range segs {
		if len(s.pcm) == 0 {
			continue
		}
		samples := int64(len(s.pcm) / 2)
		pos := (s.wallNs - baseNs) * int64(rate) / 1_000_000_000
		// Snap micro-jitter to `pen` so contiguous audio stays
		// contiguous in the WAV. Only fall back to wall-clock when the
		// inter-frame gap exceeds jitterSnapNs (real silence).
		if pos > pen && (s.wallNs-prevTailNs) < jitterSnapNs {
			pos = pen
		}
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
		// frameDurNs = samples * 1e9 / rate
		prevTailNs = s.wallNs + samples*1_000_000_000/int64(rate)
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

func (r *Recorder) partObjectKeys(seq int, ts int64) (wavKey, lKey, rKey string) {
	at := r.cfg.RecordingAt
	if at.IsZero() {
		at = time.Now()
	}
	wavKey = utils.RecordingPartObjectKey(r.cfg.TenantID, r.cfg.CallID, at, seq, ts)
	if wavKey == "" {
		base := sanitizeFilename(r.cfg.CallID)
		wavKey = fmt.Sprintf("%s-part-%d-%d.wav", base, seq, ts)
	}
	stem := strings.TrimSuffix(wavKey, ".wav")
	lKey = stem + "-L.pcm"
	rKey = stem + "-R.pcm"
	return wavKey, lKey, rKey
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

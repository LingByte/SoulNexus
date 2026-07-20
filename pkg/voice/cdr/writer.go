// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cdr

// Writer is the persistent side of the CDR pipeline.
//
// Architecture:
//
//	┌──────── call-end goroutine ────────┐
//	│  Emit(record)                      │
//	│   └─► non-blocking send on ch ─────┼──► drain goroutine ──► current.jsonl
//	└────────────────────────────────────┘                          │
//	                                                                 │ rotate when size/time
//	                                                                 ▼
//	                                                          <base>-<UTC>.jsonl
//	                                                          (Filebeat picks up)
//
// Why one goroutine + one channel:
//
//   - Producers never block. Hot-path budget rule (per design doc):
//     call-end is the most expensive observation we make, and even
//     it must finish in microseconds.
//   - All file I/O is serial on a single fd — no fsync contention,
//     no torn lines, no need for a per-write mutex.
//   - Rotation can read+rename the current file safely because only
//     this goroutine touches it.
//
// Robustness:
//
//   - Channel overflow → drop + metric increment (cdr_dropped_total).
//   - I/O errors → log + counter (cdr_write_errors_total), the next
//     write retries; we never block the drain goroutine on a stuck
//     disk.
//   - Crash safety: each line is a complete JSON object terminated
//     by '\n'. Filebeat / Vector consume line-by-line and resume
//     from the last-shipped offset; a partial final line is just
//     re-shipped, which is idempotent at the downstream layer.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/voice/metrics"
	"go.uber.org/zap"
)

const (
	// MetricCDRWrittenTotal is bumped on every successful CDR line
	// flushed to disk. Useful to verify the pipeline is live.
	MetricCDRWrittenTotal = "voiceserver_cdr_written_total"
	// MetricCDRDroppedTotal counts records lost because the channel
	// was full. Non-zero means the drain isn't keeping up — usually
	// a disk-I/O stall.
	MetricCDRDroppedTotal = "voiceserver_cdr_dropped_total"
	// MetricCDRWriteErrorsTotal counts disk-write failures. Each
	// failure is also logged once per minute (rate-limited).
	MetricCDRWriteErrorsTotal = "voiceserver_cdr_write_errors_total"

	// defaultBufSize is the channel capacity. 1024 records ≈ 256 KiB
	// of pending work at worst (records are small, ~256 B post-JSON).
	defaultBufSize = 1024

	// defaultMaxFileBytes triggers size-based rotation. 64 MiB ≈
	// 250k records — Filebeat handles files this size comfortably.
	defaultMaxFileBytes = 64 * 1024 * 1024
)

func init() {
	metrics.RegisterLabels(MetricCDRWrittenTotal)
	metrics.RegisterLabels(MetricCDRDroppedTotal)
	metrics.RegisterLabels(MetricCDRWriteErrorsTotal)
}

// Config controls Writer construction. All fields are optional;
// zero values pick sensible defaults.
type Config struct {
	// Dir is the directory where CDR files are written. Created
	// with mkdir-p semantics if missing. Default: /var/log/soulnexus.
	Dir string
	// BaseName is the file name without the rotation suffix.
	// The currently-being-written file is `<BaseName>.current.jsonl`;
	// rotated files become `<BaseName>-<UTC ISO>.jsonl`.
	BaseName string
	// BufSize sets the channel capacity. Default 1024.
	BufSize int
	// MaxFileBytes triggers rotation when the current file exceeds
	// this size. Default 64 MiB. Zero disables size rotation.
	MaxFileBytes int64
	// RotateInterval triggers periodic rotation regardless of size
	// so Filebeat sees fresh files even on a slow shift. Default
	// 1h. Zero disables time-based rotation.
	RotateInterval time.Duration
}

func (c *Config) applyDefaults() {
	if c.Dir == "" {
		// Default: <process working dir>/logs/trace. Override via
		// SOULNEXUS_CDR_DIR in .env (see env.example).
		c.Dir = "logs/trace"
	}
	if c.BaseName == "" {
		c.BaseName = "cdr"
	}
	if c.BufSize <= 0 {
		c.BufSize = defaultBufSize
	}
	if c.MaxFileBytes <= 0 {
		c.MaxFileBytes = defaultMaxFileBytes
	}
	if c.RotateInterval <= 0 {
		c.RotateInterval = time.Hour
	}
}

// Writer is the singleton-style CDR sink. One per process.
type Writer struct {
	cfg    Config
	ch     chan CallRecord
	stopCh chan struct{}
	doneCh chan struct{}

	// Drain-goroutine-only state — no mutex needed.
	file    *os.File
	written int64
	opened  time.Time

	// Atomic counters mirror what we expose via metrics so tests
	// can verify behaviour without a metrics scrape round-trip.
	dropped     atomic.Uint64
	writtenCnt  atomic.Uint64
	writeErrors atomic.Uint64

	startOnce sync.Once
	stopOnce  sync.Once
	errLogMu  sync.Mutex
	errLogAt  time.Time
}

// NewWriter constructs a Writer but does not start the drain. Call
// Start before Emit.
func NewWriter(cfg Config) *Writer {
	cfg.applyDefaults()
	return &Writer{
		cfg:    cfg,
		ch:     make(chan CallRecord, cfg.BufSize),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// Start launches the drain goroutine. Idempotent — second call is a
// no-op. Returns an error if the directory cannot be created.
func (w *Writer) Start() error {
	if w == nil {
		return fmt.Errorf("cdr: nil writer")
	}
	if err := os.MkdirAll(w.cfg.Dir, 0o755); err != nil {
		return fmt.Errorf("cdr: mkdir %s: %w", w.cfg.Dir, err)
	}
	w.startOnce.Do(func() {
		go w.run()
	})
	return nil
}

// Stop signals the drain goroutine to flush remaining records and
// exit. Blocks until the goroutine has finished or 5s elapsed —
// after that the OS handles any leftover fd cleanup. Idempotent.
func (w *Writer) Stop() {
	if w == nil {
		return
	}
	w.stopOnce.Do(func() {
		close(w.stopCh)
	})
	select {
	case <-w.doneCh:
	case <-time.After(5 * time.Second):
		if logger.Lg != nil {
			logger.Lg.Warn("cdr writer stop timeout",
				zap.Uint64("dropped", w.dropped.Load()))
		}
	}
}

// Emit hands a record off to the drain goroutine. Never blocks.
// Returns false if the channel is full (record dropped + counter
// bumped). Records are taken by value so the caller can safely
// mutate / re-use the original.
//
// Hot-path cost: one channel send. ~30 ns on x86. Zero allocation
// when the record itself has no nested slices being grown.
func (w *Writer) Emit(rec CallRecord) bool {
	if w == nil {
		return false
	}
	select {
	case w.ch <- rec:
		return true
	default:
		w.dropped.Add(1)
		metrics.Default.IncCounter(MetricCDRDroppedTotal,
			"CDR records dropped because the writer buffer was full",
			nil)
		return false
	}
}

// Stats exposes runtime counters for tests and admin endpoints.
func (w *Writer) Stats() (written, dropped, writeErrors uint64) {
	if w == nil {
		return 0, 0, 0
	}
	return w.writtenCnt.Load(), w.dropped.Load(), w.writeErrors.Load()
}

// run is the drain goroutine. Owns the open file handle and
// rotation timer.
func (w *Writer) run() {
	defer close(w.doneCh)

	if err := w.openCurrent(); err != nil {
		w.logWriteErr("open initial CDR file", err)
		// Without a file we can't write — keep draining so we don't
		// back-pressure producers, but every record will be dropped.
	}

	ticker := time.NewTicker(w.cfg.RotateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			w.drainRemaining()
			w.closeAndRotate(true)
			return

		case rec := <-w.ch:
			w.writeOne(rec)

		case <-ticker.C:
			// Periodic rotation. Only rotate if we actually wrote
			// something — empty rotations spam Filebeat.
			if w.written > 0 {
				w.closeAndRotate(false)
				if err := w.openCurrent(); err != nil {
					w.logWriteErr("reopen after rotate", err)
				}
			}
		}
	}
}

// drainRemaining flushes any queued records before final close.
// Bounded by the channel length at entry — we don't accept new ones
// after stopCh is closed (Emit's `default` branch would drop them,
// which is the right behaviour during shutdown).
func (w *Writer) drainRemaining() {
	for {
		select {
		case rec := <-w.ch:
			w.writeOne(rec)
		default:
			return
		}
	}
}

func (w *Writer) writeOne(rec CallRecord) {
	if w.file == nil {
		w.writeErrors.Add(1)
		return
	}
	line, err := json.Marshal(rec)
	if err != nil {
		// Marshal errors mean a programmer error (cycle in Extra,
		// non-encodable type). Drop and count.
		w.writeErrors.Add(1)
		metrics.Default.IncCounter(MetricCDRWriteErrorsTotal,
			"CDR records that failed to write to disk", nil)
		return
	}
	line = append(line, '\n')
	n, err := w.file.Write(line)
	if err != nil {
		w.writeErrors.Add(1)
		metrics.Default.IncCounter(MetricCDRWriteErrorsTotal,
			"CDR records that failed to write to disk", nil)
		w.logWriteErr("CDR write failed", err)
		return
	}
	w.written += int64(n)
	w.writtenCnt.Add(1)
	metrics.Default.IncCounter(MetricCDRWrittenTotal,
		"CDR records successfully written to disk", nil)

	if w.cfg.MaxFileBytes > 0 && w.written >= w.cfg.MaxFileBytes {
		w.closeAndRotate(false)
		if err := w.openCurrent(); err != nil {
			w.logWriteErr("reopen after size rotate", err)
		}
	}
}

func (w *Writer) openCurrent() error {
	path := filepath.Join(w.cfg.Dir, w.cfg.BaseName+".current.jsonl")
	// O_APPEND so a restart concatenates onto the same in-flight
	// file rather than truncating recent records. Permissions 0640
	// — readable by the log-shipper group but not world.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		return err
	}
	// Find current size so size-rotation triggers correctly on a
	// re-opened file.
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	w.file = f
	w.written = st.Size()
	w.opened = time.Now()
	return nil
}

// closeAndRotate closes the current file and renames it with a
// timestamp suffix. `final` indicates this is the shutdown path —
// we still rotate even if size is zero so Filebeat sees a closed
// artifact.
func (w *Writer) closeAndRotate(final bool) {
	if w.file == nil {
		return
	}
	_ = w.file.Sync()
	_ = w.file.Close()
	w.file = nil

	if w.written == 0 && !final {
		// Don't bother renaming an empty file — just delete it so
		// downstream doesn't ship zero-byte artifacts.
		_ = os.Remove(filepath.Join(w.cfg.Dir, w.cfg.BaseName+".current.jsonl"))
		w.written = 0
		return
	}

	// Rotate name: <base>-YYYYMMDDTHHMMSSZ.jsonl
	stamp := w.opened.Format("20060102T150405Z")
	if stamp == "00010101T000000Z" {
		stamp = time.Now().Format("20060102T150405")
	}
	src := filepath.Join(w.cfg.Dir, w.cfg.BaseName+".current.jsonl")
	dst := filepath.Join(w.cfg.Dir, fmt.Sprintf("%s-%s.jsonl", w.cfg.BaseName, stamp))
	if err := os.Rename(src, dst); err != nil {
		w.logWriteErr("rotate rename", err)
	}
	w.written = 0
}

// logWriteErr logs at most once per minute so a sustained outage
// doesn't drown the log.
func (w *Writer) logWriteErr(what string, err error) {
	w.errLogMu.Lock()
	now := time.Now()
	suppress := !w.errLogAt.IsZero() && now.Sub(w.errLogAt) < time.Minute
	w.errLogAt = now
	w.errLogMu.Unlock()
	if suppress {
		return
	}
	if logger.Lg != nil {
		logger.Lg.Warn("cdr writer error",
			zap.String("what", what),
			zap.Error(err))
	}
}

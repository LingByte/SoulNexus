package tasks

import (
	"context"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils/welcomeaudio"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AudioPrefetchWarmup periodically warms the welcomeaudio disk cache.
// Hold-music URL collection was removed with telephony;
// the ticker remains so callers can reintroduce non-telephony prefetch sources later.
type AudioPrefetchWarmup struct {
	db       *gorm.DB
	interval time.Duration
	stopCh   chan struct{}
	once     sync.Once
}

// NewAudioPrefetchWarmup builds a warmer (default interval 10m).
func NewAudioPrefetchWarmup(db *gorm.DB, interval time.Duration) *AudioPrefetchWarmup {
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	return &AudioPrefetchWarmup{db: db, interval: interval, stopCh: make(chan struct{})}
}

// Start runs an immediate pass then ticks until Stop.
func (w *AudioPrefetchWarmup) Start() {
	if w == nil || w.db == nil {
		return
	}
	logger.SafeGo("audio-prefetch-warmup", func() {
		w.runOnce()
		t := time.NewTicker(w.interval)
		defer t.Stop()
		for {
			select {
			case <-w.stopCh:
				return
			case <-t.C:
				w.runOnce()
			}
		}
	})
}

// Stop ends the background loop.
func (w *AudioPrefetchWarmup) Stop() {
	if w == nil {
		return
	}
	w.once.Do(func() { close(w.stopCh) })
}

func (w *AudioPrefetchWarmup) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	urls := collectPrefetchURLs(ctx, w.db)
	ok, fail, skipped := 0, 0, 0
	for _, u := range urls {
		if welcomeaudio.HasDiskCache(u) {
			skipped++
			continue
		}
		if err := welcomeaudio.PrefetchURL(u); err != nil {
			fail++
			if logger.Lg != nil {
				logger.Lg.Warn("audio prefetch failed", zap.String("url", u), zap.Error(err))
			}
			continue
		}
		ok++
	}
	if logger.Lg != nil && (ok > 0 || fail > 0) {
		logger.Lg.Info("audio prefetch warmup",
			zap.Int("ok", ok),
			zap.Int("failed", fail),
			zap.Int("skipped", skipped),
			zap.Int("unique_urls", len(urls)),
			zap.String("cache_dir", welcomeaudio.DiskCacheDir()),
		)
	}
}

func collectPrefetchURLs(_ context.Context, _ *gorm.DB) []string {
	return nil
}

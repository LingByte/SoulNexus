package tasks

import (
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// LogRetentionCleaner periodically purges log files older than LOG_RETENTION_DAYS.
type LogRetentionCleaner struct {
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewLogRetentionCleaner(interval time.Duration) *LogRetentionCleaner {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &LogRetentionCleaner{
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (c *LogRetentionCleaner) Start() {
	if c == nil {
		return
	}
	c.wg.Add(1)
	go c.run()
}

func (c *LogRetentionCleaner) run() {
	defer c.wg.Done()
	safePurge := func() {
		defer func() {
			if r := recover(); r != nil && logger.Lg != nil {
				logger.Lg.Error("log retention cleaner panic recovered", zap.Any("panic", r))
			}
		}()
		c.purge()
	}
	safePurge()
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			safePurge()
		}
	}
}

func (c *LogRetentionCleaner) purge() {
	cfg := config.GlobalConfig.Log
	retentionDays := cfg.RetentionDays
	if retentionDays <= 0 {
		retentionDays = cfg.MaxAge
	}
	if retentionDays <= 0 {
		return
	}
	logDir := logger.LogDirFromFilename(cfg.Filename)
	removed, err := logger.PurgeExpiredLogFiles(logDir, retentionDays)
	if err != nil {
		if logger.Lg != nil {
			logger.Lg.Warn("log retention purge failed",
				zap.String("dir", logDir),
				zap.Int("retention_days", retentionDays),
				zap.Error(err),
			)
		}
		return
	}
	if removed > 0 && logger.Lg != nil {
		logger.Lg.Info("log retention purge finished",
			zap.String("dir", logDir),
			zap.Int("retention_days", retentionDays),
			zap.Int("removed", removed),
		)
	}
}

func (c *LogRetentionCleaner) Stop() {
	if c == nil {
		return
	}
	close(c.stopCh)
	c.wg.Wait()
}

package tasks

import (
	"context"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// StatsScheduler runs daily/monthly stats rollups.
type StatsScheduler struct {
	db   *gorm.DB
	stop chan struct{}
}

func NewStatsScheduler(db *gorm.DB) *StatsScheduler {
	return &StatsScheduler{db: db, stop: make(chan struct{})}
}

func (s *StatsScheduler) Start() {
	if s == nil || s.db == nil {
		return
	}
	go s.loop()
}

func (s *StatsScheduler) Stop() {
	if s == nil {
		return
	}
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
}

func (s *StatsScheduler) loop() {
	for {
		wait := durationUntilNextStatsTick(timeutil.Location())
		select {
		case <-s.stop:
			return
		case <-time.After(wait):
			s.runDueJobs()
		}
	}
}

func durationUntilNextStatsTick(loc *time.Location) time.Duration {
	if loc == nil {
		loc = time.Local
	}
	now := time.Now().In(loc)
	candidates := []time.Time{
		time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, loc),
		time.Date(now.Year(), now.Month(), now.Day(), 2, 30, 0, 0, loc),
	}
	var next time.Time
	for _, c := range candidates {
		if c.After(now) && (next.IsZero() || c.Before(next)) {
			next = c
		}
	}
	if next.IsZero() {
		next = time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, loc).Add(24 * time.Hour)
	}
	return time.Until(next)
}

func (s *StatsScheduler) runDueJobs() {
	loc := timeutil.Location()
	now := time.Now().In(loc)
	ctx := context.Background()

	runDaily := now.Hour() == 2 && now.Minute() < 15
	runMonthly := now.Day() == 1 && now.Hour() == 2 && now.Minute() >= 15 && now.Minute() < 45

	if runDaily {
		yesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -1)
		err := models.CollectAllTenantsDailyStats(ctx, s.db, yesterday)
		if err != nil && logger.Lg != nil {
			logger.Lg.Warn("stats cron: daily collect had failures",
				zap.String("day", yesterday.Format("2006-01-02")),
				zap.Error(err))
		} else if logger.Lg != nil {
			logger.Lg.Info("stats cron: daily collect done", zap.String("day", yesterday.Format("2006-01-02")))
		}
	}

	if runMonthly {
		lastMonth := now.AddDate(0, -1, 0)
		monthStart := time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, loc)
		var tenants []models.Tenant
		if err := s.db.Where("status = ?", constants.TenantStatusActive).Find(&tenants).Error; err != nil {
			if logger.Lg != nil {
				logger.Lg.Warn("stats cron: list tenants failed", zap.Error(err))
			}
			return
		}
		for _, t := range tenants {
			if t.ID == 0 {
				continue
			}
			if err := models.RollupTenantMonthlyStats(ctx, s.db, t.ID, monthStart); err != nil && logger.Lg != nil {
				logger.Lg.Warn("stats cron: monthly rollup failed", zap.Uint("tenant_id", t.ID), zap.Error(err))
			}
		}
		if logger.Lg != nil {
			logger.Lg.Info("stats cron: monthly rollup done", zap.String("month", monthStart.Format("2006-01")))
		}
	}
}

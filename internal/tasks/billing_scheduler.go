package tasks

import (
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BillingScheduler auto-generates finalized monthly tenant bills.
type BillingScheduler struct {
	db   *gorm.DB
	stop chan struct{}
}

func NewBillingScheduler(db *gorm.DB) *BillingScheduler {
	return &BillingScheduler{db: db, stop: make(chan struct{})}
}

func (s *BillingScheduler) Start() {
	if s == nil || s.db == nil {
		return
	}
	go s.loop()
}

func (s *BillingScheduler) Stop() {
	if s == nil {
		return
	}
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
}

func (s *BillingScheduler) loop() {
	s.runOnce()
	for {
		wait := durationUntilNextDayAt(timeutil.Location(), 2, 10)
		select {
		case <-s.stop:
			return
		case <-time.After(wait):
			s.runOnce()
		}
	}
}

func (s *BillingScheduler) runOnce() {
	if err := models.EnsureAllTenantsBillsUpToDate(s.db); err != nil && logger.Lg != nil {
		logger.Lg.Warn("billing auto-generate failed", zap.Error(err))
	}
}

func durationUntilNextDayAt(loc *time.Location, hour, minute int) time.Duration {
	if loc == nil {
		loc = time.Local
	}
	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return time.Until(next)
}

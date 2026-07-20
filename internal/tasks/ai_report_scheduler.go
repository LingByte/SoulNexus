package tasks

import "gorm.io/gorm"

// AIReportScheduler previously generated telephony-centric AI reports (removed).
type AIReportScheduler struct {
	db   *gorm.DB
	stop chan struct{}
}

func NewAIReportScheduler(db *gorm.DB) *AIReportScheduler {
	return &AIReportScheduler{db: db, stop: make(chan struct{})}
}

func (s *AIReportScheduler) Start() {}

func (s *AIReportScheduler) Stop() {
	if s == nil {
		return
	}
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
}

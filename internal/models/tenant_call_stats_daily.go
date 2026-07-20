// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package models

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TenantCallStatsDaily stores pre-aggregated per-tenant call analytics for one calendar day.
type TenantCallStatsDaily struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	TenantID uint      `json:"tenantId" gorm:"uniqueIndex:idx_tenant_call_stats_day;not null;index"`
	StatDate time.Time `json:"statDate" gorm:"type:date;uniqueIndex:idx_tenant_call_stats_day;not null;index"`

	CallCount        int64 `json:"callCount"`
	ConnectedCount   int64 `json:"connectedCount"`
	BilledMinutesSum int64 `json:"billedMinutesSum"`
	DurationSecSum   int64 `json:"durationSecSum"`
	TurnCountSum     int64 `json:"turnCountSum"`
	AIToHumanCount   int64 `json:"aiToHumanCount"`
	PureAICount      int64 `json:"pureAiCount"`
	CallsWithKB      int64 `json:"callsWithKb"`
	CallsQuoted      int64 `json:"callsQuoted"`

	TurnSampleCount int64 `json:"turnSampleCount"`
	LLMFirstMsSum   int64 `json:"llmFirstMsSum"`
	LLMWallMsSum    int64 `json:"llmWallMsSum"`
	TTSMsSum        int64 `json:"ttsMsSum"`
	PipelineMsSum   int64 `json:"pipelineMsSum"`
	RecallMsSum     int64 `json:"recallMsSum"`
	RecallSampleN   int64 `json:"recallSampleN"`
	RAGHitTurns     int64 `json:"ragHitTurns"`
	RAGMissTurns    int64 `json:"ragMissTurns"`
	InterruptCount  int64 `json:"interruptCount"`

	TransferAttemptCount  int64 `json:"transferAttemptCount"`
	TransferAnsweredCount int64 `json:"transferAnsweredCount"`

	DurationBucketsJSON     datatypes.JSON `json:"durationBucketsJson,omitempty" gorm:"type:json"`
	TurnBucketsJSON         datatypes.JSON `json:"turnBucketsJson,omitempty" gorm:"type:json"`
	ProvinceBucketsJSON     datatypes.JSON `json:"provinceBucketsJson,omitempty" gorm:"type:json"`
	EndStatusBucketsJSON    datatypes.JSON `json:"endStatusBucketsJson,omitempty" gorm:"type:json"`
	TransferOutcomeJSON     datatypes.JSON `json:"transferOutcomeJson,omitempty" gorm:"type:json"`
	TransferReasonJSON      datatypes.JSON `json:"transferReasonJson,omitempty" gorm:"type:json"`
}

// AggregateDailyStatsForDay is a no-op after voice session aggregation was removed.
func AggregateDailyStatsForDay(ctx context.Context, db *gorm.DB, tenantID uint, day time.Time) error {
	_ = ctx
	_ = db
	_ = tenantID
	_ = day
	return nil
}


func normalizeStatDate(day time.Time) time.Time {
	if day.IsZero() {
		return day
	}
	loc := day.Location()
	if loc == nil {
		loc = time.Local
	}
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
}

func GetTenantCallStatsDaily(db *gorm.DB, tenantID uint, day time.Time) (*TenantCallStatsDaily, error) {
	day = normalizeStatDate(day)
	var row TenantCallStatsDaily
	err := db.Where("tenant_id = ? AND stat_date >= ? AND stat_date < ?", tenantID, day, day.AddDate(0, 0, 1)).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func ListTenantCallStatsDaily(db *gorm.DB, tenantID uint, startDay, endDay time.Time) ([]TenantCallStatsDaily, error) {
	startDay = normalizeStatDate(startDay)
	endDay = normalizeStatDate(endDay)
	var rows []TenantCallStatsDaily
	err := db.Where("tenant_id = ? AND stat_date >= ? AND stat_date < ?", tenantID, startDay, endDay.AddDate(0, 0, 1)).
		Order("stat_date ASC").
		Find(&rows).Error
	return rows, err
}

// SaveTenantCallStatsDaily upserts by (tenant_id, stat_date): load existing id first, then update or insert.
// On a create race it returns the unique-conflict error so callers (Increment) can retry from a fresh load
// instead of overwriting a full Aggregate rebuild with a one-call partial row.
func SaveTenantCallStatsDaily(db *gorm.DB, row *TenantCallStatsDaily) error {
	if db == nil || row == nil {
		return nil
	}
	row.StatDate = normalizeStatDate(row.StatDate)
	if row.ID == 0 {
		var existing TenantCallStatsDaily
		err := db.Select("id", "created_at").
			Where("tenant_id = ? AND stat_date >= ? AND stat_date < ?", row.TenantID, row.StatDate, row.StatDate.AddDate(0, 0, 1)).
			First(&existing).Error
		if err == nil {
			row.ID = existing.ID
			row.CreatedAt = existing.CreatedAt
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	if row.ID != 0 {
		return db.Save(row).Error
	}
	return db.Create(row).Error
}

// ReplaceTenantCallStatsDaily inserts or fully replaces the day row (Aggregate / backfill).
// Safe under multi-replica 02:00 cron: ON DUPLICATE KEY / ON CONFLICT updates all columns.
func ReplaceTenantCallStatsDaily(db *gorm.DB, row *TenantCallStatsDaily) error {
	if db == nil || row == nil {
		return nil
	}
	row.StatDate = normalizeStatDate(row.StatDate)
	err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_id"},
			{Name: "stat_date"},
		},
		UpdateAll: true,
	}).Create(row).Error
	if err == nil {
		return nil
	}
	if !isUniqueConflict(err) {
		return err
	}
	// Fallback when the dialect does not translate OnConflict.
	var existing TenantCallStatsDaily
	if e2 := db.Select("id", "created_at").
		Where("tenant_id = ? AND stat_date >= ? AND stat_date < ?", row.TenantID, row.StatDate, row.StatDate.AddDate(0, 0, 1)).
		First(&existing).Error; e2 != nil {
		return err
	}
	row.ID = existing.ID
	row.CreatedAt = existing.CreatedAt
	return db.Save(row).Error
}

func isUniqueConflict(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate entry") ||
		strings.Contains(msg, "unique constraint failed") ||
		strings.Contains(msg, "duplicated key") ||
		strings.Contains(msg, "1062")
}

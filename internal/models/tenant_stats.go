// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package models

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Tenant stats dimensions
const (
	TenantStatsDimensionDay   = "day"
	TenantStatsDimensionMonth = "month"

	TenantStatsTypeAssistant = "assistant"
	TenantStatsTypeKnowledge = "knowledge"
	TenantStatsTypeCallout   = "callout"
)

// TenantStats stores pre-aggregated tenant analytics (day/month rollups).
type TenantStats struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	TenantID     uint      `json:"tenantId" gorm:"uniqueIndex:idx_tenant_stats_key;not null;index"`
	AssistantID  uint      `json:"assistantId" gorm:"uniqueIndex:idx_tenant_stats_key;not null;default:0;index"`
	CallerNumber string    `json:"callerNumber" gorm:"type:varchar(64);uniqueIndex:idx_tenant_stats_key;not null;default:'';index"`
	StatsType    string    `json:"statsType" gorm:"type:varchar(20);uniqueIndex:idx_tenant_stats_key;not null;index"`
	Dimension    string    `json:"dimension" gorm:"type:varchar(20);uniqueIndex:idx_tenant_stats_key;not null;index"`
	StatsDate    time.Time `json:"statsDate" gorm:"type:date;uniqueIndex:idx_tenant_stats_key;not null;index"`

	StartTime time.Time      `json:"startTime" gorm:"not null"`
	EndTime   time.Time      `json:"endTime" gorm:"not null"`
	StatsData datatypes.JSON `json:"statsData" gorm:"type:json"`
}

func (TenantStats) TableName() string { return "tenant_stats" }

// AssistantStatsPayload mirrors AssistantStatsData (subset used by AI reports).
type AssistantStatsPayload struct {
	TotalCalls            int64   `json:"totalCalls"`
	TotalDuration         int64   `json:"totalDuration"`
	ConnectedCount        int64   `json:"connectedCount"`
	AverageCallDuration   float64 `json:"averageCallDuration"`
	AverageTurnCount      float64 `json:"averageTurnCount"`
	OneTimeResolutionRate float64 `json:"oneTimeResolutionRate"`
	AverageResponseDelay  float64 `json:"averageResponseDelay"`
	TransferToHumanCount  int64   `json:"transferToHumanCount"`
	TransferToHumanRate   float64 `json:"transferToHumanRate"`
	KnowledgeQuotedRate   float64 `json:"knowledgeQuotedRate"`
	NonWorkingHoursCalls  int64   `json:"nonWorkingHoursCalls"`
	RobotSavedManDays     float64 `json:"robotSavedManDays"`
	DurationBuckets       []int64 `json:"durationBuckets,omitempty"`
	TurnBuckets           []int64 `json:"turnBuckets,omitempty"`
}

// CalloutStatsPayload mirrors CalloutStatsData.
type CalloutStatsPayload struct {
	CallerNumber       string  `json:"callerNumber"`
	TotalAttempts      int64   `json:"totalAttempts"`
	TotalAnswered      int64   `json:"totalAnswered"`
	TotalUnanswered    int64   `json:"totalUnanswered"`
	TotalDurationSec   int64   `json:"totalDurationSec"`
	AverageDurationSec float64 `json:"averageDurationSec"`
	AnsweredPercent    float64 `json:"answeredPercent"`
	CompletedContacts  int64   `json:"completedContacts"`
	HighIntentCount    int64   `json:"highIntentCount"`
	MediumIntentCount  int64   `json:"mediumIntentCount"`
	LowIntentCount     int64   `json:"lowIntentCount"`
}

// KnowledgeStatsPayload mirrors KnowledgeStatsData (tenant-level).
type KnowledgeStatsPayload struct {
	OverallQuotedRate    float64 `json:"overallQuotedRate"`
	OverallNotQuotedRate float64 `json:"overallNotQuotedRate"`
	RecallTotal          int64   `json:"recallTotal"`
	UnansweredNew        int64   `json:"unansweredNew"`
	UnansweredOpen       int64   `json:"unansweredOpen"`
}

func SaveTenantStats(db *gorm.DB, row *TenantStats) error {
	if db == nil || row == nil {
		return gorm.ErrInvalidDB
	}
	var existing TenantStats
	err := db.Where(
		"tenant_id = ? AND assistant_id = ? AND caller_number = ? AND stats_type = ? AND dimension = ? AND stats_date = ?",
		row.TenantID, row.AssistantID, row.CallerNumber, row.StatsType, row.Dimension, row.StatsDate.Format("2006-01-02"),
	).First(&existing).Error
	if err == nil {
		row.ID = existing.ID
		return db.Save(row).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	return db.Create(row).Error
}

func ListTenantStats(db *gorm.DB, tenantID uint, statsType, dimension string, startDay, endDay time.Time) ([]TenantStats, error) {
	var rows []TenantStats
	err := db.Where(
		"tenant_id = ? AND stats_type = ? AND dimension = ? AND stats_date >= ? AND stats_date <= ?",
		tenantID, statsType, dimension, startDay.Format("2006-01-02"), endDay.Format("2006-01-02"),
	).Order("stats_date ASC").Find(&rows).Error
	return rows, err
}

func EncodeStatsPayload(v any) datatypes.JSON {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return datatypes.JSON(b)
}

func DecodeStatsPayload[T any](j datatypes.JSON, out *T) {
	if len(j) == 0 || out == nil {
		return
	}
	_ = json.Unmarshal(j, out)
}

package models

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	knconst "github.com/LingByte/SoulNexus/pkg/knowledge/constants"
	knmodels "github.com/LingByte/SoulNexus/pkg/knowledge/models"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"gorm.io/gorm"
)

// Stable bucket key order for assistant stats charts (legacy daily rollups).
var (
	durationBucketOrder = []string{"0-10", "10-30", "30-60", "60-180", "180+"}
	turnBucketOrder     = []string{"1", "2-3", "4-6", "7-10", "10+"}
)

// CollectTenantDailyStats rebuilds yesterday's rollups for one tenant
func CollectTenantDailyStats(ctx context.Context, db *gorm.DB, tenantID uint, day time.Time) error {
	if db == nil || tenantID == 0 {
		return gorm.ErrInvalidDB
	}
	loc := timeutil.Location()
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	endExclusive := day.AddDate(0, 0, 1)

	if err := AggregateDailyStatsForDay(ctx, db, tenantID, day); err != nil {
		return err
	}

	daily, err := GetTenantCallStatsDaily(db.WithContext(ctx), tenantID, day)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	asst := buildAssistantStatsPayload(daily)
	if err := SaveTenantStats(db.WithContext(ctx), &TenantStats{
		TenantID:  tenantID,
		StatsType: TenantStatsTypeAssistant,
		Dimension: TenantStatsDimensionDay,
		StatsDate: day,
		StartTime: day,
		EndTime:   endExclusive.Add(-time.Nanosecond),
		StatsData: EncodeStatsPayload(asst),
	}); err != nil {
		return err
	}

	calloutRows, err := collectCalloutStatsForDay(ctx, db, tenantID, day, endExclusive)
	if err != nil {
		return err
	}
	for _, row := range calloutRows {
		if err := SaveTenantStats(db.WithContext(ctx), &row); err != nil {
			return err
		}
	}

	kb, err := buildKnowledgeStatsPayload(ctx, db, tenantID, day, endExclusive)
	if err != nil {
		return err
	}
	return SaveTenantStats(db.WithContext(ctx), &TenantStats{
		TenantID:  tenantID,
		StatsType: TenantStatsTypeKnowledge,
		Dimension: TenantStatsDimensionDay,
		StatsDate: day,
		StartTime: day,
		EndTime:   endExclusive.Add(-time.Nanosecond),
		StatsData: EncodeStatsPayload(kb),
	})
}

// CollectAllTenantsDailyStats runs daily aggregation for every active tenant.
// Per-tenant failures are joined so one bad tenant cannot abort the rest.
func CollectAllTenantsDailyStats(ctx context.Context, db *gorm.DB, day time.Time) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	var tenants []Tenant
	if err := db.WithContext(ctx).Where("status = ?", constants.TenantStatusActive).Find(&tenants).Error; err != nil {
		return err
	}
	var errs []error
	for _, t := range tenants {
		if t.ID == 0 {
			continue
		}
		if err := CollectTenantDailyStats(ctx, db, t.ID, day); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// RollupTenantMonthlyStats folds day rows into month rows for the given calendar month.
func RollupTenantMonthlyStats(ctx context.Context, db *gorm.DB, tenantID uint, month time.Time) error {
	if db == nil || tenantID == 0 {
		return gorm.ErrInvalidDB
	}
	loc := timeutil.Location()
	month = time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, loc)
	endDay := month.AddDate(0, 1, -1)
	dailyRows, err := ListTenantStats(db.WithContext(ctx), tenantID, TenantStatsTypeAssistant, TenantStatsDimensionDay, month, endDay)
	if err != nil {
		return err
	}
	if len(dailyRows) == 0 {
		return nil
	}
	var merged AssistantStatsPayload
	for _, r := range dailyRows {
		var p AssistantStatsPayload
		DecodeStatsPayload(r.StatsData, &p)
		merged.TotalCalls += p.TotalCalls
		merged.TotalDuration += p.TotalDuration
		merged.ConnectedCount += p.ConnectedCount
		merged.TransferToHumanCount += p.TransferToHumanCount
		merged.NonWorkingHoursCalls += p.NonWorkingHoursCalls
		for i, v := range p.DurationBuckets {
			if i < len(merged.DurationBuckets) {
				merged.DurationBuckets[i] += v
			} else {
				merged.DurationBuckets = append(merged.DurationBuckets, v)
			}
		}
		for i, v := range p.TurnBuckets {
			if i < len(merged.TurnBuckets) {
				merged.TurnBuckets[i] += v
			} else {
				merged.TurnBuckets = append(merged.TurnBuckets, v)
			}
		}
	}
	if merged.ConnectedCount > 0 {
		merged.AverageCallDuration = float64(merged.TotalDuration) / float64(merged.ConnectedCount)
	}
	if merged.TotalCalls > 0 {
		merged.TransferToHumanRate = float64(merged.TransferToHumanCount) / float64(merged.TotalCalls) * 100
	}
	endExclusive := month.AddDate(0, 1, 0)
	return SaveTenantStats(db.WithContext(ctx), &TenantStats{
		TenantID:  tenantID,
		StatsType: TenantStatsTypeAssistant,
		Dimension: TenantStatsDimensionMonth,
		StatsDate: month,
		StartTime: month,
		EndTime:   endExclusive.Add(-time.Nanosecond),
		StatsData: EncodeStatsPayload(merged),
	})
}

func buildAssistantStatsPayload(daily *TenantCallStatsDaily) AssistantStatsPayload {
	var p AssistantStatsPayload
	if daily == nil {
		return p
	}
	p.TotalCalls = daily.CallCount
	p.ConnectedCount = daily.ConnectedCount
	p.TotalDuration = daily.DurationSecSum
	p.TransferToHumanCount = daily.AIToHumanCount
	if daily.ConnectedCount > 0 {
		p.AverageCallDuration = float64(daily.DurationSecSum) / float64(daily.ConnectedCount)
		p.AverageTurnCount = float64(daily.TurnCountSum) / float64(daily.ConnectedCount)
		p.OneTimeResolutionRate = float64(daily.PureAICount) / float64(daily.ConnectedCount) * 100
	}
	if daily.CallCount > 0 {
		p.TransferToHumanRate = float64(daily.AIToHumanCount) / float64(daily.CallCount) * 100
	}
	if daily.ConnectedCount > 0 && daily.CallsQuoted > 0 {
		p.KnowledgeQuotedRate = float64(daily.CallsQuoted) / float64(daily.ConnectedCount) * 100
	}
	if daily.TurnSampleCount > 0 && daily.PipelineMsSum > 0 {
		p.AverageResponseDelay = float64(daily.PipelineMsSum) / float64(daily.TurnSampleCount)
	}
	p.DurationBuckets = bucketJSONToSlice(daily.DurationBucketsJSON, durationBucketOrder)
	p.TurnBuckets = bucketJSONToSlice(daily.TurnBucketsJSON, turnBucketOrder)
	// Rough saved-man-days heuristic
	if daily.BilledMinutesSum > 0 {
		p.RobotSavedManDays = float64(daily.BilledMinutesSum) / 480.0
	}
	return p
}

func bucketJSONToSlice(j []byte, order []string) []int64 {
	if len(j) == 0 {
		return nil
	}
	var m map[string]int64
	if err := json.Unmarshal(j, &m); err != nil {
		return nil
	}
	out := make([]int64, 0, len(order))
	for _, k := range order {
		out = append(out, m[k])
	}
	return out
}

func buildKnowledgeStatsPayload(ctx context.Context, db *gorm.DB, tenantID uint, day, endExclusive time.Time) (KnowledgeStatsPayload, error) {
	var p KnowledgeStatsPayload
	_ = db.WithContext(ctx).Model(&knmodels.KnowledgeUnansweredQuestion{}).
		Where("group_id = ? AND created_at >= ? AND created_at < ?", tenantID, day, endExclusive).
		Count(&p.UnansweredNew).Error
	_ = db.WithContext(ctx).Model(&knmodels.KnowledgeUnansweredQuestion{}).
		Where("group_id = ? AND status = ?", tenantID, knconst.KnowledgeUnansweredStatusOpen).
		Count(&p.UnansweredOpen).Error
	return p, nil
}

func collectCalloutStatsForDay(_ context.Context, _ *gorm.DB, _ uint, _, _ time.Time) ([]TenantStats, error) {
	return nil, nil
}

func avgSec(total, n int64) float64 {
	if n <= 0 {
		return 0
	}
	return float64(total) / float64(n)
}

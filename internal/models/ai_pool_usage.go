package models

import (
	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"gorm.io/gorm"
)

// RecordAIPoolTransitUsage increments tenant grant + global pool quota after a settled call.
func RecordAIPoolTransitUsage(db *gorm.DB, tenantID uint, callID string, durationSec int) error {
	if db == nil || tenantID == 0 {
		return nil
	}
	poolIDs := callbinding.GetTransitPoolIDs(callID)
	if len(poolIDs) == 0 {
		return nil
	}
	credits := billedMinutesFromDurationSec(durationSec)
	if credits < 1 {
		credits = 1
	}
	seen := map[uint]struct{}{}
	for _, poolID := range poolIDs {
		if poolID == 0 {
			continue
		}
		if _, dup := seen[poolID]; dup {
			continue
		}
		seen[poolID] = struct{}{}
		if err := db.Exec(
			"UPDATE ai_provider_pools SET quota_used = quota_used + ? WHERE id = ?",
			credits, poolID,
		).Error; err != nil {
			return err
		}
		var grant TenantAIPoolGrant
		err := db.Where("tenant_id = ? AND pool_id = ?", tenantID, poolID).First(&grant).Error
		if err != nil {
			continue
		}
		if err := db.Model(&TenantAIPoolGrant{}).
			Where("id = ?", grant.ID).
			UpdateColumn("quota_used", gorm.Expr("quota_used + ?", credits)).Error; err != nil {
			return err
		}
	}
	callbinding.ClearTransitPoolIDs(callID)
	return nil
}

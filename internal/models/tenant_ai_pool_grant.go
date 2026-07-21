package models

import (
	constants2 "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

// TenantAIPoolGrant assigns a platform 号池 to a tenant with an optional quota (中转配额).
type TenantAIPoolGrant struct {
	common.BaseModel
	TenantID   uint  `json:"tenantId" gorm:"index:idx_tenant_pool_grant,unique;not null"`
	PoolID     uint  `json:"poolId" gorm:"index:idx_tenant_pool_grant,unique;not null"`
	QuotaLimit int64 `json:"quotaLimit" gorm:"column:quota_limit;not null;default:0"` // 0 = unlimited
	QuotaUsed  int64 `json:"quotaUsed" gorm:"column:quota_used;not null;default:0"`
	Enabled    bool  `json:"enabled" gorm:"not null;default:true;index"`
}

func (TenantAIPoolGrant) TableName() string {
	return constants2.TENANT_AI_POOL_GRANT_TABLE_NAME
}

func grantHasQuota(g TenantAIPoolGrant) bool {
	if g.QuotaLimit <= 0 {
		return true
	}
	return g.QuotaUsed < g.QuotaLimit
}

// TenantHasActivePoolGrant reports whether the tenant may use platform transit (号池).
func TenantHasActivePoolGrant(db *gorm.DB, tenantID uint) bool {
	if db == nil || tenantID == 0 {
		return false
	}
	var grants []TenantAIPoolGrant
	if err := db.Where("tenant_id = ? AND enabled = ?", tenantID, true).Limit(64).Find(&grants).Error; err != nil {
		return false
	}
	for _, g := range grants {
		if !grantHasQuota(g) {
			continue
		}
		var pool AIProviderPool
		if err := db.Where("id = ? AND enabled = ?", g.PoolID, true).First(&pool).Error; err != nil {
			continue
		}
		if poolHasQuota(pool) {
			return true
		}
	}
	return false
}

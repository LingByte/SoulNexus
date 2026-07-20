package models

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

// BillingPlan is a platform-managed cost tariff template (会话/LLM 单价方案).
type BillingPlan struct {
	common.BaseModel

	Name               string  `json:"name" gorm:"size:128;not null"`
	Description        string  `json:"description,omitempty" gorm:"size:512"`
	Currency           string  `json:"currency" gorm:"size:8;not null;default:CNY"`
	CallRatePerMinute  float64 `json:"callRatePerMinute" gorm:"column:call_rate_per_minute;type:decimal(16,6);not null;default:0"`
	LLMRatePer1kTokens float64 `json:"llmRatePer1kTokens" gorm:"column:llm_rate_per_1k_tokens;type:decimal(16,6);not null;default:0"`
	Status             string  `json:"status" gorm:"size:24;index;not null;default:active"` // active | disabled
}

func (BillingPlan) TableName() string {
	return constants.BILLING_PLAN_TABLE_NAME
}

func ListBillingPlansPage(db *gorm.DB, page, size int, nameContains string) ([]BillingPlan, int64, error) {
	q := db.Model(&BillingPlan{})
	if name := strings.TrimSpace(nameContains); name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}
	return utils.FindPage[BillingPlan](q, page, size, "id DESC", utils.DefaultMaxPageSize)
}

func GetBillingPlanByID(db *gorm.DB, id uint) (BillingPlan, error) {
	var row BillingPlan
	err := db.Where("id = ?", id).First(&row).Error
	return row, err
}

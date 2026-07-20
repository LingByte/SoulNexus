package models

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

const (
	LoginHistoryPrincipalTenantUser    = UserDevicePrincipalTenantUser
	LoginHistoryPrincipalPlatformAdmin = UserDevicePrincipalPlatformAdmin
)

// LoginHistory records one login attempt outcome for tenant users and platform admins.
type LoginHistory struct {
	common.BaseModel

	PrincipalType string `json:"principalType" gorm:"size:32;index;not null"`
	PrincipalID   uint   `json:"principalId,string" gorm:"index;not null"`
	TenantID      uint   `json:"tenantId,string" gorm:"index;not null;default:0"`
	Email         string `json:"email" gorm:"size:128;index"`
	ClientIP      string `json:"clientIp" gorm:"size:64;index"`
	City          string `json:"city" gorm:"size:64"`
	Location      string `json:"location" gorm:"size:256"`
	UserAgent     string `json:"userAgent" gorm:"size:512"`
	LoginMethod   string `json:"loginMethod" gorm:"size:32;index"`
	Success       bool   `json:"success" gorm:"index;not null;default:true"`
	FailureReason string `json:"failureReason,omitempty" gorm:"size:256"`
	DeviceKey     string `json:"deviceKey,omitempty" gorm:"size:128;index"`
}

func (LoginHistory) TableName() string {
	return constants.LOGIN_HISTORY_TABLE_NAME
}

// LoginHistoryInput is the write payload for a login event.
type LoginHistoryInput struct {
	PrincipalType string
	PrincipalID   uint
	TenantID      uint
	Email         string
	ClientIP      string
	City          string
	Location      string
	UserAgent     string
	LoginMethod   string
	Success       bool
	FailureReason string
	DeviceKey     string
	Operator      string
}

// RecordLoginHistory appends one login history row.
func RecordLoginHistory(db *gorm.DB, in LoginHistoryInput) error {
	if db == nil || in.PrincipalID == 0 || strings.TrimSpace(in.PrincipalType) == "" {
		return nil
	}
	row := LoginHistory{
		PrincipalType: strings.TrimSpace(in.PrincipalType),
		PrincipalID:   in.PrincipalID,
		TenantID:      in.TenantID,
		Email:         strings.TrimSpace(in.Email),
		ClientIP:      strings.TrimSpace(in.ClientIP),
		City:          strings.TrimSpace(in.City),
		Location:      strings.TrimSpace(in.Location),
		UserAgent:     truncateRunes(in.UserAgent, 500),
		LoginMethod:   strings.TrimSpace(in.LoginMethod),
		Success:       in.Success,
		FailureReason: truncateRunes(in.FailureReason, 250),
		DeviceKey:     strings.TrimSpace(in.DeviceKey),
	}
	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "system:login"
	}
	row.SetCreateInfo(op)
	return db.Create(&row).Error
}

// ListLoginHistoryPage returns paginated login history for one principal.
func ListLoginHistoryPage(db *gorm.DB, principalType string, principalID uint, page, size int) ([]LoginHistory, int64, error) {
	q := db.Model(&LoginHistory{}).Where("principal_type = ? AND principal_id = ?", principalType, principalID)
	return utils.FindPage[LoginHistory](q, page, size, "id DESC", utils.DefaultMaxPageSize)
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

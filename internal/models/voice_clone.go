package models

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

const (
	VoiceCloneStatusPending  = "pending"
	VoiceCloneStatusTraining = "training"
	VoiceCloneStatusSuccess  = "success"
	VoiceCloneStatusFailed   = "failed"
)

// VoiceCloneProfile is a tenant-scoped cloned timbre trained via Xunfei / Volcengine.
type VoiceCloneProfile struct {
	common.BaseModel

	TenantID     uint   `json:"tenantId" gorm:"index;not null"`
	Name         string `json:"name" gorm:"size:128;not null"`
	Provider     string `json:"provider" gorm:"size:32;index;not null"`
	Status       string `json:"status" gorm:"size:24;index;not null;default:pending"`
	TaskID       string `json:"taskId,omitempty" gorm:"size:128;index"`
	AssetID      string `json:"assetId,omitempty" gorm:"size:128;index"`
	SpeakerID    string `json:"speakerId,omitempty" gorm:"size:128;index"`
	Sex          int    `json:"sex,omitempty"`
	Language     string `json:"language,omitempty" gorm:"size:16"`
	TrainText    string `json:"trainText,omitempty" gorm:"size:512"`
	TextID       int64  `json:"textId,omitempty"`
	TextSegID    int64  `json:"textSegId,omitempty"`
	FailedReason string `json:"failedReason,omitempty" gorm:"type:text"`
	Progress     float64 `json:"progress,omitempty"`
}

func (VoiceCloneProfile) TableName() string {
	return constants.VOICE_CLONE_PROFILE_TABLE_NAME
}

func ListVoiceCloneProfiles(db *gorm.DB, tenantID uint, status string) ([]VoiceCloneProfile, error) {
	if db == nil {
		return nil, gorm.ErrInvalidDB
	}
	q := db.Where("tenant_id = ?", tenantID).Order("id DESC")
	if s := strings.TrimSpace(status); s != "" {
		q = q.Where("status = ?", s)
	}
	var rows []VoiceCloneProfile
	return rows, q.Find(&rows).Error
}

func GetVoiceCloneProfile(db *gorm.DB, tenantID, id uint) (VoiceCloneProfile, error) {
	var row VoiceCloneProfile
	err := db.Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	return row, err
}

func CreateVoiceCloneProfile(db *gorm.DB, row *VoiceCloneProfile) error {
	if row == nil {
		return gorm.ErrInvalidData
	}
	return db.Create(row).Error
}

func UpdateVoiceCloneProfile(db *gorm.DB, id uint, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	return db.Model(&VoiceCloneProfile{}).Where("id = ?", id).Updates(updates).Error
}

func DeleteVoiceCloneProfile(db *gorm.DB, tenantID, id uint) error {
	return db.Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&VoiceCloneProfile{}).Error
}

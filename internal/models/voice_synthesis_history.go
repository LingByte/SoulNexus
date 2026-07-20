package models

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

const (
	VoiceSynthesisStatusSuccess = "success"
	VoiceSynthesisStatusFailed  = "failed"
)

// VoiceSynthesisHistory records a tenant voice-clone synthesis attempt.
type VoiceSynthesisHistory struct {
	common.BaseModel

	TenantID     uint   `json:"tenantId" gorm:"index;not null"`
	ProfileID    uint   `json:"profileId" gorm:"index"`
	Provider     string `json:"provider" gorm:"size:32;index"`
	AssetID      string `json:"assetId" gorm:"size:128;index"`
	VoiceName    string `json:"voiceName" gorm:"size:128"`
	Text         string `json:"text" gorm:"type:text"`
	SampleRate   int    `json:"sampleRate"`
	AudioKey     string `json:"audioKey,omitempty" gorm:"size:512"`
	AudioURL     string `json:"audioUrl,omitempty" gorm:"-"`
	Status       string `json:"status" gorm:"size:24;index;not null;default:success"`
	ErrorMessage string `json:"errorMessage,omitempty" gorm:"type:text"`
	CreatedBy    uint   `json:"createdBy,omitempty"`
}

func (VoiceSynthesisHistory) TableName() string {
	return constants.VOICE_SYNTHESIS_HISTORY_TABLE_NAME
}

func CreateVoiceSynthesisHistory(db *gorm.DB, row *VoiceSynthesisHistory) error {
	if row == nil {
		return gorm.ErrInvalidData
	}
	return db.Create(row).Error
}

func ListVoiceSynthesisHistory(db *gorm.DB, tenantID uint, profileID uint, limit int) ([]VoiceSynthesisHistory, error) {
	if db == nil {
		return nil, gorm.ErrInvalidDB
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := db.Where("tenant_id = ?", tenantID).Order("id DESC").Limit(limit)
	if profileID > 0 {
		q = q.Where("profile_id = ?", profileID)
	}
	var rows []VoiceSynthesisHistory
	return rows, q.Find(&rows).Error
}

func ListAllVoiceSynthesisHistory(db *gorm.DB, tenantID uint, limit, offset int) ([]VoiceSynthesisHistory, int64, error) {
	if db == nil {
		return nil, 0, gorm.ErrInvalidDB
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	q := db.Model(&VoiceSynthesisHistory{})
	if tenantID > 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []VoiceSynthesisHistory
	err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func GetVoiceSynthesisHistory(db *gorm.DB, id uint) (VoiceSynthesisHistory, error) {
	var row VoiceSynthesisHistory
	err := db.Where("id = ?", id).First(&row).Error
	return row, err
}

func DeleteVoiceSynthesisHistory(db *gorm.DB, id uint) error {
	return db.Where("id = ?", id).Delete(&VoiceSynthesisHistory{}).Error
}

func DeleteVoiceSynthesisHistoryForTenant(db *gorm.DB, tenantID, id uint) error {
	if db == nil || tenantID == 0 || id == 0 {
		return gorm.ErrInvalidData
	}
	res := db.Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&VoiceSynthesisHistory{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func FindVoiceCloneProfileByVoiceID(db *gorm.DB, tenantID uint, voiceID string) (VoiceCloneProfile, error) {
	voiceID = strings.TrimSpace(voiceID)
	var row VoiceCloneProfile
	if db == nil || tenantID == 0 || voiceID == "" {
		return row, gorm.ErrInvalidData
	}
	err := db.Where(
		"tenant_id = ? AND status = ? AND (asset_id = ? OR speaker_id = ?)",
		tenantID, VoiceCloneStatusSuccess, voiceID, voiceID,
	).First(&row).Error
	return row, err
}

func ListAllVoiceCloneProfiles(db *gorm.DB, tenantID uint, limit, offset int) ([]VoiceCloneProfile, int64, error) {
	if db == nil {
		return nil, 0, gorm.ErrInvalidDB
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	q := db.Model(&VoiceCloneProfile{})
	if tenantID > 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []VoiceCloneProfile
	err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

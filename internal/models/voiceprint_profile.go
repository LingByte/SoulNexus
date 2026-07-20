package models

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

const (
	VoiceprintStatusActive = "active"
	VoiceprintStatusFailed = "failed"

	// VoiceprintSceneBusiness 租户业务音色，展示在声纹工作台，可绑定助手（一对多：一助手多声纹）。
	VoiceprintSceneBusiness = "business"
	// VoiceprintSceneAccount 用户账号内部声纹，用于登录二次校验，不在业务列表展示。
	VoiceprintSceneAccount = "account"
)

// VoiceprintProfile is a tenant-scoped voiceprint row (business timbre or account-internal).
// Account ownership is on tenant_users.voiceprint_id, not on this table.
// AssistantID optional: nil = tenant-wide; set = bound to one assistant (one assistant → many voiceprints).
type VoiceprintProfile struct {
	common.BaseModel

	TenantID      uint   `json:"tenantId" gorm:"uniqueIndex:uk_tenant_feature,priority:1;index;not null"`
	AssistantID   *uint  `json:"assistantId,string,omitempty" gorm:"index"`
	SubjectID     *uint  `json:"subjectId,string,omitempty" gorm:"index"` // logical speaker; optional until bound
	Scene         string `json:"scene" gorm:"size:32;index;not null;default:business"`
	Name          string `json:"name" gorm:"size:128;not null"`
	Provider      string `json:"provider" gorm:"size:32;index;not null"`
	FeatureID     string `json:"featureId" gorm:"size:128;index;not null;uniqueIndex:uk_tenant_feature,priority:2"`
	Status        string `json:"status" gorm:"size:24;index;not null;default:active"`
	Description   string `json:"description,omitempty" gorm:"size:512"`
	FeatureVector []byte `json:"-"` // dialect-portable blob (sqlite/pg/mysql via GORM)
}

func (VoiceprintProfile) TableName() string {
	return constants.VOICEPRINT_PROFILE_TABLE_NAME
}

func normalizeVoiceprintScene(scene string) string {
	s := strings.TrimSpace(scene)
	if s == VoiceprintSceneAccount {
		return VoiceprintSceneAccount
	}
	return VoiceprintSceneBusiness
}

func businessVoiceprintsQuery(db *gorm.DB, tenantID uint) *gorm.DB {
	q := db.Model(&VoiceprintProfile{}).Where("scene = ?", VoiceprintSceneBusiness)
	if tenantID > 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	return q
}

func ListVoiceprintProfiles(db *gorm.DB, tenantID uint) ([]VoiceprintProfile, error) {
	if db == nil {
		return nil, gorm.ErrInvalidDB
	}
	var rows []VoiceprintProfile
	return rows, businessVoiceprintsQuery(db, tenantID).Order("id DESC").Find(&rows).Error
}

func ListVoiceprintProfilesByAssistantID(db *gorm.DB, tenantID, assistantID uint) ([]VoiceprintProfile, error) {
	if db == nil || tenantID == 0 || assistantID == 0 {
		return nil, gorm.ErrInvalidDB
	}
	var rows []VoiceprintProfile
	err := businessVoiceprintsQuery(db, tenantID).
		Where("assistant_id = ?", assistantID).
		Order("id DESC").
		Find(&rows).Error
	return rows, err
}

func GetVoiceprintProfile(db *gorm.DB, tenantID, id uint) (VoiceprintProfile, error) {
	var row VoiceprintProfile
	err := db.Where("tenant_id = ? AND id = ? AND scene = ?", tenantID, id, VoiceprintSceneBusiness).First(&row).Error
	return row, err
}

func GetAccountVoiceprint(db *gorm.DB, tenantID, id uint) (VoiceprintProfile, error) {
	var row VoiceprintProfile
	err := db.Where("tenant_id = ? AND id = ? AND scene = ?", tenantID, id, VoiceprintSceneAccount).First(&row).Error
	return row, err
}

func FindVoiceprintProfileByFeatureID(db *gorm.DB, tenantID uint, featureID string) (VoiceprintProfile, error) {
	var row VoiceprintProfile
	err := db.Where("tenant_id = ? AND feature_id = ? AND scene = ?", tenantID, strings.TrimSpace(featureID), VoiceprintSceneBusiness).First(&row).Error
	return row, err
}

func CreateVoiceprintProfile(db *gorm.DB, row *VoiceprintProfile) error {
	if row == nil {
		return gorm.ErrInvalidData
	}
	row.Scene = normalizeVoiceprintScene(row.Scene)
	return db.Create(row).Error
}

func UpdateVoiceprintProfileAssistant(db *gorm.DB, tenantID, id uint, assistantID *uint) error {
	return db.Model(&VoiceprintProfile{}).
		Where("tenant_id = ? AND id = ? AND scene = ?", tenantID, id, VoiceprintSceneBusiness).
		Update("assistant_id", assistantID).Error
}

func UpdateVoiceprintProfileSubject(db *gorm.DB, tenantID, id uint, subjectID *uint) error {
	return db.Model(&VoiceprintProfile{}).
		Where("tenant_id = ? AND id = ? AND scene = ?", tenantID, id, VoiceprintSceneBusiness).
		Update("subject_id", subjectID).Error
}

func DeleteVoiceprintProfile(db *gorm.DB, tenantID, id uint) error {
	return db.Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&VoiceprintProfile{}).Error
}

func GetVoiceprintProfileByID(db *gorm.DB, id uint) (VoiceprintProfile, error) {
	var row VoiceprintProfile
	err := db.Where("id = ?", id).First(&row).Error
	return row, err
}

func ListAllVoiceprintProfiles(db *gorm.DB, tenantID uint, limit, offset int) ([]VoiceprintProfile, int64, error) {
	if db == nil {
		return nil, 0, gorm.ErrInvalidDB
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	q := businessVoiceprintsQuery(db, tenantID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []VoiceprintProfile
	err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func VoiceprintCandidateFeatureIDs(db *gorm.DB, tenantID uint, featureIDs []string, assistantID *uint) ([]string, error) {
	if db == nil {
		return nil, gorm.ErrInvalidDB
	}
	q := businessVoiceprintsQuery(db, tenantID).Where("status = ?", VoiceprintStatusActive)
	if assistantID != nil && *assistantID > 0 {
		// Prefer profiles explicitly bound to this assistant. If none, fall back to
		// tenant-wide (assistant_id IS NULL) plus any bound to this assistant.
		var boundCount int64
		_ = businessVoiceprintsQuery(db, tenantID).
			Where("status = ? AND assistant_id = ?", VoiceprintStatusActive, *assistantID).
			Count(&boundCount).Error
		if boundCount > 0 {
			q = q.Where("assistant_id = ?", *assistantID)
		} else {
			q = q.Where("assistant_id IS NULL OR assistant_id = ?", *assistantID)
		}
	}
	if len(featureIDs) > 0 {
		trimmed := make([]string, 0, len(featureIDs))
		for _, id := range featureIDs {
			if v := strings.TrimSpace(id); v != "" {
				trimmed = append(trimmed, v)
			}
		}
		if len(trimmed) == 0 {
			return nil, nil
		}
		q = q.Where("feature_id IN ?", trimmed)
	}
	var rows []VoiceprintProfile
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if id := strings.TrimSpace(r.FeatureID); id != "" {
			out = append(out, id)
		}
	}
	return out, nil
}

// GetAccountVoiceprintForUser loads the account-scene profile bound on tenant_users.voiceprint_id.
func GetAccountVoiceprintForUser(db *gorm.DB, tenantID, userID uint) (VoiceprintProfile, error) {
	u, err := GetAuthenticatedTenantUser(db, userID, tenantID)
	if err != nil {
		return VoiceprintProfile{}, err
	}
	if u.VoiceprintID == nil || *u.VoiceprintID == 0 {
		return VoiceprintProfile{}, gorm.ErrRecordNotFound
	}
	return GetAccountVoiceprint(db, tenantID, *u.VoiceprintID)
}

// TenantUserHasVoiceprint reports whether the user has bound an account voiceprint row.
func TenantUserHasVoiceprint(u TenantUser) bool {
	return u.VoiceprintID != nil && *u.VoiceprintID > 0
}

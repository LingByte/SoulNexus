package models

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

const (
	SpeakerSubjectStatusActive   = "active"
	SpeakerSubjectStatusDisabled = "disabled"

	SpeakerVisibilityLLM      = "llm"
	SpeakerVisibilityInternal = "internal"
	SpeakerVisibilityTool     = "tool"
)

// SpeakerSubject is a logical speaker (may own multiple voiceprint profiles).
type SpeakerSubject struct {
	common.BaseModel

	TenantID    uint   `json:"tenantId" gorm:"uniqueIndex:uk_tenant_speaker_name,priority:1;index;not null"`
	DisplayName string `json:"displayName" gorm:"size:128;not null;uniqueIndex:uk_tenant_speaker_name,priority:2"`
	Status      string `json:"status" gorm:"size:24;index;not null;default:active"`
	Notes       string `json:"notes,omitempty" gorm:"size:512"`
}

func (SpeakerSubject) TableName() string {
	return constants.SPEAKER_SUBJECT_TABLE_NAME
}

// SpeakerAttribute is a typed key/value attached to a subject.
type SpeakerAttribute struct {
	common.BaseModel

	TenantID   uint   `json:"tenantId" gorm:"index;not null"`
	SubjectID  uint   `json:"subjectId" gorm:"uniqueIndex:uk_speaker_attr,priority:1;index;not null"`
	Key        string `json:"key" gorm:"size:64;not null;uniqueIndex:uk_speaker_attr,priority:2"`
	Value      string `json:"value" gorm:"size:2048;not null"`
	Visibility string `json:"visibility" gorm:"size:16;not null;default:llm"` // llm | internal | tool
}

func (SpeakerAttribute) TableName() string {
	return constants.SPEAKER_ATTRIBUTE_TABLE_NAME
}

// SpeakerCredentialRef stores tool-runtime secrets for a subject (never injected into LLM prompts).
type SpeakerCredentialRef struct {
	common.BaseModel

	TenantID  uint   `json:"tenantId" gorm:"index;not null"`
	SubjectID uint   `json:"subjectId" gorm:"uniqueIndex:uk_speaker_cred,priority:1;index;not null"`
	Provider  string `json:"provider" gorm:"size:64;not null;uniqueIndex:uk_speaker_cred,priority:2"` // cloudsteps | crm | ...
	SecretRef string `json:"-" gorm:"size:4096;not null"`                                             // token / vault ref
	Scopes    string `json:"scopes,omitempty" gorm:"size:512"`                                      // comma-separated
}

func (SpeakerCredentialRef) TableName() string {
	return constants.SPEAKER_CREDENTIAL_TABLE_NAME
}

func NormalizeSpeakerVisibility(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case SpeakerVisibilityInternal:
		return SpeakerVisibilityInternal
	case SpeakerVisibilityTool:
		return SpeakerVisibilityTool
	default:
		return SpeakerVisibilityLLM
	}
}

func GetSpeakerSubject(db *gorm.DB, tenantID, id uint) (SpeakerSubject, error) {
	var row SpeakerSubject
	err := db.Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	return row, err
}

func FindSpeakerSubjectByName(db *gorm.DB, tenantID uint, name string) (SpeakerSubject, error) {
	var row SpeakerSubject
	err := db.Where("tenant_id = ? AND display_name = ?", tenantID, strings.TrimSpace(name)).First(&row).Error
	return row, err
}

func CreateSpeakerSubject(db *gorm.DB, row *SpeakerSubject) error {
	if row == nil {
		return gorm.ErrInvalidData
	}
	row.DisplayName = strings.TrimSpace(row.DisplayName)
	if row.DisplayName == "" {
		return gorm.ErrInvalidData
	}
	if strings.TrimSpace(row.Status) == "" {
		row.Status = SpeakerSubjectStatusActive
	}
	return db.Create(row).Error
}

func EnsureSpeakerSubject(db *gorm.DB, tenantID uint, displayName, notes string) (SpeakerSubject, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return SpeakerSubject{}, gorm.ErrInvalidData
	}
	if existing, err := FindSpeakerSubjectByName(db, tenantID, displayName); err == nil {
		if notes = strings.TrimSpace(notes); notes != "" && existing.Notes != notes {
			_ = db.Model(&existing).Update("notes", notes).Error
			existing.Notes = notes
		}
		return existing, nil
	}
	row := &SpeakerSubject{
		TenantID:    tenantID,
		DisplayName: displayName,
		Status:      SpeakerSubjectStatusActive,
		Notes:       strings.TrimSpace(notes),
	}
	if err := CreateSpeakerSubject(db, row); err != nil {
		return SpeakerSubject{}, err
	}
	return *row, nil
}

func UpdateSpeakerSubjectFields(db *gorm.DB, tenantID, id uint, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	return db.Model(&SpeakerSubject{}).Where("tenant_id = ? AND id = ?", tenantID, id).Updates(updates).Error
}

func ListSpeakerAttributes(db *gorm.DB, tenantID, subjectID uint) ([]SpeakerAttribute, error) {
	var rows []SpeakerAttribute
	err := db.Where("tenant_id = ? AND subject_id = ?", tenantID, subjectID).Order("id ASC").Find(&rows).Error
	return rows, err
}

func ReplaceSpeakerAttributes(db *gorm.DB, tenantID, subjectID uint, attrs []SpeakerAttribute) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND subject_id = ?", tenantID, subjectID).Delete(&SpeakerAttribute{}).Error; err != nil {
			return err
		}
		for i := range attrs {
			key := strings.TrimSpace(attrs[i].Key)
			if key == "" {
				continue
			}
			row := SpeakerAttribute{
				TenantID:   tenantID,
				SubjectID:  subjectID,
				Key:        key,
				Value:      strings.TrimSpace(attrs[i].Value),
				Visibility: NormalizeSpeakerVisibility(attrs[i].Visibility),
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ListSpeakerCredentials(db *gorm.DB, tenantID, subjectID uint) ([]SpeakerCredentialRef, error) {
	var rows []SpeakerCredentialRef
	err := db.Where("tenant_id = ? AND subject_id = ?", tenantID, subjectID).Order("id ASC").Find(&rows).Error
	return rows, err
}

func UpsertSpeakerCredential(db *gorm.DB, tenantID, subjectID uint, provider, secret, scopes string) error {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return gorm.ErrInvalidData
	}
	var existing SpeakerCredentialRef
	err := db.Where("tenant_id = ? AND subject_id = ? AND provider = ?", tenantID, subjectID, provider).First(&existing).Error
	if err == nil {
		updates := map[string]any{"scopes": strings.TrimSpace(scopes)}
		if strings.TrimSpace(secret) != "" {
			updates["secret_ref"] = strings.TrimSpace(secret)
		}
		return db.Model(&existing).Updates(updates).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	if strings.TrimSpace(secret) == "" {
		return gorm.ErrInvalidData
	}
	row := &SpeakerCredentialRef{
		TenantID:  tenantID,
		SubjectID: subjectID,
		Provider:  provider,
		SecretRef: strings.TrimSpace(secret),
		Scopes:    strings.TrimSpace(scopes),
	}
	return db.Create(row).Error
}

func DeleteSpeakerCredential(db *gorm.DB, tenantID, subjectID uint, provider string) error {
	provider = strings.ToLower(strings.TrimSpace(provider))
	return db.Where("tenant_id = ? AND subject_id = ? AND provider = ?", tenantID, subjectID, provider).
		Delete(&SpeakerCredentialRef{}).Error
}

func DeleteSpeakerSubjectCascade(db *gorm.DB, tenantID, subjectID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND subject_id = ?", tenantID, subjectID).Delete(&SpeakerAttribute{}).Error; err != nil {
			return err
		}
		if err := tx.Where("tenant_id = ? AND subject_id = ?", tenantID, subjectID).Delete(&SpeakerCredentialRef{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&VoiceprintProfile{}).
			Where("tenant_id = ? AND subject_id = ?", tenantID, subjectID).
			Update("subject_id", nil).Error; err != nil {
			return err
		}
		return tx.Where("tenant_id = ? AND id = ?", tenantID, subjectID).Delete(&SpeakerSubject{}).Error
	})
}

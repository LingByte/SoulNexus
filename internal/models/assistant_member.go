package models

import (
	"errors"

	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

// AssistantMember grants a tenant user access to edit/debug an assistant.
type AssistantMember struct {
	common.BaseModel
	AssistantID uint   `json:"assistantId,string" gorm:"uniqueIndex:idx_assistant_member;not null"`
	UserID      uint   `json:"userId,string" gorm:"uniqueIndex:idx_assistant_member;not null"`
	Role        string `json:"role" gorm:"size:32;default:'editor'"`
}

func (AssistantMember) TableName() string {
	return "assistant_members"
}

func ListAssistantMembers(db *gorm.DB, assistantID uint) ([]AssistantMember, error) {
	var rows []AssistantMember
	err := db.Where("assistant_id = ?", assistantID).Order("id ASC").Find(&rows).Error
	return rows, err
}

func ReplaceAssistantMembers(db *gorm.DB, assistantID uint, userIDs []uint, operator string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("assistant_id = ?", assistantID).Delete(&AssistantMember{}).Error; err != nil {
			return err
		}
		return createAssistantMembers(tx, assistantID, userIDs, operator)
	})
}

// AddAssistantMembers appends collaborators; restores soft-deleted rows when re-inviting.
func AddAssistantMembers(db *gorm.DB, assistantID uint, userIDs []uint, operator string) error {
	if len(userIDs) == 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		return createAssistantMembers(tx, assistantID, userIDs, operator)
	})
}

// RemoveAssistantMember removes one collaborator (hard delete to free unique index).
func RemoveAssistantMember(db *gorm.DB, assistantID, userID uint) error {
	if userID == 0 {
		return nil
	}
	return db.Unscoped().Where("assistant_id = ? AND user_id = ?", assistantID, userID).Delete(&AssistantMember{}).Error
}

func createAssistantMembers(tx *gorm.DB, assistantID uint, userIDs []uint, operator string) error {
	for _, uid := range userIDs {
		if uid == 0 {
			continue
		}
		var existing AssistantMember
		err := tx.Unscoped().Where("assistant_id = ? AND user_id = ?", assistantID, uid).First(&existing).Error
		if err == nil {
			if existing.DeletedAt.Valid {
				existing.Restore(operator)
				if err := tx.Unscoped().Save(&existing).Error; err != nil {
					return err
				}
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row := AssistantMember{AssistantID: assistantID, UserID: uid, Role: "editor"}
		row.SetCreateInfo(operator)
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

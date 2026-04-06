package models

import (
	"github.com/LingByte/SoulNexus/pkg/constants"
	"gorm.io/datatypes"
)

// SIPScriptTemplate stores reusable outbound dialog-flow scripts for campaigns.
type SIPScriptTemplate struct {
	BaseModel

	Name        string `json:"name" gorm:"size:128;not null;index"`
	ScriptID    string `json:"scriptId" gorm:"size:128;not null;index"`
	Version     string `json:"version" gorm:"size:64;index"`
	Description string `json:"description" gorm:"type:text"`
	Enabled     bool   `json:"enabled" gorm:"default:true;index"`

	ScriptSpec datatypes.JSON `json:"scriptSpec" gorm:"type:json;not null"`
}

func (SIPScriptTemplate) TableName() string {
	return constants.SIP_SCRIPT_TEMPLATE_TABLE_NAME
}


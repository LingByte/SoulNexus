package models

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
)

// SIPUser represents a SIP endpoint registration / online user state.
//
// This is intentionally "SIP-facing", and can later be mapped to User/Assistant/Credential.
type SIPUser struct {
	BaseModel

	// SIP identity (AOR = username@domain)
	Username string `json:"username" gorm:"size:128;not null;uniqueIndex:idx_sip_user_aor"`
	Domain   string `json:"domain" gorm:"size:128;not null;uniqueIndex:idx_sip_user_aor"`

	// Contact info (where to reach this user)
	ContactURI string `json:"contactUri" gorm:"size:512"`
	RemoteIP   string `json:"remoteIp" gorm:"size:64;index"`
	RemotePort int    `json:"remotePort" gorm:"index"`

	// Registration state
	Online     bool       `json:"online" gorm:"default:false;index"`
	ExpiresAt  *time.Time `json:"expiresAt" gorm:"index"`
	LastSeenAt *time.Time `json:"lastSeenAt" gorm:"index"`

	// Raw SIP headers for debugging / interoperability
	UserAgent string `json:"userAgent" gorm:"size:256"`
	Via       string `json:"via" gorm:"type:text"`
}

func (SIPUser) TableName() string {
	return constants.SIP_USER_TABLE_NAME
}


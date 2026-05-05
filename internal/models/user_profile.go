package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"gorm.io/gorm"
)

// UserProfile is the 1:1 extension row for User. Keep auth, OAuth linkage, roles,
// verification tokens, and login counters on users; store optional presentation,
// prefs, and rarely touched extras here.
type UserProfile struct {
	UserID uint `gorm:"primaryKey;column:user_id" json:"userId"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`

	// Contact & presentation (profile UI, admin search).
	Phone       string `json:"phone,omitempty" gorm:"size:64;index"`
	FirstName   string `json:"firstName,omitempty" gorm:"size:128"`
	LastName    string `json:"lastName,omitempty" gorm:"size:128"`
	DisplayName string `json:"displayName,omitempty" gorm:"size:128"`
	Avatar      string `json:"avatar,omitempty" gorm:"type:text"`

	// Preferences.
	Locale   string `json:"locale,omitempty" gorm:"size:20"`
	Timezone string `json:"timezone,omitempty" gorm:"size:200"`

	EmailNotifications bool `json:"emailNotifications" gorm:"default:true"`
	PushNotifications  bool `json:"pushNotifications" gorm:"default:true"`
	ProfileComplete    int  `json:"profileComplete" gorm:"default:0"`

	// Extended demographic / arbitrary payload.
	Gender string `json:"gender,omitempty" gorm:"size:32"`
	City   string `json:"city,omitempty" gorm:"size:128"`
	Region string `json:"region,omitempty" gorm:"size:128"`
	Extra  string `json:"extra,omitempty" gorm:"type:text"`
}

func (UserProfile) TableName() string {
	return constants.USER_PROFILE_TABLE_NAME
}

// EnsureUserProfile creates an empty profile row if missing.
func EnsureUserProfile(db *gorm.DB, userID uint) error {
	if db == nil || userID == 0 {
		return nil
	}
	var n int64
	if err := db.Model(&UserProfile{}).Where("user_id = ?", userID).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	return db.Create(&UserProfile{
		UserID:             userID,
		EmailNotifications: true,
		PushNotifications:  true,
	}).Error
}

// UpdateUserProfileFields upserts profile columns for the given user id.
func UpdateUserProfileFields(db *gorm.DB, userID uint, vals map[string]any) error {
	if userID == 0 {
		return errors.New("invalid user id")
	}
	if err := EnsureUserProfile(db, userID); err != nil {
		return err
	}
	return db.Model(&UserProfile{}).Where("user_id = ?", userID).Updates(vals).Error
}

func loadUserProfile(db *gorm.DB, userID uint) (*UserProfile, error) {
	var p UserProfile
	err := db.Where("user_id = ?", userID).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (u *User) absorbProfile(p *UserProfile) {
	if u == nil || p == nil {
		return
	}
	u.Profile = *p
}

// AfterFind fills Profile when missing from the main query (Joins loads it in one round-trip).
func (u *User) AfterFind(tx *gorm.DB) error {
	if u == nil || u.ID == 0 || tx == nil {
		return nil
	}
	if u.Profile.UserID == u.ID {
		return nil
	}
	p, err := loadUserProfile(tx, u.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	u.absorbProfile(p)
	return nil
}

// AfterCreate ensures each user has a profile row.
func (u *User) AfterCreate(tx *gorm.DB) error {
	if u == nil || tx == nil || u.ID == 0 {
		return nil
	}
	return EnsureUserProfile(tx, u.ID)
}

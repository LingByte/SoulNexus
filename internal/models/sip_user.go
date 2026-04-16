package models

import (
	"context"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SIPUser represents a SIP endpoint registration / online user state.
// This is intentionally "SIP-facing", and can later be mapped to User/Assistant/Credential.
type SIPUser struct {
	BaseModel
	Username   string     `json:"username" gorm:"size:128;not null;uniqueIndex:idx_sip_user_aor"` // SIP identity (AOR = username@domain)
	Domain     string     `json:"domain" gorm:"size:128;not null;uniqueIndex:idx_sip_user_aor"`
	ContactURI string     `json:"contactUri" gorm:"size:512"` // Contact info (where to reach this user)
	RemoteIP   string     `json:"remoteIp" gorm:"size:64;index"`
	RemotePort int        `json:"remotePort" gorm:"index"`
	Online     bool       `json:"online" gorm:"default:false;index"` // Registration state
	ExpiresAt  *time.Time `json:"expiresAt" gorm:"index"`
	LastSeenAt *time.Time `json:"lastSeenAt" gorm:"index"`
	UserAgent  string     `json:"userAgent" gorm:"size:256"` // Raw SIP headers for debugging / interoperability
	Via        string     `json:"via" gorm:"type:text"`
}

// TableName SIP User Table
func (SIPUser) TableName() string {
	return constants.SIP_USER_TABLE_NAME
}

// ActiveSIPUsers returns base query for non-deleted SIP users.
func ActiveSIPUsers(db *gorm.DB) *gorm.DB {
	return db.Model(&SIPUser{}).Where("is_deleted = ?", SoftDeleteStatusActive)
}

// OnlineSIPUsers returns base query for active and online SIP users.
func OnlineSIPUsers(db *gorm.DB, now time.Time) *gorm.DB {
	return ActiveSIPUsers(db).
		Where("online = ?", true).
		Where("expires_at IS NULL OR expires_at > ?", now)
}

func ListSIPUsersPage(db *gorm.DB, page, size int) ([]SIPUser, int64, error) {
	var total int64
	q := ActiveSIPUsers(db)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	var list []SIPUser
	if err := q.Order("id DESC").Offset(offset).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func GetActiveSIPUserByID(db *gorm.DB, id uint) (SIPUser, error) {
	var row SIPUser
	err := ActiveSIPUsers(db).Where("id = ?", id).First(&row).Error
	return row, err
}

func SoftDeleteSIPUserByID(db *gorm.DB, id uint) (int64, error) {
	res := db.Model(&SIPUser{}).Where("id = ?", id).Updates(map[string]any{
		"is_deleted": SoftDeleteStatusDeleted,
	})
	return res.RowsAffected, res.Error
}

func CountOnlineSIPUsersByUsername(db *gorm.DB, username string) (int64, error) {
	var n int64
	err := OnlineSIPUsers(db, time.Now()).
		Where("username = ?", strings.TrimSpace(username)).
		Count(&n).Error
	return n, err
}

func FindOnlineSIPUserByAOR(ctx context.Context, db *gorm.DB, username, domain string) (SIPUser, error) {
	q := OnlineSIPUsers(db.WithContext(ctx), time.Now()).
		Where("username = ?", strings.TrimSpace(username))
	if d := strings.TrimSpace(domain); d != "" {
		q = q.Where("domain = ?", d)
	}
	var row SIPUser
	err := q.First(&row).Error
	return row, err
}

func FindLatestOnlineSIPUserByUsername(ctx context.Context, db *gorm.DB, username string) (SIPUser, error) {
	var row SIPUser
	err := OnlineSIPUsers(db.WithContext(ctx), time.Now()).
		Where("username = ?", strings.TrimSpace(username)).
		Order("last_seen_at DESC").
		First(&row).Error
	return row, err
}

func UpsertSIPUserRegister(ctx context.Context, db *gorm.DB, user SIPUser) error {
	now := time.Now()
	user.Username = strings.TrimSpace(user.Username)
	user.Domain = strings.TrimSpace(user.Domain)
	if user.Username == "" || user.Domain == "" {
		return nil
	}
	if user.LastSeenAt == nil {
		user.LastSeenAt = &now
	}
	user.IsDeleted = SoftDeleteStatusActive
	updates := map[string]any{
		"contact_uri":  user.ContactURI,
		"remote_ip":    user.RemoteIP,
		"remote_port":  user.RemotePort,
		"user_agent":   user.UserAgent,
		"via":          user.Via,
		"online":       user.Online,
		"expires_at":   user.ExpiresAt,
		"last_seen_at": user.LastSeenAt,
		"is_deleted":   SoftDeleteStatusActive,
		"updated_at":   now,
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "username"},
			{Name: "domain"},
		},
		DoUpdates: clause.Assignments(updates),
	}).Create(&user).Error
}

func MarkSIPUserOffline(ctx context.Context, db *gorm.DB, username, domain string) error {
	return ActiveSIPUsers(db.WithContext(ctx)).
		Where("username = ? AND domain = ?", strings.TrimSpace(username), strings.TrimSpace(domain)).
		Updates(map[string]any{
			"online":       false,
			"expires_at":   nil,
			"last_seen_at": time.Now(),
		}).Error
}

// MarkExpiredSIPUsersOffline updates expired online SIP users to offline state.
func MarkExpiredSIPUsersOffline(ctx context.Context, db *gorm.DB, now time.Time) (int64, error) {
	res := ActiveSIPUsers(db.WithContext(ctx)).
		Where("online = ?", true).
		Where("expires_at IS NOT NULL AND expires_at <= ?", now).
		Updates(map[string]any{
			"online":       false,
			"last_seen_at": now,
		})
	return res.RowsAffected, res.Error
}

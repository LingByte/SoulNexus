// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package persist

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SIPUser is one row of the `sip_users` table — registrar-facing online state
// for a SIP user agent. Used at INVITE-time to look up the current contact /
// transport for a target AOR.
//
// One row per (Username, Domain). REGISTER refreshes ExpiresAt and Online; a
// 0-Expires REGISTER (de-register) flips Online=false.
type SIPUser struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" gorm:"autoUpdateTime"`

	Username   string     `json:"username" gorm:"size:128;not null;uniqueIndex:idx_sip_user_aor"`
	Domain     string     `json:"domain" gorm:"size:128;not null;uniqueIndex:idx_sip_user_aor"`
	ContactURI string     `json:"contactUri" gorm:"size:512"`
	RemoteIP   string     `json:"remoteIp" gorm:"size:64;index"`
	RemotePort int        `json:"remotePort"`
	UserAgent  string     `json:"userAgent" gorm:"size:256"`
	Online     bool       `json:"online" gorm:"default:false;index"`
	ExpiresAt  *time.Time `json:"expiresAt" gorm:"index"`
	LastSeenAt *time.Time `json:"lastSeenAt" gorm:"index"`
}

// TableName overrides GORM's default pluralization to use `sip_users`.
func (SIPUser) TableName() string { return "sip_users" }

// ---------- repository functions ------------------------------------------

// UpsertSIPUser writes a registrar update keyed by (Username, Domain). It is
// the canonical entry point for the registrar; REGISTER 200 OK should call
// this with the contact URI and a non-zero ExpiresAt; de-register sets
// Online=false and ExpiresAt to a past time.
//
// Implementation uses GORM's ON CONFLICT path so it's a single SQL statement
// on MySQL/Postgres/SQLite without a separate read.
//
// Time normalization: SQLite stores time values as ISO-8601 strings WITHOUT
// timezone, so a `time.Now()` (local) and `time.Now().UTC()` written through
// the same column will compare as different strings. We force UTC on every
// time field at write and at query time so cross-process / cross-test reads
// are deterministic.
func UpsertSIPUser(ctx context.Context, db *gorm.DB, row *SIPUser) error {
	if db == nil {
		return errors.New("persist: nil db")
	}
	if row == nil {
		return errors.New("persist: nil user row")
	}
	row.Username = strings.TrimSpace(row.Username)
	row.Domain = strings.TrimSpace(row.Domain)
	if row.Username == "" {
		return errors.New("persist: empty username")
	}
	if row.LastSeenAt == nil {
		now := time.Now().UTC()
		row.LastSeenAt = &now
	} else {
		t := row.LastSeenAt.UTC()
		row.LastSeenAt = &t
	}
	if row.ExpiresAt != nil {
		t := row.ExpiresAt.UTC()
		row.ExpiresAt = &t
	}
	return db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "username"}, {Name: "domain"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"contact_uri",
				"remote_ip",
				"remote_port",
				"user_agent",
				"online",
				"expires_at",
				"last_seen_at",
				"updated_at",
			}),
		}).
		Create(row).Error
}

// FindOnlineSIPUserByAOR returns the current online row for username[@domain]
// whose registration has not yet expired. Domain may be empty to match any.
// Returns gorm.ErrRecordNotFound if no online row is found.
func FindOnlineSIPUserByAOR(ctx context.Context, db *gorm.DB, username, domain string) (SIPUser, error) {
	var row SIPUser
	if db == nil {
		return row, errors.New("persist: nil db")
	}
	uname := strings.TrimSpace(username)
	if uname == "" {
		return row, errors.New("persist: empty username")
	}
	now := time.Now().UTC()
	q := db.WithContext(ctx).
		Model(&SIPUser{}).
		Where("username = ? AND online = ?", uname, true).
		Where("expires_at IS NULL OR expires_at > ?", now)
	if d := strings.TrimSpace(domain); d != "" {
		q = q.Where("domain = ?", d)
	}
	err := q.Order("last_seen_at DESC").First(&row).Error
	return row, err
}

// MarkSIPUserOfflineByID flips Online=false on the row. Used by the offline
// reaper sweeping rows whose ExpiresAt has elapsed without a refresh.
func MarkSIPUserOfflineByID(ctx context.Context, db *gorm.DB, id uint) (int64, error) {
	if db == nil {
		return 0, errors.New("persist: nil db")
	}
	res := db.WithContext(ctx).
		Model(&SIPUser{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"online":     false,
			"updated_at": time.Now().UTC(),
		})
	return res.RowsAffected, res.Error
}

// SweepExpiredSIPUsers flips Online=false for every row whose ExpiresAt has
// passed; returns rows-affected. Cheap to call periodically (~ once a minute)
// since the WHERE is index-backed.
func SweepExpiredSIPUsers(ctx context.Context, db *gorm.DB) (int64, error) {
	if db == nil {
		return 0, errors.New("persist: nil db")
	}
	now := time.Now().UTC()
	res := db.WithContext(ctx).
		Model(&SIPUser{}).
		Where("online = ? AND expires_at IS NOT NULL AND expires_at <= ?", true, now).
		Updates(map[string]any{
			"online":     false,
			"updated_at": now,
		})
	return res.RowsAffected, res.Error
}

// ListSIPUsersPage returns a single page of user rows ordered by id DESC.
func ListSIPUsersPage(ctx context.Context, db *gorm.DB, page, size int, onlineOnly bool) ([]SIPUser, int64, error) {
	if db == nil {
		return nil, 0, errors.New("persist: nil db")
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 500 {
		size = 50
	}
	q := db.WithContext(ctx).Model(&SIPUser{})
	if onlineOnly {
		q = q.Where("online = ?", true).
			Where("expires_at IS NULL OR expires_at > ?", time.Now().UTC())
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []SIPUser
	if err := q.Order("id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

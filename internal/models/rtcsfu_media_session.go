// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// RTCSFUMediaSession records a participant WebRTC leg connected to the embedded SFU.
type RTCSFUMediaSession struct {
	BaseModel
	RoomID    string     `json:"roomId" gorm:"size:160;index:idx_rtcsfu_media_room_peer,priority:1;not null"`
	PeerID    string     `json:"peerId" gorm:"size:160;index:idx_rtcsfu_media_room_peer,priority:2;not null"`
	RemoteIP  string     `json:"remoteIp" gorm:"size:64"`
	UserAgent string     `json:"userAgent" gorm:"size:512"`
	JoinedAt  time.Time  `json:"joinedAt" gorm:"comment:WS connect time"`
	LeftAt    *time.Time `json:"leftAt,omitempty" gorm:"comment:Disconnect time"`
}

func (RTCSFUMediaSession) TableName() string {
	return "rtcsfu_media_sessions"
}

// UpsertRTCSFURoomAssignment writes the latest join result for a room.
func UpsertRTCSFURoomAssignment(db *gorm.DB, roomID, sfuNodeID, region, signalURL, mediaURL string) error {
	var row RTCSFURoomAssignment
	err := db.Where("room_id = ?", roomID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&RTCSFURoomAssignment{
			RoomID:    roomID,
			SFUNodeID: sfuNodeID,
			Region:    region,
			SignalURL: signalURL,
			MediaURL:  mediaURL,
		}).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&row).Updates(map[string]any{
		"sfu_node_id": sfuNodeID,
		"region":      region,
		"signal_url":  signalURL,
		"media_url":   mediaURL,
	}).Error
}

// StartRTCSFUMediaSession inserts an active media session row.
func StartRTCSFUMediaSession(db *gorm.DB, roomID, peerID, remoteIP, userAgent string) error {
	now := time.Now()
	s := RTCSFUMediaSession{
		RoomID:    roomID,
		PeerID:    peerID,
		RemoteIP:  remoteIP,
		UserAgent: userAgent,
		JoinedAt:  now,
	}
	return db.Create(&s).Error
}

// EndRTCSFUMediaSession marks the latest open session for room+peer as left.
func EndRTCSFUMediaSession(db *gorm.DB, roomID, peerID string) error {
	now := time.Now()
	var list []RTCSFUMediaSession
	if err := db.Where("room_id = ? AND peer_id = ? AND left_at IS NULL", roomID, peerID).
		Order("id DESC").Limit(1).Find(&list).Error; err != nil {
		return err
	}
	if len(list) == 0 {
		return nil
	}
	return db.Model(&list[0]).Update("left_at", now).Error
}

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package models

// RTCSFURoomAssignment stores the last control-plane SFU pick for a room (join API).
type RTCSFURoomAssignment struct {
	BaseModel
	RoomID    string `json:"roomId" gorm:"size:160;uniqueIndex:ux_rtcsfu_room_assignment;not null;comment:Room identifier"`
	SFUNodeID string `json:"sfuNodeId" gorm:"size:64;comment:Assigned SFU node id"`
	Region    string `json:"region" gorm:"size:32;comment:Client region hint"`
	SignalURL string `json:"signalUrl" gorm:"size:768;comment:Signaling base URL for this node"`
	MediaURL  string `json:"mediaUrl" gorm:"size:768;comment:Optional media-specific URL"`
}

func (RTCSFURoomAssignment) TableName() string {
	return "rtcsfu_room_assignments"
}

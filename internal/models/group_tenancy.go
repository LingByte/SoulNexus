package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Organization types: each user gets one personal org by default; teams are shared orgs.
const (
	GroupTypePersonal = "personal"
	GroupTypeTeam     = "team"
)

// MemberGroupIDs returns every group the user belongs to (including personal).
func MemberGroupIDs(db *gorm.DB, userID uint) ([]uint, error) {
	var ids []uint
	err := db.Model(&GroupMember{}).Where("user_id = ?", userID).Pluck("group_id", &ids).Error
	return ids, err
}

// UserIsGroupMember reports whether the user is a member of the group.
func UserIsGroupMember(db *gorm.DB, userID, groupID uint) bool {
	if groupID == 0 {
		return false
	}
	var n int64
	_ = db.Model(&GroupMember{}).Where("user_id = ? AND group_id = ?", userID, groupID).Limit(1).Count(&n).Error
	return n > 0
}

// UserIsGroupAdmin reports creator or explicit admin role.
func UserIsGroupAdmin(db *gorm.DB, userID, groupID uint) bool {
	if groupID == 0 {
		return false
	}
	var g Group
	if err := db.First(&g, groupID).Error; err != nil {
		return false
	}
	if g.CreatorID == userID {
		return true
	}
	var m GroupMember
	err := db.Where("group_id = ? AND user_id = ? AND role = ?", groupID, userID, GroupRoleAdmin).First(&m).Error
	return err == nil
}

// ResolveWriteGroupID picks the target group for creating tenant resources.
// If requestedGroupID is nil, uses the caller's personal organization.
func ResolveWriteGroupID(db *gorm.DB, userID uint, requestedGroupID *uint) (uint, error) {
	if requestedGroupID != nil {
		gid := *requestedGroupID
		if !UserIsGroupMember(db, userID, gid) {
			return 0, errors.New("not a member of the target organization")
		}
		return gid, nil
	}
	g, err := EnsurePersonalGroupForUser(db, userID)
	if err != nil {
		return 0, err
	}
	return g.ID, nil
}

// EnsurePersonalGroupForUser creates a personal organization if missing and adds the user as admin.
func EnsurePersonalGroupForUser(db *gorm.DB, userID uint) (*Group, error) {
	var existing Group
	err := db.Where("creator_id = ? AND type = ?", userID, GroupTypePersonal).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var u User
	if err := db.First(&u, userID).Error; err != nil {
		return nil, err
	}

	name := u.EffectiveDisplayName()
	if name == "" {
		name = u.Email
	}
	if name == "" {
		name = fmt.Sprintf("user-%d", userID)
	}
	displayName := name + " · 个人"

	g := Group{
		Name:      displayName,
		Type:      GroupTypePersonal,
		CreatorID: userID,
	}
	if err := db.Create(&g).Error; err != nil {
		return nil, err
	}
	member := GroupMember{
		UserID:  userID,
		GroupID: g.ID,
		Role:    GroupRoleAdmin,
	}
	if err := db.Create(&member).Error; err != nil {
		_ = db.Delete(&g)
		return nil, err
	}
	return &g, nil
}

// PersonalGroupIDForUser returns the user's personal org ID or 0.
// CanManageTenantResource allows group admins or the original creator to mutate a resource.
func CanManageTenantResource(db *gorm.DB, userID, groupID, createdBy uint) bool {
	if UserIsGroupAdmin(db, userID, groupID) {
		return true
	}
	return createdBy != 0 && createdBy == userID
}

func PersonalGroupIDForUser(db *gorm.DB, userID uint) (uint, error) {
	var g Group
	err := db.Where("creator_id = ? AND type = ?", userID, GroupTypePersonal).First(&g).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return g.ID, nil
}

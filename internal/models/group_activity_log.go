package models

import (
	"time"
)

// GroupActivityLog 组织活动日志
type GroupActivityLog struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	GroupID      uint      `json:"groupId" gorm:"index"`
	UserID       uint      `json:"userId" gorm:"index"`
	User         *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Action       string    `json:"action" gorm:"type:varchar(50);index"`  // 操作类型
	ResourceType string    `json:"resourceType" gorm:"type:varchar(50)"`  // 资源类型
	ResourceID   *uint     `json:"resourceId,omitempty"`                  // 资源ID
	ResourceName string    `json:"resourceName" gorm:"type:varchar(255)"` // 资源名称
	Details      string    `json:"details" gorm:"type:text"`              // 详细信息（JSON格式）
	IPAddress    string    `json:"ipAddress" gorm:"type:varchar(45)"`     // IP地址
}

// 操作类型常量
const (
	// 成员操作
	ActionMemberInvited     = "member_invited"
	ActionMemberJoined      = "member_joined"
	ActionMemberRemoved     = "member_removed"
	ActionMemberRoleChanged = "member_role_changed"

	// 资源操作
	ActionResourceAdded    = "resource_added"
	ActionResourceRemoved  = "resource_removed"
	ActionResourceShared   = "resource_shared"
	ActionResourceUnshared = "resource_unshared"
	ActionResourceAccessed = "resource_accessed"

	// 组织操作
	ActionGroupCreated  = "group_created"
	ActionGroupUpdated  = "group_updated"
	ActionGroupArchived = "group_archived"
	ActionGroupRestored = "group_restored"
	ActionGroupDeleted  = "group_deleted"
	ActionGroupExported = "group_exported"
	ActionGroupCloned   = "group_cloned"

	// 配额操作
	ActionQuotaAdded   = "quota_added"
	ActionQuotaUpdated = "quota_updated"
	ActionQuotaDeleted = "quota_deleted"

	// 设置操作
	ActionSettingsUpdated = "settings_updated"
	ActionAvatarUpdated   = "avatar_updated"
)

// 资源类型常量
const (
	ResourceTypeAssistant = "assistant"
	ResourceTypeKnowledge = "knowledge"
	ResourceTypeMember    = "member"
	ResourceTypeQuota     = "quota"
	ResourceTypeGroup     = "group"
	ResourceTypeOverview  = "overview"
)

// TableName 指定表名
func (GroupActivityLog) TableName() string {
	return "group_activity_logs"
}

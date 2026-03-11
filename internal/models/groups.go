package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type GroupPermission struct {
	Permissions []string
}

type Group struct {
	ID         uint            `json:"id" gorm:"primaryKey"`
	CreatedAt  time.Time       `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt  time.Time       `json:"updatedAt" gorm:"autoUpdateTime"`
	Name       string          `json:"name" gorm:"size:200"`
	Type       string          `json:"type" gorm:"size:24;index"`
	Extra      string          `json:"extra,omitempty"`
	Avatar     string          `json:"avatar,omitempty" gorm:"size:500"` // 组织头像URL
	Permission GroupPermission `json:"permission,omitempty" gorm:"type:json"`
	CreatorID  uint            `json:"creatorId" gorm:"index"`
	Creator    User            `json:"creator,omitempty" gorm:"foreignKey:CreatorID"`

	// 归档相关
	IsArchived bool       `json:"isArchived" gorm:"default:false;index"`
	ArchivedAt *time.Time `json:"archivedAt,omitempty"`
	ArchivedBy *uint      `json:"archivedBy,omitempty"`

	// 模板相关
	IsTemplate bool  `json:"isTemplate" gorm:"default:false;index"`
	TemplateID *uint `json:"templateId,omitempty" gorm:"index"` // 如果是从模板创建的，记录模板ID
	ClonedFrom *uint `json:"clonedFrom,omitempty" gorm:"index"` // 如果是克隆的，记录源组织ID
}

// 实现 driver.Valuer 接口
func (gp GroupPermission) Value() (driver.Value, error) {
	return json.Marshal(gp)
}

// 实现 sql.Scanner 接口
func (gp *GroupPermission) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to convert value to []byte")
	}
	return json.Unmarshal(bytes, gp)
}

type GroupMember struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UserID    uint      `json:"userId" gorm:"index"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
	GroupID   uint      `json:"groupId" gorm:"index"`
	Group     Group     `json:"group,omitempty" gorm:"foreignKey:GroupID"`
	Role      string    `json:"role" gorm:"size:60;index"`
}

// GroupInvitation 组织邀请
type GroupInvitation struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time  `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updatedAt" gorm:"autoUpdateTime"`
	GroupID   uint       `json:"groupId" gorm:"index"`
	Group     Group      `json:"group,omitempty" gorm:"foreignKey:GroupID"`
	InviterID uint       `json:"inviterId" gorm:"index"`
	Inviter   User       `json:"inviter,omitempty" gorm:"foreignKey:InviterID"`
	InviteeID uint       `json:"inviteeId" gorm:"index"`
	Invitee   User       `json:"invitee,omitempty" gorm:"foreignKey:InviteeID"`
	Status    string     `json:"status" gorm:"size:20;index;default:'pending'"` // pending, accepted, rejected
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

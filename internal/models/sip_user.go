package models

import (
	"time"

	"gorm.io/gorm"
)

// SipUserStatus SIP用户状态
type SipUserStatus string

const (
	SipUserStatusRegistered   SipUserStatus = "registered"   // 已注册
	SipUserStatusUnregistered SipUserStatus = "unregistered" // 未注册
	SipUserStatusExpired      SipUserStatus = "expired"      // 已过期
)

// SipUser SIP用户表
type SipUser struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time  `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"-" gorm:"index"`

	// SIP认证信息
	Username string `json:"username" gorm:"size:128;uniqueIndex;not null"` // SIP用户名（唯一）
	Password string `json:"-" gorm:"size:128"`                             // SIP密码（可选，用于认证）

	// SIP注册信息
	Contact     string     `json:"contact,omitempty" gorm:"size:256"`  // Contact地址（完整URI）
	ContactIP   string     `json:"contactIp,omitempty" gorm:"size:64"` // Contact IP地址
	ContactPort int        `json:"contactPort,omitempty"`              // Contact端口
	Expires     int        `json:"expires" gorm:"default:3600"`        // 过期时间（秒）
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`                // 过期时间点

	// 注册状态
	Status         SipUserStatus `json:"status" gorm:"size:20;default:'unregistered';index"` // 注册状态
	LastRegister   *time.Time    `json:"lastRegister,omitempty"`                             // 最后注册时间
	LastUnregister *time.Time    `json:"lastUnregister,omitempty"`                           // 最后注销时间

	// 客户端信息
	UserAgent string `json:"userAgent,omitempty" gorm:"size:256"` // 用户代理（User-Agent）
	RemoteIP  string `json:"remoteIp,omitempty" gorm:"size:64"`   // 远程IP地址

	// 关联信息
	UserID  *uint `json:"userId,omitempty" gorm:"index"` // 关联到系统用户（可选）
	User    User  `json:"user,omitempty" gorm:"foreignKey:UserID"`
	GroupID *uint `json:"groupId,omitempty" gorm:"index"` // 关联到组织（可选）
	Group   Group `json:"group,omitempty" gorm:"foreignKey:GroupID"`

	// 显示信息
	DisplayName string `json:"displayName,omitempty" gorm:"size:128"` // 显示名称
	Alias       string `json:"alias,omitempty" gorm:"size:128"`       // 别名

	// 统计信息
	RegisterCount     int `json:"registerCount" gorm:"default:0"`     // 注册次数
	CallCount         int `json:"callCount" gorm:"default:0"`         // 通话次数
	TotalCallDuration int `json:"totalCallDuration" gorm:"default:0"` // 总通话时长（秒）

	// 配置信息
	Enabled bool   `json:"enabled" gorm:"default:true"`      // 是否启用
	Notes   string `json:"notes,omitempty" gorm:"type:text"` // 备注
}

// TableName 指定表名
func (SipUser) TableName() string {
	return "sip_users"
}

// IsRegistered 检查用户是否已注册
func (su *SipUser) IsRegistered() bool {
	return su.Status == SipUserStatusRegistered
}

// IsExpired 检查注册是否已过期
func (su *SipUser) IsExpired() bool {
	if su.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*su.ExpiresAt)
}

// UpdateExpiresAt 根据Expires字段更新过期时间
func (su *SipUser) UpdateExpiresAt() {
	if su.Expires > 0 {
		expiresAt := time.Now().Add(time.Duration(su.Expires) * time.Second)
		su.ExpiresAt = &expiresAt
	}
}

// CreateSipUser 创建SIP用户
func CreateSipUser(db *gorm.DB, sipUser *SipUser) error {
	return db.Create(sipUser).Error
}

// GetSipUserByUsername 根据用户名获取SIP用户
func GetSipUserByUsername(db *gorm.DB, username string) (*SipUser, error) {
	var sipUser SipUser
	err := db.Where("username = ?", username).First(&sipUser).Error
	if err != nil {
		return nil, err
	}
	return &sipUser, nil
}

// GetSipUserByID 根据ID获取SIP用户
func GetSipUserByID(db *gorm.DB, id uint) (*SipUser, error) {
	var sipUser SipUser
	err := db.First(&sipUser, id).Error
	if err != nil {
		return nil, err
	}
	return &sipUser, nil
}

// UpdateSipUser 更新SIP用户
func UpdateSipUser(db *gorm.DB, sipUser *SipUser) error {
	return db.Save(sipUser).Error
}

// DeleteSipUser 删除SIP用户（软删除）
func DeleteSipUser(db *gorm.DB, id uint) error {
	return db.Delete(&SipUser{}, id).Error
}

// GetRegisteredSipUsers 获取所有已注册的SIP用户
func GetRegisteredSipUsers(db *gorm.DB) ([]SipUser, error) {
	var sipUsers []SipUser
	err := db.Where("status = ?", SipUserStatusRegistered).Find(&sipUsers).Error
	return sipUsers, err
}

// GetSipUsersByUserID 根据系统用户ID获取SIP用户列表
func GetSipUsersByUserID(db *gorm.DB, userID uint) ([]SipUser, error) {
	var sipUsers []SipUser
	err := db.Where("user_id = ?", userID).Find(&sipUsers).Error
	return sipUsers, err
}

// GetSipUsersByGroupID 根据组织ID获取SIP用户列表
func GetSipUsersByGroupID(db *gorm.DB, groupID uint) ([]SipUser, error) {
	var sipUsers []SipUser
	err := db.Where("group_id = ?", groupID).Find(&sipUsers).Error
	return sipUsers, err
}

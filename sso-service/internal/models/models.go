package models

import (
	"time"

	"github.com/LingByte/SoulNexus/sso-service/pkg/constants"
)

type User struct {
	BaseModel
	Email                 string     `json:"email" gorm:"size:128;uniqueIndex"`
	Password              string     `json:"-" gorm:"size:128"`
	Phone                 string     `json:"phone,omitempty" gorm:"size:64;index"`
	FirstName             string     `json:"firstName,omitempty" gorm:"size:128"`
	LastName              string     `json:"lastName,omitempty" gorm:"size:128"`
	DisplayName           string     `json:"displayName,omitempty" gorm:"size:128"`
	IsStaff               bool       `json:"isStaff,omitempty"`
	Enabled               bool       `json:"-"`
	Activated             bool       `json:"-"`
	LastLogin             *time.Time `json:"lastLogin,omitempty"`
	LastLoginIP           string     `json:"-" gorm:"size:128"`
	Source                string     `json:"-" gorm:"size:64;index"`
	Locale                string     `json:"locale,omitempty" gorm:"size:20"`
	Timezone              string     `json:"timezone,omitempty" gorm:"size:200"`
	AuthToken             string     `json:"token,omitempty" gorm:"-"`
	Avatar                string     `json:"avatar,omitempty"`
	Gender                string     `json:"gender,omitempty"`
	City                  string     `json:"city,omitempty"`
	Region                string     `json:"region,omitempty"`
	EmailNotifications    bool       `json:"emailNotifications"`                           // 邮件通知
	PushNotifications     bool       `json:"pushNotifications" gorm:"default:true"`        // 推送通知
	SystemNotifications   bool       `json:"systemNotifications" gorm:"default:true"`      // 系统通知
	AutoCleanUnreadEmails bool       `json:"autoCleanUnreadEmails" gorm:"default:false"`   // 自动清理七天未读邮件
	EmailVerified         bool       `json:"emailVerified" gorm:"default:false"`           // 邮箱已验证
	PhoneVerified         bool       `json:"phoneVerified" gorm:"default:false"`           // 手机已验证
	TwoFactorEnabled      bool       `json:"twoFactorEnabled" gorm:"default:false"`        // 双因素认证
	TwoFactorSecret       string     `json:"-" gorm:"size:128"`                            // 双因素认证密钥
	EmailVerifyToken      string     `json:"-" gorm:"size:128"`                            // 邮箱验证令牌
	PhoneVerifyToken      string     `json:"-" gorm:"size:128"`                            // 手机验证令牌
	PasswordResetToken    string     `json:"-" gorm:"size:128"`                            // 密码重置令牌
	PasswordResetExpires  *time.Time `json:"-"`                                            // 密码重置过期时间
	EmailVerifyExpires    *time.Time `json:"-"`                                            // 邮箱验证过期时间
	LoginCount            int        `json:"loginCount" gorm:"default:0"`                  // 登录次数
	LastPasswordChange    *time.Time `json:"lastPasswordChange,omitempty"`                 // 最后密码修改时间
	ProfileComplete       int        `json:"profileComplete" gorm:"default:0"`             // 资料完整度百分比
	Role                  string     `json:"role,omitempty" gorm:"size:50;default:'user'"` // 用户角色
}

func (u *User) TableName() string {
	return constants.USER_TABLE_NAME
}

type OAuthClient struct {
	ID             string `gorm:"primaryKey;size:64"`
	Name           string `gorm:"size:120;not null"`
	Secret         string `gorm:"size:255;not null"`
	RedirectURI    string `gorm:"size:1024;not null"`
	IsConfidential bool   `gorm:"not null;default:true"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AuthorizationCode struct {
	Code                string    `gorm:"primaryKey;size:128"`
	ClientID            string    `gorm:"index;size:64;not null"`
	UserID              string    `gorm:"index;size:64;not null"`
	RedirectURI         string    `gorm:"size:1024;not null"`
	Scope               string    `gorm:"size:512"`
	CodeChallenge       string    `gorm:"size:255"`
	CodeChallengeMethod string    `gorm:"size:16"`
	ExpiresAt           time.Time `gorm:"index;not null"`
	Consumed            bool      `gorm:"not null;default:false"`
	CreatedAt           time.Time
}

type RefreshToken struct {
	TokenHash string    `gorm:"primaryKey;size:128"`
	ClientID  string    `gorm:"index;size:64;not null"`
	UserID    string    `gorm:"index;size:64;not null"`
	Scope     string    `gorm:"size:512"`
	ExpiresAt time.Time `gorm:"index;not null"`
	RevokedAt *time.Time
	CreatedAt time.Time
}

type RevokedToken struct {
	JTI       string    `gorm:"primaryKey;size:64"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time
}

type SigningKey struct {
	KID        string `gorm:"primaryKey;size:64"`
	Algorithm  string `gorm:"size:16;not null"`
	PrivatePEM string `gorm:"type:text;not null"`
	PublicPEM  string `gorm:"type:text;not null"`
	Active     bool   `gorm:"not null;default:true"`
	CreatedAt  time.Time
}

type UserSession struct {
	ID        string    `gorm:"primaryKey;size:64"`
	UserID    string    `gorm:"index;size:64;not null"`
	ClientID  string    `gorm:"index;size:64"`
	ExpiresAt time.Time `gorm:"index;not null"`
	RevokedAt *time.Time
	CreatedAt time.Time
}

type AuditLog struct {
	ID        string `gorm:"primaryKey;size:64"`
	UserID    string `gorm:"index;size:64"`
	ClientID  string `gorm:"index;size:64"`
	Action    string `gorm:"size:64;index;not null"`
	AuthMode  string `gorm:"size:16"`
	IPAddress string `gorm:"size:64"`
	UserAgent string `gorm:"size:512"`
	CreatedAt time.Time
}

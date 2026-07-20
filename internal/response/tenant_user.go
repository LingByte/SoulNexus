package response

import (
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

// TenantGroupBrief is a minimal tenant group reference in user payloads.
type TenantGroupBrief struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault,omitempty"`
}

// TenantRoleBrief is a minimal tenant role reference in user payloads.
type TenantRoleBrief struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsSystem bool   `json:"isSystem"`
}

// TenantUserResponse is the public API view of a tenant user.
type TenantUserResponse struct {
	ID                       string             `json:"id"`
	TenantID                 string             `json:"tenantId"`
	Email                    string             `json:"email"`
	Phone                    string             `json:"phone"`
	Username                 string             `json:"username"`
	DisplayName              string             `json:"displayName"`
	AvatarURL                string             `json:"avatarUrl"`
	Status                   string             `json:"status"`
	CreatedAt                time.Time          `json:"createdAt"`
	LastLogin                *time.Time         `json:"lastLogin,omitempty"`
	LastLoginIP              string             `json:"lastLoginIp"`
	LoginCount               int                `json:"loginCount"`
	TOTPEnabled              bool               `json:"totpEnabled"`
	ReceiveEmailNotify       bool               `json:"receiveEmailNotify"`
	RequireDeviceVerify      bool               `json:"requireDeviceVerify"`
	TrustDeviceLoginEnabled  bool               `json:"trustDeviceLoginEnabled"`
	RequireRemoteLoginVerify bool               `json:"requireRemoteLoginVerify"`
	PrimaryLoginCity         string             `json:"primaryLoginCity,omitempty"`
	SessionIdleTimeoutHours  int                `json:"sessionIdleTimeoutHours"`
	SessionMaxLifetimeHours  int                `json:"sessionMaxLifetimeHours"`
	DeletionRequestedAt      *time.Time         `json:"deletionRequestedAt,omitempty"`
	TenantGroups             []TenantGroupBrief `json:"tenantGroups,omitempty"`
	TenantGroup              *TenantGroupBrief  `json:"tenantGroup,omitempty"`
	Roles                    []TenantRoleBrief  `json:"roles,omitempty"`
	Source                   string             `json:"source,omitempty"`
	GitHubLogin              string             `json:"githubLogin,omitempty"`
	VoiceprintID             string             `json:"voiceprintId,omitempty"`
	VoiceprintEnrolled       bool               `json:"voiceprintEnrolled"`
}

// NewTenantUserResponse converts a tenant user model to its public API representation.
func NewTenantUserResponse(db *gorm.DB, u models.TenantUser) TenantUserResponse {
	out := TenantUserResponse{
		ID:                       formatID(u.ID),
		TenantID:                 formatID(u.TenantID),
		Email:                    u.Email,
		Phone:                    u.Phone,
		Username:                 u.Username,
		DisplayName:              u.DisplayName,
		AvatarURL:                u.AvatarURL,
		Status:                   u.Status,
		CreatedAt:                u.CreatedAt,
		LastLogin:                u.LastLogin,
		LastLoginIP:              u.LastLoginIP,
		LoginCount:               u.LoginCount,
		TOTPEnabled:              u.TOTPEnabled,
		ReceiveEmailNotify:       u.ReceiveEmailNotify,
		RequireDeviceVerify:      u.RequireDeviceVerify,
		TrustDeviceLoginEnabled:  u.TrustDeviceLoginEnabled,
		RequireRemoteLoginVerify: u.RequireRemoteLoginVerify,
		PrimaryLoginCity:         u.PrimaryLoginCity,
		SessionIdleTimeoutHours:  utils.SessionIdleTimeout(u.SessionIdleTimeoutHours),
		SessionMaxLifetimeHours:  utils.SessionMaxLifetime(u.SessionIdleTimeoutHours, u.SessionMaxLifetimeHours),
		DeletionRequestedAt:      u.DeletionRequestedAt,
		VoiceprintEnrolled:       models.TenantUserHasVoiceprint(u),
	}
	if u.VoiceprintID != nil && *u.VoiceprintID > 0 {
		out.VoiceprintID = formatID(*u.VoiceprintID)
	}
	if gs, err := models.ListTenantGroupsForUser(db, u.ID); err == nil && len(gs) > 0 {
		gpub := make([]TenantGroupBrief, 0, len(gs))
		for _, g := range gs {
			gpub = append(gpub, TenantGroupBrief{
				ID: formatID(g.ID), Name: g.Name, IsDefault: g.IsDefault,
			})
		}
		out.TenantGroups = gpub
		out.TenantGroup = &TenantGroupBrief{
			ID: formatID(gs[0].ID), Name: gs[0].Name,
		}
	}
	if roles, err := models.ListTenantRolesForUser(db, u.ID); err == nil && len(roles) > 0 {
		rpub := make([]TenantRoleBrief, 0, len(roles))
		for _, r := range roles {
			rpub = append(rpub, TenantRoleBrief{
				ID: formatID(r.ID), Name: r.Name, IsSystem: r.IsSystem,
			})
		}
		out.Roles = rpub
	}
	if src := strings.TrimSpace(u.Source); src != "" && src != constants.TenantUserSourceRegister {
		out.Source = src
	}
	if gh := strings.TrimSpace(u.GitHubLogin); gh != "" {
		out.GitHubLogin = gh
	}
	return out
}

// TenantUserTotpEnabledResponse is returned when TOTP is enabled for a tenant user.
type TenantUserTotpEnabledResponse struct {
	TenantUserResponse
	RecoveryCodes []string `json:"recoveryCodes"`
}

// NewTenantUserTotpEnabledResponse builds the TOTP enable payload for tenant users.
func NewTenantUserTotpEnabledResponse(db *gorm.DB, u models.TenantUser, recoveryCodes []string) TenantUserTotpEnabledResponse {
	return TenantUserTotpEnabledResponse{
		TenantUserResponse: NewTenantUserResponse(db, u),
		RecoveryCodes:      recoveryCodes,
	}
}

package response

import (
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

// PlatformAdminResponse is the public API view of a platform admin.
type PlatformAdminResponse struct {
	ID                       string     `json:"id"`
	Email                    string     `json:"email"`
	DisplayName              string     `json:"displayName"`
	Status                   string     `json:"status"`
	TOTPEnabled              bool       `json:"totpEnabled"`
	ReceiveEmailNotify       bool       `json:"receiveEmailNotify"`
	RequireDeviceVerify      bool       `json:"requireDeviceVerify"`
	TrustDeviceLoginEnabled  bool       `json:"trustDeviceLoginEnabled"`
	RequireRemoteLoginVerify bool       `json:"requireRemoteLoginVerify"`
	PrimaryLoginCity         string     `json:"primaryLoginCity,omitempty"`
	SessionIdleTimeoutHours  int        `json:"sessionIdleTimeoutHours"`
	SessionMaxLifetimeHours  int        `json:"sessionMaxLifetimeHours"`
	DeletionRequestedAt      *time.Time `json:"deletionRequestedAt,omitempty"`
	CreatedAt                time.Time  `json:"createdAt"`
	UpdatedAt                time.Time  `json:"updatedAt"`
	GitHubLogin              string     `json:"githubLogin,omitempty"`
}

// NewPlatformAdminResponse converts a platform admin model to its public API representation.
func NewPlatformAdminResponse(a models.PlatformAdmin) PlatformAdminResponse {
	out := PlatformAdminResponse{
		ID:                       formatID(a.ID),
		Email:                    a.Email,
		DisplayName:              a.DisplayName,
		Status:                   a.Status,
		TOTPEnabled:              a.TOTPEnabled,
		ReceiveEmailNotify:       a.ReceiveEmailNotify,
		RequireDeviceVerify:      a.RequireDeviceVerify,
		TrustDeviceLoginEnabled:  a.TrustDeviceLoginEnabled,
		RequireRemoteLoginVerify: a.RequireRemoteLoginVerify,
		PrimaryLoginCity:         a.PrimaryLoginCity,
		SessionIdleTimeoutHours:  utils.SessionIdleTimeout(a.SessionIdleTimeoutHours),
		SessionMaxLifetimeHours:  utils.SessionMaxLifetime(a.SessionIdleTimeoutHours, a.SessionMaxLifetimeHours),
		DeletionRequestedAt:      a.DeletionRequestedAt,
		CreatedAt:                a.CreatedAt,
		UpdatedAt:                a.UpdatedAt,
	}
	if gh := strings.TrimSpace(a.GitHubLogin); gh != "" {
		out.GitHubLogin = gh
	}
	return out
}

// PlatformAdminTotpEnabledResponse is returned when TOTP is enabled for a platform admin.
type PlatformAdminTotpEnabledResponse struct {
	PlatformAdminResponse
	RecoveryCodes []string `json:"recoveryCodes"`
}

// NewPlatformAdminTotpEnabledResponse builds the TOTP enable payload for platform admins.
func NewPlatformAdminTotpEnabledResponse(a models.PlatformAdmin, recoveryCodes []string) PlatformAdminTotpEnabledResponse {
	return PlatformAdminTotpEnabledResponse{
		PlatformAdminResponse: NewPlatformAdminResponse(a),
		RecoveryCodes:         recoveryCodes,
	}
}

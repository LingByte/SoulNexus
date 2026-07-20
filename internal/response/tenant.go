package response

import (
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

// TenantResponse is the public API view of a tenant (Snowflake id as string for JS).
type TenantResponse struct {
	ID                      string     `json:"id"`
	Name                    string     `json:"name"`
	Slug                    string     `json:"slug"`
	Description             string     `json:"description"`
	Status                  string     `json:"status"`
	ContactEmail            string     `json:"contactEmail"`
	MaxUserCount            int        `json:"maxUserCount"`
	CreatedAt               time.Time  `json:"createdAt"`
	BillingUnlimited        bool       `json:"billingUnlimited"`
	BillingMode             string     `json:"billingMode"`
	PrepaidMinutesRemaining int64      `json:"prepaidMinutesRemaining"`
	RemainingMinutesDisplay string     `json:"remainingMinutesDisplay"`
	MeteredBilledMinutes    int64      `json:"meteredBilledMinutes"`
	MeteredCallCount        int64      `json:"meteredCallCount"`
	MaxConcurrentCalls      int        `json:"maxConcurrentCalls"`
	DailyMinuteLimit        int64      `json:"dailyMinuteLimit"`
	MonthlyMinuteLimit      int64      `json:"monthlyMinuteLimit"`
	LicenseExpiresAt        *time.Time `json:"licenseExpiresAt,omitempty"`
	QuotaSuspended          bool       `json:"quotaSuspended"`
	VoiceMode               string     `json:"voiceMode"`
}

// TenantPlatformDetailResponse includes per-tenant AI JSON (platform admin APIs only).
type TenantPlatformDetailResponse struct {
	TenantResponse
	AsrConfig            any     `json:"asrConfig"`
	TtsConfig            any     `json:"ttsConfig"`
	LlmConfig            any     `json:"llmConfig"`
	RealtimeConfig       any     `json:"realtimeConfig"`
	BillingRatePerMinute float64 `json:"billingRatePerMinute"`
	BillingCurrency      string  `json:"billingCurrency"`
	AutomationConfig     any     `json:"automationConfig"`
}

// NewTenantResponse converts a tenant model to its public API representation.
func NewTenantResponse(t models.Tenant) TenantResponse {
	acct := models.TenantBillingAccountFrom(t)
	return TenantResponse{
		ID:                      formatID(t.ID),
		Name:                    t.Name,
		Slug:                    t.Slug,
		Description:             t.Description,
		Status:                  t.Status,
		ContactEmail:            t.ContactEmail,
		MaxUserCount:            t.MaxUserCount,
		CreatedAt:               t.CreatedAt,
		BillingUnlimited:        t.BillingUnlimited,
		BillingMode:             acct.BillingMode,
		PrepaidMinutesRemaining: t.PrepaidMinutesRemaining,
		RemainingMinutesDisplay: acct.RemainingMinutesDisplay,
		MeteredBilledMinutes:    t.MeteredBilledMinutes,
		MeteredCallCount:        t.MeteredCallCount,
		MaxConcurrentCalls:      t.MaxConcurrentCalls,
		DailyMinuteLimit:        t.DailyMinuteLimit,
		MonthlyMinuteLimit:      t.MonthlyMinuteLimit,
		LicenseExpiresAt:        t.LicenseExpiresAt,
		QuotaSuspended:          t.QuotaSuspended,
		VoiceMode:               tenantVoiceModeForPublic(t.VoiceMode),
	}
}

// NewTenantPlatformDetailResponse converts a tenant model for platform admin detail APIs.
func NewTenantPlatformDetailResponse(t models.Tenant) TenantPlatformDetailResponse {
	base := NewTenantResponse(t)
	acct := models.TenantBillingAccountFrom(t)
	base.VoiceMode = strings.TrimSpace(t.VoiceMode)
	base.BillingMode = acct.BillingMode
	base.BillingUnlimited = acct.BillingUnlimited
	base.PrepaidMinutesRemaining = acct.PrepaidMinutesRemaining
	base.RemainingMinutesDisplay = acct.RemainingMinutesDisplay
	base.MeteredBilledMinutes = acct.MeteredBilledMinutes
	base.MeteredCallCount = acct.MeteredCallCount
	return TenantPlatformDetailResponse{
		TenantResponse:       base,
		AsrConfig:            utils.JSONValueFromBytes(t.AsrConfig),
		TtsConfig:            utils.JSONValueFromBytes(t.TtsConfig),
		LlmConfig:            utils.JSONValueFromBytes(t.LlmConfig),
		RealtimeConfig:       utils.JSONValueFromBytes(t.RealtimeConfig),
		BillingRatePerMinute: acct.BillingRatePerMinute,
		BillingCurrency:      acct.BillingCurrency,
		AutomationConfig:     models.ParseTenantAutomationConfig(t.AutomationConfigRaw),
	}
}

func tenantVoiceModeForPublic(raw string) string {
	if strings.EqualFold(strings.TrimSpace(raw), "realtime") {
		return "realtime"
	}
	return "pipeline"
}

func formatID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

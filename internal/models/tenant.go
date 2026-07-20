package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	constants2 "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	apperror "github.com/LingByte/SoulNexus/pkg/errors"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	pinyinLib "github.com/mozillazg/go-pinyin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Tenant is one SaaS customer organization (multi-tenant root).
type Tenant struct {
	common.BaseModel
	Name                    string         `json:"name" gorm:"size:128;index;not null;comment:租户名称"`
	Slug                    string         `json:"slug" gorm:"size:64;uniqueIndex;not null;comment:租户标识"`
	Description             string         `json:"description,omitempty" gorm:"size:512;comment:描述"`
	Status                  string         `json:"status" gorm:"size:24;index;not null;default:active;comment:租户状态"`
	ContactEmail            string         `json:"contactEmail" gorm:"size:128;index;comment:联系邮箱"`
	MaxUserCount            int            `json:"maxUserCount" gorm:"default:5;comment:最大成员数"`
	AsrConfig               datatypes.JSON `json:"asrConfig,omitempty" gorm:"column:asr_config;comment:ASR配置JSON"`
	TtsConfig               datatypes.JSON `json:"ttsConfig,omitempty" gorm:"column:tts_config;comment:TTS配置JSON"`
	LlmConfig               datatypes.JSON `json:"llmConfig,omitempty" gorm:"column:llm_config;comment:LLM配置JSON"`
	VoiceMode               string         `json:"voiceMode,omitempty" gorm:"column:voice_mode;size:32;default:pipeline;comment:对话模式 pipeline|realtime"`
	RealtimeConfig          datatypes.JSON `json:"realtimeConfig,omitempty" gorm:"column:realtime_config;comment:实时多模态配置JSON"`
	BillingMode             string         `json:"billingMode" gorm:"column:billing_mode;size:16;not null;default:prepaid;comment:计费模式 prepaid|postpaid"`
	BillingUnlimited        bool           `json:"billingUnlimited" gorm:"column:billing_unlimited;not null;default:0;comment:不限量"`
	PrepaidMinutesRemaining int64          `json:"prepaidMinutesRemaining" gorm:"column:prepaid_minutes_remaining;not null;default:0;comment:预付费剩余分钟"`
	BillingRatePerMinute    float64        `json:"billingRatePerMinute" gorm:"column:billing_rate_per_minute;type:decimal(16,6);not null;default:0;comment:单价元/计费分钟"`
	BillingCurrency         string         `json:"billingCurrency" gorm:"column:billing_currency;size:8;not null;default:CNY;comment:计费币种"`
	MeteredBilledMinutes    int64          `json:"meteredBilledMinutes" gorm:"column:metered_billed_minutes;not null;default:0;comment:累计计费分钟"`
	MeteredCallCount        int64          `json:"meteredCallCount" gorm:"column:metered_call_count;not null;default:0;comment:累计会话数"`
	MaxConcurrentCalls      int            `json:"maxConcurrentCalls" gorm:"column:max_concurrent_calls;not null;default:0;comment:租户并发上限0=不限"`
	DailyMinuteLimit        int64          `json:"dailyMinuteLimit" gorm:"column:daily_minute_limit;not null;default:0;comment:日计费分钟上限0=不限"`
	MonthlyMinuteLimit      int64          `json:"monthlyMinuteLimit" gorm:"column:monthly_minute_limit;not null;default:0;comment:月计费分钟上限0=不限"`
	LicenseExpiresAt        *time.Time     `json:"licenseExpiresAt,omitempty" gorm:"column:license_expires_at;comment:租户License到期"`
	QuotaSuspended          bool           `json:"quotaSuspended" gorm:"column:quota_suspended;not null;default:0;comment:超限自动停机"`
	AutomationConfigRaw     datatypes.JSON `json:"-" gorm:"column:contact_center_config;comment:tenant automation JSON"`
	Balance                 float64        `json:"balance,omitempty" gorm:"type:decimal(16,4);not null;default:0;comment:legacy"` // Legacy monetary balance (unused; kept for schema compatibility).
}

func (Tenant) TableName() string {
	return constants2.TENANT_TABLE_NAME
}

// TenantProvisionInput is the payload for tenant self-register or platform provisioning.
type TenantProvisionInput struct {
	CompanyName       string
	AdminEmail        string
	AdminDisplayName  string
	TenantDescription string
	MaxUserCount      int
}

// BaseSlugFromCompanyName derives a slug base from company name (pinyin + normalization).
func BaseSlugFromCompanyName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "tenant"
	}
	args := pinyinLib.NewArgs()
	segs := pinyinLib.LazyPinyin(name, args)
	raw := strings.Join(segs, "-")
	slug := utils.DeriveTenantSlug(raw)
	if slug == "" {
		slug = utils.DeriveTenantSlug(name)
	}
	if len(slug) < 2 {
		slug = "tenant"
	}
	if len(slug) > 62 {
		slug = strings.TrimRight(strings.TrimSpace(slug[:62]), "-")
	}
	return slug
}

// AllocateUniqueTenantSlug picks an unused slug by appending a random two-digit suffix.
func AllocateUniqueTenantSlug(db *gorm.DB, base string) (string, error) {
	base = strings.Trim(base, "-")
	if base == "" {
		base = "tenant"
	}
	for attempts := 0; attempts < 80; attempts++ {
		nBig, err := rand.Int(rand.Reader, big.NewInt(100))
		if err != nil {
			return "", err
		}
		suffix := fmt.Sprintf("%02d", nBig.Int64())
		candidate := base + suffix
		if len(candidate) > 64 {
			trunc := base[:64-len(suffix)]
			trunc = strings.TrimRight(strings.TrimSpace(trunc), "-")
			if len(trunc) < 2 {
				trunc = "te"
			}
			candidate = trunc + suffix
		}
		if len(candidate) < 2 || len(candidate) > 64 {
			continue
		}
		if !utils.ValidTenantSlug(candidate) {
			continue
		}
		ok, err := TenantSlugTaken(db, candidate)
		if err != nil {
			return "", err
		}
		if !ok {
			return candidate, nil
		}
	}
	return "", apperror.ErrTenantSlugAllocate
}

// ProvisionTenantWithAdmin creates tenant, system admin role, first admin user, and full permission bindings.
func ProvisionTenantWithAdmin(db *gorm.DB, req TenantProvisionInput, passwordHash string, attachTag string) (tenant Tenant, user TenantUser, role TenantRole, err error) {
	email := utils.TrimLower(req.AdminEmail)
	display := strings.TrimSpace(req.AdminDisplayName)
	if display == "" {
		display = strings.TrimSpace(req.CompanyName)
	}
	if display == "" {
		display = strings.Split(email, "@")[0]
	}
	slugBase := BaseSlugFromCompanyName(req.CompanyName)
	err = db.Transaction(func(tx *gorm.DB) error {
		slug, e := AllocateUniqueTenantSlug(tx, slugBase)
		if e != nil {
			return e
		}
		t := &Tenant{
			Name:             req.CompanyName,
			Slug:             slug,
			Description:      req.TenantDescription,
			Status:           constants.TenantStatusActive,
			ContactEmail:     email,
			MaxUserCount:     req.MaxUserCount,
			BillingMode:      constants.TenantBillingModePrepaid,
			BillingUnlimited: false,
		}
		t.SetCreateInfo(attachTag)
		if t.MaxUserCount <= 0 {
			t.MaxUserCount = 5
		}
		if !utils.IsEmail(t.ContactEmail) {
			return apperror.ErrInvalidContactEmail
		}
		if e := CreateTenant(tx, t); e != nil {
			return e
		}
		tenant = *t

		roleRow := &TenantRole{
			TenantID:    tenant.ID,
			Name:        constants.TenantAdminRoleName,
			Description: "组织管理员，注册时自动创建",
			IsSystem:    true,
		}
		roleRow.SetCreateInfo(attachTag)
		if e := CreateTenantRole(tx, roleRow); e != nil {
			return e
		}
		role = *roleRow
		u := &TenantUser{
			TenantID:     tenant.ID,
			Email:        email,
			PasswordHash: passwordHash,
			DisplayName:  display,
			Status:       constants.TenantUserStatusActive,
			Source:       constants.TenantUserSourceRegister,
		}
		u.SetCreateInfo(attachTag)
		if e := CreateTenantUser(tx, u); e != nil {
			return e
		}
		user = *u

		tur := &TenantUserRole{
			TenantUserID: user.ID,
			RoleID:       role.ID,
		}
		tur.SetCreateInfo(attachTag)
		if e := CreateTenantUserRole(tx, tur); e != nil {
			return e
		}
		return AttachAllPermissionsToRole(tx, role.ID, attachTag)
	})
	return tenant, user, role, err
}

// CreateTenant inserts a tenant row.
func CreateTenant(db *gorm.DB, t *Tenant) error {
	return db.Create(t).Error
}

// GetActiveTenantByID returns an active tenant by primary key.
func GetActiveTenantByID(db *gorm.DB, id uint) (Tenant, error) {
	var row Tenant
	err := db.Where("id = ?", id).First(&row).Error
	return row, err
}

// TenantSlugTaken reports whether slug is already used by an active tenant.
func TenantSlugTaken(db *gorm.DB, slug string) (bool, error) {
	var n int64
	err := db.Model(&Tenant{}).Where("slug = ?", slug).Count(&n).Error
	return n > 0, err
}

// ListTenantsPage lists active tenants (platform admin).
func ListTenantsPage(db *gorm.DB, page, size int, search string) ([]Tenant, int64, error) {
	q := db.Model(&Tenant{})
	if s := strings.TrimSpace(search); s != "" {
		like := "%" + s + "%"
		q = q.Where("name LIKE ? OR slug LIKE ?", like, like)
	}
	return utils.FindPage[Tenant](q, page, size, "id DESC", utils.DefaultMaxPageSize)
}

// UpdateActiveTenant patches name / description / status for an active tenant.
func UpdateActiveTenant(db *gorm.DB, id uint, name, description, status, contactEmail string, maxUserCount int, updateBy string) error {
	meta := common.BaseModel{}
	meta.SetUpdateInfo(updateBy)
	updates := map[string]any{
		"updated_at": time.Now(),
	}
	if meta.UpdateBy != "" {
		updates["update_by"] = meta.UpdateBy
	}
	if !utils.IsEmpty(name) {
		updates["name"] = strings.TrimSpace(name)
	}
	if description != "" {
		updates["description"] = strings.TrimSpace(description)
	}
	status = utils.Trim(status)
	if status != "" {
		updates["status"] = status
	}
	if contactEmail != "" {
		updates["contact_email"] = strings.TrimSpace(contactEmail)
	}
	if maxUserCount > 0 {
		updates["max_user_count"] = maxUserCount
	}
	if len(updates) <= 1 {
		return nil
	}
	return db.Model(&Tenant{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// SoftDeleteTenant soft-deletes one tenant row (platform ops).
func SoftDeleteTenant(db *gorm.DB, id uint, updateBy string) error {
	meta := common.BaseModel{}
	meta.SoftDelete(updateBy)
	updates := map[string]any{
		"updated_at": meta.UpdatedAt,
		"deleted_at": meta.DeletedAt,
	}
	if meta.UpdateBy != "" {
		updates["update_by"] = meta.UpdateBy
	}
	return db.Model(&Tenant{}).Where("id = ?", id).Updates(updates).Error
}

// PatchTenantAIConfigJSON updates optional AI config columns.
// `voiceMode` is a non-pointer because empty string == "no change"; the
// handler is responsible for sending the literal value only when the
// operator actually flipped the switch. `realtime` mirrors `asr`/`tts`/`llm`
// semantics: nil = leave column untouched, non-nil = overwrite.
func PatchTenantAIConfigJSON(db *gorm.DB, id uint, asr, tts, llm, realtime *json.RawMessage, voiceMode string, updateBy string) error {
	patch := map[string]any{
		"updated_at": time.Now(),
		"update_by":  updateBy,
	}
	if asr != nil {
		patch["asr_config"] = datatypes.JSON(utils.CloneRawMessage(*asr))
	}
	if tts != nil {
		stripped := tenantcfg.StripTTSVoice([]byte(*tts))
		patch["tts_config"] = datatypes.JSON(utils.CloneRawMessage(json.RawMessage(stripped)))
	}
	if llm != nil {
		patch["llm_config"] = datatypes.JSON(utils.CloneRawMessage(*llm))
	}
	if realtime != nil {
		stripped := tenantcfg.StripRealtimeVoice([]byte(*realtime))
		patch["realtime_config"] = datatypes.JSON(utils.CloneRawMessage(json.RawMessage(stripped)))
	}
	if vm := strings.TrimSpace(voiceMode); vm != "" {
		if vm != "pipeline" && vm != "realtime" {
			vm = "pipeline"
		}
		patch["voice_mode"] = vm
	}
	if len(patch) <= 2 {
		return nil
	}
	return db.Model(&Tenant{}).Where("id = ?", id).Updates(patch).Error
}

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/notification/im"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

// TenantIMChannel is a tenant-scoped enterprise IM outlet (WeCom / Feishu).
type TenantIMChannel struct {
	common.BaseModel

	TenantID   uint   `json:"tenantId" gorm:"index;not null;uniqueIndex:idx_tenant_im_code"`
	Provider   string `json:"provider" gorm:"size:32;not null;index;comment:wecom|feishu"`
	Code       string `json:"code" gorm:"size:64;not null;uniqueIndex:idx_tenant_im_code;comment:渠道编码"`
	Name       string `json:"name" gorm:"size:128;not null"`
	Enabled    bool   `json:"enabled" gorm:"not null;default:true;index"`
	Remark     string `json:"remark,omitempty" gorm:"size:255"`
	ConfigJSON string `json:"-" gorm:"type:text;comment:渠道配置 JSON（含 webhook/secret）"`
}

func (TenantIMChannel) TableName() string { return constants.TENANT_IM_CHANNEL_TABLE_NAME }

// TenantIMChannelPublic is the API-safe view (secrets masked).
type TenantIMChannelPublic struct {
	ID       uint           `json:"id"`
	TenantID uint           `json:"tenantId"`
	Provider string         `json:"provider"`
	Code     string         `json:"code"`
	Name     string         `json:"name"`
	Enabled  bool           `json:"enabled"`
	Remark   string         `json:"remark,omitempty"`
	Config   map[string]any `json:"config"`
}

func (row TenantIMChannel) ToPublic() TenantIMChannelPublic {
	cfg := map[string]any{}
	_ = json.Unmarshal([]byte(strings.TrimSpace(row.ConfigJSON)), &cfg)
	// Mask secrets
	for _, k := range []string{"secret", "appSecret", "webhookUrl"} {
		if v, ok := cfg[k].(string); ok && strings.TrimSpace(v) != "" {
			if k == "webhookUrl" {
				cfg[k+"Set"] = true
				cfg[k] = maskURL(v)
			} else {
				cfg[k+"Set"] = true
				cfg[k] = ""
			}
		}
	}
	return TenantIMChannelPublic{
		ID: row.ID, TenantID: row.TenantID, Provider: row.Provider,
		Code: row.Code, Name: row.Name, Enabled: row.Enabled, Remark: row.Remark, Config: cfg,
	}
}

func maskURL(u string) string {
	u = strings.TrimSpace(u)
	if len(u) <= 16 {
		return "***"
	}
	return u[:12] + "…***"
}

func ListTenantIMChannelsPage(db *gorm.DB, tenantID uint, page, size int) ([]TenantIMChannel, int64, error) {
	q := db.Model(&TenantIMChannel{}).Where("tenant_id = ?", tenantID)
	return utils.FindPage[TenantIMChannel](q, page, size, "id DESC", utils.DefaultMaxPageSize)
}

func ListEnabledTenantIMChannels(db *gorm.DB, tenantID uint) ([]TenantIMChannel, error) {
	var rows []TenantIMChannel
	err := db.Where("tenant_id = ? AND enabled = ?", tenantID, true).Order("id ASC").Find(&rows).Error
	return rows, err
}

func GetTenantIMChannel(db *gorm.DB, id, tenantID uint) (TenantIMChannel, error) {
	var row TenantIMChannel
	err := db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&row).Error
	return row, err
}

func BuildIMChannelConfigJSON(provider string, cfg map[string]any) (string, error) {
	kind := im.NormalizeProvider(provider)
	if kind == "" {
		return "", fmt.Errorf("unsupported im provider: %s", provider)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	webhook, _ := cfg["webhookUrl"].(string)
	if strings.TrimSpace(webhook) == "" {
		return "", errors.New("webhookUrl required")
	}
	if err := utils.ValidateURLForSSRF(strings.TrimSpace(webhook)); err != nil {
		return "", err
	}
	// Keep only known keys
	clean := map[string]any{"webhookUrl": strings.TrimSpace(webhook)}
	for _, k := range []string{"secret", "corpId", "agentId", "appId", "appSecret"} {
		if v, ok := cfg[k].(string); ok && strings.TrimSpace(v) != "" {
			clean[k] = strings.TrimSpace(v)
		}
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return "", err
	}
	// Validate by constructing provider
	if _, err := im.NewProviderFromConfig(kind, string(raw)); err != nil {
		return "", err
	}
	return string(raw), nil
}

func MergeIMSecretsOnUpdate(oldJSON, newJSON string) (string, error) {
	var oldC, newC map[string]any
	if err := json.Unmarshal([]byte(oldJSON), &oldC); err != nil {
		return newJSON, err
	}
	if err := json.Unmarshal([]byte(newJSON), &newC); err != nil {
		return newJSON, err
	}
	if newC == nil {
		newC = map[string]any{}
	}
	for _, k := range []string{"secret", "appSecret", "webhookUrl"} {
		ns, _ := newC[k].(string)
		os, _ := oldC[k].(string)
		if strings.TrimSpace(ns) == "" && strings.TrimSpace(os) != "" {
			newC[k] = os
		}
	}
	out, err := json.Marshal(newC)
	if err != nil {
		return newJSON, err
	}
	return string(out), nil
}

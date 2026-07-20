package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/notification/mail"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

type NotificationChannelUpsertReq struct {
	ChannelType      string `json:"channelType" binding:"required,oneof=email sms"`
	Name             string `json:"name" binding:"required,max=128"`
	SortOrder        int    `json:"sortOrder"`
	Enabled          *bool  `json:"enabled"`
	Remark           string `json:"remark" binding:"max=255"`
	Driver           string `json:"driver"`
	SMTPHost         string `json:"smtpHost"`
	SMTPPort         int64  `json:"smtpPort"`
	SMTPUsername     string `json:"smtpUsername"`
	SMTPPassword     string `json:"smtpPassword"`
	SMTPFrom         string `json:"smtpFrom"`
	SendcloudAPIUser string `json:"sendcloudApiUser"`
	SendcloudAPIKey  string `json:"sendcloudApiKey"`
	SendcloudFrom    string `json:"sendcloudFrom"`
	FromDisplayName  string `json:"fromDisplayName"`
	SMSProvider      string `json:"smsProvider"`
	SMSConfig        any    `json:"smsConfig"`
}

func BuildChannelConfigForUpdate(req NotificationChannelUpsertReq, oldConfigJSON string) (string, error) {
	if strings.TrimSpace(oldConfigJSON) == "" {
		return BuildChannelConfig(req)
	}
	patched := req
	switch strings.ToLower(strings.TrimSpace(req.ChannelType)) {
	case models.NotificationChannelTypeEmail:
		var oldC mail.MailConfig
		if err := json.Unmarshal([]byte(oldConfigJSON), &oldC); err != nil {
			return BuildChannelConfig(req)
		}
		driver := strings.ToLower(strings.TrimSpace(req.Driver))
		if driver == mail.ProviderSendCloud {
			if strings.TrimSpace(patched.SendcloudAPIKey) == "" && oldC.APIKey != "" {
				patched.SendcloudAPIKey = oldC.APIKey
			}
			if strings.TrimSpace(patched.SendcloudFrom) == "" && oldC.From != "" {
				patched.SendcloudFrom = oldC.From
			}
			if strings.TrimSpace(patched.SendcloudAPIUser) == "" && oldC.APIUser != "" {
				patched.SendcloudAPIUser = oldC.APIUser
			}
		}
		if driver == mail.ProviderSMTP {
			if patched.SMTPPassword == "" && oldC.Password != "" {
				patched.SMTPPassword = oldC.Password
			}
			if strings.TrimSpace(patched.SMTPFrom) == "" && oldC.From != "" {
				patched.SMTPFrom = oldC.From
			}
			if strings.TrimSpace(patched.SMTPHost) == "" && oldC.Host != "" {
				patched.SMTPHost = oldC.Host
			}
			if patched.SMTPPort <= 0 && oldC.Port > 0 {
				patched.SMTPPort = oldC.Port
			}
		}
		if strings.TrimSpace(patched.FromDisplayName) == "" && strings.TrimSpace(oldC.FromName) != "" {
			patched.FromDisplayName = oldC.FromName
		}
	case models.NotificationChannelTypeSMS:
		var oldE struct {
			Provider string         `json:"provider"`
			Config   map[string]any `json:"config"`
		}
		if err := json.Unmarshal([]byte(oldConfigJSON), &oldE); err == nil {
			if strings.TrimSpace(patched.SMSProvider) == "" {
				patched.SMSProvider = oldE.Provider
			}
			if patched.SMSConfig == nil {
				patched.SMSConfig = oldE.Config
			} else if m, ok := patched.SMSConfig.(map[string]any); ok && oldE.Config != nil {
				for k, ov := range oldE.Config {
					os, ok := ov.(string)
					if !ok || strings.TrimSpace(os) == "" {
						continue
					}
					if nv, exists := m[k]; !exists || strings.TrimSpace(fmt.Sprint(nv)) == "" {
						m[k] = os
					}
				}
				patched.SMSConfig = m
			}
		}
	}
	return BuildChannelConfig(patched)
}

func BuildChannelConfig(req NotificationChannelUpsertReq) (string, error) {
	switch strings.ToLower(strings.TrimSpace(req.ChannelType)) {
	case models.NotificationChannelTypeEmail:
		switch strings.ToLower(strings.TrimSpace(req.Driver)) {
		case mail.ProviderSMTP:
			return models.BuildEmailChannelConfigJSON(
				mail.ProviderSMTP, req.Name,
				req.SMTPHost, req.SMTPPort, req.SMTPUsername, req.SMTPPassword, req.SMTPFrom, req.FromDisplayName,
				"", "", "",
			)
		case mail.ProviderSendCloud:
			return models.BuildEmailChannelConfigJSON(
				mail.ProviderSendCloud, req.Name,
				"", 0, "", "", "", req.FromDisplayName,
				req.SendcloudAPIUser, req.SendcloudAPIKey, req.SendcloudFrom,
			)
		default:
			return "", fmt.Errorf("未知邮件驱动: %q（仅支持 smtp / sendcloud）", req.Driver)
		}
	case models.NotificationChannelTypeSMS:
		return models.BuildSMSChannelConfigJSON(req.SMSProvider, req.SMSConfig)
	default:
		return "", errors.New("未知 channelType")
	}
}

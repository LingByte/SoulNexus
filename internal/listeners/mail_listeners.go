// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package listeners

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/notification/mail"
	"gorm.io/gorm"
)

func init() {
	notification.RegisterChannelLoader(EnabledMailConfigs)
	notification.RegisterTemplateLoader(loadMailTemplate)
}

// EnabledMailConfigs returns all enabled email channels for the system mail service.
func EnabledMailConfigs(db *gorm.DB) ([]mail.MailConfig, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	var rows []models.NotificationChannel
	if err := db.Where("type = ? AND enabled = ?", models.NotificationChannelTypeEmail, true).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]mail.MailConfig, 0, len(rows))
	for _, row := range rows {
		raw := strings.TrimSpace(row.ConfigJSON)
		if raw == "" {
			continue
		}
		var cfg mail.MailConfig
		if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
			continue
		}
		if strings.TrimSpace(cfg.Name) == "" {
			cfg.Name = row.Name
		}
		out = append(out, cfg)
	}
	if len(out) == 0 {
		return nil, errors.New("no enabled email notification channels")
	}
	return out, nil
}

// loadMailTemplate loads a mail template by code.
func loadMailTemplate(db *gorm.DB, code, locale string) (string, string, error) {
	if db == nil {
		return "", "", errors.New("nil db")
	}
	var tpl models.MailTemplate
	q := db.Where("code = ? AND enabled = ?", code, true)
	if loc := strings.TrimSpace(locale); loc != "" {
		q = q.Where("locale = ?", loc)
	}
	if err := q.Order("id ASC").First(&tpl).Error; err != nil {
		return "", "", err
	}
	return tpl.Subject, tpl.HTMLBody, nil
}

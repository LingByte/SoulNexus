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

// EnabledMailConfigs 返回指定 OrgID 启用的邮件渠道（按 sort_order 升序）。
// 当 orgID == 0 时取系统级渠道。
func EnabledMailConfigs(db *gorm.DB, orgID uint) ([]mail.MailConfig, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	var rows []models.NotificationChannel
	q := db.Where("type = ? AND enabled = ?", models.NotificationChannelTypeEmail, true)
	if orgID > 0 {
		q = q.Where("org_id = ?", orgID)
	} else {
		q = q.Where("org_id = ?", 0)
	}
	if err := q.Order("sort_order ASC, id ASC").Find(&rows).Error; err != nil {
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

// loadMailTemplate 从 DB 取模板；若指定 org 找不到则回退到系统级 (org_id=0)。
func loadMailTemplate(db *gorm.DB, orgID uint, code, locale string) (string, string, error) {
	if db == nil {
		return "", "", errors.New("nil db")
	}
	tpl, err := models.GetMailTemplateByCode(db, orgID, code, locale)
	if err != nil && orgID != 0 {
		tpl, err = models.GetMailTemplateByCode(db, 0, code, locale)
	}
	if err != nil {
		return "", "", err
	}
	return tpl.Subject, tpl.HTMLBody, nil
}

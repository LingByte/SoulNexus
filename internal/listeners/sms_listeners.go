// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package listeners

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/notification/sms"
	"gorm.io/gorm"
)

type smsChannelEnvelope struct {
	Provider string         `json:"provider"`
	Config   map[string]any `json:"config"`
}

// EnabledSMSChannels 返回指定 OrgID 启用的短信渠道（按 sort_order 升序）。
// orgID == 0 时取系统级渠道。
func EnabledSMSChannels(db *gorm.DB, orgID uint) ([]sms.SenderChannel, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	var rows []models.NotificationChannel
	q := db.Where("type = ? AND enabled = ?", models.NotificationChannelTypeSMS, true)
	if orgID > 0 {
		q = q.Where("org_id = ?", orgID)
	} else {
		q = q.Where("org_id = ?", 0)
	}
	if err := q.Order("sort_order ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]sms.SenderChannel, 0, len(rows))
	for _, row := range rows {
		raw := strings.TrimSpace(row.ConfigJSON)
		if raw == "" {
			continue
		}
		var env smsChannelEnvelope
		if err := json.Unmarshal([]byte(raw), &env); err != nil {
			continue
		}
		kind := sms.ProviderKind(strings.ToLower(strings.TrimSpace(env.Provider)))
		if kind == "" {
			continue
		}
		p, err := sms.NewProviderFromKindMap(kind, env.Config)
		if err != nil {
			continue
		}
		out = append(out, sms.SenderChannel{Label: row.Name, Provider: p})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no enabled sms channels for org %d", orgID)
	}
	return out, nil
}

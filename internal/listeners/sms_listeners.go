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

// EnabledSMSChannels returns all enabled SMS channels (system-global).
func EnabledSMSChannels(db *gorm.DB) ([]sms.SenderChannel, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	var rows []models.NotificationChannel
	if err := db.Where("type = ? AND enabled = ?", models.NotificationChannelTypeSMS, true).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error; err != nil {
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
		return nil, fmt.Errorf("no enabled sms channels")
	}
	return out, nil
}

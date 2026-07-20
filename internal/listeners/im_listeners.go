// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package listeners

import (
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/notification/im"
	"gorm.io/gorm"
)

func init() {
	im.RegisterChannelLoader(loadTenantIMChannels)
}

func loadTenantIMChannels(db *gorm.DB, tenantID uint) ([]im.ChannelRow, error) {
	rows, err := models.ListEnabledTenantIMChannels(db, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]im.ChannelRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, im.ChannelRow{
			ID: r.ID, TenantID: r.TenantID, Provider: r.Provider,
			Name: r.Name, Enabled: r.Enabled, ConfigJSON: r.ConfigJSON,
		})
	}
	return out, nil
}

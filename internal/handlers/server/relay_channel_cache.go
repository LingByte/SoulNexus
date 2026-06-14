// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package server

import (
	"fmt"
	"sync"
	"time"

	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"gorm.io/gorm"
)

const relayChannelCacheTTL = 30 * time.Second

type relayChannelCacheEntry struct {
	channels []svcmodels.LLMChannel
	expires  time.Time
}

var relayChannelCache sync.Map

func relayChannelsCacheKey(group, model, protocol string, orgID uint) string {
	return fmt.Sprintf("%s|%s|%s|%d", group, model, protocol, orgID)
}

func relayChannelsForModelCached(db *gorm.DB, group, model, protocol string, orgID uint) ([]svcmodels.LLMChannel, error) {
	key := relayChannelsCacheKey(group, model, protocol, orgID)
	if v, ok := relayChannelCache.Load(key); ok {
		if ent, ok := v.(relayChannelCacheEntry); ok && time.Now().Before(ent.expires) {
			return ent.channels, nil
		}
		relayChannelCache.Delete(key)
	}
	rows, err := relayChannelsForModel(db, group, model, protocol, orgID)
	if err != nil {
		return nil, err
	}
	relayChannelCache.Store(key, relayChannelCacheEntry{
		channels: rows,
		expires:  time.Now().Add(relayChannelCacheTTL),
	})
	return rows, nil
}

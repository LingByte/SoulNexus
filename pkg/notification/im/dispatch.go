// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package im

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ChannelRow is the minimal DB shape needed to build a Provider.
type ChannelRow struct {
	ID         uint
	TenantID   uint
	Provider   string
	Name       string
	Enabled    bool
	ConfigJSON string
}

// ChannelLoader loads enabled IM channels for one tenant.
type ChannelLoader func(db *gorm.DB, tenantID uint) ([]ChannelRow, error)

var channelLoader ChannelLoader

// RegisterChannelLoader wires the application DB loader (called from listeners).
func RegisterChannelLoader(fn ChannelLoader) {
	channelLoader = fn
}

// NewProviderFromConfig builds a provider from kind + JSON config map/object.
func NewProviderFromConfig(provider string, configJSON string) (Provider, error) {
	kind := NormalizeProvider(provider)
	if kind == "" {
		return nil, fmt.Errorf("%w: unknown provider %q", ErrInvalidConfig, provider)
	}
	raw := strings.TrimSpace(configJSON)
	if raw == "" {
		return nil, fmt.Errorf("%w: empty config", ErrInvalidConfig)
	}
	switch kind {
	case ProviderWeCom:
		var cfg WeComConfig
		if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
			return nil, err
		}
		return NewWeCom(cfg)
	case ProviderFeishu:
		var cfg FeishuConfig
		if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
			return nil, err
		}
		return NewFeishu(cfg)
	default:
		return nil, fmt.Errorf("%w: unknown provider %q", ErrInvalidConfig, provider)
	}
}

// Dispatch sends msg to all enabled IM channels for the tenant (best-effort).
func Dispatch(ctx context.Context, db *gorm.DB, lg *zap.Logger, tenantID uint, msg Message) (sent int, err error) {
	if db == nil || tenantID == 0 || channelLoader == nil {
		return 0, nil
	}
	rows, err := channelLoader(db, tenantID)
	if err != nil {
		return 0, err
	}
	var lastErr error
	for _, row := range rows {
		if !row.Enabled {
			continue
		}
		p, perr := NewProviderFromConfig(row.Provider, row.ConfigJSON)
		if perr != nil {
			lastErr = perr
			if lg != nil {
				lg.Warn("im channel build failed",
					zap.Uint("tenant_id", tenantID),
					zap.Uint("channel_id", row.ID),
					zap.Error(perr))
			}
			continue
		}
		if serr := p.Send(ctx, msg); serr != nil {
			lastErr = serr
			if lg != nil {
				lg.Warn("im send failed",
					zap.Uint("tenant_id", tenantID),
					zap.Uint("channel_id", row.ID),
					zap.String("provider", p.Kind()),
					zap.Error(serr))
			}
			continue
		}
		sent++
	}
	return sent, lastErr
}

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"errors"
	"strings"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/rtcsfu"
	"github.com/gin-gonic/gin"
)

func rtcsfuBearerValue(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	const pfx = "Bearer "
	if len(h) > len(pfx) && strings.EqualFold(h[:len(pfx)], pfx) {
		return strings.TrimSpace(h[len(pfx):])
	}
	return ""
}

// authorizeRTCSFUJoin enforces API key and/or join token policy for POST /join.
func (h *Handlers) authorizeRTCSFUJoin(c *gin.Context, roomID, bodyJoinToken string) error {
	cfg := config.GlobalConfig.RTCSFU
	secret := strings.TrimSpace(cfg.JoinTokenSecret)
	apiOK := cfg.APIKey != "" && c.GetHeader("X-RTCSFU-Key") == cfg.APIKey
	if secret != "" {
		tok := rtcsfuBearerValue(c)
		if tok == "" {
			tok = strings.TrimSpace(bodyJoinToken)
		}
		tokOK := tok != "" && rtcsfu.VerifyJoinToken(secret, tok, roomID) == nil
		if apiOK || tokOK {
			return nil
		}
		return errors.New("需要 X-RTCSFU-Key 或有效 join_token / Authorization: Bearer <token>")
	}
	if cfg.APIKey != "" && !apiOK {
		return errors.New("需要请求头 X-RTCSFU-Key")
	}
	return nil
}

// authorizeRTCSFUSignal enforces API key and/or join_token query for WebSocket upgrade.
func (h *Handlers) authorizeRTCSFUSignal(c *gin.Context, roomID string) error {
	cfg := config.GlobalConfig.RTCSFU
	secret := strings.TrimSpace(cfg.JoinTokenSecret)
	apiOK := cfg.APIKey != "" && (c.GetHeader("X-RTCSFU-Key") == cfg.APIKey || c.Query("api_key") == cfg.APIKey)
	if secret != "" {
		tok := strings.TrimSpace(c.Query("join_token"))
		if tok == "" {
			tok = rtcsfuBearerValue(c)
		}
		tokOK := tok != "" && rtcsfu.VerifyJoinToken(secret, tok, roomID) == nil
		if apiOK || tokOK {
			return nil
		}
		return errors.New("需要 api_key / X-RTCSFU-Key 或有效 join_token")
	}
	if cfg.APIKey != "" && !apiOK {
		return errors.New("invalid or missing API key")
	}
	return nil
}

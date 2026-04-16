// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/rtcsfu"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) registerRTCSFUTokenRoute(r *gin.RouterGroup) {
	cfg := config.GlobalConfig.RTCSFU
	if cfg.APIKey == "" || strings.TrimSpace(cfg.JoinTokenSecret) == "" {
		return
	}
	r.POST("/token", h.rtcsfuMintJoinToken)
}

func (h *Handlers) rtcsfuMintJoinToken(c *gin.Context) {
	cfg := config.GlobalConfig.RTCSFU
	if c.GetHeader("X-RTCSFU-Key") != cfg.APIKey {
		response.FailWithCode(c, 401, "未授权", nil)
		return
	}
	var req struct {
		RoomID string `json:"room_id" binding:"required"`
		TTLSec int    `json:"ttl_sec"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}
	ttl := time.Duration(cfg.JoinTokenDefaultTTLSeconds) * time.Second
	if req.TTLSec > 0 {
		ttl = time.Duration(req.TTLSec) * time.Second
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	maxTTL := cfg.JoinTokenMaxTTLSeconds
	if maxTTL <= 0 {
		maxTTL = 86400
	}
	if int(ttl.Seconds()) > maxTTL {
		response.Fail(c, "ttl_sec 过大", nil)
		return
	}
	exp := time.Now().Add(ttl).UTC()
	tok, err := rtcsfu.MintJoinToken(cfg.JoinTokenSecret, req.RoomID, exp)
	if err != nil {
		response.Fail(c, "签发失败", err.Error())
		return
	}
	response.Success(c, "ok", gin.H{
		"room_id":    req.RoomID,
		"join_token": tok,
		"expires_at": exp.Unix(),
	})
}

func (h *Handlers) rtcsfuReady(c *gin.Context) {
	cfg := config.GlobalConfig.RTCSFU
	if !cfg.Enabled || h.sfuEng == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"ready": false, "reason": "embedded_sfu_off"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ready": true})
}

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// rtcsfuAdminAuth checks RTCSFU is on, API key header, and optionally rejects replica processes (primary-only writes).
func (h *Handlers) rtcsfuAdminAuth(c *gin.Context, forbidReplica bool) bool {
	if h.rtcsfu == nil {
		response.Fail(c, "RTCSFU 未启用", nil)
		return false
	}
	if forbidReplica && strings.EqualFold(strings.TrimSpace(config.GlobalConfig.RTCSFU.ClusterRole), "replica") {
		response.FailWithCode(c, 403, "replica 进程不能使用主控写接口", nil)
		return false
	}
	if c.GetHeader("X-RTCSFU-Key") != config.GlobalConfig.RTCSFU.APIKey {
		response.FailWithCode(c, 401, "未授权", nil)
		return false
	}
	return true
}

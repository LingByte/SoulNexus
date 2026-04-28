// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"io"
	"strings"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/rtcsfu"
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

func parseNodesJSONArrayOrObject(raw []byte) ([]rtcsfu.SFUNode, error) {
	if len(raw) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	return rtcsfu.ParseNodesJSONFlexible(raw)
}

func (h *Handlers) rtcsfuRegisterReplicaNodes(c *gin.Context) {
	if !h.rtcsfuAdminAuth(c, true) {
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Fail(c, "读取请求体失败", err.Error())
		return
	}
	nodes, err := parseNodesJSONArrayOrObject(body)
	if err != nil {
		response.Fail(c, "JSON 无效", err.Error())
		return
	}
	if len(nodes) == 0 {
		response.Fail(c, "节点列表为空", nil)
		return
	}
	for _, n := range nodes {
		h.rtcsfu.UpsertReplica(n)
	}
	response.Success(c, "ok", gin.H{"registered": len(nodes)})
}

func (h *Handlers) rtcsfuUnregisterReplicaNode(c *gin.Context) {
	if !h.rtcsfuAdminAuth(c, true) {
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.Fail(c, "缺少 id", nil)
		return
	}
	if !h.rtcsfu.RemoveReplica(rtcsfu.NodeID(id)) {
		response.Fail(c, "未找到该 replica id", nil)
		return
	}
	response.Success(c, "ok", gin.H{"removed": id})
}

func (h *Handlers) rtcsfuReplicaTouch(c *gin.Context) {
	if !h.rtcsfuAdminAuth(c, true) {
		return
	}
	var req struct {
		ID         string `json:"id" binding:"required"`
		TouchToken string `json:"touch_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}
	id := strings.TrimSpace(req.ID)
	cfg := config.GlobalConfig.RTCSFU
	if ts := strings.TrimSpace(cfg.ReplicaTouchHMACSecret); ts != "" {
		tok := rtcsfuBearerValue(c)
		if tok == "" {
			tok = strings.TrimSpace(req.TouchToken)
		}
		if tok == "" {
			response.FailWithCode(c, 401, "未授权", "需要 touch_token 或 Authorization: Bearer <touch_token>")
			return
		}
		if err := rtcsfu.VerifyReplicaTouchToken(ts, tok, id); err != nil {
			response.FailWithCode(c, 401, "touch_token 无效", err.Error())
			return
		}
	}
	if !h.rtcsfu.TouchReplica(rtcsfu.NodeID(id)) {
		response.FailWithCode(c, 404, "未知 replica id", nil)
		return
	}
	response.Success(c, "ok", gin.H{"id": id})
}

func (h *Handlers) rtcsfuListNodes(c *gin.Context) {
	if !h.rtcsfuAdminAuth(c, false) {
		return
	}
	response.Success(c, "ok", gin.H{
		"cluster_role": config.GlobalConfig.RTCSFU.ClusterRole,
		"static":       h.rtcsfu.StaticSnapshot(),
		"replicas":     h.rtcsfu.ReplicaRows(),
		"merged":       h.rtcsfu.MergedSnapshot(),
	})
}

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/rtcsfu"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) registerRTCSFURoutes(r *gin.RouterGroup) {
	g := r.Group("/rtcsfu/v1")
	{
		g.GET("/ready", h.rtcsfuReady)
		g.POST("/join", h.rtcsfuJoin)
		g.GET("/status", h.rtcsfuStatus)
		g.GET("/signal/ws", h.rtcsfuSignalWS)
		g.GET("/p2p/ws", h.rtcsfuP2PSignalWS)
		h.registerRTCSFUTokenRoute(g)
	}
	if config.GlobalConfig.RTCSFU.APIKey != "" {
		g.POST("/admin/reload-nodes", h.rtcsfuReloadNodes)
		g.POST("/admin/nodes/register", h.rtcsfuRegisterReplicaNodes)
		g.POST("/admin/nodes/replica/touch", h.rtcsfuReplicaTouch)
		g.DELETE("/admin/nodes/replica/:id", h.rtcsfuUnregisterReplicaNode)
		g.GET("/admin/nodes", h.rtcsfuListNodes)
	}
}

func (h *Handlers) rtcsfuStatus(c *gin.Context) {
	cfg := config.GlobalConfig.RTCSFU
	n := 0
	if h.rtcsfu != nil {
		n = h.rtcsfu.NodeCount()
	}
	data := gin.H{
		"rtcsfu_enabled":      cfg.Enabled,
		"control_plane_ready": h.rtcsfu != nil,
		"nodes":               n,
		"embedded_sfu_ready":  h.sfuEng != nil,
		"p2p_relay_ready":     h.p2p != nil,
		"cluster_role":        strings.TrimSpace(cfg.ClusterRole),
		"sfu_require_auth":    cfg.APIKey != "" || strings.TrimSpace(cfg.JoinTokenSecret) != "",
		"signal_require_auth": cfg.SignalRequireAuth,
	}
	if h.rtcsfu != nil {
		data["replica_nodes"] = len(h.rtcsfu.ReplicaSnapshot())
		if cfg.ReplicaStaleSeconds > 0 {
			data["replica_stale_seconds"] = cfg.ReplicaStaleSeconds
		}
		if strings.TrimSpace(cfg.ReplicaTouchHMACSecret) != "" {
			data["replica_touch_hmac_enabled"] = true
		}
	}
	if len(h.rtcsfuICEClientJSON) > 0 {
		data["ice_servers"] = json.RawMessage(h.rtcsfuICEClientJSON)
	}
	if h.sfuEng != nil {
		rooms, peers := h.sfuEng.Stats()
		data["sfu_rooms"] = rooms
		data["sfu_peers"] = peers
	}
	response.Success(c, "ok", data)
}

func (h *Handlers) rtcsfuJoin(c *gin.Context) {
	if h.rtcsfu == nil {
		response.Fail(c, "RTCSFU 未启用", "设置 RTCSFU_ENABLED=true 并提供 RTCSFU_NODES JSON 数组")
		return
	}
	var req struct {
		RoomID    string `json:"room_id" binding:"required"`
		Region    string `json:"region"`
		JoinToken string `json:"join_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}
	if err := h.authorizeRTCSFUJoin(c, req.RoomID, req.JoinToken); err != nil {
		response.FailWithCode(c, 401, "未授权", err.Error())
		return
	}
	a, err := h.rtcsfu.Join(rtcsfu.RoomID(req.RoomID), rtcsfu.RegionID(req.Region))
	if err != nil {
		response.Fail(c, "分配 SFU 失败", err.Error())
		return
	}
	if h.db != nil {
		_ = models.UpsertRTCSFURoomAssignment(h.db, req.RoomID, string(a.Node.ID), req.Region, a.Node.SignalURL, a.Node.MediaURL)
	}
	sfuN, p2pN := 0, 0
	if h.sfuEng != nil {
		sfuN = h.sfuEng.PeersInRoom(req.RoomID)
	}
	if h.p2p != nil {
		p2pN = h.p2p.RoomPeerCount(req.RoomID)
	}
	mediaMode := "sfu"
	p2pPairFull := false
	if sfuN == 0 && p2pN < 2 {
		mediaMode = "p2p"
	} else if sfuN == 0 && p2pN >= 2 {
		mediaMode = "sfu"
		p2pPairFull = true
	}
	cfg := config.GlobalConfig.RTCSFU
	clusterRole := strings.TrimSpace(cfg.ClusterRole)
	if clusterRole == "" {
		clusterRole = "standalone"
	}
	out := gin.H{
		"room_id":        string(a.RoomID),
		"node_id":        string(a.Node.ID),
		"region":         string(a.Node.Region),
		"signal_url":     a.Node.SignalURL,
		"media_url":      a.Node.MediaURL,
		"assigned_at":    a.Assigned.Unix(),
		"webrtc_enabled": h.sfuEng != nil,
		"p2p_ws_path":    "/api/rtcsfu/v1/p2p/ws",
		"media_mode":     mediaMode,
		"peers_sfu":      sfuN,
		"peers_p2p_ws":   p2pN,
		"cluster_role":   clusterRole,
	}
	if p2pPairFull {
		out["p2p_pair_full"] = true
		out["migration_hint"] = "已有两人占用 P2P 槽位；新成员请走 SFU。原两人若需与第三人互通，请断开 P2P 后改用 SFU 信令。"
	}
	if len(h.rtcsfuICEClientJSON) > 0 {
		out["ice_servers"] = json.RawMessage(h.rtcsfuICEClientJSON)
	}
	response.Success(c, "ok", out)
}

// rtcsfuReloadNodes replaces the SFU node snapshot at runtime (same JSON shape as RTCSFU_NODES).
// Only registered when RTCSFU_API_KEY is set; requires X-RTCSFU-Key.
func (h *Handlers) rtcsfuReloadNodes(c *gin.Context) {
	if !h.rtcsfuAdminAuth(c, false) {
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Fail(c, "读取请求体失败", err.Error())
		return
	}
	nodes, err := rtcsfu.ParseNodesJSON(body)
	if err != nil {
		response.Fail(c, "JSON 无效", err.Error())
		return
	}
	if len(nodes) == 0 {
		response.Fail(c, "节点列表为空", nil)
		return
	}
	h.rtcsfu.Reload(nodes)
	response.Success(c, "ok", gin.H{"nodes": len(nodes)})
}

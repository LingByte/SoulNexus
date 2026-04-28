// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/sfu"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func (h *Handlers) rtcsfuSignalWS(c *gin.Context) {
	if h.sfuEng == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "RTCSFU is not enabled"})
		return
	}
	cfg := config.GlobalConfig.RTCSFU
	if cfg.SignalRequireAuth && cfg.APIKey == "" && strings.TrimSpace(cfg.JoinTokenSecret) == "" {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "RTCSFU_SIGNAL_REQUIRE_AUTH requires RTCSFU_API_KEY or RTCSFU_JOIN_TOKEN_SECRET"})
		return
	}
	roomID := c.Query("room_id")
	peerID := c.Query("peer_id")
	if roomID == "" || peerID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "room_id and peer_id query params are required"})
		return
	}
	if err := h.authorizeRTCSFUSignal(c, roomID); err != nil {
		sfu.RecordPeerRejected("auth")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	up := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin:     h.signalCheckOrigin,
	}
	conn, err := up.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		sfu.RecordPeerRejected("upgrade")
		return
	}
	if h.rtcsfuWSMaxReadBytes > 0 {
		conn.SetReadLimit(h.rtcsfuWSMaxReadBytes)
	}
	if h.db != nil {
		_ = models.StartRTCSFUMediaSession(h.db, roomID, peerID, c.ClientIP(), c.GetHeader("User-Agent"))
	}
	defer func() {
		_ = conn.Close()
		if h.db != nil {
			_ = models.EndRTCSFUMediaSession(h.db, roomID, peerID)
		}
	}()
	h.sfuEng.HandleConn(conn, roomID, peerID)
}

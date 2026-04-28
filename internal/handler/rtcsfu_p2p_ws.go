// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/sfu"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func (h *Handlers) rtcsfuP2PSignalWS(c *gin.Context) {
	if h.p2p == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "P2P relay is not enabled"})
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
	if err := h.p2p.RoomAllows(roomID, peerID); err != nil {
		if errors.Is(err, sfu.ErrP2PRoomFull) {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error": err.Error(),
				"hint":  "房间已有两人走 P2P 中继；第三人请使用 /api/rtcsfu/v1/signal/ws（SFU）。若全员需 SFU，请先断开 P2P 再连 SFU。",
			})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	_ = h.p2p.Handle(roomID, peerID, conn, h.rtcsfuWSMaxReadBytes)
}

package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

type acdPoolTargetWriteReq struct {
	Name                  string `json:"name"`
	RouteType             string `json:"routeType"`
	SipSource             string `json:"sipSource"` // internal | trunk (SIP only)
	TargetValue           string `json:"targetValue"`
	SipTrunkHost          string `json:"sipTrunkHost"`
	SipTrunkPort          int    `json:"sipTrunkPort"`
	SipTrunkSignalingAddr string `json:"sipTrunkSignalingAddr"`
	SipCallerID           string `json:"sipCallerId"`
	SipCallerDisplayName  string `json:"sipCallerDisplayName"`
	Weight                int    `json:"weight"`
	WorkState             string `json:"workState"`
}

func acdOperator(c *gin.Context) string {
	if s := strings.TrimSpace("19511899044@163.com"); s != "" {
		return s
	}
	return strconv.FormatUint(00001, 10)
}

// acdPoolTargetListItem adds live SIP registration hint for admin list (not stored in acd_pool_targets).
type acdPoolTargetListItem struct {
	models.ACDPoolTarget
	LiveLineOnline bool `json:"liveLineOnline"`
}

func (h *Handlers) listACDPoolTargets(c *gin.Context) {
	page, size := parsePageSize(c)
	list, total, err := models.ListACDPoolTargetsPage(h.db, page, size, c.Query("routeType"))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	out := make([]acdPoolTargetListItem, 0, len(list))
	for _, row := range list {
		item := acdPoolTargetListItem{ACDPoolTarget: row}
		if models.ACDSipInternalLiveLineEligible(row) {
			n, _ := models.CountOnlineSIPUsersByUsername(h.db, row.TargetValue)
			item.LiveLineOnline = n > 0
		} else if row.RouteType == models.ACDPoolRouteTypeWeb {
			item.LiveLineOnline = models.WebSeatLastSeenFresh(row.WebSeatLastSeenAt)
		}
		out = append(out, item)
	}
	response.Success(c, "success", gin.H{"list": out, "total": total, "page": page, "size": size})
}

func (h *Handlers) getACDPoolTarget(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	row, err := models.GetActiveACDPoolTargetByID(h.db, uint(id))
	if err != nil {
		response.Fail(c, "not found", nil)
		return
	}
	item := acdPoolTargetListItem{ACDPoolTarget: row}
	if models.ACDSipInternalLiveLineEligible(row) {
		n, _ := models.CountOnlineSIPUsersByUsername(h.db, row.TargetValue)
		item.LiveLineOnline = n > 0
	} else if row.RouteType == models.ACDPoolRouteTypeWeb {
		item.LiveLineOnline = models.WebSeatLastSeenFresh(row.WebSeatLastSeenAt)
	}
	response.Success(c, "success", item)
}

func (h *Handlers) createACDPoolTarget(c *gin.Context) {
	var req acdPoolTargetWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "invalid body", err.Error())
		return
	}
	rt, ok := models.ParseACDRouteType(req.RouteType)
	if !ok {
		response.Fail(c, "routeType must be sip or web", nil)
		return
	}
	ws := models.NormalizeACDWorkState(req.WorkState)
	now := time.Now()
	sipSrc := ""
	if rt == models.ACDPoolRouteTypeSIP {
		// SIP ACD rows are now unified as outbound trunk-style targets.
		sipSrc = models.ACDSipSourceTrunk
	}
	var webSeen *time.Time
	if rt == models.ACDPoolRouteTypeWeb && ws == models.ACDWorkStateAvailable {
		webSeen = &now
	}
	row := models.NewACDPoolTargetForCreate(
		req.Name, rt, sipSrc, req.TargetValue,
		req.SipTrunkHost, req.SipTrunkPort, req.SipTrunkSignalingAddr,
		req.SipCallerID, req.SipCallerDisplayName,
		req.Weight, ws, now, webSeen,
	)
	op := acdOperator(c)
	if op != "" {
		row.SetCreateInfo(op)
	}
	// Web seat rows should not be duplicated for one operator.
	// Reuse one row and clean up older duplicates.
	if rt == models.ACDPoolRouteTypeWeb && op != "" {
		ctx := c.Request.Context()
		existing, err := models.ListActiveWebACDPoolTargetsByCreateBy(ctx, h.db, op)
		if err != nil {
			response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
			return
		}
		if len(existing) > 0 {
			keep := existing[0]
			updates := models.BuildACDPoolTargetUpdateMap(
				keep, req.Name, rt, "", req.TargetValue,
				"", 0, "",
				"", "",
				req.Weight, ws, now, op,
			)
			if err := h.db.WithContext(ctx).Model(&models.ACDPoolTarget{}).Where("id = ?", keep.ID).Updates(updates).Error; err != nil {
				response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
				return
			}
			if ws == models.ACDWorkStateOffline {
				_ = models.ClearACDPoolTargetWebSeatLastSeen(h.db, keep.ID)
			}
			if len(existing) > 1 {
				dupIDs := make([]uint, 0, len(existing)-1)
				for i := 1; i < len(existing); i++ {
					dupIDs = append(dupIDs, existing[i].ID)
				}
				_, _ = models.SoftDeleteACDPoolTargetsByIDs(ctx, h.db, dupIDs, op)
			}
			updated, _ := models.ReloadACDPoolTargetByID(h.db, keep.ID)
			response.Success(c, "success", updated)
			return
		}
	}
	if err := h.db.Create(&row).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", row)
}

func (h *Handlers) updateACDPoolTarget(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	var req acdPoolTargetWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "invalid body", err.Error())
		return
	}
	rt, ok := models.ParseACDRouteType(req.RouteType)
	if !ok {
		response.Fail(c, "routeType must be sip or web", nil)
		return
	}
	row, err := models.GetActiveACDPoolTargetByID(h.db, uint(id))
	if err != nil {
		response.Fail(c, "not found", nil)
		return
	}
	ws := models.NormalizeACDWorkState(req.WorkState)
	now := time.Now()
	sipSrc := ""
	if rt == models.ACDPoolRouteTypeSIP {
		// SIP ACD rows are now unified as outbound trunk-style targets.
		sipSrc = models.ACDSipSourceTrunk
	}
	op := acdOperator(c)
	updates := models.BuildACDPoolTargetUpdateMap(
		row, req.Name, rt, sipSrc, req.TargetValue,
		req.SipTrunkHost, req.SipTrunkPort, req.SipTrunkSignalingAddr,
		req.SipCallerID, req.SipCallerDisplayName,
		req.Weight, ws, now, op,
	)
	if err := h.db.Model(&row).Updates(updates).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if rt == models.ACDPoolRouteTypeWeb && ws == models.ACDWorkStateOffline {
		_ = models.ClearACDPoolTargetWebSeatLastSeen(h.db, uint(id))
	}
	row, _ = models.ReloadACDPoolTargetByID(h.db, uint(id))
	response.Success(c, "success", row)
}

func (h *Handlers) deleteACDPoolTarget(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	n, err := models.SoftDeleteACDPoolTargetByID(h.db, uint(id), acdOperator(c))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if n == 0 {
		response.Fail(c, "not found", nil)
		return
	}
	response.Success(c, "success", gin.H{"id": id})
}

type webSeatACDHeartbeatReq struct {
	TargetID uint `json:"targetId"`
}

// webSeatACDHeartbeat refreshes web_seat_last_seen_at for the anchored browser row (keepalive).
func (h *Handlers) webSeatACDHeartbeat(c *gin.Context) {
	var req webSeatACDHeartbeatReq
	if err := c.ShouldBindJSON(&req); err != nil || req.TargetID == 0 {
		response.Fail(c, "invalid body: need targetId", nil)
		return
	}
	op := acdOperator(c)
	if op == "" {
		response.Fail(c, "unauthorized", nil)
		return
	}
	row, err := models.GetActiveACDPoolTargetByID(h.db, req.TargetID)
	if err != nil {
		response.Fail(c, "not found", nil)
		return
	}
	if row.RouteType != models.ACDPoolRouteTypeWeb {
		response.Fail(c, "not a web target", nil)
		return
	}
	if !models.WebSeatActorMayTouchRow(row, op) {
		response.Fail(c, "forbidden", nil)
		return
	}
	if err := models.UpdateACDPoolTargetWebSeatHeartbeat(h.db, req.TargetID, op, time.Now()); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", gin.H{"ok": true})
}

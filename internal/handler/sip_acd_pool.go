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
	u := models.CurrentUser(c)
	if u == nil {
		return ""
	}
	if s := strings.TrimSpace(u.Email); s != "" {
		return s
	}
	return strconv.FormatUint(uint64(u.ID), 10)
}

func normalizeACDRouteType(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case models.ACDPoolRouteTypeSIP, models.ACDPoolRouteTypeWeb:
		return s, true
	default:
		return "", false
	}
}

// acdPoolTargetListItem adds live SIP registration hint for admin list (not stored in acd_pool_targets).
type acdPoolTargetListItem struct {
	models.ACDPoolTarget
	LiveLineOnline bool `json:"liveLineOnline"`
}

func normalizeACDSipSource(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == models.ACDSipSourceTrunk || s == "external" {
		return models.ACDSipSourceTrunk
	}
	return models.ACDSipSourceInternal
}

func sipRowLiveLineEligible(row models.ACDPoolTarget) bool {
	if row.RouteType != models.ACDPoolRouteTypeSIP || strings.TrimSpace(row.TargetValue) == "" {
		return false
	}
	src := strings.ToLower(strings.TrimSpace(row.SipSource))
	if src == models.ACDSipSourceTrunk || src == "external" {
		return false
	}
	return true
}

func normalizeACDWorkState(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case models.ACDWorkStateOffline, models.ACDWorkStateAvailable, models.ACDWorkStateRinging,
		models.ACDWorkStateBusy, models.ACDWorkStateACW, models.ACDWorkStateBreak:
		return s
	default:
		return models.ACDWorkStateOffline
	}
}

func normalizeACDTrunkPort(p int) int {
	if p <= 0 || p >= 65536 {
		return 5060
	}
	return p
}

// acdTrunkStorageFields returns DB fields for trunk columns; clears them when not sip+trunk.
func acdTrunkStorageFields(rt, sipSrc string, req *acdPoolTargetWriteReq) (host string, port int, sig string) {
	if rt != models.ACDPoolRouteTypeSIP || sipSrc != models.ACDSipSourceTrunk {
		return "", 0, ""
	}
	host = strings.TrimSpace(req.SipTrunkHost)
	port = normalizeACDTrunkPort(req.SipTrunkPort)
	sig = strings.TrimSpace(req.SipTrunkSignalingAddr)
	return host, port, sig
}

func acdCallerStorageFields(rt string, req *acdPoolTargetWriteReq) (id, disp string) {
	if rt != models.ACDPoolRouteTypeSIP {
		return "", ""
	}
	return strings.TrimSpace(req.SipCallerID), strings.TrimSpace(req.SipCallerDisplayName)
}

func (h *Handlers) listACDPoolTargets(c *gin.Context) {
	page, size := parsePageSize(c)
	q := h.db.Model(&models.ACDPoolTarget{}).Where("is_deleted = ?", models.SoftDeleteStatusActive)
	if rt := strings.TrimSpace(c.Query("routeType")); rt != "" {
		if t, ok := normalizeACDRouteType(rt); ok {
			q = q.Where("route_type = ?", t)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	offset := (page - 1) * size
	var list []models.ACDPoolTarget
	if err := q.Order("weight DESC, id DESC").Offset(offset).Limit(size).Find(&list).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	out := make([]acdPoolTargetListItem, 0, len(list))
	for _, row := range list {
		item := acdPoolTargetListItem{ACDPoolTarget: row}
		if sipRowLiveLineEligible(row) {
			var n int64
			_ = h.db.Model(&models.SIPUser{}).
				Where("is_deleted = ? AND username = ? AND online = ?", models.SoftDeleteStatusActive, strings.TrimSpace(row.TargetValue), true).
				Count(&n).Error
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
	var row models.ACDPoolTarget
	if err := h.db.Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).First(&row).Error; err != nil {
		response.Fail(c, "not found", nil)
		return
	}
	item := acdPoolTargetListItem{ACDPoolTarget: row}
	if sipRowLiveLineEligible(row) {
		var n int64
		_ = h.db.Model(&models.SIPUser{}).
			Where("is_deleted = ? AND username = ? AND online = ?", models.SoftDeleteStatusActive, strings.TrimSpace(row.TargetValue), true).
			Count(&n).Error
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
	rt, ok := normalizeACDRouteType(req.RouteType)
	if !ok {
		response.Fail(c, "routeType must be sip or web", nil)
		return
	}
	ws := normalizeACDWorkState(req.WorkState)
	now := time.Now()
	sipSrc := ""
	if rt == models.ACDPoolRouteTypeSIP {
		sipSrc = normalizeACDSipSource(req.SipSource)
	}
	if rt == models.ACDPoolRouteTypeSIP && sipSrc == models.ACDSipSourceTrunk {
		if strings.TrimSpace(req.TargetValue) == "" || strings.TrimSpace(req.SipTrunkHost) == "" {
			response.Fail(c, "SIP trunk requires targetValue (dial user) and sipTrunkHost (gateway)", nil)
			return
		}
	}
	th, tp, ts := acdTrunkStorageFields(rt, sipSrc, &req)
	cid, cdn := acdCallerStorageFields(rt, &req)
	var webSeen *time.Time
	if rt == models.ACDPoolRouteTypeWeb && ws == models.ACDWorkStateAvailable {
		webSeen = &now
	}
	row := models.ACDPoolTarget{
		Name:                  strings.TrimSpace(req.Name),
		RouteType:             rt,
		SipSource:             sipSrc,
		TargetValue:           strings.TrimSpace(req.TargetValue),
		SipTrunkHost:          th,
		SipTrunkPort:          tp,
		SipTrunkSignalingAddr: ts,
		SipCallerID:           cid,
		SipCallerDisplayName:  cdn,
		Weight:                req.Weight,
		WorkState:             ws,
		WorkStateAt:           &now,
		WebSeatLastSeenAt:     webSeen,
	}
	if op := acdOperator(c); op != "" {
		row.SetCreateInfo(op)
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
	rt, ok := normalizeACDRouteType(req.RouteType)
	if !ok {
		response.Fail(c, "routeType must be sip or web", nil)
		return
	}
	var row models.ACDPoolTarget
	if err := h.db.Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).First(&row).Error; err != nil {
		response.Fail(c, "not found", nil)
		return
	}
	ws := normalizeACDWorkState(req.WorkState)
	now := time.Now()
	sipSrc := ""
	if rt == models.ACDPoolRouteTypeSIP {
		sipSrc = normalizeACDSipSource(req.SipSource)
	}
	if rt == models.ACDPoolRouteTypeSIP && sipSrc == models.ACDSipSourceTrunk {
		if strings.TrimSpace(req.TargetValue) == "" || strings.TrimSpace(req.SipTrunkHost) == "" {
			response.Fail(c, "SIP trunk requires targetValue (dial user) and sipTrunkHost (gateway)", nil)
			return
		}
	}
	th, tp, ts := acdTrunkStorageFields(rt, sipSrc, &req)
	cid, cdn := acdCallerStorageFields(rt, &req)
	updates := map[string]interface{}{
		"name":                      strings.TrimSpace(req.Name),
		"route_type":                rt,
		"sip_source":                sipSrc,
		"target_value":              strings.TrimSpace(req.TargetValue),
		"sip_trunk_host":            th,
		"sip_trunk_port":            tp,
		"sip_trunk_signaling_addr":  ts,
		"sip_caller_id":             cid,
		"sip_caller_display_name":   cdn,
		"weight":                    req.Weight,
		"work_state":                ws,
		"updated_at":                now,
	}
	if row.WorkState != ws {
		updates["work_state_at"] = now
	}
	if rt == models.ACDPoolRouteTypeWeb && ws == models.ACDWorkStateAvailable {
		updates["web_seat_last_seen_at"] = now
	}
	if op := acdOperator(c); op != "" {
		updates["update_by"] = op
	}
	if err := h.db.Model(&row).Updates(updates).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if rt == models.ACDPoolRouteTypeWeb && ws == models.ACDWorkStateOffline {
		_ = h.db.Model(&models.ACDPoolTarget{}).Where("id = ?", id).Update("web_seat_last_seen_at", nil).Error
	}
	_ = h.db.Where("id = ?", id).First(&row).Error
	response.Success(c, "success", row)
}

func (h *Handlers) deleteACDPoolTarget(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	updates := map[string]interface{}{
		"is_deleted": models.SoftDeleteStatusDeleted,
		"updated_at": time.Now(),
	}
	if op := acdOperator(c); op != "" {
		updates["update_by"] = op
	}
	res := h.db.Model(&models.ACDPoolTarget{}).Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).Updates(updates)
	if res.Error != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		response.Fail(c, "not found", nil)
		return
	}
	response.Success(c, "success", gin.H{"id": id})
}

type webSeatACDHeartbeatReq struct {
	TargetID uint `json:"targetId"`
}

func acdWebSeatActorMayTouchRow(row models.ACDPoolTarget, op string) bool {
	op = strings.TrimSpace(op)
	cb := strings.TrimSpace(row.CreateBy)
	if cb == "" {
		return true
	}
	return cb == op
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
	var row models.ACDPoolTarget
	if err := h.db.Where("id = ? AND is_deleted = ?", req.TargetID, models.SoftDeleteStatusActive).First(&row).Error; err != nil {
		response.Fail(c, "not found", nil)
		return
	}
	if row.RouteType != models.ACDPoolRouteTypeWeb {
		response.Fail(c, "not a web target", nil)
		return
	}
	if !acdWebSeatActorMayTouchRow(row, op) {
		response.Fail(c, "forbidden", nil)
		return
	}
	now := time.Now()
	updates := map[string]interface{}{
		"work_state":            models.ACDWorkStateAvailable,
		"work_state_at":         now,
		"web_seat_last_seen_at": now,
		"updated_at":            now,
		"update_by":             op,
	}
	if err := h.db.Model(&models.ACDPoolTarget{}).Where("id = ?", req.TargetID).Updates(updates).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", gin.H{"ok": true})
}

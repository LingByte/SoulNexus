package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"
	"strconv"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

func parsePageSize(c *gin.Context) (page, size int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ = strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

func (h *Handlers) registerSIPContactCenterRoutes(r *gin.RouterGroup) {
	g := r.Group("sip-center")
	{
		g.GET("/users", h.listSIPUsers)
		g.GET("/users/:id", h.getSIPUser)
		g.DELETE("/users/:id", h.deleteSIPUser)

		g.GET("/calls", h.listSIPCalls)
		g.GET("/calls/:id", h.getSIPCall)

		g.GET("/acd-pool", h.listACDPoolTargets)
		g.POST("/acd-pool/web-seat/heartbeat", h.webSeatACDHeartbeat)
		g.GET("/acd-pool/:id", h.getACDPoolTarget)
		g.POST("/acd-pool", h.createACDPoolTarget)
		g.PUT("/acd-pool/:id", h.updateACDPoolTarget)
		g.DELETE("/acd-pool/:id", h.deleteACDPoolTarget)

		g.GET("/scripts", h.listSIPScriptTemplates)
		g.GET("/scripts/:id", h.getSIPScriptTemplate)
		g.POST("/scripts", h.createSIPScriptTemplate)
		g.PUT("/scripts/:id", h.updateSIPScriptTemplate)
		g.DELETE("/scripts/:id", h.deleteSIPScriptTemplate)

		g.POST("/campaigns", h.createSIPCampaign)
		g.GET("/campaigns", h.listSIPCampaigns)
		g.POST("/campaigns/:id/contacts", h.addSIPCampaignContacts)
		g.GET("/campaigns/:id/contacts", h.listSIPCampaignContacts)
		g.POST("/campaigns/:id/contacts/reset-suppressed", h.resetSIPCampaignSuppressedContacts)
		g.POST("/campaigns/:id/start", h.startSIPCampaign)
		g.POST("/campaigns/:id/pause", h.pauseSIPCampaign)
		g.POST("/campaigns/:id/resume", h.resumeSIPCampaign)
		g.POST("/campaigns/:id/stop", h.stopSIPCampaign)
		g.DELETE("/campaigns/:id", h.deleteSIPCampaign)
		g.GET("/campaigns/metrics", h.getSIPCampaignMetrics)
		g.GET("/campaigns/worker-metrics", h.getSIPCampaignWorkerMetrics)
		g.GET("/campaigns/:id/logs", h.getSIPCampaignLogs)
	}
}

func (h *Handlers) listSIPUsers(c *gin.Context) {
	page, size := parsePageSize(c)
	list, total, err := models.ListSIPUsersPage(h.db, page, size)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", gin.H{"list": list, "total": total, "page": page, "size": size})
}

func (h *Handlers) getSIPUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	row, err := models.GetActiveSIPUserByID(h.db, uint(id))
	if err != nil {
		response.Fail(c, "not found", nil)
		return
	}
	response.Success(c, "success", row)
}

func (h *Handlers) deleteSIPUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	rows, err := models.SoftDeleteSIPUserByID(h.db, uint(id))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if rows == 0 {
		response.Fail(c, "not found", nil)
		return
	}
	response.Success(c, "success", gin.H{"id": id})
}

func (h *Handlers) listSIPCalls(c *gin.Context) {
	page, size := parsePageSize(c)
	list, total, err := models.ListSIPCallsPage(h.db, page, size, c.Query("callId"), c.Query("state"))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", gin.H{"list": list, "total": total, "page": page, "size": size})
}

func (h *Handlers) getSIPCall(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	row, err := models.GetActiveSIPCallByID(h.db, uint(id))
	if err != nil {
		response.Fail(c, "not found", nil)
		return
	}
	response.Success(c, "success", row)
}

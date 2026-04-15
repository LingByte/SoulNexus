package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/gin-gonic/gin"
)

// registerGroupRoutes Group Module
func (h *Handlers) registerGroupRoutes(r *gin.RouterGroup) {
	group := r.Group("group")
	group.Use(models.AuthRequired)
	{
		group.POST("", h.CreateGroup)
		group.GET("", h.ListGroups)

		group.GET("/search-users", h.SearchUsers)

		group.GET("/invitations", h.ListInvitations)
		group.POST("/invitations/:id/accept", h.AcceptInvitation)
		group.POST("/invitations/:id/reject", h.RejectInvitation)

		group.GET("/:id/overview/config", h.GetOverviewConfig)
		group.POST("/:id/overview/config", h.SaveOverviewConfig)
		group.PUT("/:id/overview/config", h.SaveOverviewConfig)
		group.DELETE("/:id/overview/config", h.DeleteOverviewConfig)

		group.GET("/:id/statistics", h.GetGroupStatistics)

		group.POST("/:id/leave", h.LeaveGroup)
		group.DELETE("/:id/members/:memberId", h.RemoveMember)
		group.PUT("/:id/members/:memberId/role", h.UpdateMemberRole)

		group.POST("/:id/invite", h.InviteUser)

		group.GET("/:id/resources", h.GetGroupSharedResources)

		group.POST("/:id/avatar", h.UploadGroupAvatar)

		group.POST("/:id/archive", h.ArchiveGroup)
		group.POST("/:id/restore", h.RestoreGroup)
		group.POST("/:id/clone", h.CloneGroup)
		group.GET("/:id/export", h.ExportGroup)
		group.GET("/:id/activity-logs", h.GetGroupActivityLogs)

		group.GET("/:id", h.GetGroup)
		group.PUT("/:id", h.UpdateGroup)
		group.DELETE("/:id", h.DeleteGroup)
	}
}

// registerQuotaRoutes registers quota routes
func (h *Handlers) registerQuotaRoutes(r *gin.RouterGroup) {
	quota := r.Group("quota")
	quota.Use(models.AuthRequired)
	{
		quota.GET("/user", h.ListUserQuotas)
		quota.GET("/user/:type", h.GetUserQuota)
		quota.POST("/user", h.CreateUserQuota)
		quota.PUT("/user/:type", h.UpdateUserQuota)
		quota.DELETE("/user/:type", h.DeleteUserQuota)

		quota.GET("/group/:id", h.ListGroupQuotas)
		quota.GET("/group/:id/:type", h.GetGroupQuota)
		quota.POST("/group/:id", h.CreateGroupQuota)
		quota.PUT("/group/:id/:type", h.UpdateGroupQuota)
		quota.DELETE("/group/:id/:type", h.DeleteGroupQuota)
	}
}

// registerAlertRoutes registers alert routes
func (h *Handlers) registerAlertRoutes(r *gin.RouterGroup) {
	alert := r.Group("alert")
	alert.Use(models.AuthRequired)
	{
		alert.POST("/rules", h.CreateAlertRule)
		alert.GET("/rules", h.ListAlertRules)
		alert.GET("/rules/:id", h.GetAlertRule)
		alert.PUT("/rules/:id", h.UpdateAlertRule)
		alert.DELETE("/rules/:id", h.DeleteAlertRule)

		alert.GET("", h.ListAlerts)
		alert.GET("/:id", h.GetAlert)
		alert.POST("/:id/resolve", h.ResolveAlert)
		alert.POST("/:id/mute", h.MuteAlert)
	}
}

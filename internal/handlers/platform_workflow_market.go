package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) registerPlatformWorkflowMarketRoutes(r *humax.Group) {
	g := r.Group("admin/workflow-market")
	g.Use(middleware.RequirePlatformAdmin())
	{
		g.GET("/stats", h.adminWorkflowMarketStats)
		g.GET("/plugins", h.adminListWorkflowPlugins)
		g.GET("/plugins/:id", h.adminGetWorkflowPlugin)
		g.PUT("/plugins/:id/status", h.adminUpdateWorkflowPluginStatus)
		g.DELETE("/plugins/:id", h.adminDeleteWorkflowPlugin)

		g.GET("/workflows", h.adminListWorkflowDefinitions)
		g.GET("/workflows/:id", h.adminGetWorkflowDefinition)
		g.DELETE("/workflows/:id", h.adminDeleteWorkflowDefinition)

		g.GET("/node-plugins", h.adminListNodePlugins)
		g.GET("/node-plugins/:id", h.adminGetNodePlugin)
		g.DELETE("/node-plugins/:id", h.adminDeleteNodePlugin)
	}
}

type workflowMarketStats struct {
	PluginTotal     int64            `json:"pluginTotal"`
	PluginPublished int64            `json:"pluginPublished"`
	PluginDraft     int64            `json:"pluginDraft"`
	PluginArchived  int64            `json:"pluginArchived"`
	InstallTotal    int64            `json:"installTotal"`
	DownloadTotal   int64            `json:"downloadTotal"`
	WorkflowTotal   int64            `json:"workflowTotal"`
	WorkflowActive  int64            `json:"workflowActive"`
	NodePluginTotal int64            `json:"nodePluginTotal"`
	NodePluginPub   int64            `json:"nodePluginPublished"`
	ByCategory      map[string]int64 `json:"byCategory"`
	TopDownloaded   []gin.H          `json:"topDownloaded"`
	RecentPublished []gin.H          `json:"recentPublished"`
}

func (h *Handlers) adminWorkflowMarketStats(c *gin.Context) {
	stats := workflowMarketStats{ByCategory: map[string]int64{}}

	_ = h.db.Model(&models.WorkflowPlugin{}).Count(&stats.PluginTotal).Error
	_ = h.db.Model(&models.WorkflowPlugin{}).Where("status = ?", models.WorkflowPluginStatusPublished).Count(&stats.PluginPublished).Error
	_ = h.db.Model(&models.WorkflowPlugin{}).Where("status = ?", models.WorkflowPluginStatusDraft).Count(&stats.PluginDraft).Error
	_ = h.db.Model(&models.WorkflowPlugin{}).Where("status = ?", models.WorkflowPluginStatusArchived).Count(&stats.PluginArchived).Error
	_ = h.db.Model(&models.WorkflowPluginInstallation{}).Where("status = ?", "active").Count(&stats.InstallTotal).Error
	_ = h.db.Model(&models.WorkflowDefinition{}).Count(&stats.WorkflowTotal).Error
	_ = h.db.Model(&models.WorkflowDefinition{}).Where("status = ?", "active").Count(&stats.WorkflowActive).Error
	_ = h.db.Model(&models.NodePlugin{}).Count(&stats.NodePluginTotal).Error
	_ = h.db.Model(&models.NodePlugin{}).Where("status = ?", models.NodePluginStatusPublished).Count(&stats.NodePluginPub).Error

	var downloadSum int64
	_ = h.db.Model(&models.WorkflowPlugin{}).Select("COALESCE(SUM(download_count),0)").Scan(&downloadSum).Error
	stats.DownloadTotal = downloadSum

	type catRow struct {
		Category string
		Cnt      int64
	}
	var cats []catRow
	_ = h.db.Model(&models.WorkflowPlugin{}).
		Select("category, COUNT(*) as cnt").
		Group("category").
		Scan(&cats).Error
	for _, row := range cats {
		stats.ByCategory[row.Category] = row.Cnt
	}

	var top []models.WorkflowPlugin
	_ = h.db.Model(&models.WorkflowPlugin{}).
		Order("download_count DESC, id DESC").
		Limit(10).
		Find(&top).Error
	for _, p := range top {
		stats.TopDownloaded = append(stats.TopDownloaded, gin.H{
			"id":            p.ID,
			"name":          p.Name,
			"displayName":   p.DisplayName,
			"category":      p.Category,
			"status":        p.Status,
			"downloadCount": p.DownloadCount,
			"starCount":     p.StarCount,
			"rating":        p.Rating,
		})
	}

	var recent []models.WorkflowPlugin
	_ = h.db.Model(&models.WorkflowPlugin{}).
		Where("status = ?", models.WorkflowPluginStatusPublished).
		Order("updated_at DESC").
		Limit(10).
		Find(&recent).Error
	for _, p := range recent {
		stats.RecentPublished = append(stats.RecentPublished, gin.H{
			"id":          p.ID,
			"name":        p.Name,
			"displayName": p.DisplayName,
			"category":    p.Category,
			"updatedAt":   p.UpdatedAt,
		})
	}

	response.Success(c, "ok", stats)
}

func (h *Handlers) adminListWorkflowPlugins(c *gin.Context) {
	page, size := ginutil.QueryPage(c, 20)
	q := h.db.Model(&models.WorkflowPlugin{})
	if s := strings.TrimSpace(c.Query("status")); s != "" {
		q = q.Where("status = ?", s)
	}
	if cat := strings.TrimSpace(c.Query("category")); cat != "" {
		q = q.Where("category = ?", cat)
	}
	if kw := strings.TrimSpace(c.Query("search")); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name LIKE ? OR slug LIKE ? OR display_name LIKE ? OR author LIKE ?", like, like, like, like)
	}
	var total int64
	if ginutil.WriteInternalError(c, q.Count(&total).Error) {
		return
	}
	var list []models.WorkflowPlugin
	if ginutil.WriteInternalError(c, q.Order("id DESC").Offset((page-1)*size).Limit(size).Find(&list).Error) {
		return
	}
	ginutil.PageSuccess(c, list, total, page, size)
}

func (h *Handlers) adminGetWorkflowPlugin(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var row models.WorkflowPlugin
	if err := h.db.Preload("Versions").Preload("Reviews").First(&row, id).Error; ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	response.Success(c, "ok", row)
}

func (h *Handlers) adminUpdateWorkflowPluginStatus(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if !ginutil.BindJSON(c, &req) {
		return
	}
	status := strings.TrimSpace(req.Status)
	switch models.WorkflowPluginStatus(status) {
	case models.WorkflowPluginStatusDraft, models.WorkflowPluginStatusPublished, models.WorkflowPluginStatusArchived:
	default:
		response.Fail(c, "invalid status", nil)
		return
	}
	res := h.db.Model(&models.WorkflowPlugin{}).Where("id = ?", id).Update("status", status)
	if ginutil.WriteInternalError(c, res.Error) {
		return
	}
	if res.RowsAffected == 0 {
		response.Fail(c, "not found", nil)
		return
	}
	response.Success(c, "ok", gin.H{"id": id, "status": status})
}

func (h *Handlers) adminDeleteWorkflowPlugin(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("plugin_id = ?", id).Delete(&models.WorkflowPluginInstallation{}).Error; err != nil {
			return err
		}
		if err := tx.Where("plugin_id = ?", id).Delete(&models.WorkflowPluginVersion{}).Error; err != nil {
			return err
		}
		if err := tx.Where("plugin_id = ?", id).Delete(&models.WorkflowPluginReview{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.WorkflowPlugin{}, id).Error
	})
	if ginutil.WriteInternalError(c, err) {
		return
	}
	response.Success(c, "ok", gin.H{"id": id})
}

func (h *Handlers) adminListWorkflowDefinitions(c *gin.Context) {
	page, size := ginutil.QueryPage(c, 20)
	q := h.db.Model(&models.WorkflowDefinition{})
	if s := strings.TrimSpace(c.Query("status")); s != "" {
		q = q.Where("status = ?", s)
	}
	if kw := strings.TrimSpace(c.Query("search")); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name LIKE ? OR slug LIKE ? OR description LIKE ?", like, like, like)
	}
	var total int64
	if ginutil.WriteInternalError(c, q.Count(&total).Error) {
		return
	}
	var list []models.WorkflowDefinition
	if ginutil.WriteInternalError(c, q.Order("id DESC").Offset((page-1)*size).Limit(size).Find(&list).Error) {
		return
	}
	ginutil.PageSuccess(c, list, total, page, size)
}

func (h *Handlers) adminGetWorkflowDefinition(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var row models.WorkflowDefinition
	if err := h.db.First(&row, id).Error; ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	response.Success(c, "ok", row)
}

func (h *Handlers) adminDeleteWorkflowDefinition(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if err := h.db.Delete(&models.WorkflowDefinition{}, id).Error; ginutil.WriteInternalError(c, err) {
		return
	}
	response.Success(c, "ok", gin.H{"id": id})
}

func (h *Handlers) adminListNodePlugins(c *gin.Context) {
	page, size := ginutil.QueryPage(c, 20)
	q := h.db.Model(&models.NodePlugin{})
	if s := strings.TrimSpace(c.Query("status")); s != "" {
		q = q.Where("status = ?", s)
	}
	if kw := strings.TrimSpace(c.Query("search")); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name LIKE ? OR slug LIKE ? OR display_name LIKE ?", like, like, like)
	}
	var total int64
	if ginutil.WriteInternalError(c, q.Count(&total).Error) {
		return
	}
	var list []models.NodePlugin
	if ginutil.WriteInternalError(c, q.Order("id DESC").Offset((page-1)*size).Limit(size).Find(&list).Error) {
		return
	}
	ginutil.PageSuccess(c, list, total, page, size)
}

func (h *Handlers) adminGetNodePlugin(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var row models.NodePlugin
	if err := h.db.First(&row, id).Error; ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	response.Success(c, "ok", row)
}

func (h *Handlers) adminDeleteNodePlugin(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if err := h.db.Delete(&models.NodePlugin{}, id).Error; ginutil.WriteInternalError(c, err) {
		return
	}
	response.Success(c, "ok", gin.H{"id": id})
}

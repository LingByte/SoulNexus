// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/notification/im"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
)

type tenantIMChannelCreateReq struct {
	Provider string         `json:"provider"`
	Code     string         `json:"code"`
	Name     string         `json:"name"`
	Remark   string         `json:"remark"`
	Enabled  *bool          `json:"enabled"`
	Config   map[string]any `json:"config"`
}

type tenantIMChannelUpdateReq struct {
	Name    *string        `json:"name"`
	Remark  *string        `json:"remark"`
	Enabled *bool          `json:"enabled"`
	Config  map[string]any `json:"config"`
}

func (h *Handlers) registerIMChannelRoutes(r *humax.Group) {
	g := r.Group("im-channels")
	g.Use(middleware.RequireHumanJWTUser())
	read := g.Group("")
	read.Use(middleware.RequireTenantPermissionAll("api.webhooks.read"))
	{
		read.GET("", h.listTenantIMChannels)
		read.GET("/providers", h.listIMProviders)
	}
	write := g.Group("")
	write.Use(middleware.RequireTenantPermissionAll("api.webhooks.write"))
	{
		write.POST("", h.createTenantIMChannel)
		write.PUT("/:id", h.updateTenantIMChannel)
		write.DELETE("/:id", h.deleteTenantIMChannel)
		write.POST("/:id/test", h.testTenantIMChannel)
	}
}

func (h *Handlers) listIMProviders(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"providers": []string{im.ProviderWeCom, im.ProviderFeishu},
	})
}

func (h *Handlers) listTenantIMChannels(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	page, size := ginutil.QueryPage(c, 50)
	list, total, err := models.ListTenantIMChannelsPage(h.db, tid, page, size)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	out := make([]models.TenantIMChannelPublic, 0, len(list))
	for _, row := range list {
		out = append(out, row.ToPublic())
	}
	ginutil.PageSuccess(c, out, total, page, size)
}

func (h *Handlers) createTenantIMChannel(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	var req tenantIMChannelCreateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	provider := im.NormalizeProvider(req.Provider)
	if provider == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidParams))
		return
	}
	name := strings.TrimSpace(req.Name)
	code := strings.TrimSpace(req.Code)
	if code == "" {
		code = provider
	}
	if name == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyNameURLRequired))
		return
	}
	cfgJSON, err := models.BuildIMChannelConfigJSON(provider, req.Config)
	if err != nil {
		response.Render(c, response.WrapErr(response.CodeBadRequest, err))
		return
	}
	row := models.TenantIMChannel{
		TenantID:   tid,
		Provider:   provider,
		Code:       code,
		Name:       name,
		Remark:     strings.TrimSpace(req.Remark),
		ConfigJSON: cfgJSON,
		Enabled:    true,
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if op := middleware.AuditOperator(c); op != "" {
		row.SetCreateInfo(op)
	}
	if err := h.db.Create(&row).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionCreate,
		Resource: "tenant_im_channel", ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Created IM channel %s", row.Name), Detail: audit.Redact(req),
	}, nil, row.ToPublic())
	response.SuccessI18n(c, i18n.KeySuccess, row.ToPublic())
}

func (h *Handlers) updateTenantIMChannel(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantIMChannel(h.db, id, tid)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	before := row.ToPublic()
	var req tenantIMChannelUpdateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	u := map[string]any{}
	if req.Name != nil {
		if n := strings.TrimSpace(*req.Name); n != "" {
			u["name"] = n
		}
	}
	if req.Remark != nil {
		u["remark"] = strings.TrimSpace(*req.Remark)
	}
	if req.Enabled != nil {
		u["enabled"] = *req.Enabled
	}
	if req.Config != nil {
		cfgJSON, err := models.BuildIMChannelConfigJSON(row.Provider, req.Config)
		if err != nil {
			// Allow empty webhook when keeping existing secret via merge
			merged, merr := models.MergeIMSecretsOnUpdate(row.ConfigJSON, mustJSON(req.Config))
			if merr != nil {
				response.Render(c, response.WrapErr(response.CodeBadRequest, err))
				return
			}
			if _, perr := im.NewProviderFromConfig(row.Provider, merged); perr != nil {
				response.Render(c, response.WrapErr(response.CodeBadRequest, err))
				return
			}
			u["config_json"] = merged
		} else {
			merged, _ := models.MergeIMSecretsOnUpdate(row.ConfigJSON, cfgJSON)
			u["config_json"] = merged
		}
	}
	if len(u) == 0 {
		response.SuccessI18n(c, i18n.KeySuccess, row.ToPublic())
		return
	}
	if op := middleware.AuditOperator(c); op != "" {
		u["update_by"] = op
	}
	if err := h.db.Model(&row).Updates(u).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	row, _ = models.GetTenantIMChannel(h.db, id, tid)
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionUpdate,
		Resource: "tenant_im_channel", ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Updated IM channel %s", row.Name), Detail: audit.Redact(req),
	}, before, row.ToPublic())
	response.SuccessI18n(c, i18n.KeySuccess, row.ToPublic())
}

func mustJSON(m map[string]any) string {
	raw, err := json.Marshal(m)
	if err != nil || len(raw) == 0 {
		return "{}"
	}
	return string(raw)
}

func (h *Handlers) deleteTenantIMChannel(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantIMChannel(h.db, id, tid)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	if err := h.db.Delete(&row).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionDelete,
		Resource: "tenant_im_channel", ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Deleted IM channel %s", row.Name),
	}, row.ToPublic(), nil)
	response.SuccessI18n(c, i18n.KeySuccess, nil)
}

func (h *Handlers) testTenantIMChannel(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantIMChannel(h.db, id, tid)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	p, err := im.NewProviderFromConfig(row.Provider, row.ConfigJSON)
	if err != nil {
		response.Render(c, response.WrapErr(response.CodeBadRequest, err))
		return
	}
	if err := p.Send(context.Background(), im.Message{
		Title:   "SoulNexus 测试通知",
		Content: "这是一条来自 SoulNexus 的 IM 渠道测试消息。",
	}); err != nil {
		response.Render(c, response.WrapErr(response.CodeBadRequest, err))
		return
	}
	response.SuccessI18n(c, i18n.KeySent, gin.H{"provider": row.Provider})
}

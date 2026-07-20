// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/chat"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) dialogChat() *chat.Service {
	return chat.New(h.db, logger.Lg.Named("dialog-chat"))
}

func (h *Handlers) registerDialogProtectedRoutes(r *humax.Group) {
	g := r.Group(constants.LingechoDialogPathPrefix)
	g.POST("/conversations", h.createDialogConversation)
	g.POST("/conversations/:id/messages", h.postDialogMessage)
	g.GET("/conversations/:id/messages", h.listDialogMessages)
	g.POST("/conversations/:id/end", h.endDialogConversation)

	ch := g.Group("channels")
	ch.Use(middleware.RequireHumanJWTUser())
	read := ch.Group("")
	read.Use(middleware.RequireTenantPermissionAll("api.webhooks.read"))
	{
		read.GET("", h.listDialogChannels)
		read.GET("/providers", h.listDialogChannelProviders)
	}
	write := ch.Group("")
	write.Use(middleware.RequireTenantPermissionAll("api.webhooks.write"))
	{
		write.POST("", h.createDialogChannel)
		write.PUT("/:id", h.updateDialogChannel)
		write.DELETE("/:id", h.deleteDialogChannel)
	}

	sk := g.Group("skills")
	sk.Use(middleware.RequireHumanJWTUser())
	skRead := sk.Group("")
	skRead.Use(middleware.RequireTenantPermissionAny("api.assistants.read", "menu.res.assistant"))
	{
		skRead.GET("", h.listDialogSkills)
		skRead.GET("/:id", h.getDialogSkill)
	}
	skWrite := sk.Group("")
	skWrite.Use(middleware.RequireTenantPermissionAny("api.assistants.write", "menu.res.assistant"))
	{
		skWrite.POST("", h.createDialogSkill)
		skWrite.PUT("/:id", h.updateDialogSkill)
		skWrite.DELETE("/:id", h.deleteDialogSkill)
		skWrite.POST("/:id/assets", h.uploadDialogSkillAssets)
	}
}

type createDialogConvReq struct {
	AssistantID string `json:"assistantId"`
	Channel     string `json:"channel"`
	ExternalID  string `json:"externalUserId"`
}

func (h *Handlers) createDialogConversation(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	var req createDialogConvReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	assistantID := utils.ParseOptionalID(req.AssistantID)
	if assistantID == 0 {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidParams))
		return
	}
	ch := strings.TrimSpace(req.Channel)
	if ch == "" {
		ch = models.DialogChannelAPI
	}
	ext := strings.TrimSpace(req.ExternalID)
	if ext == "" {
		ext = "api-user"
		if op := middleware.AuditOperator(c); op != "" {
			ext = "user-" + op
		}
	}
	svc := h.dialogChat()
	conv, err := svc.EnsureConversation(c.Request.Context(), chat.EnsureParams{
		TenantID:       tid,
		AssistantID:    assistantID,
		Channel:        ch,
		ExternalUserID: ext,
	})
	if ginutil.WriteInternalError(c, err) {
		return
	}
	welcome := ""
	callbinding.SetAssistantID(conv.CallKey(), conv.AssistantID)
	callbinding.SetTenantID(conv.CallKey(), conv.TenantID)
	if env, ok, rErr := tenantcfg.Resolve(c.Request.Context(), tid, conv.CallKey()); rErr == nil && ok {
		welcome = tenantcfg.ResolvedAssistantWelcome(env)
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"id":          strconv.FormatUint(uint64(conv.ID), 10),
		"assistantId": strconv.FormatUint(uint64(conv.AssistantID), 10),
		"channel":     conv.Channel,
		"status":      conv.Status,
		"welcomeText": welcome,
	})
}

type postDialogMsgReq struct {
	Text   string `json:"text"`
	Stream *bool  `json:"stream"`
}

func (h *Handlers) postDialogMessage(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var req postDialogMsgReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	wantStream := c.Query("stream") == "1" || c.Query("stream") == "true"
	if req.Stream != nil && *req.Stream {
		wantStream = true
	}
	if wantStream {
		h.postDialogMessageSSE(c, tid, id, req.Text)
		return
	}
	res, err := h.dialogChat().HandleUserText(c.Request.Context(), tid, id, req.Text)
	if err != nil {
		if strings.Contains(err.Error(), "closed") || strings.Contains(err.Error(), "not found") {
			response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
			return
		}
		ginutil.WriteInternalError(c, err)
		return
	}
	payload := gin.H{
		"reply":     res.Reply,
		"latencyMs": res.LatencyMs,
	}
	if res.Confidence != nil {
		payload["confidence"] = *res.Confidence
	}
	response.SuccessI18n(c, i18n.KeySuccess, payload)
}

func (h *Handlers) postDialogMessageSSE(c *gin.Context, tid, conversationID uint, text string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		ginutil.WriteInternalError(c, errors.New("streaming unsupported"))
		return
	}
	writeEvent := func(event string, data any) {
		raw, _ := json.Marshal(data)
		_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, raw)
		flusher.Flush()
	}
	res, err := h.dialogChat().HandleUserTextStream(c.Request.Context(), tid, conversationID, text, chat.StreamHandlers{
		OnStage: func(stage string) {
			writeEvent("message.stage", gin.H{"stage": stage})
		},
		OnDelta: func(delta string) {
			writeEvent("message.delta", gin.H{"delta": delta})
		},
	})
	if err != nil {
		writeEvent("error", gin.H{"message": err.Error()})
		return
	}
	completed := gin.H{
		"reply":     res.Reply,
		"latencyMs": res.LatencyMs,
	}
	if res.Confidence != nil {
		completed["confidence"] = *res.Confidence
	}
	if res.ToolsJSON != "" {
		completed["toolsJson"] = res.ToolsJSON
	}
	writeEvent("message.completed", completed)
}

func (h *Handlers) listDialogMessages(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if _, err := h.dialogChat().GetConversation(c.Request.Context(), tid, id); err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	msgs, err := h.dialogChat().ListMessages(c.Request.Context(), id, 200)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(msgs))
	for _, m := range msgs {
		row := gin.H{
			"id": strconv.FormatUint(uint64(m.ID), 10), "role": m.Role, "content": m.Content,
			"latencyMs": m.LatencyMs, "createdAt": m.CreatedAt,
		}
		if strings.TrimSpace(m.ToolsJSON) != "" {
			row["toolsJson"] = m.ToolsJSON
		}
		if m.Confidence != nil {
			row["confidence"] = *m.Confidence
		}
		if strings.TrimSpace(m.ConfidenceJSON) != "" {
			row["confidenceJson"] = m.ConfidenceJSON
		}
		out = append(out, row)
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"messages": out})
}

func (h *Handlers) endDialogConversation(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if err := h.dialogChat().EndConversation(c.Request.Context(), tid, id); err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"ok": true})
}

type dialogChannelCreateReq struct {
	Provider    string         `json:"provider"`
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	AssistantID string         `json:"assistantId"`
	Remark      string         `json:"remark"`
	Enabled     *bool          `json:"enabled"`
	Config      map[string]any `json:"config"`
}

type dialogChannelUpdateReq struct {
	Name        *string        `json:"name"`
	AssistantID *string        `json:"assistantId"`
	Remark      *string        `json:"remark"`
	Enabled     *bool          `json:"enabled"`
	Config      map[string]any `json:"config"`
}

func (h *Handlers) listDialogChannelProviders(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"providers": []string{models.DialogProviderWeComApp, models.DialogProviderWeChatOA},
	})
}

func (h *Handlers) listDialogChannels(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	page, size := ginutil.QueryPage(c, 50)
	list, total, err := models.ListTenantDialogChannelsPage(h.db, tid, page, size)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	out := make([]models.TenantDialogChannelPublic, 0, len(list))
	for _, row := range list {
		out = append(out, row.ToPublic())
	}
	ginutil.PageSuccess(c, out, total, page, size)
}

func (h *Handlers) createDialogChannel(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	var req dialogChannelCreateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	provider := models.NormalizeDialogProvider(req.Provider)
	assistantID := utils.ParseOptionalID(req.AssistantID)
	if provider == "" || assistantID == 0 || strings.TrimSpace(req.Name) == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidParams))
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		code = provider
	}
	cfgJSON, err := models.BuildDialogChannelConfigJSON(provider, req.Config)
	if err != nil {
		response.Render(c, response.WrapErr(response.CodeBadRequest, err))
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := models.TenantDialogChannel{
		TenantID: tid, Provider: provider, Code: code, Name: strings.TrimSpace(req.Name),
		AssistantID: assistantID, Enabled: enabled, Remark: strings.TrimSpace(req.Remark),
		ConfigJSON: cfgJSON,
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
		Resource: "tenant_dialog_channel", ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Created dialog channel %s", row.Name), Detail: audit.Redact(req),
	}, nil, row.ToPublic())
	response.SuccessI18n(c, i18n.KeySuccess, row.ToPublic())
}

func (h *Handlers) updateDialogChannel(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantDialogChannel(h.db, id, tid)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	before := row.ToPublic()
	var req dialogChannelUpdateReq
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
	if req.AssistantID != nil {
		if aid := utils.ParseOptionalID(*req.AssistantID); aid > 0 {
			u["assistant_id"] = aid
		}
	}
	if req.Config != nil {
		oldCfg := map[string]any{}
		_ = json.Unmarshal([]byte(row.ConfigJSON), &oldCfg)
		for k, v := range req.Config {
			if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
				continue
			}
			oldCfg[k] = v
		}
		cfgJSON, err := models.BuildDialogChannelConfigJSON(row.Provider, oldCfg)
		if err != nil {
			response.Render(c, response.WrapErr(response.CodeBadRequest, err))
			return
		}
		u["config_json"] = cfgJSON
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
	row, _ = models.GetTenantDialogChannel(h.db, id, tid)
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionUpdate,
		Resource: "tenant_dialog_channel", ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Updated dialog channel %s", row.Name), Detail: audit.Redact(req),
	}, before, row.ToPublic())
	response.SuccessI18n(c, i18n.KeySuccess, row.ToPublic())
}

func (h *Handlers) deleteDialogChannel(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantDialogChannel(h.db, id, tid)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	before := row.ToPublic()
	if err := h.db.Delete(&row).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionDelete,
		Resource: "tenant_dialog_channel", ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Deleted dialog channel %s", row.Name),
	}, before, nil)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"ok": true})
}

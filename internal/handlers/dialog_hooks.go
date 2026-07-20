// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/channels/wechat_oa"
	"github.com/LingByte/SoulNexus/pkg/dialog/channels/wecom"
	"github.com/LingByte/SoulNexus/pkg/dialog/chat"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *Handlers) registerDialogPublicRoutes(r *humax.Group) {
	g := r.Group(constants.LingechoDialogPathPrefix)
	g.GET("/hooks/wecom/:tenantId/:code", h.wecomDialogHook)
	g.POST("/hooks/wecom/:tenantId/:code", h.wecomDialogHook)
	g.GET("/hooks/wechat_oa/:tenantId/:code", h.wechatOADialogHook)
	g.POST("/hooks/wechat_oa/:tenantId/:code", h.wechatOADialogHook)
}

func (h *Handlers) loadDialogChannel(c *gin.Context, wantProvider string) (models.TenantDialogChannel, bool) {
	tidStr := strings.TrimSpace(c.Param("tenantId"))
	code := strings.TrimSpace(c.Param("code"))
	tid, err := utils.ParseID(tidStr)
	if err != nil || code == "" {
		c.String(http.StatusBadRequest, "bad tenant/code")
		return models.TenantDialogChannel{}, false
	}
	row, err := models.GetTenantDialogChannelByCode(h.db, tid, code)
	if err != nil || !row.Enabled || row.Provider != wantProvider {
		c.String(http.StatusNotFound, "channel not found")
		return models.TenantDialogChannel{}, false
	}
	return row, true
}

func (h *Handlers) wecomDialogHook(c *gin.Context) {
	row, ok := h.loadDialogChannel(c, models.DialogProviderWeComApp)
	if !ok {
		return
	}
	cfgMap, err := models.ParseDialogChannelConfig(row)
	if err != nil {
		c.String(http.StatusInternalServerError, "bad config")
		return
	}
	client, err := wecom.NewClient(wecom.Config{
		CorpID: cfgMap["corpId"], AgentID: cfgMap["agentId"], Secret: cfgMap["secret"],
		Token: cfgMap["token"], EncodingAESKey: cfgMap["encodingAESKey"],
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "crypt init failed")
		return
	}

	msgSig := firstQuery(c, "msg_signature", "msg_signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")

	if c.Request.Method == http.MethodGet {
		echostr := c.Query("echostr")
		plain, err := client.VerifyURL(msgSig, timestamp, nonce, echostr)
		if err != nil {
			c.String(http.StatusForbidden, "verify failed")
			return
		}
		c.String(http.StatusOK, plain)
		return
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		c.String(http.StatusBadRequest, "read body")
		return
	}
	msg, err := client.ParseInbound(msgSig, timestamp, nonce, body)
	if err != nil {
		logger.Warn("wecom dialog parse failed", zap.Error(err))
		c.String(http.StatusForbidden, "parse failed")
		return
	}
	if !strings.EqualFold(msg.MsgType, "text") || strings.TrimSpace(msg.Content) == "" {
		c.String(http.StatusOK, "success")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	svc := h.dialogChat()
	conv, err := svc.EnsureConversation(ctx, chat.EnsureParams{
		TenantID:         row.TenantID,
		AssistantID:      row.AssistantID,
		Channel:          models.DialogChannelWeCom,
		ChannelAccountID: cfgMap["corpId"] + ":" + cfgMap["agentId"],
		ExternalUserID:   msg.FromUserName,
	})
	if err != nil {
		logger.Warn("wecom ensure conversation failed", zap.Error(err))
		c.String(http.StatusOK, "success")
		return
	}
	turn, err := svc.HandleUserText(ctx, row.TenantID, conv.ID, msg.Content)
	if err != nil {
		logger.Warn("wecom dialog turn failed", zap.Error(err))
		// Fall back to proactive send empty / skip
		c.String(http.StatusOK, "success")
		return
	}

	// Prefer proactive send so long LLM turns don't miss the callback window.
	go func() {
		sendCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := client.SendText(sendCtx, msg.FromUserName, turn.Reply); err != nil {
			logger.Warn("wecom proactive send failed", zap.Error(err))
		}
	}()
	c.String(http.StatusOK, "success")
}

func (h *Handlers) wechatOADialogHook(c *gin.Context) {
	row, ok := h.loadDialogChannel(c, models.DialogProviderWeChatOA)
	if !ok {
		return
	}
	cfgMap, err := models.ParseDialogChannelConfig(row)
	if err != nil {
		c.String(http.StatusInternalServerError, "bad config")
		return
	}
	client, err := wechat_oa.NewClient(wechat_oa.Config{
		AppID: cfgMap["appId"], AppSecret: cfgMap["appSecret"],
		Token: cfgMap["token"], EncodingAESKey: cfgMap["encodingAESKey"],
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "crypt init failed")
		return
	}

	encryptMode := strings.TrimSpace(c.Query("encrypt_type")) == "aes" ||
		strings.TrimSpace(c.Query("msg_signature")) != ""
	signature := firstQuery(c, "msg_signature", "signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")

	if c.Request.Method == http.MethodGet {
		echostr := c.Query("echostr")
		plain, err := client.VerifyURL(signature, timestamp, nonce, echostr, encryptMode)
		if err != nil {
			c.String(http.StatusForbidden, "verify failed")
			return
		}
		c.String(http.StatusOK, plain)
		return
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		c.String(http.StatusBadRequest, "read body")
		return
	}
	msg, err := client.ParseInbound(signature, timestamp, nonce, body, encryptMode)
	if err != nil {
		logger.Warn("wechat_oa dialog parse failed", zap.Error(err))
		c.String(http.StatusForbidden, "parse failed")
		return
	}

	// Must acknowledge within 5s; process async via customer-service API.
	c.String(http.StatusOK, "success")

	if !strings.EqualFold(msg.MsgType, "text") || strings.TrimSpace(msg.Content) == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		svc := h.dialogChat()
		conv, err := svc.EnsureConversation(ctx, chat.EnsureParams{
			TenantID:         row.TenantID,
			AssistantID:      row.AssistantID,
			Channel:          models.DialogChannelWeChatOA,
			ChannelAccountID: cfgMap["appId"],
			ExternalUserID:   msg.FromUserName,
		})
		if err != nil {
			logger.Warn("wechat_oa ensure conversation failed", zap.Error(err))
			return
		}
		turn, err := svc.HandleUserText(ctx, row.TenantID, conv.ID, msg.Content)
		if err != nil {
			logger.Warn("wechat_oa dialog turn failed", zap.Error(err))
			return
		}
		if err := client.SendCustomerText(ctx, msg.FromUserName, turn.Reply); err != nil {
			logger.Warn("wechat_oa custom send failed", zap.Error(err))
		}
	}()
}

func firstQuery(c *gin.Context, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(c.Query(k)); v != "" {
			return v
		}
	}
	return ""
}

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// LLM Relay：对外暴露 OpenAI / Anthropic 兼容 API。
// 鉴权：LLMToken.APIKey（独立于 UserCredential，不带 API Secret，专注 /v1/* 调用）。
// 路由：上游渠道按 ability(token.group, model) 选出 channel 列表，按 priority 从高到低重试。
// 计费：成功后按 LLMModelMeta 倍率折算 quota_delta，扣减 LLMToken.token_used / quota_used / request_used。
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const relayMaxBodyBytes = 8 * 1024 * 1024

// extractRelayAPIKey 优先 Authorization: Bearer，其次 x-api-key。
func extractRelayAPIKey(c *gin.Context) string {
	if h := strings.TrimSpace(c.GetHeader("Authorization")); h != "" {
		const p = "Bearer "
		if strings.HasPrefix(h, p) {
			return strings.TrimSpace(h[len(p):])
		}
		return h
	}
	return strings.TrimSpace(c.GetHeader("x-api-key"))
}

// relayTokenFromRequest 解析并校验 LLMToken：状态/过期/封禁。无效时直接写入 401/403 并返回 false。
func relayTokenFromRequest(db *gorm.DB, c *gin.Context) (*models.LLMToken, bool) {
	key := extractRelayAPIKey(c)
	if key == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"type": "auth_error", "message": "missing API key"}})
		return nil, false
	}
	t, err := models.LookupActiveLLMToken(db, key)
	if err != nil || t == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"type": "auth_error", "message": "invalid or inactive API key"}})
		return nil, false
	}
	return t, true
}

// relayChannelsForModel 按 (group, model, protocol) 选出可用渠道，priority 从高到低排序。
// orgID 用于多租户隔离：>0 表示只看 OrgID=0(全局) 或 OrgID=orgID(该租户) 的 ability/channel。
func relayChannelsForModel(db *gorm.DB, group, model, protocol string, orgID uint) ([]models.LLMChannel, error) {
	if strings.TrimSpace(group) == "" {
		group = "default"
	}
	type abilityRow struct {
		ChannelID int
	}
	var ab []abilityRow
	abQ := db.Model(&models.LLMAbility{}).
		Select("channel_id").
		Where("`group` = ? AND model = ? AND enabled = ?", group, model, true)
	abQ = models.ScopeOrgVisible(abQ, orgID)
	if err := abQ.Order("priority DESC, weight DESC").Scan(&ab).Error; err != nil {
		return nil, err
	}
	if len(ab) == 0 {
		return nil, nil
	}
	ids := make([]int, 0, len(ab))
	for _, r := range ab {
		ids = append(ids, r.ChannelID)
	}
	var rows []models.LLMChannel
	q := db.Model(&models.LLMChannel{}).Where("id IN ? AND status = ? AND protocol = ?", ids, 1, protocol)
	q = models.ScopeOrgVisible(q, orgID)
	if err := q.Order("priority DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func toUpstreamChannel(ch *models.LLMChannel) llm.UpstreamChannel {
	// 多 Key 模式（is_multi_key=true）下从 ch.Key（多行/逗号分隔）按 random/polling 选一把。
	// 单 Key 模式直接透传 ch.Key。
	mode := string(ch.ChannelInfo.MultiKeyMode)
	picked := llm.PickChannelKey(ch.Id, ch.Key, ch.ChannelInfo.IsMultiKey, mode)
	return llm.UpstreamChannel{
		ID:                 ch.Id,
		Key:                picked,
		BaseURL:            ch.BaseURL,
		OpenAIOrganization: ch.OpenAIOrganization,
	}
}

func parseRelayBodyModel(body []byte) string {
	var m struct {
		Model string `json:"model"`
	}
	_ = json.Unmarshal(body, &m)
	return strings.TrimSpace(m.Model)
}

func parseRelayBodyStream(body []byte) bool {
	var m struct {
		Stream bool `json:"stream"`
	}
	_ = json.Unmarshal(body, &m)
	return m.Stream
}

// parseUsageFromOpenAIResponse 解析 OpenAI /v1/chat/completions 返回体的 usage 字段。
func parseUsageFromOpenAIResponse(buf []byte) (prompt, completion, total int) {
	var w struct {
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(buf, &w) != nil || w.Usage == nil {
		return 0, 0, 0
	}
	prompt = w.Usage.PromptTokens
	completion = w.Usage.CompletionTokens
	total = w.Usage.TotalTokens
	if total == 0 {
		total = prompt + completion
	}
	return
}

// parseUsageFromAnthropicResponse 解析 Anthropic /v1/messages 返回体的 usage 字段。
func parseUsageFromAnthropicResponse(buf []byte) (prompt, completion, total int) {
	var w struct {
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(buf, &w) != nil || w.Usage == nil {
		return 0, 0, 0
	}
	prompt = w.Usage.InputTokens
	completion = w.Usage.OutputTokens
	total = prompt + completion
	return
}

// computeQuotaDelta 按 LLMModelMeta 倍率折算"额度单位"：
//
//	delta = (prompt × prompt_ratio + completion × completion_ratio) × model_ratio
//
// 没有 meta 时按 1.0 倍率返回总 token 数。
func computeQuotaDelta(db *gorm.DB, model string, prompt, completion int) int {
	var meta models.LLMModelMeta
	if err := db.Where("model_name = ?", model).First(&meta).Error; err != nil {
		return prompt + completion
	}
	pr := meta.QuotaPromptRatio
	if pr <= 0 {
		pr = 1
	}
	cr := meta.QuotaCompletionRatio
	if cr <= 0 {
		cr = 1
	}
	mr := meta.QuotaModelRatio
	if mr <= 0 {
		mr = 1
	}
	v := (float64(prompt)*pr + float64(completion)*cr) * mr
	return int(v + 0.5)
}

func relayMeta(c *gin.Context, t *models.LLMToken) llm.RelayUsageMeta {
	uid := ""
	if t != nil && t.UserID > 0 {
		uid = formatUint(t.UserID)
	}
	return llm.RelayUsageMeta{
		UserIDStr: uid,
		UserAgent: c.GetHeader("User-Agent"),
		ClientIP:  c.ClientIP(),
	}
}

func formatUint(u uint) string {
	const digits = "0123456789"
	if u == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for u > 0 {
		i--
		buf[i] = digits[u%10]
		u /= 10
	}
	return string(buf[i:])
}

// preflightTokenForModel 在请求 upstream 前完成 (额度 / 模型白名单) 校验。
func (h *Handlers) preflightTokenForModel(c *gin.Context, t *models.LLMToken, model string) bool {
	if exceeded, why := t.QuotaExceeded(); exceeded {
		// 异步发一条站内通知（best-effort，不阻塞客户端响应）
		go func(userID uint, name, reason string) {
			defer func() { _ = recover() }()
			n := name
			if strings.TrimSpace(n) == "" {
				n = "Token"
			}
			_ = models.NewInternalNotificationService(h.db).Send(
				userID,
				"LLM Token 调用被拦截",
				"您的 LLM Token「"+n+"」因 "+reason+" 被拦截，请检查配额或调整额度。",
			)
		}(t.UserID, t.Name, why)
		c.JSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"type": "quota_exceeded", "message": why}})
		return false
	}
	if !t.AllowsModel(model) {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"type": "model_not_allowed", "message": "model not allowed by this token"}})
		return false
	}
	return true
}

// requireTokenType 校验 token.type 必须等于期望值；为空兼容历史数据（视为 llm）。
func requireTokenType(c *gin.Context, tok *models.LLMToken, want models.LLMTokenType) bool {
	if tok == nil {
		return false
	}
	got := tok.Type
	if got == "" {
		got = models.LLMTokenTypeLLM
	}
	if got != want {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
			"type":    "token_type_mismatch",
			"message": "this token type (" + string(got) + ") cannot access " + string(want) + " endpoints",
		}})
		return false
	}
	return true
}

// handleRelayOpenAIChat POST /v1/chat/completions
func (h *Handlers) handleRelayOpenAIChat(c *gin.Context) {
	tok, ok := relayTokenFromRequest(h.db, c)
	if !ok {
		return
	}
	if !requireTokenType(c, tok, models.LLMTokenTypeLLM) {
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, relayMaxBodyBytes))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "bad_request", "message": "read body failed"}})
		return
	}
	model := parseRelayBodyModel(body)
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "bad_request", "message": "model is required"}})
		return
	}
	if !h.preflightTokenForModel(c, tok, model) {
		return
	}
	channels, err := relayChannelsForModel(h.db, tok.Group, model, models.LLMChannelProtocolOpenAI, models.TokenOrgID(tok))
	if err != nil || len(channels) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"type": "no_channel", "message": "no channel available for model"}})
		return
	}
	meta := relayMeta(c, tok)
	stream := parseRelayBodyStream(body)
	if stream {
		body = llm.EnsureOpenAIChatStreamIncludeUsage(body)
		// 多渠道流式 fallback：在收到 upstream 响应头之前不向 client 写入；429/408/5xx 自动切下一个 channel。
		ups := make([]llm.UpstreamChannel, 0, len(channels))
		for i := range channels {
			ups = append(ups, toUpstreamChannel(&channels[i]))
		}
		cap, attempts, winID, err := llm.RelayOpenAIStreamMulti(c.Request.Context(), body, c.GetHeader("Accept"), ups, c.Writer)
		if err != nil || cap == nil {
			emsg := "all_channels_failed"
			if err != nil {
				emsg = err.Error()
			}
			llm.EmitRelayOpenAIUsageFailure(body, &llm.RelayResult{
				WinChannelID: winID,
				Attempts:     attempts,
				AllFailed:    true,
			}, meta, "stream_error", emsg)
			return
		}
		quota := computeQuotaDelta(h.db, model, cap.PromptTokens, cap.CompletionTokens)
		// 流式必须用 *Stream* 版本：非流版会因 FinalBody 为空而 early-return，导致 llm_usage 不落库。
		baseURL := ""
		for i := range channels {
			if channels[i].Id == winID {
				if channels[i].BaseURL != nil {
					baseURL = *channels[i].BaseURL
				}
				break
			}
		}
		llm.EmitRelayOpenAIStreamUsageSuccess(body, meta, cap, winID, baseURL, attempts, quota)
		_ = models.AddLLMTokenUsage(h.db, tok.ID, cap.PromptTokens+cap.CompletionTokens, quota)
		return
	}

	res := relayMultiOpenAINonStream(c.Request.Context(), body, c.GetHeader("Accept"), channels)
	if res.AllFailed {
		llm.EmitRelayOpenAIUsageFailure(body, res, meta, "all_failed", "")
		c.Data(res.FinalStatus, "application/json", res.FinalBody)
		return
	}
	llm.CopyRelayResponseHeaders(c.Writer.Header(), res.FinalHeader)
	c.Data(res.FinalStatus, "application/json", res.FinalBody)
	pt, ct, tt := parseUsageFromOpenAIResponse(res.FinalBody)
	quota := computeQuotaDelta(h.db, model, pt, ct)
	llm.EmitRelayOpenAIUsageSuccess(body, res, meta, quota)
	_ = models.AddLLMTokenUsage(h.db, tok.ID, tt, quota)
}

// relayMultiOpenAINonStream 按 priority 顺序遍历 channels，第一个 BusinessOK 即返回。
func relayMultiOpenAINonStream(ctx context.Context, body []byte, accept string, channels []models.LLMChannel) *llm.RelayResult {
	t0 := time.Now()
	res := &llm.RelayResult{}
	for i := range channels {
		one := llm.RelayOpenAIChatCompletionsOnce(ctx, body, accept, toUpstreamChannel(&channels[i]), i+1)
		res.Attempts = append(res.Attempts, one.Attempt)
		if one.BusinessOK {
			res.WinChannelID = channels[i].Id
			if channels[i].BaseURL != nil {
				res.WinBaseURL = *channels[i].BaseURL
			}
			res.FinalStatus = one.Status
			res.FinalBody = one.Body
			res.FinalHeader = one.Header
			res.WinHopMs = one.Attempt.LatencyMs
			res.WallLatencyMs = time.Since(t0).Milliseconds()
			return res
		}
		res.FinalStatus = one.Status
		res.FinalBody = one.Body
		res.FinalHeader = one.Header
	}
	res.AllFailed = true
	res.WallLatencyMs = time.Since(t0).Milliseconds()
	if res.FinalStatus == 0 {
		res.FinalStatus = http.StatusBadGateway
	}
	if len(res.FinalBody) == 0 {
		res.FinalBody = []byte(`{"error":{"type":"all_channels_failed","message":"all upstream channels failed"}}`)
	}
	return res
}

// handleRelayAnthropicMessages POST /v1/messages
func (h *Handlers) handleRelayAnthropicMessages(c *gin.Context) {
	tok, ok := relayTokenFromRequest(h.db, c)
	if !ok {
		return
	}
	if !requireTokenType(c, tok, models.LLMTokenTypeLLM) {
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, relayMaxBodyBytes))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "bad_request", "message": "read body failed"}})
		return
	}
	model := parseRelayBodyModel(body)
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "bad_request", "message": "model is required"}})
		return
	}
	if !h.preflightTokenForModel(c, tok, model) {
		return
	}
	channels, err := relayChannelsForModel(h.db, tok.Group, model, models.LLMChannelProtocolAnthropic, models.TokenOrgID(tok))
	if err != nil || len(channels) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"type": "no_channel", "message": "no channel available for model"}})
		return
	}
	meta := relayMeta(c, tok)
	stream := parseRelayBodyStream(body)
	anthropicVer := c.GetHeader("anthropic-version")
	anthropicBeta := c.GetHeader("anthropic-beta")
	if stream {
		ups := make([]llm.UpstreamChannel, 0, len(channels))
		for i := range channels {
			ups = append(ups, toUpstreamChannel(&channels[i]))
		}
		cap, attempts, winID, err := llm.RelayAnthropicStreamMulti(c.Request.Context(), body, anthropicVer, anthropicBeta, ups, c.Writer)
		if err != nil || cap == nil {
			emsg := "all_channels_failed"
			if err != nil {
				emsg = err.Error()
			}
			llm.EmitRelayAnthropicUsageFailure(body, &llm.RelayResult{
				WinChannelID: winID,
				Attempts:     attempts,
				AllFailed:    true,
			}, meta, "stream_error", emsg)
			return
		}
		quota := computeQuotaDelta(h.db, model, cap.PromptTokens, cap.CompletionTokens)
		baseURL := ""
		for i := range channels {
			if channels[i].Id == winID {
				if channels[i].BaseURL != nil {
					baseURL = *channels[i].BaseURL
				}
				break
			}
		}
		llm.EmitRelayAnthropicStreamUsageSuccess(body, meta, cap, winID, baseURL, attempts, quota)
		_ = models.AddLLMTokenUsage(h.db, tok.ID, cap.PromptTokens+cap.CompletionTokens, quota)
		// 纯转发调用不再写 chat_sessions / chat_messages；全量访问走 llm_usage。
		return
	}

	res := relayMultiAnthropicNonStream(c.Request.Context(), body, anthropicVer, anthropicBeta, channels)
	if res.AllFailed {
		llm.EmitRelayAnthropicUsageFailure(body, res, meta, "all_failed", "")
		c.Data(res.FinalStatus, "application/json", res.FinalBody)
		return
	}
	llm.CopyRelayResponseHeaders(c.Writer.Header(), res.FinalHeader)
	c.Data(res.FinalStatus, "application/json", res.FinalBody)
	pt, ct, tt := parseUsageFromAnthropicResponse(res.FinalBody)
	quota := computeQuotaDelta(h.db, model, pt, ct)
	llm.EmitRelayAnthropicUsageSuccess(body, res, meta, quota)
	_ = models.AddLLMTokenUsage(h.db, tok.ID, tt, quota)
}

func relayMultiAnthropicNonStream(ctx context.Context, body []byte, anthropicVer, anthropicBeta string, channels []models.LLMChannel) *llm.RelayResult {
	t0 := time.Now()
	res := &llm.RelayResult{}
	for i := range channels {
		one := llm.RelayAnthropicMessagesOnce(ctx, body, anthropicVer, anthropicBeta, toUpstreamChannel(&channels[i]), i+1)
		res.Attempts = append(res.Attempts, one.Attempt)
		if one.BusinessOK {
			res.WinChannelID = channels[i].Id
			if channels[i].BaseURL != nil {
				res.WinBaseURL = *channels[i].BaseURL
			}
			res.FinalStatus = one.Status
			res.FinalBody = one.Body
			res.FinalHeader = one.Header
			res.WinHopMs = one.Attempt.LatencyMs
			res.WallLatencyMs = time.Since(t0).Milliseconds()
			return res
		}
		res.FinalStatus = one.Status
		res.FinalBody = one.Body
		res.FinalHeader = one.Header
	}
	res.AllFailed = true
	res.WallLatencyMs = time.Since(t0).Milliseconds()
	if res.FinalStatus == 0 {
		res.FinalStatus = http.StatusBadGateway
	}
	if len(res.FinalBody) == 0 {
		res.FinalBody = []byte(`{"type":"error","error":{"type":"all_channels_failed","message":"all upstream channels failed"}}`)
	}
	return res
}

// handleRelayOpenAIModelsList GET /v1/models — 返回当前 token group 下可用的模型 id 列表。
//
// 解析顺序见 models.BuildOpenAPIModelListForToken：
//  1. token.OpenAPIModelCatalogJSON 显式列表（管理员/用户配置）
//  2. llm_abilities WHERE group = token.group AND enabled
//  3. llm_channels WHERE group = token.group AND status=1 的 models 字段
//
// 最终再用 token.ModelWhitelist 裁剪。
func (h *Handlers) handleRelayOpenAIModelsList(c *gin.Context) {
	tok, ok := relayTokenFromRequest(h.db, c)
	if !ok {
		return
	}
	items, err := models.BuildOpenAPIModelListForToken(h.db, tok)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusInternalServerError, errors.New("list models failed"))
		return
	}
	now := time.Now().Unix()
	out := make([]gin.H, 0, len(items))
	for _, it := range items {
		owned := strings.TrimSpace(it.OwnedBy)
		if owned == "" {
			owned = "soulnexus"
		}
		out = append(out, gin.H{
			"id":       it.ID,
			"object":   "model",
			"created":  now,
			"owned_by": owned,
		})
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": out})
}

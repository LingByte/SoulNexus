package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
)

type tenantLLMTestReq struct {
	Prompt    string          `json:"prompt"`
	LLMConfig json.RawMessage `json:"llmConfig"` // optional draft override (unsaved form)
}

// testTenantLLMStream runs a short streaming LLM call against the tenant's
// (or draft) LLM config and returns JSON with first-token latency.
//
// POST /tenants/:id/llm-test  (platform admin)
func (h *Handlers) testTenantLLMStream(c *gin.Context) {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidID))
		return
	}
	var req tenantLLMTestReq
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid params", err))
		return
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = "请用一句话自我介绍。"
	}

	tenant, err := models.GetActiveTenantByID(h.db, id)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}

	llmRaw := []byte(tenant.LlmConfig)
	if len(req.LLMConfig) > 0 && string(req.LLMConfig) != "null" {
		llmRaw = req.LLMConfig
	}
	env, err := tenantcfg.VoiceEnvFromJSON(nil, nil, llmRaw, nil, "")
	if err != nil {
		response.Fail(c, fmt.Sprintf("invalid llmConfig: %v", err), nil)
		return
	}
	if strings.TrimSpace(env.LLMProvider) == "" {
		response.Fail(c, "llm provider is empty", nil)
		return
	}
	if strings.TrimSpace(env.LLMAPIKey) == "" && !strings.EqualFold(env.LLMProvider, "ollama") {
		response.Fail(c, "llm apiKey is empty", nil)
		return
	}

	model := strings.TrimSpace(env.LLMModel)
	if model == "" {
		model = "gpt-4o-mini"
	}
	baseURL := strings.TrimSpace(env.LLMBaseURL)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()

	llm, err := providers.NewChatLLM(ctx, env.LLMProvider, env.LLMAPIKey, baseURL, "你是连通性测试助手，请简短回复。")
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	start := time.Now()
	var firstTokenAt time.Time
	var full strings.Builder
	_, err = llm.QueryStream(prompt, providers.LLMQueryOptions{
		Model:        model,
		Stream:       true,
		DisableTools: true,
		Context:      ctx,
	}, func(segment string, isComplete bool) error {
		if segment != "" {
			if firstTokenAt.IsZero() {
				firstTokenAt = time.Now()
			}
			full.WriteString(segment)
		}
		_ = isComplete
		return nil
	})
	wallMs := int(time.Since(start).Milliseconds())
	firstMs := 0
	if !firstTokenAt.IsZero() {
		firstMs = int(firstTokenAt.Sub(start).Milliseconds())
	}
	if err != nil {
		response.FailWithCode(c, http.StatusBadGateway, err.Error(), nil)
		return
	}
	reply := strings.TrimSpace(full.String())
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"provider":     env.LLMProvider,
		"model":        model,
		"prompt":       prompt,
		"reply":        reply,
		"firstTokenMs": firstMs,
		"wallMs":       wallMs,
		"ok":           reply != "",
	})
}

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

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
)

type credentialLLMTestReq struct {
	Prompt       string          `json:"prompt"`
	CredentialID string          `json:"credentialId"`
	LLMConfig    json.RawMessage `json:"llmConfig"` // draft override (create/edit form)
}

// testCredentialLLMStream runs a short streaming LLM call for Access Keys UI.
// POST /credentials/llm-test (human JWT)
func (h *Handlers) testCredentialLLMStream(c *gin.Context) {
	tenantID := middleware.CurrentTenantID(c)
	if tenantID == 0 {
		response.Render(c, response.NewI18n(response.CodeUnauthorized, i18n.KeyUnauthorized))
		return
	}
	var req credentialLLMTestReq
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid params", err))
		return
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = "请用一句话自我介绍。"
	}

	llmRaw, err := h.resolveCredentialLLMJSON(c, tenantID, req)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
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

func (h *Handlers) resolveCredentialLLMJSON(c *gin.Context, tenantID uint, req credentialLLMTestReq) ([]byte, error) {
	if len(req.LLMConfig) > 0 && string(req.LLMConfig) != "null" {
		return req.LLMConfig, nil
	}
	credID := utils.ParseOptionalID(req.CredentialID)
	if credID == 0 {
		return nil, fmt.Errorf("llmConfig or credentialId is required")
	}
	row, err := models.GetCredentialByIDForTenant(h.db, credID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("credential not found")
	}
	if models.NormalizeCredentialKind(row.Kind) == constants.CredentialKindPlatformBundle {
		bundle := models.ApplyAIProviderPoolsToBundle(h.db, tenantID, "", tenantcfg.VoiceConfigBundle{}, true)
		if len(bundle.Llm) == 0 {
			return nil, fmt.Errorf("platform key has no LLM pool grant for this tenant")
		}
		return bundle.Llm, nil
	}
	raw := []byte(row.LlmConfig)
	if len(strings.TrimSpace(string(raw))) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("credential llmConfig is empty")
	}
	return raw, nil
}

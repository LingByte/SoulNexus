// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"go.uber.org/zap"
)

var scoreRe = regexp.MustCompile(`(?i)(?:score|置信度|分数)\s*[:=：]?\s*(0(?:\.\d+)?|1(?:\.0+)?)`)

type confidenceResult struct {
	Score  float64 `json:"score"`
	Reason string  `json:"reason,omitempty"`
}

// scoreTurnConfidence asks the tenant LLM for a 0–1 confidence on the reply.
// Failures return ok=false; callers should ignore silently.
func scoreTurnConfidence(ctx context.Context, env tenantcfg.VoiceEnv, userText, reply string, lg *zap.Logger) (confidenceResult, bool) {
	if lg == nil {
		lg = zap.NewNop()
	}
	userText = strings.TrimSpace(userText)
	reply = strings.TrimSpace(reply)
	if userText == "" || reply == "" {
		return confidenceResult{}, false
	}
	if !tenantcfg.LLMReady(env) {
		return confidenceResult{}, false
	}
	judgeCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	system := "你是对话质量评审。根据用户问题与助手回答，给出置信度 0~1（1=完全正确且切题）。" +
		"只输出一行：score: <小数> reason: <不超过20字>"
	llm, err := providers.NewChatLLM(judgeCtx, env.LLMProvider, env.LLMAPIKey, llmBaseURL(env), system)
	if err != nil {
		lg.Debug("dialog confidence: llm init failed", zap.Error(err))
		return confidenceResult{}, false
	}
	prompt := fmt.Sprintf("用户：%s\n助手：%s", truncateRunes(userText, 400), truncateRunes(reply, 600))
	temp := float32(0)
	maxTok := 64
	out, err := llm.QueryWithOptions(prompt, providers.LLMQueryOptions{
		Model:       env.LLMModel,
		Temperature: &temp,
		MaxTokens:   &maxTok,
		Context:     judgeCtx,
		DisableTools: true,
	})
	if err != nil {
		lg.Debug("dialog confidence: query failed", zap.Error(err))
		return confidenceResult{}, false
	}
	score, reason, ok := parseConfidenceOutput(out)
	if !ok {
		return confidenceResult{}, false
	}
	return confidenceResult{Score: score, Reason: reason}, true
}

func llmBaseURL(env tenantcfg.VoiceEnv) string {
	return strings.TrimSpace(env.LLMBaseURL)
}

func parseConfidenceOutput(s string) (float64, string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, "", false
	}
	m := scoreRe.FindStringSubmatch(s)
	if len(m) < 2 {
		// bare float fallback
		fields := strings.Fields(s)
		if len(fields) == 0 {
			return 0, "", false
		}
		v, err := strconv.ParseFloat(strings.TrimSuffix(fields[0], ","), 64)
		if err != nil || v < 0 || v > 1 {
			return 0, "", false
		}
		return v, "", true
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil || v < 0 || v > 1 {
		return 0, "", false
	}
	reason := ""
	if i := strings.Index(strings.ToLower(s), "reason"); i >= 0 {
		rest := s[i:]
		if j := strings.IndexAny(rest, ":："); j >= 0 {
			reason = strings.TrimSpace(rest[j+1:])
		}
	}
	return v, truncateRunes(reason, 40), true
}

func confidenceJSON(r confidenceResult) string {
	b, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	return string(b)
}

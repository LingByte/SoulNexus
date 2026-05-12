// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package llm

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

const maxRelayUsageBodyClip = 512 * 1024

func clipForRelayUsageStore(s string) string {
	if len(s) <= maxRelayUsageBodyClip {
		return s
	}
	b := []byte(s)
	n := maxRelayUsageBodyClip
	for n > 0 && n < len(b) && b[n-1]&0xC0 == 0x80 {
		n--
	}
	return string(b[:n]) + "…(truncated)"
}

// ClipRelayUsageBody truncates large JSON/text for usage audit or storage.
func ClipRelayUsageBody(s string) string {
	return clipForRelayUsageStore(s)
}

// RelayUsageMeta is filled by the HTTP layer for relay usage signals (no gin/models).
type RelayUsageMeta struct {
	UserIDStr string
	UserAgent string
	ClientIP  string
}

func relayUsageFailRequestID(prefix string) string {
	if utils.SnowflakeUtil != nil {
		return prefix + utils.SnowflakeUtil.GenID()
	}
	return prefix + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func tpsFromOutputTokens(outTok int, hopMs int64) float64 {
	if outTok <= 0 || hopMs <= 0 {
		return 0
	}
	return float64(outTok) / (float64(hopMs) / 1000.0)
}

// EmitRelayOpenAIUsageSuccess emits a usage signal after non-stream OpenAI-compatible chat completion success.
func EmitRelayOpenAIUsageSuccess(reqBody []byte, res *RelayResult, meta RelayUsageMeta, quotaDelta int) {
	if res == nil || res.WinChannelID <= 0 || len(res.FinalBody) == 0 {
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(res.FinalBody, &raw); err != nil {
		return
	}
	if _, hasErr := raw["error"]; hasErr {
		return
	}
	idBytes, ok := raw["id"]
	if !ok {
		return
	}
	var idStr string
	if err := json.Unmarshal(idBytes, &idStr); err != nil || strings.TrimSpace(idStr) == "" {
		return
	}
	modelStr := ""
	if m, ok := raw["model"]; ok {
		_ = json.Unmarshal(m, &modelStr)
	}
	var usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
		Details          *struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	}
	if u, ok := raw["usage"]; ok {
		_ = json.Unmarshal(u, &usage)
	}
	inTok := usage.PromptTokens
	outTok := usage.CompletionTokens
	tot := usage.TotalTokens
	if tot == 0 {
		tot = inTok + outTok
	}
	now := time.Now()
	ms := now.UnixMilli()
	reqMs := ms
	if cr, ok := raw["created"]; ok {
		var sec int64
		if json.Unmarshal(cr, &sec) == nil && sec > 0 {
			reqMs = sec * 1000
		}
	}
	tps := tpsFromOutputTokens(outTok, res.WinHopMs)
	if quotaDelta < 0 {
		quotaDelta = 0
	}
	payload := &LLMUsageSignalPayload{
		RequestID:       idStr,
		UserID:          strings.TrimSpace(meta.UserIDStr),
		Provider:        "openai",
		Model:           strings.TrimSpace(modelStr),
		BaseURL:         strings.TrimSpace(res.WinBaseURL),
		RequestType:     "openapi_openai_chat_completions",
		ChannelID:       res.WinChannelID,
		ChannelAttempts: res.Attempts,
		InputTokens:     inTok,
		OutputTokens:    outTok,
		TotalTokens:     tot,
		QuotaDelta:      quotaDelta,
		LatencyMs:       res.WallLatencyMs,
		TTFTMs:          res.WinHopMs,
		TPS:             tps,
		QueueTimeMs:     res.QueueMs,
		RequestContent:  clipForRelayUsageStore(string(reqBody)),
		ResponseContent: clipForRelayUsageStore(string(res.FinalBody)),
		UserAgent:       meta.UserAgent,
		IPAddress:       meta.ClientIP,
		StatusCode:      res.FinalStatus,
		Success:         true,
		RequestedAtMs:   reqMs,
		StartedAtMs:     ms,
		FirstTokenAtMs:  0,
		CompletedAtMs:   ms,
	}
	utils.Sig().Emit(SignalLLMUsage, payload)
}

// EmitRelayOpenAIUsageFailure emits usage on non-stream or stream entry failure.
func EmitRelayOpenAIUsageFailure(reqBody []byte, res *RelayResult, meta RelayUsageMeta, errorCode string, extraMsg string) {
	rid := relayUsageFailRequestID("ling-relay-openai-fail-")
	msg := strings.TrimSpace(extraMsg)
	if msg == "" && res != nil && len(res.Attempts) > 0 {
		last := res.Attempts[len(res.Attempts)-1]
		msg = strings.TrimSpace(last.ErrorMessage)
		if msg == "" {
			msg = strings.TrimSpace(last.ErrorCode)
		}
	}
	if msg == "" {
		msg = "openai upstream failed"
	}
	if strings.TrimSpace(errorCode) == "" {
		errorCode = "all_channels_exhausted"
	}
	var wall, queue int64
	var atts []UsageChannelAttempt
	respClip := ""
	httpCode := 502
	if res != nil {
		wall = res.WallLatencyMs
		queue = res.QueueMs
		atts = res.Attempts
		if res.FinalStatus > 0 {
			httpCode = res.FinalStatus
		}
		if len(res.FinalBody) > 0 {
			respClip = clipForRelayUsageStore(string(res.FinalBody))
		}
	}
	ms := time.Now().UnixMilli()
	payload := &LLMUsageSignalPayload{
		RequestID:       rid,
		UserID:          strings.TrimSpace(meta.UserIDStr),
		Provider:        "openai",
		Model:           "",
		BaseURL:         "",
		RequestType:     "openapi_openai_chat_completions",
		ChannelID:       0,
		ChannelAttempts: atts,
		QuotaDelta:      0,
		LatencyMs:       wall,
		TTFTMs:          0,
		TPS:             0,
		QueueTimeMs:     queue,
		RequestContent:  clipForRelayUsageStore(string(reqBody)),
		ResponseContent: respClip,
		UserAgent:       meta.UserAgent,
		IPAddress:       meta.ClientIP,
		StatusCode:      httpCode,
		Success:         false,
		ErrorCode:       strings.TrimSpace(errorCode),
		ErrorMessage:    truncateRelayAttemptMsg(msg, maxRelayAttemptErrBytes),
		RequestedAtMs:   ms,
		StartedAtMs:     ms,
		FirstTokenAtMs:  0,
		CompletedAtMs:   ms,
	}
	utils.Sig().Emit(SignalLLMUsage, payload)
}

// EmitRelayAnthropicUsageSuccess emits usage after non-stream Anthropic /v1/messages success.
func EmitRelayAnthropicUsageSuccess(reqBody []byte, res *RelayResult, meta RelayUsageMeta, quotaDelta int) {
	if res == nil || res.WinChannelID <= 0 {
		return
	}
	var a struct {
		Type  string `json:"type"`
		ID    string `json:"id"`
		Model string `json:"model"`
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(res.FinalBody, &a); err != nil || a.Type == "error" || strings.TrimSpace(a.ID) == "" {
		return
	}
	inTok, outTok := 0, 0
	if a.Usage != nil {
		inTok = a.Usage.InputTokens
		outTok = a.Usage.OutputTokens
	}
	ms := time.Now().UnixMilli()
	tps := tpsFromOutputTokens(outTok, res.WinHopMs)
	if quotaDelta < 0 {
		quotaDelta = 0
	}
	payload := &LLMUsageSignalPayload{
		RequestID:       a.ID,
		UserID:          strings.TrimSpace(meta.UserIDStr),
		Provider:        "anthropic",
		Model:           strings.TrimSpace(a.Model),
		BaseURL:         strings.TrimSpace(res.WinBaseURL),
		RequestType:     "openapi_anthropic_messages",
		ChannelID:       res.WinChannelID,
		ChannelAttempts: res.Attempts,
		InputTokens:     inTok,
		OutputTokens:    outTok,
		TotalTokens:     inTok + outTok,
		QuotaDelta:      quotaDelta,
		LatencyMs:       res.WallLatencyMs,
		TTFTMs:          res.WinHopMs,
		TPS:             tps,
		QueueTimeMs:     res.QueueMs,
		RequestContent:  clipForRelayUsageStore(string(reqBody)),
		ResponseContent: clipForRelayUsageStore(string(res.FinalBody)),
		UserAgent:       meta.UserAgent,
		IPAddress:       meta.ClientIP,
		StatusCode:      res.FinalStatus,
		Success:         true,
		RequestedAtMs:   ms,
		StartedAtMs:     ms,
		FirstTokenAtMs:  0,
		CompletedAtMs:   ms,
	}
	utils.Sig().Emit(SignalLLMUsage, payload)
}

// EmitRelayAnthropicUsageFailure emits usage on Anthropic relay failure.
func EmitRelayAnthropicUsageFailure(reqBody []byte, res *RelayResult, meta RelayUsageMeta, errorCode string, extraMsg string) {
	rid := relayUsageFailRequestID("ling-relay-anthropic-fail-")
	msg := strings.TrimSpace(extraMsg)
	if msg == "" && res != nil && len(res.Attempts) > 0 {
		last := res.Attempts[len(res.Attempts)-1]
		msg = strings.TrimSpace(last.ErrorMessage)
		if msg == "" {
			msg = strings.TrimSpace(last.ErrorCode)
		}
	}
	if msg == "" {
		msg = "anthropic upstream failed"
	}
	if strings.TrimSpace(errorCode) == "" {
		errorCode = "all_channels_exhausted"
	}
	var wall, queue int64
	var atts []UsageChannelAttempt
	respClip := ""
	httpCode := 502
	if res != nil {
		wall = res.WallLatencyMs
		queue = res.QueueMs
		atts = res.Attempts
		if res.FinalStatus > 0 {
			httpCode = res.FinalStatus
		}
		if len(res.FinalBody) > 0 {
			respClip = clipForRelayUsageStore(string(res.FinalBody))
		}
	}
	ms := time.Now().UnixMilli()
	payload := &LLMUsageSignalPayload{
		RequestID:       rid,
		UserID:          strings.TrimSpace(meta.UserIDStr),
		Provider:        "anthropic",
		Model:           "",
		BaseURL:         "",
		RequestType:     "openapi_anthropic_messages",
		ChannelID:       0,
		ChannelAttempts: atts,
		QuotaDelta:      0,
		LatencyMs:       wall,
		QueueTimeMs:     queue,
		RequestContent:  clipForRelayUsageStore(string(reqBody)),
		ResponseContent: respClip,
		UserAgent:       meta.UserAgent,
		IPAddress:       meta.ClientIP,
		StatusCode:      httpCode,
		Success:         false,
		ErrorCode:       strings.TrimSpace(errorCode),
		ErrorMessage:    truncateRelayAttemptMsg(msg, maxRelayAttemptErrBytes),
		RequestedAtMs:   ms,
		StartedAtMs:     ms,
		FirstTokenAtMs:  0,
		CompletedAtMs:   ms,
	}
	utils.Sig().Emit(SignalLLMUsage, payload)
}

// EmitRelayOpenAIStreamUsageSuccess emits usage after streaming chat completion (prefer stream_options.include_usage in request).
func EmitRelayOpenAIStreamUsageSuccess(reqBody []byte, meta RelayUsageMeta, cap *OpenAIStreamCapture, channelID int, baseURL string, attempts []UsageChannelAttempt, quotaDelta int) {
	if cap == nil || strings.TrimSpace(meta.UserIDStr) == "" {
		return
	}
	rid := cap.effectiveRequestID()
	ttft := int64(0)
	if cap.FirstTokenAtMs > 0 && cap.StartedAtMs > 0 {
		ttft = cap.FirstTokenAtMs - cap.StartedAtMs
	}
	tps := tpsFromOutputTokens(cap.CompletionTokens, cap.WallLatencyMs)
	reqMs := cap.StartedAtMs
	if reqMs <= 0 {
		reqMs = time.Now().UnixMilli()
	}
	if quotaDelta < 0 {
		quotaDelta = 0
	}
	payload := &LLMUsageSignalPayload{
		RequestID:       rid,
		UserID:          strings.TrimSpace(meta.UserIDStr),
		Provider:        "openai",
		Model:           strings.TrimSpace(cap.Model),
		BaseURL:         strings.TrimSpace(baseURL),
		RequestType:     "openapi_openai_chat_completions_stream",
		ChannelID:       channelID,
		ChannelAttempts: attempts,
		InputTokens:     cap.PromptTokens,
		OutputTokens:    cap.CompletionTokens,
		TotalTokens:     cap.TotalTokens,
		QuotaDelta:      quotaDelta,
		LatencyMs:       cap.WallLatencyMs,
		TTFTMs:          ttft,
		TPS:             tps,
		QueueTimeMs:     0,
		RequestContent:  clipForRelayUsageStore(string(reqBody)),
		ResponseContent: cap.streamResponseContentForUsage(),
		UserAgent:       meta.UserAgent,
		IPAddress:       meta.ClientIP,
		StatusCode:      cap.StatusCode,
		Success:         true,
		RequestedAtMs:   reqMs,
		StartedAtMs:     cap.StartedAtMs,
		FirstTokenAtMs:  cap.FirstTokenAtMs,
		CompletedAtMs:   cap.CompletedAtMs,
	}
	utils.Sig().Emit(SignalLLMUsage, payload)
}

// EmitRelayAnthropicStreamUsageSuccess emits usage after Anthropic streaming messages.
func EmitRelayAnthropicStreamUsageSuccess(reqBody []byte, meta RelayUsageMeta, cap *AnthropicStreamCapture, channelID int, baseURL string, attempts []UsageChannelAttempt, quotaDelta int) {
	if cap == nil || strings.TrimSpace(meta.UserIDStr) == "" {
		return
	}
	rid := cap.effectiveRequestID()
	ttft := int64(0)
	if cap.FirstTokenAtMs > 0 && cap.StartedAtMs > 0 {
		ttft = cap.FirstTokenAtMs - cap.StartedAtMs
	}
	tps := tpsFromOutputTokens(cap.CompletionTokens, cap.WallLatencyMs)
	reqMs := cap.StartedAtMs
	if reqMs <= 0 {
		reqMs = time.Now().UnixMilli()
	}
	tot := cap.TotalTokens
	if tot == 0 {
		tot = cap.PromptTokens + cap.CompletionTokens
	}
	if quotaDelta < 0 {
		quotaDelta = 0
	}
	payload := &LLMUsageSignalPayload{
		RequestID:       rid,
		UserID:          strings.TrimSpace(meta.UserIDStr),
		Provider:        "anthropic",
		Model:           strings.TrimSpace(cap.Model),
		BaseURL:         strings.TrimSpace(baseURL),
		RequestType:     "openapi_anthropic_messages_stream",
		ChannelID:       channelID,
		ChannelAttempts: attempts,
		InputTokens:     cap.PromptTokens,
		OutputTokens:    cap.CompletionTokens,
		TotalTokens:     tot,
		QuotaDelta:      quotaDelta,
		LatencyMs:       cap.WallLatencyMs,
		TTFTMs:          ttft,
		TPS:             tps,
		QueueTimeMs:     0,
		RequestContent:  clipForRelayUsageStore(string(reqBody)),
		ResponseContent: cap.streamResponseContentForUsage(),
		UserAgent:       meta.UserAgent,
		IPAddress:       meta.ClientIP,
		StatusCode:      cap.StatusCode,
		Success:         true,
		RequestedAtMs:   reqMs,
		StartedAtMs:     cap.StartedAtMs,
		FirstTokenAtMs:  cap.FirstTokenAtMs,
		CompletedAtMs:   cap.CompletedAtMs,
	}
	utils.Sig().Emit(SignalLLMUsage, payload)
}

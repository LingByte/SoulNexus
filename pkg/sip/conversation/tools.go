package conversation

import (
	"encoding/json"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/llm"
	"go.uber.org/zap"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

func registerSIPTransferTool(handler llm.LLMHandler, callID string, lg *zap.Logger) {
	if handler == nil || strings.TrimSpace(callID) == "" {
		return
	}
	params := json.RawMessage(`{
		"type":"object",
		"properties":{
			"reason":{"type":"string","description":"用户请求转人工的简短原因"},
			"confidence":{"type":"number","description":"0到1，当前意图置信度"}
		},
		"required":[],
		"additionalProperties":true
	}`)
	handler.RegisterFunctionTool(
		"transfer_to_agent",
		"当且仅当用户明确要求转人工客服时调用该工具。调用后系统会执行转人工，不要让用户重复确认。",
		params,
		func(args map[string]interface{}, _ interface{}) (string, error) {
			if lg != nil {
				lg.Info("sip voice: transfer tool invoked",
					zap.String("call_id", callID),
					zap.Any("args", args),
				)
			}
			// Keep UX consistent with Alibaba flow: transfer only after current TTS reply finishes.
			markSIPTransferPending(callID)
			return "transfer_requested", nil
		},
	)
}

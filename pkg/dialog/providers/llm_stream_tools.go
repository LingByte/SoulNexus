package providers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import "strings"

// NeedsNonStreamToolRound reports whether this turn should use QueryWithOptions
// (tool chain) instead of QueryStream.
//
// Builtin knowledge / speaker tools must NOT force non-stream:
// knowledge is injected via SearchBlockForQuery. Catalog / MCP / HTTP tools
// require the tool chain when the utterance looks action-oriented.
func NeedsNonStreamToolRound(toolNames []string, userText string) bool {
	if len(toolNames) == 0 {
		return false
	}
	hasCatalog := false
	for _, name := range toolNames {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "search_knowledge_base",
			"get_speaker_context", "identify_speaker",
			"hangup":
			continue
		default:
			if name != "" {
				hasCatalog = true
			}
		}
	}
	if !hasCatalog {
		return false
	}
	t := strings.TrimSpace(userText)
	if t == "" {
		return false
	}
	lower := strings.ToLower(t)
	actionHints := []string{
		"预约", "查询", "订单", "课时", "报名", "取消", "改期", "退款",
		"结束会话",
		"book", "order", "cancel", "refund", "schedule",
	}
	for _, h := range actionHints {
		if strings.Contains(t, h) || strings.Contains(lower, strings.ToLower(h)) {
			return true
		}
	}
	if len([]rune(t)) <= 16 {
		return false
	}
	return true
}

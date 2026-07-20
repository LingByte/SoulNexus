// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package chat

import (
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
)

// ChannelSurface is prompt policy for a dialog channel.
type ChannelSurface struct {
	// SystemAppendix is appended to the text-dialog system prompt.
	SystemAppendix string
}

// SurfaceForChannel returns channel-specific prompt guidance.
func SurfaceForChannel(channel string) ChannelSurface {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case models.DialogChannelWeCom:
		return ChannelSurface{
			SystemAppendix: "【通道·企业微信】当前对话来自企业微信应用消息。" +
				"回复简洁、可直接发到聊天框；避免过长 markdown 表格。" +
				"如需人工协助，引导用户通过企业微信客服入口联系。",
		}
	case models.DialogChannelWeChatOA:
		return ChannelSurface{
			SystemAppendix: "【通道·微信公众号】当前对话来自微信公众号客服消息。" +
				"回复简洁、口语清晰；避免复杂排版。",
		}
	case models.DialogChannelDebug:
		return ChannelSurface{
			SystemAppendix: "【通道·调试】当前为助手调试台文本对话，可较完整展示推理与分点说明。",
		}
	default: // api
		return ChannelSurface{
			SystemAppendix: "【通道·API】当前为开放 API / 嵌入式文本对话。" +
				"回答准确、可结构化。",
		}
	}
}

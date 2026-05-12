// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 把 OpenAIStreamCapture / AnthropicStreamCapture 转换成单元素 attempts 数组，
// 便于流式路径直接复用 EmitRelayXxxUsageSuccess。
package llm

// OpenAIStreamAttemptsCompat 用 cap 中的状态码/耗时构建一条成功 attempt。
func OpenAIStreamAttemptsCompat(channelID int, cap *OpenAIStreamCapture) []UsageChannelAttempt {
	if cap == nil {
		return []UsageChannelAttempt{{Order: 1, ChannelID: channelID, Success: false, ErrorCode: "stream_error", ErrorMessage: "no capture"}}
	}
	return []UsageChannelAttempt{{
		Order:      1,
		ChannelID:  channelID,
		Success:    cap.StatusCode >= 200 && cap.StatusCode < 300,
		StatusCode: cap.StatusCode,
		LatencyMs:  cap.WallLatencyMs,
		TTFTMs:     cap.FirstTokenAtMs - cap.StartedAtMs,
	}}
}

// AnthropicStreamAttemptsCompat 同上。
func AnthropicStreamAttemptsCompat(channelID int, cap *AnthropicStreamCapture) []UsageChannelAttempt {
	if cap == nil {
		return []UsageChannelAttempt{{Order: 1, ChannelID: channelID, Success: false, ErrorCode: "stream_error", ErrorMessage: "no capture"}}
	}
	return []UsageChannelAttempt{{
		Order:      1,
		ChannelID:  channelID,
		Success:    cap.StatusCode >= 200 && cap.StatusCode < 300,
		StatusCode: cap.StatusCode,
		LatencyMs:  cap.WallLatencyMs,
		TTFTMs:     cap.FirstTokenAtMs - cap.StartedAtMs,
	}}
}

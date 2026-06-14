// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voicedialog

import (
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
)

// HardwareToolHooks wires xiaozhi-era LLM tools to the voice gateway.
type HardwareToolHooks struct {
	SendCommand   func(cmd gateway.Command)
	GoodbyePending func() // mark call to hang up after next TTS span
}

// RegisterHardwareTools registers goodbye / switch_speaker / voiceprint_identify
// for hardware voice sessions (parity with pkg/voice).
func RegisterHardwareTools(provider llm.LLMHandler, hooks HardwareToolHooks) {
	if provider == nil || hooks.SendCommand == nil {
		return
	}

	speakerIDs := []string{
		"101001", "101002", "101003", "101004", "101005", "101006", "101007", "101008", "101009", "101010",
		"101040", "101019", "101050", "101015",
		"BV700_streaming", "BV700_V2_streaming", "BV213_streaming", "BV025_streaming",
		"BV019_streaming", "BV221_streaming", "BV423_streaming",
	}

	provider.RegisterFunctionTool(
		"goodbye",
		"当用户表达告别意图时调用（再见、拜拜、晚安、挂断等）。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "告别原因（可选）",
				},
			},
		},
		func(args map[string]interface{}, _ interface{}) (string, error) {
			if hooks.GoodbyePending != nil {
				hooks.GoodbyePending()
			}
			reason, _ := args["reason"].(string)
			msg := "好的，再见"
			if strings.TrimSpace(reason) != "" {
				msg = fmt.Sprintf("好的，%s，再见", strings.TrimSpace(reason))
			}
			return msg, nil
		},
	)

	provider.RegisterFunctionTool(
		"switch_speaker",
		"切换语音合成发音人。用户要求换声音、男声女声、方言时调用。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"speaker_id": map[string]interface{}{
					"type":        "string",
					"description": "发音人ID",
					"enum":        speakerIDs,
				},
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "切换原因（可选）",
				},
			},
			"required": []string{"speaker_id"},
		},
		func(args map[string]interface{}, _ interface{}) (string, error) {
			speakerID, _ := args["speaker_id"].(string)
			speakerID = strings.TrimSpace(speakerID)
			if speakerID == "" {
				return "", fmt.Errorf("缺少 speaker_id")
			}
			hooks.SendCommand(gateway.Command{
				Type:      gateway.CmdTTSSwitchSpeaker,
				SpeakerID: speakerID,
			})
			return fmt.Sprintf("好的，已切换到发音人 %s", speakerID), nil
		},
	)

	provider.RegisterFunctionTool(
		"voiceprint_identify",
		"声纹识别：用户问「我是谁」「你认识我吗」时调用 identify。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"description": "identify 或 describe",
					"enum":        []string{"identify", "describe"},
				},
			},
			"required": []string{"action"},
		},
		func(args map[string]interface{}, _ interface{}) (string, error) {
			action, _ := args["action"].(string)
			switch strings.TrimSpace(action) {
			case "identify":
				return "我这边还没连上声纹库，先凭声音跟你聊；你可以直接告诉我你是谁。", nil
			case "describe":
				return "声纹描述功能在此通道暂未启用。", nil
			default:
				return "", fmt.Errorf("未知 action: %s", action)
			}
		},
	)
}

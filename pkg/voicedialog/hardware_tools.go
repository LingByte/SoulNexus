// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voicedialog

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
)

// HardwareToolHooks wires session-control tools to the voice gateway.
type HardwareToolHooks struct {
	SendCommand    func(cmd gateway.Command)
	GoodbyePending func() // mark call to hang up after next TTS span
}

// voiceStyleToSpeaker maps compact style slugs to provider speaker IDs.
var voiceStyleToSpeaker = map[string]string{
	"female_gentle":  "101001",
	"female_general": "101002",
	"female_sweet":   "101003",
	"male_mature":    "101004",
	"male_energetic": "101005",
	"cantonese":      "101019",
	"sichuan":        "101040",
}

var voiceStyleEnum = []string{
	"female_gentle",
	"female_general",
	"female_sweet",
	"male_mature",
	"male_energetic",
	"cantonese",
	"sichuan",
}

func toolResultOK(action string) string {
	b, _ := json.Marshal(map[string]string{"status": "ok", "action": action})
	return string(b)
}

func toolResultErr(msg string) string {
	b, _ := json.Marshal(map[string]string{"status": "error", "message": msg})
	return string(b)
}

// RegisterHardwareTools registers goodbye and switch_speaker for hardware voice sessions.
// Voiceprint is not registered on the dialog plane (no live audio path here).
func RegisterHardwareTools(provider llm.LLMHandler, hooks HardwareToolHooks) {
	if provider == nil || hooks.SendCommand == nil {
		return
	}

	provider.RegisterFunctionTool(
		"goodbye",
		"用户结束或告别通话时调用。",
		map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		func(_ map[string]interface{}, _ interface{}) (string, error) {
			if hooks.GoodbyePending != nil {
				hooks.GoodbyePending()
			}
			return toolResultOK("goodbye"), nil
		},
	)

	provider.RegisterFunctionTool(
		"switch_speaker",
		"用户要求换音色、男声女声或方言时调用。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"voice_style": map[string]interface{}{
					"type":        "string",
					"description": "音色风格",
					"enum":        voiceStyleEnum,
				},
				"speaker_id": map[string]interface{}{
					"type":        "string",
					"description": "可选：具体发音人ID（voice_style 无法满足时使用）",
				},
			},
		},
		func(args map[string]interface{}, _ interface{}) (string, error) {
			speakerID := strings.TrimSpace(fmt.Sprint(args["speaker_id"]))
			if speakerID == "" {
				style := strings.TrimSpace(fmt.Sprint(args["voice_style"]))
				speakerID = voiceStyleToSpeaker[style]
				if speakerID == "" {
					return toolResultErr("invalid voice_style or speaker_id"), nil
				}
			}
			hooks.SendCommand(gateway.Command{
				Type:      gateway.CmdTTSSwitchSpeaker,
				SpeakerID: speakerID,
			})
			return toolResultOK("switch_speaker"), nil
		},
	)
}

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package app

import (
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/persist"
)

func TestVoiceTurnsToConversation(t *testing.T) {
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	end := start.Add(30 * time.Second)
	turns := []persist.VoiceCallDialogTurn{
		{ASRText: "你好", LLMText: "你好呀", At: start.Add(2 * time.Second), E2EFirstByteMs: 400},
	}
	details := voiceTurnsToConversation("xz-1", turns, start, end)
	if details.TotalTurns != 2 {
		t.Fatalf("expected 2 turns, got %d", details.TotalTurns)
	}
	if details.UserTurns != 1 || details.AITurns != 1 {
		t.Fatalf("unexpected user/ai counts: %+v", details)
	}
	if details.Turns[0].Type != "user" || details.Turns[1].Type != "ai" {
		t.Fatalf("unexpected turn types: %+v", details.Turns)
	}
}

func TestNormalizeRecordingURL(t *testing.T) {
	if got := normalizeRecordingURL("https://cdn.example.com/a.wav", "a.wav"); got != "https://cdn.example.com/a.wav" {
		t.Fatalf("cdn url: %q", got)
	}
	if got := normalizeRecordingURL("/media/xz-1.wav", "voiceserver-recordings/xz-1.wav"); got != "/api/device/recordings/voiceserver-recordings/xz-1.wav" {
		t.Fatalf("local key: %q", got)
	}
}

func TestMapVoiceCallStatus(t *testing.T) {
	if mapVoiceCallStatus(persist.VoiceCallEndCompletedRemote, "") != "completed" {
		t.Fatal("expected completed")
	}
	if mapVoiceCallStatus(persist.VoiceCallEndServerError, "") != "error" {
		t.Fatal("expected error")
	}
}

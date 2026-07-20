package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"testing"
)

func TestCheckLocalTTSAvailable(t *testing.T) {
	available := CheckLocalTTSAvailable()
	t.Logf("Available local TTS commands: %v", available)
}

func TestDetectLocalTTSCommand(t *testing.T) {
	detected := DetectLocalTTSCommand()
	t.Logf("Detected local TTS command: %s", detected)
}

func TestGetLocalTTSInfo(t *testing.T) {
	info := GetLocalTTSInfo()
	t.Logf("Local TTS info: %+v", info)
}

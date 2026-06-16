// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package tts

import (
	"os"
	"testing"
)

func TestVolcengineCloneEnvOption_missingEnv(t *testing.T) {
	os.Setenv("VOLCENGINE_CLONE_APP_ID", "")
	os.Setenv("VOLCENGINE_CLONE_TOKEN", "")
	_, err := VolcengineCloneEnvOption("asset-1")
	if err == nil {
		t.Fatal("expected error when clone env credentials missing")
	}
}

func TestVolcengineCloneEnvOption_ok(t *testing.T) {
	os.Setenv("VOLCENGINE_CLONE_APP_ID", "3279503365")
	os.Setenv("VOLCENGINE_CLONE_TOKEN", "test-token")
	os.Setenv("VOLCENGINE_CLONE_CLUSTER", "volcano_icl")
	os.Setenv("VOLCENGINE_CLONE_SAMPLE_RATE", "16000")
	os.Setenv("VOLCENGINE_CLONE_ENCODING", "pcm")
	os.Setenv("VOLCENGINE_CLONE_SPEED_RATIO", "1.0")

	opt, err := VolcengineCloneEnvOption("my-asset")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt.AppID != "3279503365" || opt.AccessToken != "test-token" {
		t.Fatalf("unexpected credentials: %+v", opt)
	}
	if opt.AssetID != "my-asset" || opt.Cluster != "volcano_icl" {
		t.Fatalf("unexpected clone fields: %+v", opt)
	}
	if opt.Rate != 16000 || opt.Encoding != "pcm" || opt.SpeedRatio != 1.0 {
		t.Fatalf("unexpected audio params: %+v", opt)
	}
}

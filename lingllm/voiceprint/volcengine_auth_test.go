// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voiceprint

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignVolcengineRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://rtc.volcengineapi.com/?Action=IotVoicePrintIdentify&Version=2025-08-01", nil)
	req.Host = "rtc.volcengineapi.com"
	body := []byte(`{"Audio":"dGVzdA=="}`)
	if err := signVolcengineRequest(req, "ak-test", "sk-test", "cn-north-1", body); err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Authorization") == "" {
		t.Fatal("expected Authorization header")
	}
	if req.Header.Get("X-Date") == "" || req.Header.Get("X-Content-Sha256") == "" {
		t.Fatal("expected signed headers")
	}
}

func TestVolcengineIdentifyResultItems(t *testing.T) {
	r := &VolcengineIdentifyResult{
		Matches: []VolcengineScoreItem{{UUID: "u1", Score: 0.9}},
	}
	if len(r.Items()) != 1 || r.Items()[0].UUID != "u1" {
		t.Fatalf("unexpected items: %+v", r.Items())
	}
	r2 := &VolcengineIdentifyResult{
		ScoreList: []VolcengineScoreItem{{UUID: "u2", Score: 0.8}},
	}
	if len(r2.Items()) != 1 || r2.Items()[0].UUID != "u2" {
		t.Fatalf("unexpected items: %+v", r2.Items())
	}
}

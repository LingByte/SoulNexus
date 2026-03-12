package text

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

func TestNewAliyunTextCensor(t *testing.T) {
	accessKeyID := utils.GetEnv("ALIYUN_ACCESS_KEY_ID")
	accessKeySecret := utils.GetEnv("ALIYUN_ACCESS_KEY_SECRET")
	if accessKeyID == "" || accessKeySecret == "" {
		t.Skipf("not found ALIYUN_ACCESS_KEY_ID or ALIYUN_ACCESS_KEY_SECRET")
	}
	client, err := NewAliyunTextCensor()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if client.AccessKeyID != utils.GetEnv("ALIYUN_ACCESS_KEY_ID") {
		t.Error("AccessKeyID was set incorrectly")
	}
	if client.AccessKeySecret != utils.GetEnv("ALIYUN_ACCESS_KEY_SECRET") {
		t.Error("AccessKeySecret was set incorrectly")
	}
	if client.Client == nil {
		t.Error("Green Client was not initialized")
	}
	// Check default endpoint
	endpoint := utils.GetEnv("ALIYUN_GREEN_ENDPOINT")
	if endpoint == "" {
		endpoint = "green-cip.cn-shanghai.aliyuncs.com"
	}
	if client.Endpoint != endpoint {
		t.Errorf("Endpoint was set incorrectly: expected %s, got %s", endpoint, client.Endpoint)
	}
}

func TestAliyunCensorText(t *testing.T) {
	accessKeyID := utils.GetEnv("ALIYUN_ACCESS_KEY_ID")
	accessKeySecret := utils.GetEnv("ALIYUN_ACCESS_KEY_SECRET")
	if accessKeyID == "" || accessKeySecret == "" {
		t.Skipf("not found ALIYUN_ACCESS_KEY_ID or ALIYUN_ACCESS_KEY_SECRET")
	}
	client, err := NewAliyunTextCensor()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test with normal text
	result, err := client.CensorText("hello world")
	if err != nil {
		t.Errorf("Failed to censor text: %v", err)
	}
	if result == nil {
		t.Error("Result is nil")
		return
	}
	if result.Suggestion != SuggestionPass && result.Suggestion != SuggestionReview && result.Suggestion != SuggestionBlock {
		t.Errorf("Unexpected suggestion: %s", result.Suggestion)
	}
	t.Logf("Censor result: Suggestion=%s, Label=%s, Score=%.4f, Msg=%s, Details=%s",
		result.Suggestion, result.Label, result.Score, result.Msg, result.Details)
}

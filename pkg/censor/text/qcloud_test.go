package text

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

func TestNewQCloudTextCensor(t *testing.T) {
	secretID := utils.GetEnv("QCLOUD_SECRET_ID")
	secretKey := utils.GetEnv("QCLOUD_SECRET_KEY")
	if secretID == "" || secretKey == "" {
		t.Skipf("not found QCLOUD_SECRET_ID or QCLOUD_SECRET_KEY")
	}
	client, err := NewQCloudTextCensor()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if client.SecretID != utils.GetEnv("QCLOUD_SECRET_ID") {
		t.Error("SecretID was set incorrectly")
	}
	if client.SecretKey != utils.GetEnv("QCLOUD_SECRET_KEY") {
		t.Error("SecretKey was set incorrectly")
	}
	if client.Client == nil {
		t.Error("TMS Client was not initialized")
	}
	// Check default region
	region := utils.GetEnv("QCLOUD_REGION")
	if region == "" {
		region = "ap-guangzhou"
	}
	if client.Region != region {
		t.Errorf("Region was set incorrectly: expected %s, got %s", region, client.Region)
	}
}

func TestQCloudCensorText(t *testing.T) {
	secretID := utils.GetEnv("QCLOUD_SECRET_ID")
	secretKey := utils.GetEnv("QCLOUD_SECRET_KEY")
	if secretID == "" || secretKey == "" {
		t.Skipf("not found QCLOUD_SECRET_ID or QCLOUD_SECRET_KEY")
	}
	client, err := NewQCloudTextCensor()
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

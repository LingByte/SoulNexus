// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voiceprint

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestNewXunfeiAuth(t *testing.T) {
	auth := NewXunfeiAuth("test-key", "test-secret", "api.xf-yun.com")
	if auth == nil {
		t.Error("NewXunfeiAuth returned nil")
	}
	if auth.APIKey != "test-key" {
		t.Errorf("APIKey = %s, want test-key", auth.APIKey)
	}
	if auth.APISecret != "test-secret" {
		t.Errorf("APISecret = %s, want test-secret", auth.APISecret)
	}
	if auth.Host != "api.xf-yun.com" {
		t.Errorf("Host = %s, want api.xf-yun.com", auth.Host)
	}
}

func TestGenerateAuthHeader(t *testing.T) {
	auth := NewXunfeiAuth("test-key", "test-secret", "api.xf-yun.com")

	authorization, date, err := auth.GenerateAuthHeader("POST", "/v1/private/s782b4996")
	if err != nil {
		t.Errorf("GenerateAuthHeader() error = %v", err)
		return
	}

	if authorization == "" {
		t.Error("authorization is empty")
	}

	if date == "" {
		t.Error("date is empty")
	}

	// 验证authorization是base64编码的
	decoded, err := base64.StdEncoding.DecodeString(authorization)
	if err != nil {
		t.Errorf("authorization is not valid base64: %v", err)
		return
	}

	// 验证解码后的内容包含必要的字段
	decodedStr := string(decoded)
	if !strings.Contains(decodedStr, "api_key=") {
		t.Error("decoded authorization does not contain api_key")
	}
	if !strings.Contains(decodedStr, "algorithm=") {
		t.Error("decoded authorization does not contain algorithm")
	}
	if !strings.Contains(decodedStr, "signature=") {
		t.Error("decoded authorization does not contain signature")
	}
}

func TestBuildAuthURL(t *testing.T) {
	auth := NewXunfeiAuth("test-key", "test-secret", "api.xf-yun.com")

	baseURL := "https://api.xf-yun.com/v1/private/s782b4996"
	url, err := auth.BuildAuthURL(baseURL, "/v1/private/s782b4996")
	if err != nil {
		t.Errorf("BuildAuthURL() error = %v", err)
		return
	}

	if !strings.HasPrefix(url, baseURL) {
		t.Errorf("URL does not start with base URL: %s", url)
	}

	if !strings.Contains(url, "authorization=") {
		t.Error("URL does not contain authorization parameter")
	}

	if !strings.Contains(url, "host=") {
		t.Error("URL does not contain host parameter")
	}

	if !strings.Contains(url, "date=") {
		t.Error("URL does not contain date parameter")
	}
}

func TestVerifyResponse(t *testing.T) {
	auth := NewXunfeiAuth("test-key", "test-secret", "api.xf-yun.com")

	tests := []struct {
		name    string
		date    string
		wantErr bool
	}{
		{
			name:    "valid date",
			date:    time.Now().UTC().Format(time.RFC1123),
			wantErr: false,
		},
		{
			name:    "invalid date format",
			date:    "invalid",
			wantErr: true,
		},
		{
			name:    "old date",
			date:    time.Now().UTC().Add(-400 * time.Second).Format(time.RFC1123),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.VerifyResponse(tt.date)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthConsistency(t *testing.T) {
	auth := NewXunfeiAuth("test-key", "test-secret", "api.xf-yun.com")

	// 生成两个认证头，应该不同（因为日期不同）
	auth1, date1, err1 := auth.GenerateAuthHeader("POST", "/v1/private/s782b4996")
	if err1 != nil {
		t.Errorf("First GenerateAuthHeader() error = %v", err1)
		return
	}

	time.Sleep(1 * time.Second)

	auth2, date2, err2 := auth.GenerateAuthHeader("POST", "/v1/private/s782b4996")
	if err2 != nil {
		t.Errorf("Second GenerateAuthHeader() error = %v", err2)
		return
	}

	// 日期应该不同
	if date1 == date2 {
		t.Error("dates should be different")
	}

	// authorization应该不同（因为日期不同）
	if auth1 == auth2 {
		t.Error("authorizations should be different")
	}
}

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package humax

import (
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

func TestGinPathToOpenAPI(t *testing.T) {
	got := GinPathToOpenAPI("/api/users/:id/roles/:rid")
	want := "/api/users/{id}/roles/{rid}"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestHumanSummary_SidebarNames(t *testing.T) {
	cases := map[string]string{
		"POST /api/admin/notification-channels":        "创建通知渠道",
		"GET /api/admin/notification-channels":         "列出通知渠道",
		"GET /api/admin/notification-channels/{id}":    "获取通知渠道",
		"DELETE /api/admin/notification-channels/{id}": "删除通知渠道",
		"PUT /api/admin/notification-channels/{id}":    "更新通知渠道",
		"POST /api/account/deletion/revoke":            "撤销账号注销",
	}
	for key, want := range cases {
		parts := splitMethodPath(key)
		got := HumanSummary(parts[0], parts[1])
		if got != want {
			t.Fatalf("%s: got %q want %q", key, got, want)
		}
	}
}

func TestHumanTag(t *testing.T) {
	cases := map[string]string{
		"/api/admin/notification-channels": "平台管理 · 通知渠道",
		"/api/admin/execution-tasks":       "平台管理 · 执行任务",
		"/api/account/deletion/revoke":     "账号安全 · 账号注销",
		"/api/me/devices":                  "个人中心 · 登录设备",
	}
	for path, want := range cases {
		got := HumanTag(path)
		if got != want {
			t.Fatalf("%s: got %q want %q", path, got, want)
		}
	}
}

func TestDocument_AddsPathAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := humagin.New(r, huma.DefaultConfig("t", "1"))
	EnsureSecuritySchemes(api)
	document(api, "POST", "/api/admin/notification-channels/:id")

	item := api.OpenAPI().Paths["/api/admin/notification-channels/{id}"]
	if item == nil || item.Post == nil {
		t.Fatal("missing post op")
	}
	op := item.Post
	if op.Summary == "" {
		t.Fatal("empty summary")
	}
	if op.RequestBody == nil {
		t.Fatal("expected request body for POST")
	}
	foundPath := false
	for _, p := range op.Parameters {
		if p.Name == "id" && p.In == "path" {
			foundPath = true
		}
	}
	if !foundPath {
		t.Fatal("expected path param id")
	}
}

func TestSchemaFromStruct_NotificationChannel(t *testing.T) {
	RegisterJSONBody("POST", "/api/admin/notification-channels", struct {
		ChannelType string `json:"channelType" binding:"required,oneof=email sms"`
		Name        string `json:"name" binding:"required"`
		SMTPHost    string `json:"smtpHost"`
	}{})
	s := LookupBodyForTest("POST", "/api/admin/notification-channels")
	if s == nil || s.Properties["channelType"] == nil || s.Properties["name"] == nil {
		t.Fatalf("schema=%v", s)
	}
	if len(s.Required) < 2 {
		t.Fatalf("required=%v", s.Required)
	}
}

func splitMethodPath(s string) [2]string {
	i := 0
	for i < len(s) && s[i] != ' ' {
		i++
	}
	return [2]string{s[:i], s[i+1:]}
}

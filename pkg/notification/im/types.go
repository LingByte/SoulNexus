// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package im provides tenant-scoped enterprise IM notification channels
// (WeCom / Feishu webhook bots and app messaging).
package im

import (
	"context"
	"errors"
	"strings"
)

const (
	ProviderWeCom  = "wecom"
	ProviderFeishu = "feishu"
)

var (
	ErrInvalidConfig   = errors.New("im: invalid config")
	ErrInvalidArgument = errors.New("im: invalid argument")
	ErrProviderReject  = errors.New("im: provider rejected")
)

// Message is a markdown/text notification payload.
type Message struct {
	Title   string
	Content string // markdown or plain text
}

// Provider sends one message through a configured IM outlet.
type Provider interface {
	Kind() string
	Send(ctx context.Context, msg Message) error
}

// NormalizeProvider returns a canonical provider kind or empty string.
func NormalizeProvider(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case ProviderWeCom, "wechat_work", "qiyeweixin", "企业微信":
		return ProviderWeCom
	case ProviderFeishu, "lark", "飞书":
		return ProviderFeishu
	default:
		return ""
	}
}

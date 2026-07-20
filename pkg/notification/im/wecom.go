// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package im

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// WeComConfig supports group webhook bots and (optionally) app credentials.
// Prefer WebhookURL for operational alerts; App* fields are reserved for future
// targeted user messaging.
type WeComConfig struct {
	WebhookURL string `json:"webhookUrl"`
	CorpID     string `json:"corpId,omitempty"`
	AgentID    string `json:"agentId,omitempty"`
	Secret     string `json:"secret,omitempty"`
}

type weComProvider struct {
	cfg WeComConfig
}

// NewWeCom builds a WeCom provider. WebhookURL is required for send.
func NewWeCom(cfg WeComConfig) (Provider, error) {
	if strings.TrimSpace(cfg.WebhookURL) == "" {
		return nil, fmt.Errorf("%w: wecom requires webhookUrl", ErrInvalidConfig)
	}
	return &weComProvider{cfg: cfg}, nil
}

func (p *weComProvider) Kind() string { return ProviderWeCom }

func (p *weComProvider) Send(ctx context.Context, msg Message) error {
	if ctx == nil {
		ctx = context.Background()
	}
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return fmt.Errorf("%w: empty content", ErrInvalidArgument)
	}
	title := strings.TrimSpace(msg.Title)
	text := content
	if title != "" {
		text = "**" + title + "**\n" + content
	}
	body := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": text,
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(p.cfg.WebhookURL), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var r struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(b, &r)
	if resp.StatusCode >= 400 || r.ErrCode != 0 {
		msg := strings.TrimSpace(r.ErrMsg)
		if msg == "" {
			msg = string(b)
		}
		return fmt.Errorf("%w: wecom status=%d errcode=%d %s", ErrProviderReject, resp.StatusCode, r.ErrCode, msg)
	}
	return nil
}

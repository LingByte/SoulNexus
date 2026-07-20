// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package im

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// FeishuConfig supports group custom bot webhooks (optional signing secret).
type FeishuConfig struct {
	WebhookURL string `json:"webhookUrl"`
	Secret     string `json:"secret,omitempty"` // bot signing secret
	AppID      string `json:"appId,omitempty"`
	AppSecret  string `json:"appSecret,omitempty"`
}

type feishuProvider struct {
	cfg FeishuConfig
}

// NewFeishu builds a Feishu/Lark webhook bot provider.
func NewFeishu(cfg FeishuConfig) (Provider, error) {
	if strings.TrimSpace(cfg.WebhookURL) == "" {
		return nil, fmt.Errorf("%w: feishu requires webhookUrl", ErrInvalidConfig)
	}
	return &feishuProvider{cfg: cfg}, nil
}

func (p *feishuProvider) Kind() string { return ProviderFeishu }

func (p *feishuProvider) Send(ctx context.Context, msg Message) error {
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
		text = title + "\n" + content
	}

	payload := map[string]any{
		"msg_type": "text",
		"content":  map[string]string{"text": text},
	}
	if sec := strings.TrimSpace(p.cfg.Secret); sec != "" {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		payload["timestamp"] = ts
		payload["sign"] = feishuSign(ts, sec)
	}

	raw, err := json.Marshal(payload)
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
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	_ = json.Unmarshal(b, &r)
	if resp.StatusCode >= 400 || r.Code != 0 {
		msg := strings.TrimSpace(r.Msg)
		if msg == "" {
			msg = string(b)
		}
		return fmt.Errorf("%w: feishu status=%d code=%d %s", ErrProviderReject, resp.StatusCode, r.Code, msg)
	}
	return nil
}

func feishuSign(timestamp, secret string) string {
	// Custom bot: sign = base64(HMAC-SHA256(secret, timestamp + "\n" + secret))
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(timestamp + "\n" + secret))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package wechat_oa

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/channels/wxcrypt"
)

const apiBase = "https://api.weixin.qq.com"

// Config holds WeChat official account credentials.
type Config struct {
	AppID          string
	AppSecret      string
	Token          string
	EncodingAESKey string
}

// Client verifies OA callbacks and sends customer-service messages.
type Client struct {
	cfg   Config
	crypt *wxcrypt.Crypt
	http  *http.Client

	tokMu    sync.Mutex
	token    string
	tokenExp time.Time
}

// NewClient builds an OA dialog client.
func NewClient(cfg Config) (*Client, error) {
	crypt, err := wxcrypt.New(cfg.Token, cfg.EncodingAESKey, cfg.AppID)
	if err != nil {
		return nil, err
	}
	return &Client{
		cfg:   cfg,
		crypt: crypt,
		http:  &http.Client{Timeout: 12 * time.Second},
	}, nil
}

// VerifyURL handles GET echostr (plaintext or encrypted).
func (c *Client) VerifyURL(signature, timestamp, nonce, echostr string, encryptMode bool) (string, error) {
	if !encryptMode {
		if !plainSignatureOK(c.cfg.Token, signature, timestamp, nonce) {
			return "", fmt.Errorf("wechat_oa: bad signature")
		}
		return echostr, nil
	}
	return c.crypt.VerifyURL(signature, timestamp, nonce, echostr)
}

// InboundMessage is a parsed OA callback message.
type InboundMessage struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
	MsgID        int64  `xml:"MsgId"`
	Event        string `xml:"Event"`
}

// ParseInbound decrypts (if needed) and unmarshals POST body.
func (c *Client) ParseInbound(signature, timestamp, nonce string, body []byte, encryptMode bool) (*InboundMessage, error) {
	plain := body
	var err error
	if encryptMode {
		plain, err = c.crypt.DecryptMsg(signature, timestamp, nonce, body)
		if err != nil {
			return nil, err
		}
	} else if !plainSignatureOK(c.cfg.Token, signature, timestamp, nonce) {
		return nil, fmt.Errorf("wechat_oa: bad signature")
	}
	var msg InboundMessage
	if err := xml.Unmarshal(plain, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// SendCustomerText sends an async reply via the customer service API.
func (c *Client) SendCustomerText(ctx context.Context, toUser, content string) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"touser":  toUser,
		"msgtype": "text",
		"text":    map[string]string{"content": content},
	}
	raw, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/cgi-bin/message/custom/send?access_token=%s", apiBase, url.QueryEscape(token))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(body, &out)
	if out.ErrCode != 0 {
		return fmt.Errorf("wechat_oa custom send: %d %s", out.ErrCode, out.ErrMsg)
	}
	return nil
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.tokMu.Lock()
	defer c.tokMu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return c.token, nil
	}
	u := fmt.Sprintf("%s/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		apiBase, url.QueryEscape(c.cfg.AppID), url.QueryEscape(c.cfg.AppSecret))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("wechat_oa token: %d %s", out.ErrCode, out.ErrMsg)
	}
	exp := out.ExpiresIn
	if exp <= 0 {
		exp = 7200
	}
	c.token = out.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(exp-120) * time.Second)
	return c.token, nil
}

func plainSignatureOK(token, signature, timestamp, nonce string) bool {
	return strings.EqualFold(sortedSHA1(token, timestamp, nonce), signature)
}

func sortedSHA1(parts ...string) string {
	cp := append([]string(nil), parts...)
	sort.Strings(cp)
	h := sha1.New()
	_, _ = io.WriteString(h, strings.Join(cp, ""))
	return fmt.Sprintf("%x", h.Sum(nil))
}

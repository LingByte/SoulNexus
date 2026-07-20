// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package wecom

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/channels/wxcrypt"
)

const apiBase = "https://qyapi.weixin.qq.com"

// Config holds WeCom app credentials for inbound + outbound messaging.
type Config struct {
	CorpID         string
	AgentID        string
	Secret         string
	Token          string
	EncodingAESKey string
}

// Client talks to WeCom APIs and verifies callbacks.
type Client struct {
	cfg   Config
	crypt *wxcrypt.Crypt
	http  *http.Client

	tokMu     sync.Mutex
	token     string
	tokenExp  time.Time
}

// NewClient builds a WeCom dialog client.
func NewClient(cfg Config) (*Client, error) {
	crypt, err := wxcrypt.New(cfg.Token, cfg.EncodingAESKey, cfg.CorpID)
	if err != nil {
		return nil, err
	}
	return &Client{
		cfg:   cfg,
		crypt: crypt,
		http:  &http.Client{Timeout: 12 * time.Second},
	}, nil
}

// VerifyURL handles GET callback verification.
func (c *Client) VerifyURL(msgSig, timestamp, nonce, echostr string) (string, error) {
	return c.crypt.VerifyURL(msgSig, timestamp, nonce, echostr)
}

// InboundMessage is a parsed WeCom callback text message.
type InboundMessage struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
	MsgID        int64  `xml:"MsgId"`
	AgentID      int64  `xml:"AgentID"`
	Event        string `xml:"Event"`
}

// ParseInbound decrypts and unmarshals a POST body.
func (c *Client) ParseInbound(msgSig, timestamp, nonce string, body []byte) (*InboundMessage, error) {
	plain, err := c.crypt.DecryptMsg(msgSig, timestamp, nonce, body)
	if err != nil {
		return nil, err
	}
	var msg InboundMessage
	if err := xml.Unmarshal(plain, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// BuildTextReplyXML builds a plaintext passive reply (before optional encrypt).
func BuildTextReplyXML(toUser, fromUser, content string) []byte {
	type xmlMsg struct {
		XMLName      xml.Name      `xml:"xml"`
		ToUserName   wxcrypt.CDATA `xml:"ToUserName"`
		FromUserName wxcrypt.CDATA `xml:"FromUserName"`
		CreateTime   int64         `xml:"CreateTime"`
		MsgType      wxcrypt.CDATA `xml:"MsgType"`
		Content      wxcrypt.CDATA `xml:"Content"`
	}
	out, _ := xml.Marshal(xmlMsg{
		ToUserName:   wxcrypt.CDATA(toUser),
		FromUserName: wxcrypt.CDATA(fromUser),
		CreateTime:   time.Now().Unix(),
		MsgType:      wxcrypt.CDATA("text"),
		Content:      wxcrypt.CDATA(content),
	})
	return out
}

// EncryptReply wraps a plaintext XML reply for encrypted mode responses.
func (c *Client) EncryptReply(plain []byte, timestamp, nonce string) ([]byte, error) {
	return c.crypt.EncryptMsg(plain, timestamp, nonce)
}

// SendText proactively sends an app text message to a user.
func (c *Client) SendText(ctx context.Context, toUser, content string) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	agentID, _ := strconv.Atoi(strings.TrimSpace(c.cfg.AgentID))
	payload := map[string]any{
		"touser":  toUser,
		"msgtype": "text",
		"agentid": agentID,
		"text":    map[string]string{"content": content},
	}
	raw, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/cgi-bin/message/send?access_token=%s", apiBase, url.QueryEscape(token))
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
		return fmt.Errorf("wecom send: %d %s", out.ErrCode, out.ErrMsg)
	}
	return nil
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.tokMu.Lock()
	defer c.tokMu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return c.token, nil
	}
	u := fmt.Sprintf("%s/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		apiBase, url.QueryEscape(c.cfg.CorpID), url.QueryEscape(c.cfg.Secret))
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
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	if out.ErrCode != 0 || out.AccessToken == "" {
		return "", fmt.Errorf("wecom gettoken: %d %s", out.ErrCode, out.ErrMsg)
	}
	exp := out.ExpiresIn
	if exp <= 0 {
		exp = 7200
	}
	c.token = out.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(exp-120) * time.Second)
	return c.token, nil
}

package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type RongcloudConfig struct {
	AppKey    string `json:"appKey"`
	AppSecret string `json:"appSecret"`
	Endpoint  string `json:"endpoint,omitempty"` // default https://api.rong-api.com
}

type RongcloudProvider struct {
	cfg RongcloudConfig
}

func NewRongcloud(cfg RongcloudConfig) (*RongcloudProvider, error) {
	if strings.TrimSpace(cfg.AppKey) == "" || strings.TrimSpace(cfg.AppSecret) == "" {
		return nil, fmt.Errorf("%w: rongcloud requires appKey/appSecret", ErrInvalidConfig)
	}
	return &RongcloudProvider{cfg: cfg}, nil
}

func (p *RongcloudProvider) Kind() ProviderKind { return ProviderRongcloud }

func (p *RongcloudProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	// Rongcloud verify uses template + data; notify can also be template-based.
	if strings.TrimSpace(req.Message.Template) == "" {
		return nil, fmt.Errorf("%w: rongcloud requires template", ErrInvalidArgument)
	}

	to, err := firstRecipient(req)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimSpace(p.cfg.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.rong-api.com"
	}
	nonce := randHex(8)
	ts := fmt.Sprintf("%d", time.Now().Unix())
	sig := sha1Hex(strings.TrimSpace(p.cfg.AppSecret) + nonce + ts)
	headers := map[string]string{
		"App-Key":   strings.TrimSpace(p.cfg.AppKey),
		"Nonce":     nonce,
		"Timestamp": ts,
		"Signature": sig,
	}

	// Use sendCode as a template SMS: /sms/sendCode.json
	// TemplateId is required; region default 86.
	form := url.Values{}
	form.Set("mobile", strings.TrimPrefix(to, "+"))
	form.Set("templateId", strings.TrimSpace(req.Message.Template))
	if req.Extras != nil {
		if v, ok := req.Extras["region"]; ok {
			form.Set("region", fmt.Sprint(v))
		}
	}
	status, b, err := postForm(ctx, endpoint+"/sms/sendCode.json", form, headers, "", "")
	raw := truncateRaw(string(b), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}
	var r struct {
		Code         int    `json:"code"`
		Session      string `json:"sessionId"`
		ErrorMessage string `json:"errorMessage"`
	}
	_ = json.Unmarshal(b, &r)
	if !is2xx(status) || r.Code != 200 {
		msg := strings.TrimSpace(r.ErrorMessage)
		if msg == "" {
			msg = "provider rejected"
		}
		return &SendResult{Provider: p.Kind(), MessageID: strings.TrimSpace(r.Session), Accepted: false, Status: fmt.Sprintf("%d", r.Code), Error: msg, Raw: raw, SentAtUnix: time.Now().Unix()}, errProviderRejected
	}
	return &SendResult{Provider: p.Kind(), MessageID: strings.TrimSpace(r.Session), Accepted: true, Status: "ok", Raw: raw, SentAtUnix: time.Now().Unix()}, nil
}

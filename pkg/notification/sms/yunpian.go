package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type YunpianConfig struct {
	APIKey    string `json:"apiKey"`
	Signature string `json:"signature"` // optional fallback signature when content has no signature
}

type YunpianProvider struct {
	cfg YunpianConfig
}

func NewYunpian(cfg YunpianConfig) (*YunpianProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("%w: yunpian requires apiKey", ErrInvalidConfig)
	}
	return &YunpianProvider{cfg: cfg}, nil
}

func (p *YunpianProvider) Kind() ProviderKind { return ProviderYunpian }

func (p *YunpianProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	// Yunpian usually uses Content.
	if strings.TrimSpace(req.Message.Content) == "" {
		return nil, fmt.Errorf("%w: yunpian requires content", ErrInvalidArgument)
	}

	type yunpianResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		// Detail string `json:"detail"`
		Sid int64 `json:"sid"`
	}

	// Yunpian single send endpoint.
	// https://sms.yunpian.com/v2/sms/single_send.json
	to, err := firstRecipient(req)
	if err != nil {
		return nil, err
	}
	text := normalizeContent(req.Message.Content, p.cfg.Signature)
	form := url.Values{}
	form.Set("apikey", strings.TrimSpace(p.cfg.APIKey))
	form.Set("mobile", to)
	form.Set("text", text)
	status, b, err := postForm(ctx, "https://sms.yunpian.com/v2/sms/single_send.json", form, nil, "", "")
	raw := truncateRaw(string(b), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}
	var r yunpianResp
	_ = json.Unmarshal(b, &r)
	if !is2xx(status) || r.Code != 0 {
		msg := strings.TrimSpace(r.Msg)
		if msg == "" {
			msg = "provider rejected"
		}
		return &SendResult{
			Provider:   p.Kind(),
			Accepted:   false,
			Status:     fmt.Sprintf("http_%d", status),
			Error:      msg,
			Raw:        raw,
			SentAtUnix: nowUnix(),
		}, errProviderRejected
	}
	return &SendResult{
		Provider:   p.Kind(),
		MessageID:  fmt.Sprintf("%d", r.Sid),
		Accepted:   true,
		Status:     "ok",
		Raw:        raw,
		SentAtUnix: nowUnix(),
	}, nil
}

package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type LuosimaoConfig struct {
	APIKey string `json:"apiKey"`
}

type LuosimaoProvider struct {
	cfg LuosimaoConfig
}

func NewLuosimao(cfg LuosimaoConfig) (*LuosimaoProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("%w: luosimao requires apiKey", ErrInvalidConfig)
	}
	return &LuosimaoProvider{cfg: cfg}, nil
}

func (p *LuosimaoProvider) Kind() ProviderKind { return ProviderLuosimao }

func (p *LuosimaoProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Message.Content) == "" {
		return nil, fmt.Errorf("%w: luosimao requires content", ErrInvalidArgument)
	}

	type luosimaoResp struct {
		Error int    `json:"error"`
		Msg   string `json:"msg"`
	}

	// https://sms-api.luosimao.com/v1/send.json
	to, err := firstRecipient(req)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("mobile", to)
	// Luosimao requires message ending with "【签名】" or included; we keep as-is.
	form.Set("message", strings.TrimSpace(req.Message.Content))
	// Luosimao basic auth: username="api", password=APIKey.
	status, b, err := postForm(ctx, "https://sms-api.luosimao.com/v1/send.json", form, nil, "api", strings.TrimSpace(p.cfg.APIKey))
	raw := truncateRaw(string(b), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}
	var r luosimaoResp
	_ = json.Unmarshal(b, &r)
	if !is2xx(status) || r.Error != 0 {
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
		Accepted:   true,
		Status:     "ok",
		Raw:        raw,
		SentAtUnix: nowUnix(),
	}, nil
}

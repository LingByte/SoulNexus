package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type HuyiConfig struct {
	APIID     string `json:"apiId"`
	APIKey    string `json:"apiKey"`
	Signature string `json:"signature"`
}

type HuyiProvider struct {
	cfg HuyiConfig
}

func NewHuyi(cfg HuyiConfig) (*HuyiProvider, error) {
	if strings.TrimSpace(cfg.APIID) == "" || strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("%w: huyi requires apiId/apiKey", ErrInvalidConfig)
	}
	return &HuyiProvider{cfg: cfg}, nil
}

func (p *HuyiProvider) Kind() ProviderKind { return ProviderHuyi }

func (p *HuyiProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	content := strings.TrimSpace(req.Message.Content)
	if content == "" && strings.TrimSpace(req.Message.Template) != "" {
		// Best-effort: render content by replacing ${key} with data.
		content = strings.TrimSpace(req.Message.Template)
		for k, v := range req.Message.Data {
			content = strings.ReplaceAll(content, "${"+k+"}", v)
		}
	}
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("%w: huyi requires content or template", ErrInvalidArgument)
	}

	// Ihuyi classic endpoint.
	// http://106.ihuyi.com/webservice/sms.php?method=Submit
	to, err := firstRecipient(req)
	if err != nil {
		return nil, err
	}
	text := normalizeContent(content, p.cfg.Signature)
	form := url.Values{}
	form.Set("account", strings.TrimSpace(p.cfg.APIID))
	form.Set("password", strings.TrimSpace(p.cfg.APIKey))
	form.Set("mobile", to)
	form.Set("content", text)
	form.Set("format", "json")
	status, b, err := postForm(ctx, "http://106.ihuyi.com/webservice/sms.php?method=Submit", form, nil, "", "")
	raw := truncateRaw(string(b), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}
	type respWrap struct {
		Smsid  string `json:"smsid"`
		Code   int    `json:"code"`
		Msg    string `json:"msg"`
		Submit string `json:"submit"`
	}
	var r struct {
		SubmitResult respWrap `json:"SubmitResult"`
	}
	_ = json.Unmarshal(b, &r)
	if !is2xx(status) || r.SubmitResult.Code != 2 {
		msg := strings.TrimSpace(r.SubmitResult.Msg)
		if msg == "" {
			msg = "provider rejected"
		}
		return &SendResult{Provider: p.Kind(), Accepted: false, Status: fmt.Sprintf("http_%d", status), Error: msg, Raw: raw, SentAtUnix: nowUnix()}, errProviderRejected
	}
	return &SendResult{Provider: p.Kind(), MessageID: strings.TrimSpace(r.SubmitResult.Smsid), Accepted: true, Status: "ok", Raw: raw, SentAtUnix: nowUnix()}, nil
}

package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ChuanglanConfig struct {
	Account  string `json:"account"`
	Password string `json:"password"`

	// Optional: international account
	IntelAccount  string `json:"intelAccount"`
	IntelPassword string `json:"intelPassword"`

	// Optional: channel/sign/unsubscribe (marketing)
	Channel     string `json:"channel"`
	Sign        string `json:"sign"`
	Unsubscribe string `json:"unsubscribe"`
}

type ChuanglanProvider struct {
	cfg ChuanglanConfig
}

func NewChuanglan(cfg ChuanglanConfig) (*ChuanglanProvider, error) {
	if strings.TrimSpace(cfg.Account) == "" || strings.TrimSpace(cfg.Password) == "" {
		return nil, fmt.Errorf("%w: chuanglan requires account/password", ErrInvalidConfig)
	}
	return &ChuanglanProvider{cfg: cfg}, nil
}

func (p *ChuanglanProvider) Kind() ProviderKind { return ProviderChuanglan }

func (p *ChuanglanProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Message.Content) == "" {
		// best-effort: if template provided, render as plain text
		if strings.TrimSpace(req.Message.Template) == "" {
			return nil, fmt.Errorf("%w: chuanglan requires content", ErrInvalidArgument)
		}
	}

	to, err := firstRecipient(req)
	if err != nil {
		return nil, err
	}
	content := strings.TrimSpace(req.Message.Content)
	if content == "" {
		content = strings.TrimSpace(req.Message.Template)
		for k, v := range req.Message.Data {
			content = strings.ReplaceAll(content, "${"+k+"}", v)
		}
	}
	content = normalizeContent(content, p.cfg.Sign)
	type payload struct {
		Account     string `json:"account"`
		Password    string `json:"password"`
		Phone       string `json:"phone"`
		Msg         string `json:"msg"`
		Report      string `json:"report,omitempty"`
		SmsType     string `json:"smsType,omitempty"`
		Extend      string `json:"extend,omitempty"`
		UID         string `json:"uid,omitempty"`
		Channel     string `json:"channel,omitempty"`
		Unsubscribe string `json:"unsub,omitempty"`
	}
	pl := payload{
		Account:     strings.TrimSpace(p.cfg.Account),
		Password:    strings.TrimSpace(p.cfg.Password),
		Phone:       to,
		Msg:         content,
		Report:      "true",
		Channel:     strings.TrimSpace(p.cfg.Channel),
		Unsubscribe: strings.TrimSpace(p.cfg.Unsubscribe),
	}
	bj, _ := json.Marshal(pl)
	// Chuanglan json endpoint (domestic).
	status, b, err := postJSON(ctx, "https://smssh1.253.com/msg/send/json", bj, nil, "", "")
	raw := truncateRaw(string(b), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}
	var r struct {
		Code  string `json:"code"`
		Msg   string `json:"msg"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal(b, &r)
	if !is2xx(status) || strings.TrimSpace(r.Code) != "0" {
		msg := strings.TrimSpace(r.Msg)
		if msg == "" {
			msg = strings.TrimSpace(r.Error)
		}
		if msg == "" {
			msg = "provider rejected"
		}
		return &SendResult{Provider: p.Kind(), Accepted: false, Status: fmt.Sprintf("http_%d", status), Error: msg, Raw: raw, SentAtUnix: nowUnix()}, errProviderRejected
	}
	return &SendResult{Provider: p.Kind(), Accepted: true, Status: "ok", Raw: raw, SentAtUnix: nowUnix()}, nil
}

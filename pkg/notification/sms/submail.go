package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type SubmailConfig struct {
	AppID   string `json:"appId"`
	AppKey  string `json:"appKey"`
	Project string `json:"project"` // default project
}

type SubmailProvider struct {
	cfg SubmailConfig
}

func NewSubmail(cfg SubmailConfig) (*SubmailProvider, error) {
	if strings.TrimSpace(cfg.AppID) == "" || strings.TrimSpace(cfg.AppKey) == "" {
		return nil, fmt.Errorf("%w: submail requires appId/appKey", ErrInvalidConfig)
	}
	return &SubmailProvider{cfg: cfg}, nil
}

func (p *SubmailProvider) Kind() ProviderKind { return ProviderSubmail }

func (p *SubmailProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	// Submail often uses template(project) + data; we accept either template or content but require one.
	if strings.TrimSpace(req.Message.Template) == "" && strings.TrimSpace(req.Message.Content) == "" {
		return nil, fmt.Errorf("%w: submail requires template or content", ErrInvalidArgument)
	}

	to, err := firstRecipient(req)
	if err != nil {
		return nil, err
	}
	project := strings.TrimSpace(req.Message.Template)
	if project == "" {
		project = strings.TrimSpace(p.cfg.Project)
	}
	form := url.Values{}
	form.Set("appid", strings.TrimSpace(p.cfg.AppID))
	form.Set("signature", strings.TrimSpace(p.cfg.AppKey))
	form.Set("to", to)
	if project != "" {
		// XSend (template).
		form.Set("project", project)
		if len(req.Message.Data) > 0 {
			form.Set("vars", jsonString(req.Message.Data))
		}
		status, b, err := postForm(ctx, "https://api.mysubmail.com/message/xsend.json", form, nil, "", "")
		raw := truncateRaw(string(b), 4000)
		if err != nil {
			return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
		}
		var r struct {
			Status string `json:"status"`
			SendID string `json:"send_id"`
			Msg    string `json:"msg"`
		}
		_ = json.Unmarshal(b, &r)
		if !is2xx(status) || strings.ToLower(strings.TrimSpace(r.Status)) != "success" {
			msg := strings.TrimSpace(r.Msg)
			if msg == "" {
				msg = "provider rejected"
			}
			return &SendResult{Provider: p.Kind(), Accepted: false, Status: fmt.Sprintf("http_%d", status), Error: msg, Raw: raw, SentAtUnix: nowUnix()}, errProviderRejected
		}
		return &SendResult{Provider: p.Kind(), MessageID: strings.TrimSpace(r.SendID), Accepted: true, Status: "ok", Raw: raw, SentAtUnix: nowUnix()}, nil
	}

	// Content mode fallback: send raw content.
	form.Set("content", strings.TrimSpace(req.Message.Content))
	status, b, err := postForm(ctx, "https://api.mysubmail.com/message/send.json", form, nil, "", "")
	raw := truncateRaw(string(b), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}
	var r struct {
		Status string `json:"status"`
		SendID string `json:"send_id"`
		Msg    string `json:"msg"`
	}
	_ = json.Unmarshal(b, &r)
	if !is2xx(status) || strings.ToLower(strings.TrimSpace(r.Status)) != "success" {
		msg := strings.TrimSpace(r.Msg)
		if msg == "" {
			msg = "provider rejected"
		}
		return &SendResult{Provider: p.Kind(), Accepted: false, Status: fmt.Sprintf("http_%d", status), Error: msg, Raw: raw, SentAtUnix: nowUnix()}, errProviderRejected
	}
	return &SendResult{Provider: p.Kind(), MessageID: strings.TrimSpace(r.SendID), Accepted: true, Status: "ok", Raw: raw, SentAtUnix: nowUnix()}, nil
}

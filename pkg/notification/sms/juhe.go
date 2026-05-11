package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type JuheConfig struct {
	AppKey string `json:"appKey"`
}

type JuheProvider struct {
	cfg JuheConfig
}

func NewJuhe(cfg JuheConfig) (*JuheProvider, error) {
	if strings.TrimSpace(cfg.AppKey) == "" {
		return nil, fmt.Errorf("%w: juhe requires appKey", ErrInvalidConfig)
	}
	return &JuheProvider{cfg: cfg}, nil
}

func (p *JuheProvider) Kind() ProviderKind { return ProviderJuhe }

func (p *JuheProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	// Juhe uses template + data.
	tpl := strings.TrimSpace(req.Message.Template)
	if tpl == "" {
		// fallback: allow passing template id via extras.tpl_id when using content-only
		if v, ok := req.Extras["tpl_id"]; ok {
			tpl = fmt.Sprint(v)
		}
	}
	if strings.TrimSpace(tpl) == "" {
		return nil, fmt.Errorf("%w: juhe requires template id", ErrInvalidArgument)
	}
	to, err := firstRecipient(req)
	if err != nil {
		return nil, err
	}
	// tpl_value format: #key#=value&#key2#=value2
	var parts []string
	for k, v := range req.Message.Data {
		kk := "#" + strings.TrimSpace(k) + "#"
		parts = append(parts, kk+"="+strings.TrimSpace(v))
	}
	form := url.Values{}
	form.Set("key", strings.TrimSpace(p.cfg.AppKey))
	form.Set("mobile", to)
	form.Set("tpl_id", tpl)
	if len(parts) > 0 {
		form.Set("tpl_value", strings.Join(parts, "&"))
	}
	status, b, err := postForm(ctx, "http://v.juhe.cn/sms/send", form, nil, "", "")
	raw := truncateRaw(string(b), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}
	var r struct {
		ErrorCode int    `json:"error_code"`
		Reason    string `json:"reason"`
		Result    any    `json:"result"`
	}
	_ = json.Unmarshal(b, &r)
	if !is2xx(status) || r.ErrorCode != 0 {
		msg := strings.TrimSpace(r.Reason)
		if msg == "" {
			msg = "provider rejected"
		}
		return &SendResult{Provider: p.Kind(), Accepted: false, Status: fmt.Sprintf("http_%d", status), Error: msg, Raw: raw, SentAtUnix: nowUnix()}, errProviderRejected
	}
	return &SendResult{Provider: p.Kind(), Accepted: true, Status: "ok", Raw: raw, SentAtUnix: nowUnix()}, nil
}

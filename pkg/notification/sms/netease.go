package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type NeteaseConfig struct {
	AppKey    string `json:"appKey"`
	AppSecret string `json:"appSecret"`
	Endpoint  string `json:"endpoint,omitempty"` // default https://api.netease.im
}

type NeteaseProvider struct {
	cfg NeteaseConfig
}

func NewNetease(cfg NeteaseConfig) (*NeteaseProvider, error) {
	if strings.TrimSpace(cfg.AppKey) == "" || strings.TrimSpace(cfg.AppSecret) == "" {
		return nil, fmt.Errorf("%w: netease requires appKey/appSecret", ErrInvalidConfig)
	}
	return &NeteaseProvider{cfg: cfg}, nil
}

func (p *NeteaseProvider) Kind() ProviderKind { return ProviderNeteaseYunx }

func (p *NeteaseProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Message.Template) == "" {
		return nil, fmt.Errorf("%w: netease requires template", ErrInvalidArgument)
	}

	endpoint := strings.TrimSpace(p.cfg.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.netease.im"
	}
	nonce := randHex(8)
	cur := fmt.Sprintf("%d", time.Now().Unix())
	checksum := sha1Hex(strings.TrimSpace(p.cfg.AppSecret) + nonce + cur)
	headers := map[string]string{
		"AppKey":       strings.TrimSpace(p.cfg.AppKey),
		"Nonce":        nonce,
		"CurTime":      cur,
		"CheckSum":     checksum,
		"Content-Type": "application/x-www-form-urlencoded",
	}

	// sendtemplate.action: templateid + mobiles + params(JSON array)
	var mobiles []string
	for _, pn := range req.To {
		mobiles = append(mobiles, strings.TrimPrefix(pn.String(), "+"))
	}
	mobilesJSON, _ := json.Marshal(mobiles)
	var params []string
	if req.Extras != nil {
		if arr, ok := req.Extras["params"]; ok {
			b, _ := json.Marshal(arr)
			_ = json.Unmarshal(b, &params)
		}
	}
	if len(params) == 0 {
		for _, v := range req.Message.Data {
			params = append(params, v)
		}
	}
	paramsJSON, _ := json.Marshal(params)

	form := url.Values{}
	form.Set("templateid", strings.TrimSpace(req.Message.Template))
	form.Set("mobiles", string(mobilesJSON))
	form.Set("params", string(paramsJSON))

	status, b, err := postForm(ctx, endpoint+"/sms/sendtemplate.action", form, headers, "", "")
	raw := truncateRaw(string(b), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}
	var r struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Obj  string `json:"obj"`
	}
	_ = json.Unmarshal(b, &r)
	if !is2xx(status) || r.Code != 200 {
		msg := strings.TrimSpace(r.Msg)
		if msg == "" {
			msg = "provider rejected"
		}
		return &SendResult{Provider: p.Kind(), MessageID: strings.TrimSpace(r.Obj), Accepted: false, Status: fmt.Sprintf("%d", r.Code), Error: msg, Raw: raw, SentAtUnix: time.Now().Unix()}, errProviderRejected
	}
	return &SendResult{Provider: p.Kind(), MessageID: strings.TrimSpace(r.Obj), Accepted: true, Status: "ok", Raw: raw, SentAtUnix: time.Now().Unix()}, nil
}

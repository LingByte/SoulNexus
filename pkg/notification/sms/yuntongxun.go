package sms

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type YuntongxunConfig struct {
	AppID         string `json:"appId"`
	AccountSID    string `json:"accountSid"`
	AccountToken  string `json:"accountToken"`
	IsSubAccount  bool
	SubAccountSID string
	SubToken      string
	Endpoint      string `json:"endpoint,omitempty"` // default https://app.cloopen.com:8883
}

type YuntongxunProvider struct {
	cfg YuntongxunConfig
}

func NewYuntongxun(cfg YuntongxunConfig) (*YuntongxunProvider, error) {
	if strings.TrimSpace(cfg.AppID) == "" || strings.TrimSpace(cfg.AccountSID) == "" || strings.TrimSpace(cfg.AccountToken) == "" {
		return nil, fmt.Errorf("%w: yuntongxun requires appId/accountSid/accountToken", ErrInvalidConfig)
	}
	return &YuntongxunProvider{cfg: cfg}, nil
}

func (p *YuntongxunProvider) Kind() ProviderKind { return ProviderYuntongxun }

func (p *YuntongxunProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	// Cloopen uses template + data.
	if strings.TrimSpace(req.Message.Template) == "" {
		return nil, fmt.Errorf("%w: yuntongxun requires template", ErrInvalidArgument)
	}

	endpoint := strings.TrimSpace(p.cfg.Endpoint)
	if endpoint == "" {
		endpoint = "https://app.cloopen.com:8883"
	}
	ts := time.Now().Format("20060102150405")
	sigRaw := strings.ToUpper(md5Hex(strings.TrimSpace(p.cfg.AccountSID) + strings.TrimSpace(p.cfg.AccountToken) + ts))
	auth := base64.StdEncoding.EncodeToString([]byte(strings.TrimSpace(p.cfg.AccountSID) + ":" + ts))

	// datas ordered
	var datas []string
	if req.Extras != nil {
		if arr, ok := req.Extras["params"]; ok {
			b, _ := json.Marshal(arr)
			_ = json.Unmarshal(b, &datas)
		}
	}
	if len(datas) == 0 {
		for _, v := range req.Message.Data {
			datas = append(datas, v)
		}
	}
	to, err := firstRecipient(req)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"to":         strings.TrimPrefix(to, "+"),
		"appId":      strings.TrimSpace(p.cfg.AppID),
		"templateId": strings.TrimSpace(req.Message.Template),
		"datas":      datas,
	}
	bj, _ := json.Marshal(payload)
	headers := map[string]string{
		"Accept":        "application/json",
		"Content-Type":  "application/json;charset=utf-8",
		"Authorization": auth,
	}
	url := fmt.Sprintf("%s/2013-12-26/Accounts/%s/SMS/TemplateSMS?sig=%s", endpoint, strings.TrimSpace(p.cfg.AccountSID), sigRaw)
	status, b, err := postJSON(ctx, url, bj, headers, "", "")
	raw := truncateRaw(string(b), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}
	var r struct {
		StatusCode  string `json:"statusCode"`
		StatusMsg   string `json:"statusMsg"`
		TemplateSMS struct {
			SmsMessageSid string `json:"smsMessageSid"`
			DateCreated   string `json:"dateCreated"`
		} `json:"templateSMS"`
	}
	_ = json.Unmarshal(b, &r)
	if !is2xx(status) || strings.TrimSpace(r.StatusCode) != "000000" {
		msg := strings.TrimSpace(r.StatusMsg)
		if msg == "" {
			msg = "provider rejected"
		}
		return &SendResult{Provider: p.Kind(), MessageID: strings.TrimSpace(r.TemplateSMS.SmsMessageSid), Accepted: false, Status: r.StatusCode, Error: msg, Raw: raw, SentAtUnix: time.Now().Unix()}, errProviderRejected
	}
	return &SendResult{Provider: p.Kind(), MessageID: strings.TrimSpace(r.TemplateSMS.SmsMessageSid), Accepted: true, Status: "ok", Raw: raw, SentAtUnix: time.Now().Unix()}, nil
}

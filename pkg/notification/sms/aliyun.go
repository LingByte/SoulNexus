package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/alibabacloud-go/tea/tea"
)

// AliyunConfig aligns with easy-sms style fields.
type AliyunConfig struct {
	AccessKeyID     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	SignName        string `json:"signName"`
	Endpoint        string `json:"endpoint,omitempty"` // default dysmsapi.aliyuncs.com
}

type AliyunProvider struct {
	cfg AliyunConfig
}

func NewAliyun(cfg AliyunConfig) (*AliyunProvider, error) {
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.AccessKeySecret) == "" {
		return nil, fmt.Errorf("%w: aliyun requires accessKeyId/accessKeySecret", ErrInvalidConfig)
	}
	if strings.TrimSpace(cfg.SignName) == "" {
		return nil, fmt.Errorf("%w: aliyun requires signName", ErrInvalidConfig)
	}
	return &AliyunProvider{cfg: cfg}, nil
}

func (p *AliyunProvider) Kind() ProviderKind { return ProviderAliyun }

func (p *AliyunProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	// Aliyun uses template + data. Content mode is not implemented here.
	if strings.TrimSpace(req.Message.Template) == "" {
		return nil, fmt.Errorf("%w: aliyun requires template", ErrInvalidArgument)
	}

	endpoint := strings.TrimSpace(p.cfg.Endpoint)
	if endpoint == "" {
		endpoint = "dysmsapi.aliyuncs.com"
	}
	cfg := &openapi.Config{
		AccessKeyId:     tea.String(strings.TrimSpace(p.cfg.AccessKeyID)),
		AccessKeySecret: tea.String(strings.TrimSpace(p.cfg.AccessKeySecret)),
		Endpoint:        tea.String(endpoint),
	}
	client, err := dysmsapi.NewClient(cfg)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), SentAtUnix: nowUnix()}, err
	}
	to, err := firstRecipient(req)
	if err != nil {
		return nil, err
	}
	// Template params: allow ordered params via Extras.params, otherwise map values.
	var params []string
	if req.Extras != nil {
		if arr, ok := req.Extras["params"]; ok {
			b, _ := json.Marshal(arr)
			_ = json.Unmarshal(b, &params)
		}
	}
	tplParam := "{}"
	if len(params) > 0 {
		// Aliyun expects JSON object or array depending on template; we pass object {"0":"a","1":"b"} if ordered.
		m := map[string]string{}
		for i, v := range params {
			m[fmt.Sprintf("%d", i)] = v
		}
		if b, err := json.Marshal(m); err == nil {
			tplParam = string(b)
		}
	} else if len(req.Message.Data) > 0 {
		if b, err := json.Marshal(req.Message.Data); err == nil {
			tplParam = string(b)
		}
	}

	sign := strings.TrimSpace(req.Message.SignName)
	if sign == "" {
		sign = strings.TrimSpace(p.cfg.SignName)
	}
	r := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(to),
		SignName:      tea.String(sign),
		TemplateCode:  tea.String(strings.TrimSpace(req.Message.Template)),
		TemplateParam: tea.String(tplParam),
	}
	resp, err := client.SendSms(r)
	raw := truncateRaw(jsonString(resp), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}
	code := ""
	bizID := ""
	msg := ""
	if resp != nil && resp.Body != nil {
		if resp.Body.Code != nil {
			code = *resp.Body.Code
		}
		if resp.Body.BizId != nil {
			bizID = *resp.Body.BizId
		}
		if resp.Body.Message != nil {
			msg = *resp.Body.Message
		}
	}
	if strings.ToUpper(strings.TrimSpace(code)) != "OK" {
		if strings.TrimSpace(msg) == "" {
			msg = "provider rejected"
		}
		return &SendResult{Provider: p.Kind(), MessageID: bizID, Accepted: false, Status: code, Error: msg, Raw: raw, SentAtUnix: time.Now().Unix()}, errProviderRejected
	}
	return &SendResult{Provider: p.Kind(), MessageID: bizID, Accepted: true, Status: code, Raw: raw, SentAtUnix: time.Now().Unix()}, nil
}

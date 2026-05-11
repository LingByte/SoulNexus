package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tccommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tcsms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
)

type TencentConfig struct {
	SdkAppID  string `json:"sdkAppId"`
	SecretID  string `json:"secretId"`
	SecretKey string `json:"secretKey"`
	SignName  string `json:"signName"`
	Region    string `json:"region,omitempty"`
}

type TencentProvider struct {
	cfg TencentConfig
}

func NewTencent(cfg TencentConfig) (*TencentProvider, error) {
	if strings.TrimSpace(cfg.SdkAppID) == "" || strings.TrimSpace(cfg.SecretID) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, fmt.Errorf("%w: tencent requires sdkAppId/secretId/secretKey", ErrInvalidConfig)
	}
	if strings.TrimSpace(cfg.SignName) == "" {
		return nil, fmt.Errorf("%w: tencent requires signName", ErrInvalidConfig)
	}
	return &TencentProvider{cfg: cfg}, nil
}

func (p *TencentProvider) Kind() ProviderKind { return ProviderTencent }

func (p *TencentProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Message.Template) == "" {
		return nil, fmt.Errorf("%w: tencent requires template", ErrInvalidArgument)
	}

	region := strings.TrimSpace(p.cfg.Region)
	if region == "" {
		region = "ap-guangzhou"
	}
	cred := tccommon.NewCredential(strings.TrimSpace(p.cfg.SecretID), strings.TrimSpace(p.cfg.SecretKey))
	cp := profile.NewClientProfile()
	client, err := tcsms.NewClient(cred, region, cp)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), SentAtUnix: nowUnix()}, err
	}
	r := tcsms.NewSendSmsRequest()
	r.SmsSdkAppId = tccommon.StringPtr(strings.TrimSpace(p.cfg.SdkAppID))
	r.SignName = tccommon.StringPtr(strings.TrimSpace(p.cfg.SignName))
	r.TemplateId = tccommon.StringPtr(strings.TrimSpace(req.Message.Template))

	var params []*string
	if req.Extras != nil {
		if arr, ok := req.Extras["params"]; ok {
			b, _ := json.Marshal(arr)
			var ss []string
			if json.Unmarshal(b, &ss) == nil {
				for _, s := range ss {
					v := s
					params = append(params, &v)
				}
			}
		}
	}
	if len(params) == 0 {
		for _, v := range req.Message.Data {
			vv := v
			params = append(params, &vv)
		}
	}
	if len(params) > 0 {
		r.TemplateParamSet = params
	}

	var phones []*string
	for _, pn := range req.To {
		s := pn.String()
		if !strings.HasPrefix(s, "+") {
			s = "+86" + strings.TrimPrefix(s, "+")
		}
		v := s
		phones = append(phones, &v)
	}
	r.PhoneNumberSet = phones

	resp, err := client.SendSms(r)
	raw := truncateRaw(jsonString(resp), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}
	accepted := false
	msgID := ""
	status := "unknown"
	if resp != nil && resp.Response != nil && len(resp.Response.SendStatusSet) > 0 {
		s0 := resp.Response.SendStatusSet[0]
		if s0 != nil {
			if s0.SerialNo != nil {
				msgID = *s0.SerialNo
			}
			if s0.Code != nil {
				status = *s0.Code
				accepted = strings.EqualFold(status, "Ok")
			}
		}
	}
	if !accepted {
		return &SendResult{Provider: p.Kind(), MessageID: msgID, Accepted: false, Status: status, Error: "provider rejected", Raw: raw, SentAtUnix: time.Now().Unix()}, errProviderRejected
	}
	return &SendResult{Provider: p.Kind(), MessageID: msgID, Accepted: true, Status: status, Raw: raw, SentAtUnix: time.Now().Unix()}, nil
}

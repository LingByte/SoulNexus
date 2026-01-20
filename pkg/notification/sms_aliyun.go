package notification

import (
	"context"
	"fmt"
)

type AliyunSMSConfig struct {
	AccessKeyId     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	Endpoint        string // 默认 cn-hangzhou
}

type AliyunSMS struct {
	cfg AliyunSMSConfig
	cli AliyunSMSClient
}

// AliyunSMSClient 便于替换/注入的发送接口（适配真实 SDK）
type AliyunSMSClient interface {
	Send(ctx context.Context, phone, sign, template string, params map[string]string) error
}

func NewAliyunSMS(cfg AliyunSMSConfig, cli AliyunSMSClient) *AliyunSMS {
	return &AliyunSMS{cfg: cfg, cli: cli}
}

func (a *AliyunSMS) SendCode(ctx context.Context, phone, code string) error {
	if a.cli == nil {
		return fmt.Errorf("AliyunSMSClient not configured")
	}
	params := map[string]string{"code": code}
	return a.cli.Send(ctx, phone, a.cfg.SignName, a.cfg.TemplateCode, params)
}

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sms

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ProviderKind identifies an SMS provider implementation.
type ProviderKind string

const (
	ProviderAliyun      ProviderKind = "aliyun"
	ProviderTencent     ProviderKind = "tencent"
	ProviderHuawei      ProviderKind = "huawei"
	ProviderYunpian     ProviderKind = "yunpian"
	ProviderSubmail     ProviderKind = "submail"
	ProviderLuosimao    ProviderKind = "luosimao"
	ProviderYuntongxun  ProviderKind = "yuntongxun" // 容联云通讯 / Cloopen
	ProviderHuyi        ProviderKind = "huyi"
	ProviderJuhe        ProviderKind = "juhe"
	ProviderBaidu       ProviderKind = "baidu"
	ProviderHuaxin      ProviderKind = "huaxin"
	ProviderChuanglan   ProviderKind = "chuanglan"
	ProviderRongcloud   ProviderKind = "rongcloud"
	ProviderTwilio      ProviderKind = "twilio"
	ProviderTiniyo      ProviderKind = "tiniyo"
	ProviderUCloud      ProviderKind = "ucloud"
	ProviderNeteaseYunx ProviderKind = "netease" // 网易云信

	// NOTE: ProviderQiniu intentionally NOT included per requirement.
)

var (
	ErrInvalidConfig   = errors.New("sms: invalid config")
	ErrNotImplemented  = errors.New("sms: not implemented")
	ErrInvalidArgument = errors.New("sms: invalid argument")
)

// PhoneNumber is a normalized phone recipient.
// CountryCode uses E.164 integer representation (e.g. 86, 1). If 0, treat as provider default.
type PhoneNumber struct {
	Number      string `json:"number"`
	CountryCode int    `json:"countryCode,omitempty"`
}

func (p PhoneNumber) String() string {
	n := strings.TrimSpace(p.Number)
	if n == "" {
		return ""
	}
	cc := p.CountryCode
	if cc <= 0 {
		return n
	}
	// E.164-ish output (do not enforce '+', because providers differ)
	return fmt.Sprintf("+%d%s", cc, n)
}

// Message describes the sms content selection (content or template + data).
// Different providers may use:
// - Content (云片/螺丝帽等)
// - Template + Data (阿里云/腾讯云/华为云等)
type Message struct {
	Content  string            `json:"content,omitempty"`
	Template string            `json:"template,omitempty"`
	Data     map[string]string `json:"data,omitempty"`
	SignName string            `json:"signName,omitempty"`
}

// SendRequest is the unified input for SMS sending.
type SendRequest struct {
	To      []PhoneNumber `json:"to"`
	Message Message       `json:"message"`

	// Optional: provider-specific extras (kept generic to avoid over-encoding).
	Extras map[string]any `json:"extras,omitempty"`
}

// SendResult is the normalized send response.
type SendResult struct {
	Provider   ProviderKind `json:"provider"`
	MessageID  string       `json:"messageId,omitempty"`
	Accepted   bool         `json:"accepted"`
	Status     string       `json:"status,omitempty"` // provider-defined or normalized status
	Error      string       `json:"error,omitempty"`
	Raw        string       `json:"raw,omitempty"` // raw provider response (best-effort)
	SentAtUnix int64        `json:"sentAt,omitempty"`
}

// Provider is the minimal interface we align across platforms (go-easy-sms-style gateways).
type Provider interface {
	Kind() ProviderKind
	Send(ctx context.Context, req SendRequest) (*SendResult, error)
}

// ValidateBasic validates common request fields.
func ValidateBasic(req SendRequest) error {
	if len(req.To) == 0 {
		return fmt.Errorf("%w: empty recipients", ErrInvalidArgument)
	}
	for i, p := range req.To {
		if strings.TrimSpace(p.Number) == "" {
			return fmt.Errorf("%w: empty recipient at %d", ErrInvalidArgument, i)
		}
	}
	if strings.TrimSpace(req.Message.Content) == "" && strings.TrimSpace(req.Message.Template) == "" {
		return fmt.Errorf("%w: message content/template required", ErrInvalidArgument)
	}
	return nil
}

func nowUnix() int64 { return time.Now().Unix() }

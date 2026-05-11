package sms

import (
	"context"
	"fmt"
	"strings"
)

type HuaxinConfig struct {
	UserID   string
	Password string
	Account  string
	IP       string
	ExtNo    string
}

type HuaxinProvider struct {
	cfg HuaxinConfig
}

func NewHuaxin(cfg HuaxinConfig) (*HuaxinProvider, error) {
	if strings.TrimSpace(cfg.UserID) == "" || strings.TrimSpace(cfg.Password) == "" {
		return nil, fmt.Errorf("%w: huaxin requires userId/password", ErrInvalidConfig)
	}
	return &HuaxinProvider{cfg: cfg}, nil
}

func (p *HuaxinProvider) Kind() ProviderKind { return ProviderHuaxin }

func (p *HuaxinProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	_ = ctx
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Message.Content) == "" {
		return nil, fmt.Errorf("%w: huaxin requires content", ErrInvalidArgument)
	}
	return &SendResult{Provider: p.Kind(), Accepted: false, Error: ErrNotImplemented.Error(), SentAtUnix: nowUnix()}, ErrNotImplemented
}

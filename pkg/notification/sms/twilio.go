package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type TwilioConfig struct {
	AccountSID string `json:"accountSid"`
	Token      string `json:"token"`
	From       string `json:"from"`
}

type TwilioProvider struct {
	cfg TwilioConfig
}

func NewTwilio(cfg TwilioConfig) (*TwilioProvider, error) {
	if strings.TrimSpace(cfg.AccountSID) == "" || strings.TrimSpace(cfg.Token) == "" || strings.TrimSpace(cfg.From) == "" {
		return nil, fmt.Errorf("%w: twilio requires accountSid/token/from", ErrInvalidConfig)
	}
	return &TwilioProvider{cfg: cfg}, nil
}

func (p *TwilioProvider) Kind() ProviderKind { return ProviderTwilio }

func (p *TwilioProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	// Twilio uses content text.
	if strings.TrimSpace(req.Message.Content) == "" {
		return nil, fmt.Errorf("%w: twilio requires content", ErrInvalidArgument)
	}

	type twilioResp struct {
		Sid   string `json:"sid"`
		Error string `json:"message"`
	}

	to, err := firstRecipient(req)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", strings.TrimSpace(p.cfg.AccountSID))
	form := url.Values{}
	form.Set("To", to)
	form.Set("From", strings.TrimSpace(p.cfg.From))
	form.Set("Body", strings.TrimSpace(req.Message.Content))
	status, b, err := postForm(ctx, endpoint, form, nil, strings.TrimSpace(p.cfg.AccountSID), strings.TrimSpace(p.cfg.Token))
	raw := truncateRaw(string(b), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}
	var r twilioResp
	_ = json.Unmarshal(b, &r)
	if !is2xx(status) || strings.TrimSpace(r.Sid) == "" {
		msg := strings.TrimSpace(r.Error)
		if msg == "" {
			msg = fmt.Sprintf("http_%d", status)
		}
		return &SendResult{
			Provider:   p.Kind(),
			Accepted:   false,
			Status:     fmt.Sprintf("http_%d", status),
			Error:      msg,
			Raw:        raw,
			SentAtUnix: nowUnix(),
		}, errProviderRejected
	}
	return &SendResult{
		Provider:   p.Kind(),
		MessageID:  strings.TrimSpace(r.Sid),
		Accepted:   true,
		Status:     "queued",
		Raw:        raw,
		SentAtUnix: nowUnix(),
	}, nil
}

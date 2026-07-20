package mail

import (
	"testing"
)

func TestChannelLabel(t *testing.T) {
	tests := []struct {
		name string
		cfg  MailConfig
		want string
	}{
		{
			"custom name takes priority",
			MailConfig{Name: "my-channel", Provider: ProviderSendCloud},
			"my-channel",
		},
		{
			"sendcloud with api user",
			MailConfig{Provider: ProviderSendCloud, APIUser: "alice"},
			"sendcloud:alice",
		},
		{
			"sendcloud without api user",
			MailConfig{Provider: ProviderSendCloud},
			ProviderSendCloud,
		},
		{
			"smtp with host",
			MailConfig{Provider: ProviderSMTP, Host: "smtp.example.com", Port: 587},
			"smtp:smtp.example.com:587",
		},
		{
			"default provider",
			MailConfig{},
			ProviderSendCloud,
		},
	}
	for _, tt := range tests {
		got := channelLabel(tt.cfg)
		if got != tt.want {
			t.Errorf("%s: channelLabel() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestNewProviderFromConfig_sendcloud(t *testing.T) {
	cfg := MailConfig{
		Provider: ProviderSendCloud,
		APIUser:  "user",
		APIKey:   "key",
		From:     "sender@example.com",
	}
	p, err := NewProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewProviderFromConfig: %v", err)
	}
	if p.Kind() != ProviderSendCloud {
		t.Errorf("Kind = %q", p.Kind())
	}
}

func TestNewProviderFromConfig_sendcloud_missingFields(t *testing.T) {
	tests := []MailConfig{
		{Provider: ProviderSendCloud, APIKey: "key", From: "sender@example.com"},
		{Provider: ProviderSendCloud, APIUser: "user", From: "sender@example.com"},
		{Provider: ProviderSendCloud, APIUser: "user", APIKey: "key"},
	}
	for _, cfg := range tests {
		_, err := NewProviderFromConfig(cfg)
		if err == nil {
			t.Errorf("expected error for config missing required field: %+v", cfg)
		}
	}
}

func TestNewProviderFromConfig_smtp(t *testing.T) {
	cfg := MailConfig{
		Provider: ProviderSMTP,
		Host:     "smtp.example.com",
		Port:     587,
		From:     "sender@example.com",
	}
	p, err := NewProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewProviderFromConfig: %v", err)
	}
	if p.Kind() != ProviderSMTP {
		t.Errorf("Kind = %q", p.Kind())
	}
}

func TestNewProviderFromConfig_smtp_missingFields(t *testing.T) {
	tests := []MailConfig{
		{Provider: ProviderSMTP, Port: 587, From: "sender@example.com"},
		{Provider: ProviderSMTP, Host: "smtp.example.com", From: "sender@example.com"},
		{Provider: ProviderSMTP, Host: "smtp.example.com", Port: 587},
	}
	for _, cfg := range tests {
		_, err := NewProviderFromConfig(cfg)
		if err == nil {
			t.Errorf("expected error for config missing required field: %+v", cfg)
		}
	}
}

func TestNewMailer_emptyChannels(t *testing.T) {
	_, err := NewMailer(nil, nil, "")
	if err == nil {
		t.Error("expected error for empty channels")
	}
}

func TestNewMailer_allInvalid(t *testing.T) {
	cfg := MailConfig{Provider: ProviderSendCloud} // missing required fields
	_, err := NewMailer([]MailConfig{cfg}, nil, "")
	if err == nil {
		t.Error("expected error for all-invalid channels")
	}
}

func TestNewMailer_valid(t *testing.T) {
	cfg := MailConfig{
		Name:     "test",
		Provider: ProviderSendCloud,
		APIUser:  "user",
		APIKey:   "key",
		From:     "sender@example.com",
	}
	m, err := NewMailer([]MailConfig{cfg}, nil, "")
	if err != nil {
		t.Fatalf("NewMailer: %v", err)
	}
	if len(m.channels) != 1 {
		t.Errorf("channels = %d", len(m.channels))
	}
	if m.retry.MaxAttempts != 3 {
		t.Errorf("retry.MaxAttempts = %d", m.retry.MaxAttempts)
	}
}

func TestNewMailer_withOptions(t *testing.T) {
	cfg := MailConfig{
		Provider: ProviderSendCloud,
		APIUser:  "user",
		APIKey:   "key",
		From:     "sender@example.com",
	}
	m, err := NewMailer(
		[]MailConfig{cfg},
		nil,
		"",
		WithRetry(RetryPolicy{MaxAttempts: 5}),
		WithMailLogUserID(123),
	)
	if err != nil {
		t.Fatalf("NewMailer: %v", err)
	}
	if m.retry.MaxAttempts != 5 {
		t.Errorf("retry.MaxAttempts = %d", m.retry.MaxAttempts)
	}
	if m.rt.userID != 123 {
		t.Errorf("rt.userID = %d", m.rt.userID)
	}
}

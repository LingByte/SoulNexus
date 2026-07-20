package mail

import (
	"testing"
)

func TestNewSMTPClient_valid(t *testing.T) {
	cfg := SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "Test Sender <sender@example.com>",
	}
	client, err := NewSMTPClient(cfg)
	if err != nil {
		t.Fatalf("NewSMTPClient: %v", err)
	}
	if client == nil {
		t.Fatal("NewSMTPClient returned nil")
	}
}

func TestNewSMTPClient_emptyFrom(t *testing.T) {
	cfg := SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "",
	}
	_, err := NewSMTPClient(cfg)
	if err == nil {
		t.Error("expected error for empty from")
	}
}

func TestNewSMTPClient_invalidFrom(t *testing.T) {
	cfg := SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "<invalid>",
	}
	_, err := NewSMTPClient(cfg)
	if err == nil {
		t.Error("expected error for invalid from")
	}
}

func TestNewSMTPClient_withFromNameFallback(t *testing.T) {
	cfg := SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "sender@example.com",
		FromName: "Display Name",
	}
	client, err := NewSMTPClient(cfg)
	if err != nil {
		t.Fatalf("NewSMTPClient: %v", err)
	}
	if client.sender.Display != "Display Name" {
		t.Errorf("Display = %q, want %q", client.sender.Display, "Display Name")
	}
	if client.sender.Envelope != "sender@example.com" {
		t.Errorf("Envelope = %q, want %q", client.sender.Envelope, "sender@example.com")
	}
}

func TestSMTPClient_Kind(t *testing.T) {
	cfg := SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "sender@example.com",
	}
	client, _ := NewSMTPClient(cfg)
	if got := client.Kind(); got != ProviderSMTP {
		t.Errorf("Kind() = %q, want %q", got, ProviderSMTP)
	}
}

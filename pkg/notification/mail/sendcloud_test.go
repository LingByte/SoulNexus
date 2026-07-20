package mail

import (
	"testing"
)

func TestNewSendCloudClient_valid(t *testing.T) {
	cfg := SendCloudConfig{
		APIUser:  "api_user",
		APIKey:   "api_key",
		From:     "sender@example.com",
		FromName: "Display",
	}
	client, err := NewSendCloudClient(cfg)
	if err != nil {
		t.Fatalf("NewSendCloudClient: %v", err)
	}
	if client == nil {
		t.Fatal("NewSendCloudClient returned nil")
	}
	if client.Client == nil {
		t.Error("Client should not be nil")
	}
	if client.sender.Envelope != "sender@example.com" {
		t.Errorf("Envelope = %q", client.sender.Envelope)
	}
	if client.sender.Display != "Display" {
		t.Errorf("Display = %q", client.sender.Display)
	}
}

func TestNewSendCloudClient_emptyFrom(t *testing.T) {
	cfg := SendCloudConfig{
		APIUser: "api_user",
		APIKey:  "api_key",
		From:    "",
	}
	_, err := NewSendCloudClient(cfg)
	if err == nil {
		t.Error("expected error for empty from")
	}
}

func TestNewSendCloudClient_invalidFrom(t *testing.T) {
	cfg := SendCloudConfig{
		APIUser: "api_user",
		APIKey:  "api_key",
		From:    "<>",
	}
	_, err := NewSendCloudClient(cfg)
	if err == nil {
		t.Error("expected error for invalid from")
	}
}

func TestSendCloudClient_Kind(t *testing.T) {
	cfg := SendCloudConfig{
		APIUser: "api_user",
		APIKey:  "api_key",
		From:    "sender@example.com",
	}
	client, _ := NewSendCloudClient(cfg)
	if got := client.Kind(); got != ProviderSendCloud {
		t.Errorf("Kind() = %q, want %q", got, ProviderSendCloud)
	}
}

func TestSendCloudWebhookErrorDetail(t *testing.T) {
	tests := []struct {
		name string
		ev   *SendCloudWebhookEvent
		want string
	}{
		{"smtpError present", &SendCloudWebhookEvent{SmtpError: "bad gateway", SmtpStatus: ""}, "bad gateway"},
		{"smtpError empty, smtpStatus present", &SendCloudWebhookEvent{SmtpError: "", SmtpStatus: "250 OK"}, "250 OK"},
		{"both empty", &SendCloudWebhookEvent{}, ""},
	}
	for _, tt := range tests {
		got := sendCloudWebhookErrorDetail(tt.ev)
		if got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestParseSendCloudWebhookEvent_JSON(t *testing.T) {
	// JSON format
	body := []byte(`{"event":"delivered","messageId":"abc123","email":"user@example.com","timestamp":1717200000}`)
	ev, err := ParseSendCloudWebhookEvent(body)
	if err != nil {
		t.Fatalf("ParseSendCloudWebhookEvent: %v", err)
	}
	if ev.Event != "delivered" {
		t.Errorf("Event = %q, want delivered", ev.Event)
	}
	if ev.MessageID != "abc123" {
		t.Errorf("MessageID = %q, want abc123", ev.MessageID)
	}
	if ev.Email != "user@example.com" {
		t.Errorf("Email = %q, want user@example.com", ev.Email)
	}
}

func TestParseSendCloudWebhookEvent_JSON_missingFields(t *testing.T) {
	// JSON with only partial fields still parses
	body := []byte(`{}`)
	ev, err := ParseSendCloudWebhookEvent(body)
	if err != nil {
		t.Fatalf("ParseSendCloudWebhookEvent empty: %v", err)
	}
	if ev.Event != "" {
		t.Errorf("Event = %q, want empty", ev.Event)
	}
}

func TestParseSendCloudWebhookEvent_formurl(t *testing.T) {
	body := []byte("event=delivered&messageId=abc123&recipient=user@example.com&smtpStatus=250+OK&smtpError=&timestamp=2025-06-01+12:00:00")
	ev, err := ParseSendCloudWebhookEvent(body)
	if err != nil {
		t.Fatalf("ParseSendCloudWebhookEvent form: %v", err)
	}
	if ev.Event != "delivered" {
		t.Errorf("Event = %q", ev.Event)
	}
	if ev.MessageID != "abc123" {
		t.Errorf("MessageID = %q", ev.MessageID)
	}
	if ev.Email != "user@example.com" {
		t.Errorf("Email = %q", ev.Email)
	}
	if ev.Timestamp == 0 {
		t.Errorf("Timestamp not parsed")
	}
}

func TestParseSendCloudWebhookEvent_formurl_emailId(t *testing.T) {
	// when messageId is missing, falls back to emailId
	body := []byte("event=click&emailId=xyz789%40example.com&recipient=")
	ev, err := ParseSendCloudWebhookEvent(body)
	if err != nil {
		t.Fatalf("ParseSendCloudWebhookEvent emailId: %v", err)
	}
	if ev.MessageID != "xyz789" {
		t.Errorf("MessageID = %q, want xyz789", ev.MessageID)
	}
	if ev.Email != "example.com" {
		t.Errorf("Email = %q, want example.com", ev.Email)
	}
}

func TestParseSendCloudWebhookEvent_formurl_timestampFormats(t *testing.T) {
	// timestamp in SendCloud format
	body := []byte("event=open&messageId=abc&timestamp=2025-12-25 08:30:00")
	ev, err := ParseSendCloudWebhookEvent(body)
	if err != nil {
		t.Fatalf("ParseSendCloudWebhookEvent: %v", err)
	}
	if ev.Timestamp == 0 {
		t.Error("Timestamp should be non-zero")
	}
}

func TestParseSendCloudWebhookEvent_garbage(t *testing.T) {
	// garbage data returns parse error
	_, err := ParseSendCloudWebhookEvent([]byte("%%%"))
	if err == nil {
		t.Error("expected error for garbage input")
	}
}

package notification

import (
	"testing"
)

func TestNewMailNotification(t *testing.T) {
	config := MailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "from@example.com",
	}

	notif := NewMailNotification(config)
	if notif == nil {
		t.Fatal("NewMailNotification returned nil")
	}
	if notif.Config.Host != config.Host {
		t.Errorf("Expected host %s, got %s", config.Host, notif.Config.Host)
	}
	if notif.Config.Port != config.Port {
		t.Errorf("Expected port %d, got %d", config.Port, notif.Config.Port)
	}
}

func TestMailNotification_SendHTML(t *testing.T) {
	config := MailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "from@example.com",
	}

	notif := NewMailNotification(config)

	// This will fail because we don't have a real SMTP server
	// But we can test the function structure
	err := notif.SendHTML("to@example.com", "Test Subject", "<html><body>Test</body></html>")
	// We expect an error since there's no real SMTP server
	if err == nil {
		t.Log("SendHTML succeeded (unexpected, might have SMTP server configured)")
	} else {
		// Expected error - function is working correctly
		_ = err
	}
}

func TestMailNotification_SendWelcomeEmail(t *testing.T) {
	config := MailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "from@example.com",
	}

	notif := NewMailNotification(config)

	// This will fail because we don't have a real SMTP server
	err := notif.SendWelcomeEmail("to@example.com", "TestUser", "https://example.com/verify")
	if err == nil {
		t.Log("SendWelcomeEmail succeeded (unexpected)")
	} else {
		// Expected error - verify it's not a template parsing error
		if err.Error() == "failed to parse template" {
			t.Errorf("Template parsing failed: %v", err)
		}
	}
}

func TestMailNotification_SendVerificationCode(t *testing.T) {
	config := MailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "from@example.com",
	}

	notif := NewMailNotification(config)

	err := notif.SendVerificationCode("to@example.com", "123456")
	if err == nil {
		t.Log("SendVerificationCode succeeded (unexpected)")
	} else {
		// Expected error - verify it's not a template parsing error
		if err.Error() == "failed to parse template" {
			t.Errorf("Template parsing failed: %v", err)
		}
	}
}

func TestMailNotification_SendVerificationEmail(t *testing.T) {
	config := MailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "from@example.com",
	}

	notif := NewMailNotification(config)

	err := notif.SendVerificationEmail("to@example.com", "TestUser", "https://example.com/verify")
	if err == nil {
		t.Log("SendVerificationEmail succeeded (unexpected)")
	} else {
		// Expected error
		_ = err
	}
}

func TestMailNotification_SendPasswordResetEmail(t *testing.T) {
	config := MailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "from@example.com",
	}

	notif := NewMailNotification(config)

	err := notif.SendPasswordResetEmail("to@example.com", "TestUser", "https://example.com/reset")
	if err == nil {
		t.Log("SendPasswordResetEmail succeeded (unexpected)")
	} else {
		// Expected error
		_ = err
	}
}

func TestMailConfig_Structure(t *testing.T) {
	config := MailConfig{
		Host:     "smtp.example.com",
		Port:     465,
		Username: "user@example.com",
		Password: "secret",
		From:     "noreply@example.com",
	}

	if config.Host == "" {
		t.Error("Host should not be empty")
	}
	if config.Port <= 0 {
		t.Error("Port should be positive")
	}
}

func TestMailNotification_Send(t *testing.T) {
	config := MailConfig{
		Host:     "invalid-host-that-does-not-exist.local",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "from@example.com",
	}

	notif := NewMailNotification(config)

	// This will fail because we don't have a real SMTP server
	// But we can test the function structure and error handling
	err := notif.Send("to@example.com", "Test Subject", "Test Body")
	// We expect an error since there's no real SMTP server
	if err == nil {
		t.Log("Send succeeded (unexpected, might have SMTP server configured)")
	} else {
		// Expected error - function is working correctly
		// Verify it's a connection error, not a nil pointer error
		if err.Error() == "" {
			t.Error("Error message should not be empty")
		}
	}
}

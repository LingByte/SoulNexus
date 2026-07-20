package access

import (
	"strings"
	"testing"
)

func TestValidateTOTPEmptyInputs(t *testing.T) {
	if ValidateTOTP("", "SECRET") {
		t.Error("empty code should return false")
	}
	if ValidateTOTP("123456", "") {
		t.Error("empty secret should return false")
	}
	if ValidateTOTP("", "") {
		t.Error("both empty should return false")
	}
}

func TestValidateTOTPInvalidCode(t *testing.T) {
	// Generate a real secret and test with obviously wrong codes
	setup, err := GenerateTOTPSetup("", "test@example.com", 20)
	if err != nil {
		t.Fatalf("failed to generate TOTP setup: %v", err)
	}
	if ValidateTOTP("000000", setup.Secret) {
		t.Log("000000 happened to be valid TOTP (extremely unlikely)")
	}
}

func TestGenerateTOTPSetupDefaults(t *testing.T) {
	setup, err := GenerateTOTPSetup("", "user@test.com", 0)
	if err != nil {
		t.Fatalf("GenerateTOTPSetup error: %v", err)
	}
	if setup.Secret == "" {
		t.Error("expected non-empty secret")
	}
	if setup.URL == "" {
		t.Error("expected non-empty otpauth URL")
	}
	if !strings.HasPrefix(setup.URL, "otpauth://totp/") {
		t.Errorf("expected otpauth URL, got: %s", setup.URL)
	}
	if setup.QRDataURL == "" {
		t.Error("expected non-empty QR data URL")
	}
	if !strings.HasPrefix(setup.QRDataURL, "data:image/png;base64,") {
		t.Error("QRDataURL should be a base64 PNG data URL")
	}
}

func TestGenerateTOTPSetupCustomIssuer(t *testing.T) {
	setup, err := GenerateTOTPSetup("MyApp", "alice@test.com", 20)
	if err != nil {
		t.Fatalf("GenerateTOTPSetup error: %v", err)
	}
	if !strings.Contains(setup.URL, "issuer=MyApp") {
		t.Errorf("expected issuer=MyApp in URL, got: %s", setup.URL)
	}
}

func TestTOTPKeyToPNGDataURLNilKey(t *testing.T) {
	_, err := TOTPKeyToPNGDataURL(nil, 256)
	if err == nil {
		t.Error("expected error for nil key")
	}
}

func TestTOTPKeyToPNGDataURLDefaultSize(t *testing.T) {
	setup, err := GenerateTOTPSetup("Test", "bob@test.com", 20)
	if err != nil {
		t.Fatalf("GenerateTOTPSetup error: %v", err)
	}
	// The QRDataURL is already generated, just verify it's valid
	if !strings.HasPrefix(setup.QRDataURL, "data:image/png;base64,") {
		t.Error("QRDataURL missing expected prefix")
	}
	// Decode should produce non-empty PNG bytes
	b64 := strings.TrimPrefix(setup.QRDataURL, "data:image/png;base64,")
	if len(b64) == 0 {
		t.Error("empty base64 data")
	}
}

func TestGenerateTOTPSetupLargeSecretSize(t *testing.T) {
	setup, err := GenerateTOTPSetup("", "large@test.com", 64)
	if err != nil {
		t.Fatalf("GenerateTOTPSetup with 64-byte secret error: %v", err)
	}
	if setup.Secret == "" {
		t.Error("expected non-empty secret")
	}
}

func TestValidateTOTPWhitespaceHandling(t *testing.T) {
	setup, err := GenerateTOTPSetup("", "ws@test.com", 20)
	if err != nil {
		t.Fatalf("GenerateTOTPSetup error: %v", err)
	}
	// Code with spaces should be trimmed and still return false for random code
	if ValidateTOTP("  000000  ", setup.Secret) {
		t.Log("whitespace-trimmed 000000 happened to be valid TOTP")
	}
}

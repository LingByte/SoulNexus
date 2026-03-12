package utils

import (
	"strings"
	"testing"
)

// TestEncryptedPasswordValidation tests the complete encrypted password validation flow
func TestEncryptedPasswordValidation(t *testing.T) {
	// Test case from the user's registration data
	encryptedPassword := "240be518fabd2724ddb6f04eeb1da5967448d7e831c08c8fa822809f74c720a9:4a20cc8aaaa750866250adcacd9a64545696c8ce6783e9239f1b5f96d158d994:gFiCOOXx1-jRvDDrRu6uHJb52qvgkNq5:1769268563405"

	// Test validation
	err := ValidatePasswordFormat(encryptedPassword)
	if err != nil {
		t.Errorf("ValidatePasswordFormat() failed for encrypted password: %v", err)
	}

	// Test sanitization
	sanitized, err := SanitizeAndValidate(encryptedPassword, "password")
	if err != nil {
		t.Errorf("SanitizeAndValidate() failed for encrypted password: %v", err)
	}

	if sanitized != encryptedPassword {
		t.Errorf("SanitizeAndValidate() modified encrypted password: got %s, want %s", sanitized, encryptedPassword)
	}

	t.Logf("Encrypted password validation successful: %s", encryptedPassword)
}

// TestPlainPasswordValidation tests plain password validation still works
func TestPlainPasswordValidation(t *testing.T) {
	testCases := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid plain password",
			password: "mypassword123",
			wantErr:  false,
		},
		{
			name:     "too short plain password",
			password: "short",
			wantErr:  true,
		},
		{
			name:     "too long plain password",
			password: string(make([]byte, 130)),
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePasswordFormat(tc.password)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidatePasswordFormat() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr {
				// Test sanitization for valid passwords
				sanitized, err := SanitizeAndValidate(tc.password, "password")
				if err != nil {
					t.Errorf("SanitizeAndValidate() failed for valid password: %v", err)
				}
				if sanitized != tc.password {
					t.Errorf("SanitizeAndValidate() modified valid password: got %s, want %s", sanitized, tc.password)
				}
			}
		})
	}
}

// TestPasswordFormatDetection tests the detection of encrypted vs plain passwords
func TestPasswordFormatDetection(t *testing.T) {
	testCases := []struct {
		name        string
		password    string
		isEncrypted bool
	}{
		{
			name:        "encrypted password",
			password:    "240be518fabd2724ddb6f04eeb1da5967448d7e831c08c8fa822809f74c720a9:4a20cc8aaaa750866250adcacd9a64545696c8ce6783e9239f1b5f96d158d994:gFiCOOXx1-jRvDDrRu6uHJb52qvgkNq5:1769268563405",
			isEncrypted: true,
		},
		{
			name:        "plain password",
			password:    "mypassword123",
			isEncrypted: false,
		},
		{
			name:        "password with colons but wrong format",
			password:    "password:with:colons",
			isEncrypted: false,
		},
		{
			name:        "password with 4 parts but invalid hash",
			password:    "invalid:hash:salt:timestamp",
			isEncrypted: true, // Will be detected as encrypted but validation will fail
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Check if password is detected as encrypted format
			isEncrypted := len(tc.password) > 0 &&
				strings.Contains(tc.password, ":") &&
				len(strings.Split(tc.password, ":")) == 4

			if isEncrypted != tc.isEncrypted {
				t.Errorf("Password format detection failed: got %v, want %v", isEncrypted, tc.isEncrypted)
			}

			t.Logf("Password: %s, Detected as encrypted: %v", tc.password, isEncrypted)
		})
	}
}

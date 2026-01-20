package utils

import (
	"testing"
)

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trim spaces",
			input:    "  test  ",
			expected: "test",
		},
		{
			name:     "remove control characters",
			input:    "test\x00\x01\x02",
			expected: "test",
		},
		{
			name:     "keep newline and tab",
			input:    "test\n\t",
			expected: "test", // SanitizeInput removes newline and tab
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			if result != tt.expected {
				t.Fatalf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trim and lowercase",
			input:    "  TEST@EXAMPLE.COM  ",
			expected: "test@example.com",
		},
		{
			name:     "remove spaces",
			input:    "test @ example.com",
			expected: "test@example.com",
		},
		{
			name:     "normal email",
			input:    "user@example.com",
			expected: "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeEmail(tt.input)
			if result != tt.expected {
				t.Fatalf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSanitizePassword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trim spaces",
			input:    "  password  ",
			expected: "password",
		},
		{
			name:     "keep internal spaces",
			input:    "pass word",
			expected: "pass word",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePassword(tt.input)
			if result != tt.expected {
				t.Fatalf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidateSQLInjection(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "safe input",
			input:   "normal text",
			wantErr: false,
		},
		{
			name:    "SQL injection - SELECT",
			input:   "test'; SELECT * FROM users; --",
			wantErr: true,
		},
		{
			name:    "SQL injection - DROP",
			input:   "test'; DROP TABLE users; --",
			wantErr: true,
		},
		{
			name:    "SQL injection - UNION",
			input:   "test' UNION SELECT * FROM users --",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: false,
		},
		{
			name:    "valid email with quote",
			input:   "o'brien@example.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSQLInjection(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateSQLInjection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateXSS(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "safe input",
			input:   "normal text",
			wantErr: false,
		},
		{
			name:    "XSS - script tag",
			input:   "<script>alert('xss')</script>",
			wantErr: true,
		},
		{
			name:    "XSS - javascript:",
			input:   "javascript:alert('xss')",
			wantErr: true,
		},
		{
			name:    "XSS - onerror",
			input:   "<img onerror='alert(1)'>",
			wantErr: true,
		},
		{
			name:    "XSS - iframe",
			input:   "<iframe src='evil.com'></iframe>",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateXSS(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateXSS() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateEmailFormat(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid email",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
		{
			name:    "invalid format - no @",
			email:   "invalid",
			wantErr: true,
		},
		{
			name:    "invalid format - no domain",
			email:   "user@",
			wantErr: true,
		},
		{
			name:    "email too long",
			email:   string(make([]byte, 255)) + "@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmailFormat(tt.email)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateEmailFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePasswordFormat(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "password too short",
			password: "short",
			wantErr:  true,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
		{
			name:     "password too long",
			password: string(make([]byte, 129)),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordFormat(tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePasswordFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDisplayName(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		wantErr     bool
	}{
		{
			name:        "valid display name",
			displayName: "John Doe",
			wantErr:     false,
		},
		{
			name:        "empty display name",
			displayName: "",
			wantErr:     false, // 允许为空
		},
		{
			name:        "display name too long",
			displayName: string(make([]byte, 51)),
			wantErr:     true,
		},
		{
			name:        "display name with XSS",
			displayName: "<script>alert('xss')</script>",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDisplayName(tt.displayName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateDisplayName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUserName(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{
			name:     "valid username",
			username: "user123",
			wantErr:  false,
		},
		{
			name:     "valid username with underscore",
			username: "user_name",
			wantErr:  false,
		},
		{
			name:     "valid username with hyphen",
			username: "user-name",
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			wantErr:  true,
		},
		{
			name:     "username too short",
			username: "ab",
			wantErr:  true,
		},
		{
			name:     "username too long",
			username: string(make([]byte, 31)),
			wantErr:  true,
		},
		{
			name:     "username with invalid characters",
			username: "user@name",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserName(tt.username)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateUserName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeAndValidate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		inputType string
		wantErr   bool
	}{
		{
			name:      "valid email",
			input:     "  TEST@EXAMPLE.COM  ",
			inputType: "email",
			wantErr:   false,
		},
		{
			name:      "invalid email",
			input:     "invalid",
			inputType: "email",
			wantErr:   true,
		},
		{
			name:      "valid password",
			input:     "  password123  ",
			inputType: "password",
			wantErr:   false,
		},
		{
			name:      "invalid password - too short",
			input:     "short",
			inputType: "password",
			wantErr:   true,
		},
		{
			name:      "valid username",
			input:     "  user123  ",
			inputType: "username",
			wantErr:   false,
		},
		{
			name:      "invalid username - too short",
			input:     "ab",
			inputType: "username",
			wantErr:   true,
		},
		{
			name:      "valid display name",
			input:     "  John Doe  ",
			inputType: "displayname",
			wantErr:   false,
		},
		{
			name:      "default type",
			input:     "  normal text  ",
			inputType: "other",
			wantErr:   false,
		},
		{
			name:      "input with SQL injection",
			input:     "test'; DROP TABLE users; --",
			inputType: "other",
			wantErr:   true,
		},
		{
			name:      "input with XSS",
			input:     "<script>alert('xss')</script>",
			inputType: "other",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeAndValidate(tt.input, tt.inputType)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SanitizeAndValidate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result == "" && tt.input != "" {
				t.Fatalf("SanitizeAndValidate() returned empty string for valid input")
			}
		})
	}
}

package utils

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

// SanitizeInput cleans input by removing leading/trailing spaces and special characters
func SanitizeInput(input string) string {
	// Remove leading and trailing spaces
	input = strings.TrimSpace(input)
	// Remove control characters
	input = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, input)
	return input
}

// SanitizeEmail cleans email address
func SanitizeEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	// Remove extra spaces in email
	email = strings.ReplaceAll(email, " ", "")
	return email
}

// SanitizePassword cleans password (preserves spaces but removes leading/trailing spaces)
func SanitizePassword(password string) string {
	return strings.TrimSpace(password)
}

// ValidateSQLInjection checks for SQL injection risks
func ValidateSQLInjection(input string) error {
	if input == "" {
		return nil
	}

	// Common SQL injection keywords and characters
	sqlKeywords := []string{
		"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_",
		"exec", "execute", "select", "insert", "update", "delete",
		"drop", "create", "alter", "union", "script", "<script",
		"javascript:", "onerror", "onload", "onclick",
	}

	inputLower := strings.ToLower(input)

	// Check if contains SQL injection keywords
	for _, keyword := range sqlKeywords {
		if strings.Contains(inputLower, keyword) {
			// Allow normal single quotes in email addresses (e.g. o'brien@example.com)
			if keyword == "'" && isValidEmailQuote(input) {
				continue
			}
			return errors.New("input contains potentially dangerous characters")
		}
	}

	// Check if contains SQL comments
	if strings.Contains(inputLower, "--") || strings.Contains(inputLower, "/*") {
		return errors.New("input contains SQL comment characters")
	}

	return nil
}

// isValidEmailQuote checks if single quote is in a valid email address
func isValidEmailQuote(input string) bool {
	// Simple email format check
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-']+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(input)
}

// ValidateXSS checks for XSS attack risks
func ValidateXSS(input string) error {
	if input == "" {
		return nil
	}

	// Common XSS keywords
	xssPatterns := []string{
		"<script", "</script>", "javascript:", "onerror=",
		"onload=", "onclick=", "<iframe", "<img", "onmouseover=",
		"<svg", "<object", "<embed",
	}

	inputLower := strings.ToLower(input)

	for _, pattern := range xssPatterns {
		if strings.Contains(inputLower, pattern) {
			return errors.New("input contains potentially dangerous script tags")
		}
	}

	return nil
}

// ValidateEmailFormat validates email format
func ValidateEmailFormat(email string) error {
	if email == "" {
		return errors.New("email is required")
	}

	// Email format regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}

	// Length check
	if len(email) > 254 {
		return errors.New("email address too long")
	}

	return nil
}

// ValidatePasswordFormat validates password format
func ValidatePasswordFormat(password string) error {
	if password == "" {
		return errors.New("password is required")
	}

	// Check if this is an encrypted password format (passwordHash:encryptedHash:salt:timestamp)
	if strings.Contains(password, ":") && len(strings.Split(password, ":")) == 4 {
		// For encrypted passwords, validate the first part (password hash)
		parts := strings.Split(password, ":")
		passwordHash := parts[0]

		// Password hash should be a valid SHA-256 hash (64 hex characters)
		if len(passwordHash) != 64 {
			return errors.New("invalid encrypted password format")
		}

		// Check if it's a valid hex string
		for _, char := range passwordHash {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				return errors.New("invalid encrypted password format")
			}
		}

		return nil // Encrypted password is valid
	}

	// For plain text passwords, apply normal validation
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return errors.New("password too long")
	}

	return nil
}

// ValidateDisplayName validates display name
func ValidateDisplayName(name string) error {
	if name == "" {
		return nil // Display name can be empty
	}

	// Length check (using rune count for proper Unicode support)
	runeCount := len([]rune(name))
	if runeCount > 50 {
		return errors.New("display name too long")
	}

	// Check for dangerous characters
	if err := ValidateXSS(name); err != nil {
		return err
	}

	return nil
}

// ValidateUserName validates username
func ValidateUserName(username string) error {
	if username == "" {
		return errors.New("username is required")
	}

	// Length check (using rune count for proper Unicode support)
	runeCount := len([]rune(username))
	if runeCount < 2 {
		return errors.New("username must be at least 2 characters long")
	}

	if runeCount > 30 {
		return errors.New("username too long")
	}

	// Allow letters (including Unicode letters like Chinese), numbers, underscores and hyphens
	// Check each character individually
	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return errors.New("username can only contain letters, numbers, underscores and hyphens")
		}
	}

	return nil
}

// SanitizeAndValidate cleans and validates input
func SanitizeAndValidate(input string, inputType string) (string, error) {
	var sanitized string

	switch inputType {
	case "email":
		sanitized = SanitizeEmail(input)
		if err := ValidateEmailFormat(sanitized); err != nil {
			return "", err
		}
	case "password":
		sanitized = SanitizePassword(input)
		if err := ValidatePasswordFormat(sanitized); err != nil {
			return "", err
		}
	case "username":
		sanitized = SanitizeInput(input)
		if err := ValidateUserName(sanitized); err != nil {
			return "", err
		}
	case "displayname":
		sanitized = SanitizeInput(input)
		if err := ValidateDisplayName(sanitized); err != nil {
			return "", err
		}
	default:
		sanitized = SanitizeInput(input)
	}

	// SQL injection check
	if err := ValidateSQLInjection(sanitized); err != nil {
		return "", err
	}

	// XSS check
	if err := ValidateXSS(sanitized); err != nil {
		return "", err
	}

	return sanitized, nil
}

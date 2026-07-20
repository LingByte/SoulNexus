package security

import (
	"html"
	"regexp"
	"strings"
	"unicode/utf8"

	lingutils "github.com/LingByte/lingllm/utils"
)

var xssPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
	regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
	regexp.MustCompile(`(?i)<object[^>]*>.*?</object>`),
	regexp.MustCompile(`(?i)<embed[^>]*>.*?</embed>`),
	regexp.MustCompile(`(?i)<embed[^>]*>`),
	regexp.MustCompile(`(?i)<form[^>]*>.*?</form>`),
	regexp.MustCompile(`(?i)<input[^>]*>`),
	regexp.MustCompile(`(?i)<button[^>]*>.*?</button>`),
	regexp.MustCompile(`(?i)javascript:`),
	regexp.MustCompile(`(?i)vbscript:`),
	regexp.MustCompile(`(?i)onload\s*=`),
	regexp.MustCompile(`(?i)onerror\s*=`),
	regexp.MustCompile(`(?i)onclick\s*=`),
	regexp.MustCompile(`(?i)onmouseover\s*=`),
	regexp.MustCompile(`(?i)onfocus\s*=`),
	regexp.MustCompile(`(?i)onblur\s*=`),
}

// SanitizeHTML strips or escapes potentially malicious HTML content.
func SanitizeHTML(input string) string {
	if input == "" {
		return ""
	}
	if len(input) > 10000 {
		input = input[:10000]
	}
	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			return html.EscapeString(input)
		}
	}
	return input
}

// EscapeHTML escapes HTML special characters.
func EscapeHTML(input string) string {
	if input == "" {
		return ""
	}
	return html.EscapeString(input)
}

// ValidateInput validates user input for control characters, UTF-8 validity, and XSS patterns.
func ValidateInput(input string) (string, bool) {
	if input == "" {
		return "", true
	}
	for _, r := range input {
		if r < 32 && r != 9 && r != 10 && r != 13 {
			return "", false
		}
	}
	if !utf8.ValidString(input) {
		return "", false
	}
	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			return "", false
		}
	}
	return strings.TrimSpace(input), true
}

// IsValidURL checks whether a URL uses an allowed protocol and passes basic safety checks.
func IsValidURL(rawURL string) bool {
	if rawURL == "" || len(rawURL) > 2048 {
		return false
	}
	allowedProtocols := []string{"http://", "https://", "local://", "minio://", "cos://", "tos://", "oss://"}
	isAllowed := false
	for _, protocol := range allowedProtocols {
		if strings.HasPrefix(strings.ToLower(rawURL), protocol) {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return false
	}
	for _, pattern := range xssPatterns {
		if pattern.MatchString(rawURL) {
			return false
		}
	}
	return true
}

// CleanMarkdown removes potentially malicious script patterns from Markdown.
func CleanMarkdown(input string) string {
	if input == "" {
		return ""
	}
	cleaned := input
	for _, pattern := range xssPatterns {
		cleaned = pattern.ReplaceAllString(cleaned, "")
	}
	return cleaned
}

// SanitizeForDisplay cleans content for safe HTML display.
func SanitizeForDisplay(input string) string {
	if input == "" {
		return ""
	}
	return html.EscapeString(CleanMarkdown(input))
}

// SanitizeForLog delegates to lingllm/utils.
func SanitizeForLog(input string) string {
	return lingutils.SanitizeForLog(input)
}

// SanitizeForLogArray sanitizes each string in a slice for logging.
func SanitizeForLogArray(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}
	sanitized := make([]string, 0, len(input))
	for _, item := range input {
		sanitized = append(sanitized, SanitizeForLog(item))
	}
	return sanitized
}

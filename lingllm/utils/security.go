package utils

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// AllowedStdioCommands is the whitelist for MCP stdio transport launchers.
var AllowedStdioCommands = map[string]bool{
	"uvx": true,
	"npx": true,
}

var dangerousArgPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^-c$`),
	regexp.MustCompile(`(?i)^--command$`),
	regexp.MustCompile(`(?i)^-e$`),
	regexp.MustCompile(`(?i)^--eval$`),
	regexp.MustCompile(`(?i)[;&|]`),
	regexp.MustCompile(`(?i)\$\(`),
	regexp.MustCompile("(?i)`"),
	regexp.MustCompile(`(?i)>\s*[/~]`),
	regexp.MustCompile(`(?i)<\s*[/~]`),
	regexp.MustCompile(`(?i)^/bin/`),
	regexp.MustCompile(`(?i)^/usr/bin/`),
	regexp.MustCompile(`(?i)^/sbin/`),
	regexp.MustCompile(`(?i)^/usr/sbin/`),
	regexp.MustCompile(`(?i)^\.\./`),
	regexp.MustCompile(`(?i)/\.\./`),
	regexp.MustCompile(`(?i)^(bash|sh|zsh|ksh|csh|tcsh|fish|dash)$`),
	regexp.MustCompile(`(?i)^(curl|wget|nc|netcat|ncat)$`),
	regexp.MustCompile(`(?i)^(rm|dd|mkfs|fdisk)$`),
}

var dangerousEnvVarPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^LD_PRELOAD$`),
	regexp.MustCompile(`(?i)^LD_LIBRARY_PATH$`),
	regexp.MustCompile(`(?i)^DYLD_`),
	regexp.MustCompile(`(?i)^PATH$`),
	regexp.MustCompile(`(?i)^PYTHONPATH$`),
	regexp.MustCompile(`(?i)^NODE_OPTIONS$`),
	regexp.MustCompile(`(?i)^BASH_ENV$`),
	regexp.MustCompile(`(?i)^ENV$`),
	regexp.MustCompile(`(?i)^SHELL$`),
}

// SafePathUnderBase returns absPath when filePath is under baseDir.
func SafePathUnderBase(baseDir, filePath string) (string, error) {
	if baseDir == "" || filePath == "" {
		return "", fmt.Errorf("baseDir and filePath cannot be empty")
	}
	absBase, err := filepath.Abs(filepath.Clean(baseDir))
	if err != nil {
		return "", fmt.Errorf("invalid base dir: %w", err)
	}
	absPath, err := filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}
	sep := string(filepath.Separator)
	if absPath != absBase && !strings.HasPrefix(absPath, absBase+sep) {
		return "", fmt.Errorf("path traversal denied: path is outside base directory")
	}
	return absPath, nil
}

// SafeFileName returns a safe basename-only filename.
func SafeFileName(fileName string) (string, error) {
	if fileName == "" {
		return "", fmt.Errorf("fileName cannot be empty")
	}
	base := filepath.Base(filepath.Clean(fileName))
	if base == "" || base == "." || base == ".." {
		return "", fmt.Errorf("invalid fileName: path traversal or empty name")
	}
	if strings.Contains(base, "..") {
		return "", fmt.Errorf("invalid fileName: contains path traversal")
	}
	if len(base) > 255 {
		return "", fmt.Errorf("fileName too long")
	}
	return base, nil
}

// SafeObjectKey validates an object storage key.
func SafeObjectKey(objectKey string) error {
	if objectKey == "" {
		return fmt.Errorf("object key cannot be empty")
	}
	if strings.Contains(objectKey, "..") {
		return fmt.Errorf("object key contains path traversal")
	}
	return nil
}

// SanitizeForLog removes control characters and newlines to prevent log injection.
func SanitizeForLog(input string) string {
	if input == "" {
		return ""
	}
	sanitized := strings.ReplaceAll(input, "\n", " ")
	sanitized = strings.ReplaceAll(sanitized, "\r", " ")
	sanitized = strings.ReplaceAll(sanitized, "\t", " ")
	var builder strings.Builder
	for _, r := range sanitized {
		if r >= 32 || r == ' ' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// IsDangerousEnvVar reports whether an environment variable name is unsafe for subprocesses.
func IsDangerousEnvVar(name string) bool {
	for _, pattern := range dangerousEnvVarPatterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

// FilterSafeEnvVars returns env entries with dangerous names removed.
func FilterSafeEnvVars(extra map[string]string) map[string]string {
	if len(extra) == 0 {
		return nil
	}
	out := make(map[string]string, len(extra))
	for key, value := range extra {
		if IsDangerousEnvVar(key) {
			continue
		}
		out[key] = value
	}
	return out
}

// ValidateStdioCommand validates an MCP stdio launcher command.
func ValidateStdioCommand(command string) error {
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}
	baseCommand := command
	if strings.Contains(command, "/") {
		parts := strings.Split(command, "/")
		baseCommand = parts[len(parts)-1]
	}
	if !AllowedStdioCommands[baseCommand] {
		return fmt.Errorf("command '%s' is not in the allowed list (uvx, npx)", baseCommand)
	}
	if strings.Contains(command, "..") {
		return fmt.Errorf("command path contains invalid characters")
	}
	return nil
}

// ValidateStdioArgs validates MCP stdio arguments.
func ValidateStdioArgs(args []string) error {
	if len(args) == 0 {
		return nil
	}
	for i, arg := range args {
		if len(arg) > 1024 {
			return fmt.Errorf("argument %d exceeds maximum length (1024 characters)", i)
		}
		for _, pattern := range dangerousArgPatterns {
			if pattern.MatchString(arg) {
				return fmt.Errorf("argument %d contains potentially dangerous pattern: %s", i, SanitizeForLog(arg))
			}
		}
		if strings.Contains(arg, "\x00") {
			return fmt.Errorf("argument %d contains null bytes", i)
		}
	}
	return nil
}

// ValidateStdioEnvVars validates MCP stdio environment variables.
func ValidateStdioEnvVars(envVars map[string]string) error {
	if len(envVars) == 0 {
		return nil
	}
	for key, value := range envVars {
		if IsDangerousEnvVar(key) {
			return fmt.Errorf("environment variable '%s' is not allowed for security reasons", key)
		}
		if len(key) > 256 {
			return fmt.Errorf("environment variable name '%s' exceeds maximum length", SanitizeForLog(key[:50]))
		}
		if len(value) > 4096 {
			return fmt.Errorf("environment variable '%s' value exceeds maximum length", key)
		}
		if strings.Contains(value, "\x00") {
			return fmt.Errorf("environment variable '%s' value contains null bytes", key)
		}
		for _, pattern := range dangerousArgPatterns {
			if pattern.MatchString(value) {
				return fmt.Errorf("environment variable '%s' value contains potentially dangerous pattern", key)
			}
		}
	}
	return nil
}

// ValidateStdioConfig validates MCP stdio transport configuration.
func ValidateStdioConfig(command string, args []string, envVars map[string]string) error {
	if err := ValidateStdioCommand(command); err != nil {
		return fmt.Errorf("invalid command: %w", err)
	}
	if err := ValidateStdioArgs(args); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	if err := ValidateStdioEnvVars(envVars); err != nil {
		return fmt.Errorf("invalid environment variables: %w", err)
	}
	return nil
}

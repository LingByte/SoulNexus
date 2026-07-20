package asr

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// SensitiveFilterConfig contains configuration for sensitive filter component.
type SensitiveFilterConfig struct {
	// Blacklist: words/patterns to filter out
	Blacklist []string
	// Whitelist: words/patterns to allow (takes precedence over blacklist)
	Whitelist []string
	// FilterEmoji: whether to filter out emoji characters (default: true)
	FilterEmoji bool
	// CaseSensitive: whether filtering is case-sensitive (default: false)
	CaseSensitive bool
	// ReplaceWith: character to replace filtered content with (default: "*")
	ReplaceWith string
}

// DefaultSensitiveFilterConfig returns the default sensitive filter configuration.
func DefaultSensitiveFilterConfig() SensitiveFilterConfig {
	return SensitiveFilterConfig{
		Blacklist:     []string{},
		Whitelist:     []string{},
		FilterEmoji:   true,
		CaseSensitive: false,
		ReplaceWith:   "*",
	}
}

// SensitiveFilterComponent filters sensitive information from recognized text.
// It supports:
// - Blacklist/whitelist filtering
// - Emoji filtering
// - Case-insensitive matching
type SensitiveFilterComponent struct {
	mu                sync.RWMutex
	blacklistPatterns []*regexp.Regexp
	whitelistPatterns []*regexp.Regexp
	filterEmoji       bool
	caseSensitive     bool
	replaceWith       string
}

// NewSensitiveFilterComponent creates a new sensitive filter component with the given configuration.
func NewSensitiveFilterComponent(config SensitiveFilterConfig) (*SensitiveFilterComponent, error) {
	if config.ReplaceWith == "" {
		config.ReplaceWith = "*"
	}

	component := &SensitiveFilterComponent{
		filterEmoji:   config.FilterEmoji,
		caseSensitive: config.CaseSensitive,
		replaceWith:   config.ReplaceWith,
	}

	// Compile blacklist patterns
	for _, pattern := range config.Blacklist {
		if pattern == "" {
			continue
		}

		regexPattern := pattern
		if !config.CaseSensitive {
			regexPattern = "(?i)" + regexPattern
		}

		regex, err := regexp.Compile(regexPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid blacklist pattern %q: %w", pattern, err)
		}

		component.blacklistPatterns = append(component.blacklistPatterns, regex)
	}

	// Compile whitelist patterns
	for _, pattern := range config.Whitelist {
		if pattern == "" {
			continue
		}

		regexPattern := pattern
		if !config.CaseSensitive {
			regexPattern = "(?i)" + regexPattern
		}

		regex, err := regexp.Compile(regexPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid whitelist pattern %q: %w", pattern, err)
		}

		component.whitelistPatterns = append(component.whitelistPatterns, regex)
	}

	return component, nil
}

// Name returns the component name.
func (s *SensitiveFilterComponent) Name() string {
	return "sensitive_filter"
}

// Process filters sensitive information from text.
// Returns (filteredText, shouldContinue, error)
func (s *SensitiveFilterComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	text, ok := data.(string)
	if !ok {
		return nil, false, fmt.Errorf("%w: expected string, got %T", ErrInvalidDataType, data)
	}

	if text == "" {
		return text, true, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := text

	// Filter emoji if enabled
	if s.filterEmoji {
		result = s.filterEmojis(result)
	}

	// Apply blacklist filtering
	// We need to process matches in reverse order to maintain correct indices
	for _, pattern := range s.blacklistPatterns {
		matches := pattern.FindAllStringIndex(result, -1)
		if len(matches) == 0 {
			continue
		}

		// Process matches in reverse order to maintain indices
		for i := len(matches) - 1; i >= 0; i-- {
			match := matches[i]
			matchedText := result[match[0]:match[1]]

			// Check if this match is whitelisted
			isWhitelisted := false
			for _, whitelistPattern := range s.whitelistPatterns {
				if whitelistPattern.MatchString(matchedText) {
					isWhitelisted = true
					break
				}
			}

			// If not whitelisted, replace it
			if !isWhitelisted {
				replacement := strings.Repeat(s.replaceWith, len(matchedText))
				result = result[:match[0]] + replacement + result[match[1]:]
			}
		}
	}

	return result, true, nil
}

// filterEmojis removes emoji characters from text.
func (s *SensitiveFilterComponent) filterEmojis(text string) string {
	var result strings.Builder

	for _, r := range text {
		// Check if the rune is an emoji
		// Emoji ranges:
		// - 0x1F300-0x1F9FF: Miscellaneous Symbols and Pictographs, Emoticons, Transport, etc.
		// - 0x2600-0x27BF: Miscellaneous Symbols
		// - 0x2300-0x23FF: Miscellaneous Technical
		// - 0x2000-0x206F: General Punctuation (some emoji)
		if isEmoji(r) {
			result.WriteString(s.replaceWith)
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// isEmoji checks if a rune is an emoji character.
func isEmoji(r rune) bool {
	// Check common emoji ranges
	if r >= 0x1F300 && r <= 0x1F9FF {
		return true
	}
	if r >= 0x2600 && r <= 0x27BF {
		return true
	}
	if r >= 0x2300 && r <= 0x23FF {
		return true
	}
	if r >= 0x2000 && r <= 0x206F {
		return true
	}
	if r >= 0x1F000 && r <= 0x1F02F {
		return true
	}
	if r >= 0x1F080 && r <= 0x1F18F {
		return true
	}
	if r >= 0x1F900 && r <= 0x1F9FF {
		return true
	}
	// Variation selectors for emoji
	if r == 0xFE0F || r == 0xFE0E {
		return true
	}
	// Zero-width joiner
	if r == 0x200D {
		return true
	}

	return false
}

// SetBlacklist updates the blacklist patterns.
func (s *SensitiveFilterComponent) SetBlacklist(patterns []string) error {
	var blacklistPatterns []*regexp.Regexp

	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}

		regexPattern := pattern
		if !s.caseSensitive {
			regexPattern = "(?i)" + regexPattern
		}

		regex, err := regexp.Compile(regexPattern)
		if err != nil {
			return fmt.Errorf("invalid blacklist pattern %q: %w", pattern, err)
		}

		blacklistPatterns = append(blacklistPatterns, regex)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.blacklistPatterns = blacklistPatterns

	return nil
}

// SetWhitelist updates the whitelist patterns.
func (s *SensitiveFilterComponent) SetWhitelist(patterns []string) error {
	var whitelistPatterns []*regexp.Regexp

	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}

		regexPattern := pattern
		if !s.caseSensitive {
			regexPattern = "(?i)" + regexPattern
		}

		regex, err := regexp.Compile(regexPattern)
		if err != nil {
			return fmt.Errorf("invalid whitelist pattern %q: %w", pattern, err)
		}

		whitelistPatterns = append(whitelistPatterns, regex)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.whitelistPatterns = whitelistPatterns

	return nil
}

// SetFilterEmoji enables or disables emoji filtering.
func (s *SensitiveFilterComponent) SetFilterEmoji(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filterEmoji = enabled
}

// SetReplaceWith sets the replacement character.
func (s *SensitiveFilterComponent) SetReplaceWith(replacement string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replaceWith = replacement
}

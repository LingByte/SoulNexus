package coerce

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils/validate"
)

// ParseCommaList splits a comma-separated config value into trimmed non-empty tokens.
func ParseCommaList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// FirstNonEmpty returns the first non-blank trimmed value.
func FirstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// ParseCommaSeparatedIDs parses comma-separated decimal IDs.
func ParseCommaSeparatedIDs(raw string) ([]uint, error) {
	parts := ParseCommaList(raw)
	if len(parts) == 0 {
		return nil, nil
	}
	return validate.ParseIDStrings(parts)
}

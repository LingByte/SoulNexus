package common

// TruncateRunes returns s trimmed to at most max runes, with an ellipsis when truncated.
// max <= 0 returns s unchanged.
func TruncateRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

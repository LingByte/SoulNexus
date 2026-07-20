package realtime

import "strings"

func IsBenignOmniError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "none active response") ||
		strings.Contains(lower, "no active response") ||
		strings.Contains(lower, "already has an active response")
}

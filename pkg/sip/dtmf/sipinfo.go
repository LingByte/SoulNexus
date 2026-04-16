package dtmf

import (
	"strconv"
	"strings"
)

// DigitFromSIPINFO extracts a DTMF digit from SIP INFO body (Linphone / many UAs use this instead of RTP).
// Supports:
//   - application/dtmf-relay: "Signal=X\r\n" where X is 0-9, *, #
//   - application/dtmf: "Signal=X"
// If the body contains Signal=, it is parsed even when Content-Type is missing or non-dtmf.
func DigitFromSIPINFO(contentType, body string) (digit string, ok bool) {
	_ = contentType
	if !strings.Contains(strings.ToLower(body), "signal=") {
		return "", false
	}
	body = strings.ReplaceAll(body, "\r\n", "\n")
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Signal=5 or Signal=# 
		if strings.HasPrefix(strings.ToLower(line), "signal=") {
			v := strings.TrimSpace(line[7:])
			v = strings.Trim(v, "\"'")
			return normalizeSignal(v)
		}
	}
	return "", false
}

func normalizeSignal(v string) (string, bool) {
	if v == "" {
		return "", false
	}
	if len(v) == 1 {
		switch v {
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "*", "#", "A", "B", "C", "D":
			return v, true
		}
	}
	// Some stacks send numeric event codes
	if n, err := strconv.Atoi(v); err == nil {
		if n >= 0 && n <= 9 {
			return string(rune('0' + n)), true
		}
		if n == 10 {
			return "*", true
		}
		if n == 11 {
			return "#", true
		}
	}
	return "", false
}

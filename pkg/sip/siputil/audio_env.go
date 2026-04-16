package siputil

import (
	"os"
	"strconv"
	"strings"
)

const envWelcomeBargeInThreshold = "SIP_WELCOME_BARGE_IN_THRESHOLD"

// WelcomeBargeInThresholdFromEnv returns the RMS threshold used for welcome-prompt barge-in.
// Default is tuned to be slightly more sensitive than a typical VAD gate.
func WelcomeBargeInThresholdFromEnv() float64 {
	s := strings.TrimSpace(os.Getenv(envWelcomeBargeInThreshold))
	if s == "" {
		return 1800.0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 {
		return 1800.0
	}
	return f
}

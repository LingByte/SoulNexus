package mail

import "strings"

// mailStatusRank orders lifecycle states for webhook updates (higher = more advanced).
// Terminal failure states use negative ranks so they always override success states.
func mailStatusRank(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case StatusSent:
		return 1
	case StatusDelivered:
		return 2
	case StatusOpened:
		return 3
	case StatusClicked:
		return 4
	case StatusUnsubscribed:
		return 5
	case StatusUnknown:
		return 0
	default:
		return 0
	}
}

func isTerminalFailureStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case StatusFailed, StatusSoftBounce, StatusInvalid, StatusSpam:
		return true
	default:
		return false
	}
}

// ResolveMailLogStatusTransition decides whether to apply an incoming status to a mail log row.
func ResolveMailLogStatusTransition(current, incoming string) (next string, apply bool) {
	current = strings.TrimSpace(current)
	incoming = strings.TrimSpace(incoming)
	if incoming == "" {
		return current, false
	}
	if current == incoming {
		return current, false
	}
	if isTerminalFailureStatus(incoming) {
		return incoming, true
	}
	if isTerminalFailureStatus(current) {
		return current, false
	}
	if mailStatusRank(incoming) > mailStatusRank(current) {
		return incoming, true
	}
	return current, false
}

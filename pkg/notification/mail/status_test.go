package mail

import (
	"testing"
)

func TestMailStatusRank(t *testing.T) {
	tests := []struct {
		status string
		want   int
	}{
		{StatusSent, 1},
		{StatusDelivered, 2},
		{StatusOpened, 3},
		{StatusClicked, 4},
		{StatusUnsubscribed, 5},
		{StatusUnknown, 0},
		{"unknown-status", 0},
		{"  sent  ", 1},
		{"DELIVERED", 2},
		{"", 0},
	}
	for _, tt := range tests {
		got := mailStatusRank(tt.status)
		if got != tt.want {
			t.Errorf("mailStatusRank(%q) = %d, want %d", tt.status, got, tt.want)
		}
	}
}

func TestIsTerminalFailureStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{StatusFailed, true},
		{StatusSoftBounce, true},
		{StatusInvalid, true},
		{StatusSpam, true},
		{"  failed  ", true},
		{"SOFT_BOUNCE", true},
		{StatusSent, false},
		{StatusDelivered, false},
		{StatusOpened, false},
		{StatusClicked, false},
		{StatusUnsubscribed, false},
		{"", false},
		{"garbage", false},
	}
	for _, tt := range tests {
		got := isTerminalFailureStatus(tt.status)
		if got != tt.want {
			t.Errorf("isTerminalFailureStatus(%q) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestResolveMailLogStatusTransition(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		incoming string
		wantNext string
		wantAppl bool
	}{
		// empty incoming should be no-op
		{"empty incoming", StatusSent, "", StatusSent, false},

		// same status is no-op
		{"same status", StatusDelivered, StatusDelivered, StatusDelivered, false},

		// terminal failure always overrides
		{"failed overrides delivered", StatusDelivered, StatusFailed, StatusFailed, true},
		{"failed overrides opened", StatusOpened, StatusSoftBounce, StatusSoftBounce, true},
		{"spam overrides sent", StatusSent, StatusSpam, StatusSpam, true},

		// terminal failure is sticky — non-failure incoming cannot replace it
		{"failed is sticky", StatusFailed, StatusDelivered, StatusFailed, false},
		{"soft_bounce is sticky", StatusSoftBounce, StatusOpened, StatusSoftBounce, false},
		{"invalid is sticky", StatusInvalid, StatusClicked, StatusInvalid, false},

		// normal progression: only higher rank wins
		{"sent -> delivered", StatusSent, StatusDelivered, StatusDelivered, true},
		{"delivered -> opened", StatusDelivered, StatusOpened, StatusOpened, true},
		{"opened -> clicked", StatusOpened, StatusClicked, StatusClicked, true},
		{"clicked -> unsubscribed", StatusClicked, StatusUnsubscribed, StatusUnsubscribed, true},

		// reverse progression should be rejected
		{"delivered -> sent", StatusDelivered, StatusSent, StatusDelivered, false},
		{"opened -> delivered", StatusOpened, StatusDelivered, StatusOpened, false},
		{"clicked -> opened", StatusClicked, StatusOpened, StatusClicked, false},
		{"unsubscribed -> clicked", StatusUnsubscribed, StatusClicked, StatusUnsubscribed, false},

		// unknown status has rank 0, so it cannot override anything
		{"unknown cannot override sent", StatusSent, StatusUnknown, StatusSent, false},
		{"unknown cannot override delivered", StatusDelivered, "garbage", StatusDelivered, false},
	}
	for _, tt := range tests {
		gotNext, gotAppl := ResolveMailLogStatusTransition(tt.current, tt.incoming)
		if gotNext != tt.wantNext || gotAppl != tt.wantAppl {
			t.Errorf("%s: ResolveMailLogStatusTransition(%q, %q) = (%q, %v), want (%q, %v)",
				tt.name, tt.current, tt.incoming, gotNext, gotAppl, tt.wantNext, tt.wantAppl)
		}
	}
}

func TestResolveMailLogStatusTransition_whitespace(t *testing.T) {
	// leading/trailing whitespace in current is trimmed
	next, apply := ResolveMailLogStatusTransition("  sent  ", "delivered")
	if next != StatusDelivered || !apply {
		t.Errorf("whitespace current: got (%q, %v), want (%q, true)", next, apply, StatusDelivered)
	}
	// whitespace in incoming is trimmed
	next, apply = ResolveMailLogStatusTransition(StatusDelivered, "  opened  ")
	if next != StatusOpened || !apply {
		t.Errorf("whitespace incoming: got (%q, %v), want (%q, true)", next, apply, StatusOpened)
	}
}

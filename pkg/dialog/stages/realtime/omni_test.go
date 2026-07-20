package realtime

import (
	"errors"
	"testing"
)

func TestIsBenignRealtimeOmniError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("other"), false},
		{errors.New("aliyunomni: server error: Conversation has none active response"), true},
		{errors.New("no active response to cancel"), true},
		{errors.New("aliyunomni: server error: Conversation already has an active response"), true},
	}
	for _, tc := range cases {
		if got := IsBenignOmniError(tc.err); got != tc.want {
			t.Fatalf("IsBenignOmniError(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

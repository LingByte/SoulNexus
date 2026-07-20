package common

import "testing"

func TestSessionTimeoutNormalize(t *testing.T) {
	if SessionIdleTimeout(0) != DefaultSessionIdleTimeoutHours {
		t.Fatal("idle default")
	}
	if SessionMaxLifetime(12, 0) != DefaultSessionMaxLifetimeHours {
		t.Fatal("max default")
	}
	if SessionMaxLifetime(12, 6) != 12 {
		t.Fatal("max should not be less than idle")
	}
}

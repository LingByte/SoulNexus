package knowledge

import (
	"testing"
	"time"
)

func TestFormatSearchBlock_TimedOutSkipsInjection(t *testing.T) {
	got := FormatSearchBlock(`{"ok":true,"timedOut":true,"hitCount":0,"results":[]}`)
	if got != "" {
		t.Fatalf("timedOut block = %q, want empty", got)
	}
}

func TestSetSearchTimeout(t *testing.T) {
	prev := searchTimeoutDuration()
	t.Cleanup(func() { SetSearchTimeout(prev) })

	SetSearchTimeout(0)
	if d := searchTimeoutDuration(); d != defaultSearchTimeout {
		t.Fatalf("zero timeout = %v, want default %v", d, defaultSearchTimeout)
	}
	SetSearchTimeout(5 * time.Second)
	if d := searchTimeoutDuration(); d != 5*time.Second {
		t.Fatalf("timeout = %v, want 5s", d)
	}
}

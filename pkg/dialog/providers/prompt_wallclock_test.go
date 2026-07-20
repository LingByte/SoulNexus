package providers

import (
	"strings"
	"testing"
	"time"
)

func TestWallClockPromptHint_ContainsTodayAndTomorrow(t *testing.T) {
	now := time.Date(2026, 7, 18, 23, 17, 0, 0, time.FixedZone("CST", 8*3600))
	hint := WallClockPromptHint(now)
	if !strings.Contains(hint, "今天=2026-07-18") {
		t.Fatalf("missing today: %s", hint)
	}
	if !strings.Contains(hint, "明天=2026-07-19") {
		t.Fatalf("missing tomorrow: %s", hint)
	}
	if !strings.Contains(hint, "23:17") {
		t.Fatalf("missing clock: %s", hint)
	}
}

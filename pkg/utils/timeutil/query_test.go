package timeutil

import "testing"

func TestParseQueryTime(t *testing.T) {
	Init("Asia/Shanghai")
	cases := []struct {
		raw string
		ok  bool
	}{
		{"", false},
		{"2026-01-15", true},
		{"2026-01-15 10:20:30", true},
		{"2026-01-15T10:20:30+08:00", true},
		{"bad", false},
	}
	for _, tc := range cases {
		_, ok := ParseQueryTime(tc.raw)
		if ok != tc.ok {
			t.Fatalf("ParseQueryTime(%q) ok=%v want %v", tc.raw, ok, tc.ok)
		}
	}
}

func TestParseQueryDay(t *testing.T) {
	Init("Asia/Shanghai")
	tm, ok := ParseQueryDay("2026-06-28")
	if !ok || tm.Year() != 2026 || tm.Month() != 6 || tm.Day() != 28 {
		t.Fatalf("ParseQueryDay got %v ok=%v", tm, ok)
	}
}

func TestDefaultRollingDateRange(t *testing.T) {
	Init("Asia/Shanghai")
	from, to := DefaultRollingDateRange("2026-01-01", "2026-01-10", 7)
	if from.Format(queryDayLayout) != "2026-01-01" {
		t.Fatalf("from=%v", from)
	}
	if to.Format(queryDayLayout) != "2026-01-11" {
		t.Fatalf("to=%v", to)
	}
}

func TestSlidingWindowNow(t *testing.T) {
	from, to := SlidingWindowNow(10, 30, 365)
	if !to.After(from) {
		t.Fatalf("from=%v to=%v", from, to)
	}
}

func TestParseOptionalRFC3339(t *testing.T) {
	got, err := ParseOptionalRFC3339(nil)
	if err != nil || got != nil {
		t.Fatalf("nil: got=%v err=%v", got, err)
	}
	s := "2026-01-01T00:00:00Z"
	got, err = ParseOptionalRFC3339(&s)
	if err != nil || got == nil {
		t.Fatalf("valid: got=%v err=%v", got, err)
	}
}

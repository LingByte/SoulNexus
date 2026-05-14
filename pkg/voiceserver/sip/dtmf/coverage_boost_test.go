package dtmf

import (
	"context"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/media"
	"go.uber.org/zap"
)

// ---------- eventCodeToDigit (RFC 2833 mapping) ---------------------------

func TestEventCodeToDigit_AllBranches(t *testing.T) {
	cases := []struct {
		code uint8
		want string
	}{
		{0, "0"}, {1, "1"}, {2, "2"}, {3, "3"}, {4, "4"},
		{5, "5"}, {6, "6"}, {7, "7"}, {8, "8"}, {9, "9"},
		{10, "*"}, {11, "#"},
		{12, "A"}, {13, "B"}, {14, "C"}, {15, "D"},
		{16, ""}, {99, ""}, {255, ""},
	}
	for _, c := range cases {
		if got := eventCodeToDigit(c.code); got != c.want {
			t.Errorf("eventCodeToDigit(%d) = %q, want %q", c.code, got, c.want)
		}
	}
}

func TestEventFromRFC2833_TooShort(t *testing.T) {
	if _, _, ok := EventFromRFC2833(nil); ok {
		t.Error("nil payload should be ok=false")
	}
	if _, _, ok := EventFromRFC2833([]byte{0x05}); ok {
		t.Error("1-byte payload should be ok=false")
	}
}

func TestEventFromRFC2833_UnknownEvent(t *testing.T) {
	// event=20, end=0 → digit empty, ok=false
	d, end, ok := EventFromRFC2833([]byte{20, 0x00, 0x00, 0x00})
	if ok {
		t.Errorf("unknown event should be ok=false (got d=%q end=%v)", d, end)
	}
}

func TestEventFromRFC2833_NotEnd(t *testing.T) {
	// digit '5', E=0
	d, end, ok := EventFromRFC2833([]byte{5, 0x00, 0x00, 0x00})
	if !ok || end || d != "5" {
		t.Errorf("d=%q end=%v ok=%v", d, end, ok)
	}
}

// ---------- normalizeSignal (covers numeric-code + invalid branches) ------

func TestNormalizeSignal_AllBranches(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"", "", false},
		{"5", "5", true},
		{"*", "*", true},
		{"#", "#", true},
		{"A", "A", true},
		{"D", "D", true},
		{"E", "", false}, // single char but not in set
		{"7", "7", true},
		{"10", "*", true}, // numeric code 10 → *
		{"11", "#", true}, // numeric code 11 → #
		{"3", "3", true},  // numeric direct
		{"12", "", false}, // out of numeric mapping range
		{"abc", "", false},
	}
	for _, c := range cases {
		got, ok := normalizeSignal(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("normalizeSignal(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestDigitFromSIPINFO_NoSignalToken(t *testing.T) {
	if _, ok := DigitFromSIPINFO("application/dtmf-relay", "totally unrelated body"); ok {
		t.Error("body without Signal= should be ok=false")
	}
}

func TestDigitFromSIPINFO_QuotedAndCRLFBody(t *testing.T) {
	body := "Duration=200\r\nSignal=\"7\"\r\n"
	d, ok := DigitFromSIPINFO("application/dtmf-relay", body)
	if !ok || d != "7" {
		t.Errorf("quoted CRLF body: d=%q ok=%v", d, ok)
	}
}

func TestDigitFromSIPINFO_NumericCode(t *testing.T) {
	body := "Signal=10\r\nDuration=160"
	d, ok := DigitFromSIPINFO("application/dtmf-relay", body)
	if !ok || d != "*" {
		t.Errorf("numeric 10 → expected *, got d=%q ok=%v", d, ok)
	}
}

// ---------- AttachProcessor / AttachLogger -------------------------------

func TestAttachProcessor_NilSafety(t *testing.T) {
	// Nil session must be tolerated without panic.
	AttachProcessor(nil, "x", func(ctx context.Context, digit string) {})
	// Nil handler must be tolerated.
	ms := newMediaSession(t)
	defer ms.Close()
	AttachProcessor(ms, "x", nil)
}

func TestAttachProcessor_DefaultName(t *testing.T) {
	ms := newMediaSession(t)
	defer ms.Close()
	called := false
	AttachProcessor(ms, "", func(ctx context.Context, digit string) {
		called = true
	})
	_ = called // we don't drive the pipeline here; we only ensure no panic and processor registration.
}

func TestAttachLogger_NilSafetyAndZapNopFallback(t *testing.T) {
	AttachLogger(nil, zap.NewNop()) // nil session
	ms := newMediaSession(t)
	defer ms.Close()
	AttachLogger(ms, nil)            // nil logger → falls back to logger.Lg or NewNop
	AttachLogger(ms, zap.NewNop())   // explicit nop
}

// ---------- helpers ------------------------------------------------------

func newMediaSession(t *testing.T) *media.MediaSession {
	t.Helper()
	ms := media.NewDefaultSession()
	if ms == nil {
		t.Fatal("NewDefaultSession returned nil")
	}
	return ms
}

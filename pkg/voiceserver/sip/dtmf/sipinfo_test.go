package dtmf

import "testing"

func TestDigitFromSIPINFO_DTMFRelay(t *testing.T) {
	body := "Signal=0\r\nDuration=200"
	d, ok := DigitFromSIPINFO("application/dtmf-relay", body)
	if !ok || d != "0" {
		t.Fatalf("got %q ok=%v", d, ok)
	}
}

func TestDigitFromSIPINFO_Hash(t *testing.T) {
	d, ok := DigitFromSIPINFO("application/dtmf-relay", "Signal=#\n")
	if !ok || d != "#" {
		t.Fatalf("got %q ok=%v", d, ok)
	}
}

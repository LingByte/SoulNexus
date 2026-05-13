package dtmf

import "testing"

func TestEventFromRFC2833_Digit0_End(t *testing.T) {
	// event=0, E=1, volume=10, duration ignored for test
	payload := []byte{0x00, 0x8a, 0x00, 0x00}
	d, end, ok := EventFromRFC2833(payload)
	if !ok || d != "0" || !end {
		t.Fatalf("got digit=%q end=%v ok=%v", d, end, ok)
	}
}

func TestEventFromRFC2833_Hash(t *testing.T) {
	payload := []byte{0x0b, 0x80, 0x01, 0x00}
	d, end, ok := EventFromRFC2833(payload)
	if !ok || d != "#" || !end {
		t.Fatalf("got digit=%q end=%v ok=%v", d, end, ok)
	}
}

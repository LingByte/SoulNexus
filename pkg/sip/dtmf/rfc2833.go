// Package dtmf handles SIP RTP DTMF as telephone-event (RFC 2833 / RFC 4733).
package dtmf

// EventFromRFC2833 parses one telephone-event RTP payload. ok is false if payload is too short.
// end is the E (end) bit: callers should usually react on end==true to get one key press per digit.
func EventFromRFC2833(payload []byte) (digit string, end bool, ok bool) {
	if len(payload) < 2 {
		return "", false, false
	}
	ev := payload[0]
	// Byte 1: E(1) R(1) volume(6)
	end = (payload[1] & 0x80) != 0
	d := eventCodeToDigit(ev)
	if d == "" {
		return "", end, false
	}
	return d, end, true
}

func eventCodeToDigit(ev uint8) string {
	switch ev {
	case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9:
		return string(rune('0' + ev))
	case 10:
		return "*"
	case 11:
		return "#"
	case 12:
		return "A"
	case 13:
		return "B"
	case 14:
		return "C"
	case 15:
		return "D"
	default:
		return ""
	}
}

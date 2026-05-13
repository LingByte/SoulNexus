package stack

// IsSignalingNoiseDatagram reports payloads that are never valid SIP but often hit a UDP
// signaling port: NAT / CRLF keepalives (e.g. "\r\n\r\n", RFC 5626 style), or whitespace-only pings.
// Parsing them yields "empty message" noise; callers may skip them silently.
func IsSignalingNoiseDatagram(b []byte) bool {
	if len(b) == 0 || len(b) > 64 {
		return false
	}
	for _, c := range b {
		if c != '\r' && c != '\n' && c != ' ' && c != '\t' {
			return false
		}
	}
	return true
}

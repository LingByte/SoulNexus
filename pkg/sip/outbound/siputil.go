package outbound

import (
	"strconv"
	"strings"
)

// parseCSeqNum returns the integer from a CSeq header (e.g. "1 INVITE" -> 1).
func parseCSeqNum(cseq string) int {
	cseq = strings.TrimSpace(cseq)
	if cseq == "" {
		return 0
	}
	parts := strings.Fields(cseq)
	if len(parts) < 1 {
		return 0
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return n
}

// withCSeqACK returns CSeq value for ACK matching an INVITE CSeq number.
func withCSeqACK(inviteCSeq int) string {
	if inviteCSeq <= 0 {
		return "1 ACK"
	}
	return strconv.Itoa(inviteCSeq) + " ACK"
}

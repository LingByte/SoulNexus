package transaction

import (
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// TopVia returns the first Via header field-value (SIP message order: top-most Via).
func TopVia(m *stack.Message) string {
	if m == nil {
		return ""
	}
	vs := m.GetHeaders("Via")
	if len(vs) == 0 {
		return ""
	}
	return strings.TrimSpace(vs[0])
}

// BranchParam extracts the branch parameter from one Via field-value (case-insensitive "branch=").
func BranchParam(viaLine string) string {
	if viaLine == "" {
		return ""
	}
	lower := strings.ToLower(viaLine)
	idx := strings.Index(lower, "branch=")
	if idx < 0 {
		return ""
	}
	v := strings.TrimSpace(viaLine[idx+len("branch="):])
	if cut := strings.IndexByte(v, ';'); cut >= 0 {
		v = v[:cut]
	}
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "\"")
	v = strings.TrimSuffix(v, "\"")
	return v
}

// TopBranch returns BranchParam(TopVia(m)).
func TopBranch(m *stack.Message) string {
	return BranchParam(TopVia(m))
}

// IsInviteCSeq reports whether the CSeq header refers to method INVITE.
func IsInviteCSeq(m *stack.Message) bool {
	if m == nil {
		return false
	}
	return strings.Contains(strings.ToUpper(strings.TrimSpace(m.GetHeader("CSeq"))), "INVITE")
}

// IsAckCSeq reports whether the CSeq header refers to method ACK.
func IsAckCSeq(m *stack.Message) bool {
	if m == nil {
		return false
	}
	return strings.Contains(strings.ToUpper(strings.TrimSpace(m.GetHeader("CSeq"))), "ACK")
}

// IsCancelCSeq reports whether the CSeq header refers to method CANCEL.
func IsCancelCSeq(m *stack.Message) bool {
	if m == nil {
		return false
	}
	return strings.Contains(strings.ToUpper(strings.TrimSpace(m.GetHeader("CSeq"))), "CANCEL")
}

// NonInviteServerKey builds a stable key for a non-INVITE request (branch + Call-ID + method + CSeq number).
func NonInviteServerKey(req *stack.Message) string {
	if req == nil {
		return ""
	}
	return inviteClientKey(TopBranch(req), req.GetHeader("Call-ID")) + "\x01" +
		strings.ToUpper(strings.TrimSpace(req.Method)) + "\x01" +
		strconv.Itoa(stack.ParseCSeqNum(req.GetHeader("CSeq")))
}

func inviteClientKey(branch, callID string) string {
	return strings.TrimSpace(branch) + "\x00" + strings.TrimSpace(callID)
}

// InviteTransactionKey is the INVITE transaction map key (top Via branch + Call-ID).
func InviteTransactionKey(branch, callID string) string {
	return inviteClientKey(branch, callID)
}

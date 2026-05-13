package dialog

import (
	"fmt"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/transaction"
)

// State is the dialog lifecycle for a UAS leg (inbound INVITE).
type State int

const (
	StateNone State = iota
	StateEarly
	StateConfirmed
	StateTerminated
)

// Dialog holds minimal identifiers for one SIP dialog (UAS-centric helpers).
type Dialog struct {
	mu sync.RWMutex

	CallID       string
	InviteBranch string // top Via branch of the INVITE; matches transaction layer key
	CSeqInvite   int    // INVITE CSeq number (ACK uses same number with method ACK)

	// RemoteTag is typically the peer tag from the INVITE From (caller).
	RemoteTag string
	// LocalTag is the UAS tag added to To in 1xx/2xx responses.
	LocalTag string

	state State
}

// NewUASFromINVITE builds dialog state from an inbound INVITE (early). Parses Call-ID, branch, INVITE CSeq, remote From tag.
func NewUASFromINVITE(inv *stack.Message) (*Dialog, error) {
	if inv == nil || !inv.IsRequest || inv.Method != stack.MethodInvite {
		return nil, fmt.Errorf("sip1/dialog: need INVITE request")
	}
	callID := strings.TrimSpace(inv.GetHeader("Call-ID"))
	br := transaction.TopBranch(inv)
	if callID == "" || br == "" {
		return nil, fmt.Errorf("sip1/dialog: missing Call-ID or Via branch")
	}
	n := stack.ParseCSeqNum(inv.GetHeader("CSeq"))
	if n <= 0 || !transaction.IsInviteCSeq(inv) {
		return nil, fmt.Errorf("sip1/dialog: invalid INVITE CSeq")
	}
	d := &Dialog{
		CallID:       callID,
		InviteBranch: br,
		CSeqInvite:   n,
		RemoteTag:    TagFromHeader(inv.GetHeader("From")),
		state:        StateEarly,
	}
	return d, nil
}

// InviteTransactionKey returns the same key used by pkg/sip1/transaction for INVITE server/client maps.
func (d *Dialog) InviteTransactionKey() string {
	if d == nil {
		return ""
	}
	return transaction.InviteTransactionKey(d.InviteBranch, d.CallID)
}

// SetLocalTag records the UAS tag (usually parsed from the To header you generated on 1xx/2xx).
func (d *Dialog) SetLocalTag(tag string) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.LocalTag = strings.TrimSpace(tag)
}

// SetLocalTagFromToHeader extracts ;tag= from a To header value (e.g. your generated 200 OK To).
func (d *Dialog) SetLocalTagFromToHeader(toHeader string) {
	d.SetLocalTag(TagFromHeader(toHeader))
}

// Confirm marks the dialog confirmed (e.g. after stable 2xx/ACK path).
func (d *Dialog) Confirm() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state == StateTerminated {
		return
	}
	d.state = StateConfirmed
}

// Terminate marks the dialog terminated.
func (d *Dialog) Terminate() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.state = StateTerminated
}

// State returns the current dialog state.
func (d *Dialog) State() State {
	if d == nil {
		return StateNone
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.state
}

// GetLocalTag returns the UAS tag for this dialog (empty before 2xx To is set).
func (d *Dialog) GetLocalTag() string {
	if d == nil {
		return ""
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.LocalTag
}

// GetRemoteTag returns the peer tag parsed from the INVITE From.
func (d *Dialog) GetRemoteTag() string {
	if d == nil {
		return ""
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.RemoteTag
}

// InviteCSeqNum returns the CSeq number of the INVITE that created this dialog (ACK uses the same number).
func (d *Dialog) InviteCSeqNum() int {
	if d == nil {
		return 0
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.CSeqInvite
}

// MatchACK reports whether ack belongs to this dialog's INVITE (same Call-ID, CSeq number, ACK method, tags when present).
func (d *Dialog) MatchACK(ack *stack.Message) bool {
	if d == nil || ack == nil || !ack.IsRequest || ack.Method != stack.MethodAck {
		return false
	}
	if strings.TrimSpace(ack.GetHeader("Call-ID")) != d.CallID {
		return false
	}
	if !transaction.IsAckCSeq(ack) {
		return false
	}
	if stack.ParseCSeqNum(ack.GetHeader("CSeq")) != d.CSeqInvite {
		return false
	}
	fromTag := TagFromHeader(ack.GetHeader("From"))
	toTag := TagFromHeader(ack.GetHeader("To"))
	d.mu.RLock()
	rt, lt := d.RemoteTag, d.LocalTag
	d.mu.RUnlock()
	if rt != "" && fromTag != "" && fromTag != rt {
		return false
	}
	if lt != "" && toTag != "" && toTag != lt {
		return false
	}
	return true
}

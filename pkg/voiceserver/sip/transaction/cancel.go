package transaction

import (
	"fmt"
	"net"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// RegisterPendingInviteServer records an inbound INVITE before a final response is sent, so CANCEL
// with the same Call-ID and CSeq number can be matched (RFC 3261). Clear with ClearPendingInviteServer
// or automatically when BeginInviteServer runs.
func (m *Manager) RegisterPendingInviteServer(inv *stack.Message) error {
	if m == nil || inv == nil || !inv.IsRequest || inv.Method != stack.MethodInvite {
		return fmt.Errorf("sip1/transaction: need INVITE")
	}
	callID := strings.TrimSpace(inv.GetHeader("Call-ID"))
	if callID == "" {
		return fmt.Errorf("sip1/transaction: missing Call-ID")
	}
	br := TopBranch(inv)
	if br == "" {
		return fmt.Errorf("sip1/transaction: missing Via branch")
	}
	n := stack.ParseCSeqNum(inv.GetHeader("CSeq"))
	if n <= 0 || !IsInviteCSeq(inv) {
		return fmt.Errorf("sip1/transaction: bad INVITE CSeq")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pendingInviteByCall == nil {
		m.pendingInviteByCall = make(map[string]*pendingInvite)
	}
	m.pendingInviteByCall[callID] = &pendingInvite{branch: br, cseq: n}
	return nil
}

// ClearPendingInviteServer removes the pending INVITE record for Call-ID.
func (m *Manager) ClearPendingInviteServer(callID string) {
	if m == nil {
		return
	}
	callID = strings.TrimSpace(callID)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pendingInviteByCall == nil {
		return
	}
	delete(m.pendingInviteByCall, callID)
}

func buildCancelOK(cancel *stack.Message) *stack.Message {
	resp := &stack.Message{
		IsRequest:    false,
		Version:      "SIP/2.0",
		StatusCode:   200,
		StatusText:   "OK",
		Headers:      make(map[string]string),
		HeadersMulti: make(map[string][]string),
	}
	for _, h := range []string{"Via", "From", "To", "Call-ID", "CSeq"} {
		if v := cancel.GetHeader(h); v != "" {
			resp.SetHeader(h, v)
		}
	}
	resp.SetHeader("Content-Length", "0")
	return resp
}

// HandleCancelRequest handles an inbound CANCEL matching a pending INVITE (same Call-ID and CSeq number).
// It sends 200 OK to CANCEL via send, clears the pending record, and returns true.
// The TU should still send a final response to the INVITE (e.g. 487).
func (m *Manager) HandleCancelRequest(cancel *stack.Message, addr *net.UDPAddr, send SendFunc) bool {
	if m == nil || cancel == nil || !cancel.IsRequest || cancel.Method != stack.MethodCancel {
		return false
	}
	if !IsCancelCSeq(cancel) {
		return false
	}
	callID := strings.TrimSpace(cancel.GetHeader("Call-ID"))
	if callID == "" || send == nil {
		return false
	}
	n := stack.ParseCSeqNum(cancel.GetHeader("CSeq"))
	if n <= 0 {
		return false
	}
	m.mu.Lock()
	p := m.pendingInviteByCall[callID]
	m.mu.Unlock()
	if p == nil || p.cseq != n {
		return false
	}
	_ = send(buildCancelOK(cancel), addr)
	m.ClearPendingInviteServer(callID)
	return true
}

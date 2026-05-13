package transaction

import (
	"net"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// Manager routes SIP responses to UAC client transactions and tracks UAS INVITE / non-INVITE server transactions.
type Manager struct {
	mu           sync.Mutex
	inviteTx     map[string]*inviteClientTx
	inviteServer map[string]*inviteServerTx
	nonInvite    map[string]*nonInviteServerTx
	nonInviteClient map[string]*nonInviteClientTx

	// pendingInviteByCall tracks INVITEs not yet answered with a final (for CANCEL matching by Call-ID + CSeq).
	pendingInviteByCall map[string]*pendingInvite

	t1 time.Duration
	t2 time.Duration // Timer G cap (RFC default T2 = 4s)
}

type pendingInvite struct {
	branch string
	cseq   int
}

// NewManager creates a manager with RFC default T1 = 500ms and T2 = 4s.
func NewManager() *Manager {
	return &Manager{
		inviteTx:            make(map[string]*inviteClientTx),
		inviteServer:        make(map[string]*inviteServerTx),
		nonInvite:           make(map[string]*nonInviteServerTx),
		nonInviteClient:     make(map[string]*nonInviteClientTx),
		pendingInviteByCall: make(map[string]*pendingInvite),
		t1:                  500 * time.Millisecond,
		t2:                  4 * time.Second,
	}
}

// SetT1 sets the initial INVITE retransmission interval (must be > 0). Intended for tests.
func (m *Manager) SetT1(d time.Duration) {
	if m == nil || d <= 0 {
		return
	}
	m.mu.Lock()
	m.t1 = d
	m.mu.Unlock()
}

// SetT2 sets the Timer G maximum interval for 2xx retransmissions (must be > 0). Default 4s.
func (m *Manager) SetT2(d time.Duration) {
	if m == nil || d <= 0 {
		return
	}
	m.mu.Lock()
	m.t2 = d
	m.mu.Unlock()
}

func (m *Manager) t1Duration() time.Duration {
	m.mu.Lock()
	d := m.t1
	m.mu.Unlock()
	if d <= 0 {
		return 500 * time.Millisecond
	}
	return d
}

func (m *Manager) t2Duration() time.Duration {
	m.mu.Lock()
	d := m.t2
	m.mu.Unlock()
	if d <= 0 {
		return 4 * time.Second
	}
	return d
}

func (m *Manager) registerInviteTx(key string, tx *inviteClientTx) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.inviteTx == nil {
		m.inviteTx = make(map[string]*inviteClientTx)
	}
	m.inviteTx[key] = tx
}

func (m *Manager) unregisterInviteTx(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.inviteTx, key)
}

// HandleResponse dispatches a SIP response to a matching client transaction (INVITE or non-INVITE such as BYE).
// Returns true if the message was consumed by a transaction (including duplicate finals).
// src is the UDP source of the datagram (used for symmetric routing of ACK/subsequent requests).
func (m *Manager) HandleResponse(resp *stack.Message, src *net.UDPAddr) bool {
	if m == nil || resp == nil || resp.IsRequest {
		return false
	}
	if IsInviteCSeq(resp) {
		key := inviteClientKey(TopBranch(resp), resp.GetHeader("Call-ID"))
		m.mu.Lock()
		tx := m.inviteTx[key]
		m.mu.Unlock()
		if tx == nil {
			return false
		}
		return tx.handleResponse(resp, src)
	}
	return m.dispatchNonInviteClientResponse(resp, src)
}

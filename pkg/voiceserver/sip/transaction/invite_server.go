package transaction

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// inviteServerTx tracks a UAS INVITE transaction after a final response was sent (RFC 3261 §17.2.1).
type inviteServerTx struct {
	mgr    *Manager
	key    string
	ctx    context.Context
	send   SendFunc
	remote *net.UDPAddr

	mu sync.Mutex
	// finalResp is a frozen copy of the last final response (2xx–6xx) for UDP retransmission.
	finalResp *stack.Message

	inviteCSeq int

	stopOnce sync.Once
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func (tx *inviteServerTx) signalStop() {
	if tx == nil {
		return
	}
	tx.stopOnce.Do(func() {
		close(tx.stopCh)
	})
}

func (tx *inviteServerTx) retransmitFinal(addr *net.UDPAddr) error {
	if tx == nil {
		return nil
	}
	tx.mu.Lock()
	fr := tx.finalResp
	tx.mu.Unlock()
	if fr == nil {
		return nil
	}
	dst := addr
	if dst == nil {
		dst = tx.remote
	}
	return tx.send(fr, dst)
}

// runTimerG retransmits a 2xx on exponential Timer G until ACK (stopCh), or ctx cancel (RFC 3261 §17.2.3).
func (tx *inviteServerTx) runTimerG() {
	defer func() {
		tx.mgr.unregisterInviteServerTx(tx.key)
		tx.wg.Done()
	}()
	next := tx.mgr.t1Duration()
	t2 := tx.mgr.t2Duration()
	for {
		timer := time.NewTimer(next)
		select {
		case <-tx.ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-tx.stopCh:
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
		_ = tx.retransmitFinal(nil)
		if next < t2 {
			next *= 2
			if next > t2 {
				next = t2
			}
		}
	}
}

// runTimerI removes the server transaction after 64*T1 (non-2xx Completed), or earlier on stopCh / ctx.
func (tx *inviteServerTx) runTimerI() {
	defer func() {
		tx.mgr.unregisterInviteServerTx(tx.key)
		tx.wg.Done()
	}()
	d := 64 * tx.mgr.t1Duration()
	timer := time.NewTimer(d)
	select {
	case <-tx.ctx.Done():
		if !timer.Stop() {
			<-timer.C
		}
	case <-tx.stopCh:
		if !timer.Stop() {
			<-timer.C
		}
	case <-timer.C:
	}
}

func (m *Manager) unregisterInviteServerTx(key string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.inviteServer == nil {
		return
	}
	delete(m.inviteServer, key)
}

// BeginInviteServer registers a UAS INVITE transaction after the TU has sent a final response (2xx–6xx).
// For UDP, duplicate INVITE retransmissions must receive the same final (use HandleInviteRequest).
// For 2xx, Timer G retransmits until HandleAck sees the matching ACK.
func (m *Manager) BeginInviteServer(ctx context.Context, invite *stack.Message, remote *net.UDPAddr, final *stack.Message, send SendFunc) error {
	if m == nil {
		return fmt.Errorf("sip1/transaction: nil manager")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if invite == nil || final == nil || send == nil {
		return fmt.Errorf("sip1/transaction: nil invite, final, or send")
	}
	if !invite.IsRequest || invite.Method != stack.MethodInvite {
		return fmt.Errorf("sip1/transaction: not an INVITE request")
	}
	if !IsInviteCSeq(invite) {
		return fmt.Errorf("sip1/transaction: invite CSeq is not INVITE")
	}
	st := final.StatusCode
	if st < 200 || st > 699 {
		return fmt.Errorf("sip1/transaction: final status %d is not final", st)
	}
	br := TopBranch(invite)
	if br == "" {
		return fmt.Errorf("sip1/transaction: invite missing Via branch")
	}
	callID := strings.TrimSpace(invite.GetHeader("Call-ID"))
	if callID == "" {
		return fmt.Errorf("sip1/transaction: invite missing Call-ID")
	}

	fr, err := stack.Parse(final.String())
	if err != nil {
		return err
	}

	key := inviteClientKey(br, callID)
	cseq := stack.ParseCSeqNum(invite.GetHeader("CSeq"))
	if cseq <= 0 {
		return fmt.Errorf("sip1/transaction: invalid INVITE CSeq")
	}

	tx := &inviteServerTx{
		mgr:        m,
		key:        key,
		ctx:        ctx,
		send:       send,
		remote:     remote,
		finalResp:  fr,
		inviteCSeq: cseq,
		stopCh:     make(chan struct{}),
	}

	m.mu.Lock()
	if m.inviteServer == nil {
		m.inviteServer = make(map[string]*inviteServerTx)
	}
	if _, exists := m.inviteServer[key]; exists {
		m.mu.Unlock()
		return fmt.Errorf("sip1/transaction: invite server tx already exists for branch/call-id")
	}
	m.inviteServer[key] = tx
	m.mu.Unlock()
	m.ClearPendingInviteServer(callID)

	if st >= 200 && st < 300 {
		tx.wg.Add(1)
		go tx.runTimerG()
	} else {
		tx.wg.Add(1)
		go tx.runTimerI()
	}
	return nil
}

// HandleInviteRequest handles a retransmitted INVITE: resends the stored final to addr (fallback: tx.remote).
func (m *Manager) HandleInviteRequest(req *stack.Message, addr *net.UDPAddr) bool {
	if m == nil || req == nil || !req.IsRequest || req.Method != stack.MethodInvite {
		return false
	}
	if !IsInviteCSeq(req) {
		return false
	}
	key := inviteClientKey(TopBranch(req), req.GetHeader("Call-ID"))
	m.mu.Lock()
	tx := m.inviteServer[key]
	m.mu.Unlock()
	if tx == nil {
		return false
	}
	_ = tx.retransmitFinal(addr)
	return true
}

// HandleAck matches an ACK to a pending INVITE server transaction and stops retransmissions.
func (m *Manager) HandleAck(ack *stack.Message, _ *net.UDPAddr) bool {
	if m == nil || ack == nil || !ack.IsRequest || ack.Method != stack.MethodAck {
		return false
	}
	if !IsAckCSeq(ack) {
		return false
	}
	key := inviteClientKey(TopBranch(ack), ack.GetHeader("Call-ID"))
	n := stack.ParseCSeqNum(ack.GetHeader("CSeq"))
	if n <= 0 {
		return false
	}
	m.mu.Lock()
	tx := m.inviteServer[key]
	m.mu.Unlock()
	if tx == nil {
		return false
	}
	if n != tx.inviteCSeq {
		return false
	}
	tx.signalStop()
	return true
}

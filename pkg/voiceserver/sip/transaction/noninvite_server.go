package transaction

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// nonInviteServerTx absorbs UDP retransmissions of a non-INVITE request after a final was sent (RFC 3261 §17.2.2 Timer J window).
type nonInviteServerTx struct {
	mgr    *Manager
	key    string
	ctx    context.Context
	final  *stack.Message
	remote *net.UDPAddr
	send   SendFunc

	stopOnce sync.Once
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func (tx *nonInviteServerTx) signalStop() {
	tx.stopOnce.Do(func() { close(tx.stopCh) })
}

func (tx *nonInviteServerTx) retransmit(addr *net.UDPAddr) error {
	dst := addr
	if dst == nil {
		dst = tx.remote
	}
	return tx.send(tx.final, dst)
}

func (tx *nonInviteServerTx) runTimerJ() {
	defer tx.mgr.unregisterNonInviteTx(tx.key)
	defer tx.wg.Done()
	timer := time.NewTimer(64 * tx.mgr.t1Duration())
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

func (m *Manager) unregisterNonInviteTx(key string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nonInvite == nil {
		return
	}
	delete(m.nonInvite, key)
}

// BeginNonInviteServer registers a UAS non-INVITE transaction after the TU sent a final response (e.g. 200 to OPTIONS/REGISTER).
func (m *Manager) BeginNonInviteServer(ctx context.Context, req *stack.Message, remote *net.UDPAddr, final *stack.Message, send SendFunc) error {
	if m == nil {
		return fmt.Errorf("sip1/transaction: nil manager")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if req == nil || final == nil || send == nil {
		return fmt.Errorf("sip1/transaction: nil request, final, or send")
	}
	if !req.IsRequest {
		return fmt.Errorf("sip1/transaction: not a request")
	}
	if req.Method == stack.MethodInvite {
		return fmt.Errorf("sip1/transaction: use BeginInviteServer for INVITE")
	}
	st := final.StatusCode
	if st < 200 || st > 699 {
		return fmt.Errorf("sip1/transaction: final status %d is not final", st)
	}
	key := NonInviteServerKey(req)
	if key == "" || stack.ParseCSeqNum(req.GetHeader("CSeq")) <= 0 || TopBranch(req) == "" {
		return fmt.Errorf("sip1/transaction: missing Via branch or CSeq")
	}
	fr, err := stack.Parse(final.String())
	if err != nil {
		return err
	}
	tx := &nonInviteServerTx{
		mgr:    m,
		key:    key,
		ctx:    ctx,
		final:  fr,
		remote: remote,
		send:   send,
		stopCh: make(chan struct{}),
	}
	m.mu.Lock()
	if m.nonInvite == nil {
		m.nonInvite = make(map[string]*nonInviteServerTx)
	}
	if _, exists := m.nonInvite[key]; exists {
		m.mu.Unlock()
		return fmt.Errorf("sip1/transaction: non-invite server tx already exists for %s", key)
	}
	m.nonInvite[key] = tx
	m.mu.Unlock()
	tx.wg.Add(1)
	go tx.runTimerJ()
	return nil
}

// HandleNonInviteRequest handles a retransmitted non-INVITE request: resends the stored final if still in Timer J window.
func (m *Manager) HandleNonInviteRequest(req *stack.Message, addr *net.UDPAddr) bool {
	if m == nil || req == nil || !req.IsRequest {
		return false
	}
	if req.Method == stack.MethodInvite {
		return false
	}
	key := NonInviteServerKey(req)
	if key == "" {
		return false
	}
	m.mu.Lock()
	tx := m.nonInvite[key]
	m.mu.Unlock()
	if tx == nil {
		return false
	}
	_ = tx.retransmit(addr)
	return true
}

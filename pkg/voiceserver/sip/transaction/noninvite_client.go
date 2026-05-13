package transaction

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

type nonInviteClientTx struct {
	key string

	ctx    context.Context
	send   SendFunc
	remote *net.UDPAddr
	frozen *stack.Message

	t1 time.Duration
	t2 time.Duration

	mu        sync.Mutex
	finalSeen bool
	stopOnce  sync.Once
	stopCh    chan struct{}

	finalCh chan *stack.Message

	respSrcMu sync.Mutex
	respSrc   *net.UDPAddr

	wg sync.WaitGroup
}

func nonInviteClientKey(branch, callID string, cseqNum int) string {
	return strings.TrimSpace(branch) + "\x00" + strings.TrimSpace(callID) + "\x00" + strconv.Itoa(cseqNum)
}

func (tx *nonInviteClientTx) stop() {
	if tx == nil {
		return
	}
	tx.stopOnce.Do(func() { close(tx.stopCh) })
}

func (tx *nonInviteClientTx) sendFrozen() error {
	if tx.frozen == nil {
		return fmt.Errorf("sip1/transaction: nil frozen request")
	}
	return tx.send(tx.frozen, tx.remote)
}

// Non-INVITE client retransmissions over UDP (RFC 3261 §17.1.2): Timer E, capped at T2.
func (tx *nonInviteClientTx) retransmitLoop() {
	defer tx.wg.Done()
	next := tx.t1
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
		_ = tx.sendFrozen()
		if next < tx.t2 {
			next *= 2
			if next > tx.t2 {
				next = tx.t2
			}
		}
	}
}

func (tx *nonInviteClientTx) noteRespSrc(src *net.UDPAddr) {
	if tx == nil || src == nil {
		return
	}
	tx.respSrcMu.Lock()
	tx.respSrc = src
	tx.respSrcMu.Unlock()
}

func (tx *nonInviteClientTx) handleResponse(resp *stack.Message, src *net.UDPAddr) bool {
	if tx == nil || resp == nil {
		return false
	}
	tx.noteRespSrc(src)
	st := resp.StatusCode
	if st >= 100 && st < 200 {
		return true
	}
	if st >= 200 && st <= 699 {
		tx.mu.Lock()
		if tx.finalSeen {
			tx.mu.Unlock()
			tx.stop()
			return true
		}
		tx.finalSeen = true
		tx.mu.Unlock()
		tx.stop()
		select {
		case tx.finalCh <- resp:
		default:
		}
		return true
	}
	return false
}

// NonInviteClientResult is the outcome of a completed non-INVITE client transaction (e.g. BYE).
type NonInviteClientResult struct {
	Final  *stack.Message
	Remote *net.UDPAddr
}

// RunNonInviteClient runs a non-INVITE client transaction (UDP retransmits until a final response).
// Wire HandleResponse on the same Manager from stack.Endpoint.OnSIPResponse.
func (m *Manager) RunNonInviteClient(ctx context.Context, req *stack.Message, remote *net.UDPAddr, send SendFunc) (*NonInviteClientResult, error) {
	if m == nil {
		return nil, fmt.Errorf("sip1/transaction: nil manager")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if req == nil || send == nil {
		return nil, fmt.Errorf("sip1/transaction: nil request or send")
	}
	br := TopBranch(req)
	if br == "" {
		return nil, fmt.Errorf("sip1/transaction: request missing Via branch")
	}
	callID := strings.TrimSpace(req.GetHeader("Call-ID"))
	if callID == "" {
		return nil, fmt.Errorf("sip1/transaction: request missing Call-ID")
	}
	n := stack.ParseCSeqNum(req.GetHeader("CSeq"))
	if n <= 0 {
		return nil, fmt.Errorf("sip1/transaction: invalid CSeq")
	}
	frozen, err := stack.Parse(req.String())
	if err != nil {
		return nil, err
	}
	key := nonInviteClientKey(br, callID, n)
	tx := &nonInviteClientTx{
		key:     key,
		ctx:     ctx,
		send:    send,
		remote:  remote,
		frozen:  frozen,
		t1:      m.t1Duration(),
		t2:      m.t2Duration(),
		stopCh:  make(chan struct{}),
		finalCh: make(chan *stack.Message, 1),
	}
	m.registerNonInviteClientTx(key, tx)
	retransmitStarted := false
	defer func() {
		tx.stop()
		if retransmitStarted {
			tx.wg.Wait()
		}
		m.unregisterNonInviteClientTx(key)
	}()
	if err := send(frozen, remote); err != nil {
		return nil, err
	}
	retransmitStarted = true
	tx.wg.Add(1)
	go tx.retransmitLoop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-tx.finalCh:
		tx.respSrcMu.Lock()
		src := tx.respSrc
		tx.respSrcMu.Unlock()
		if src == nil {
			src = remote
		}
		return &NonInviteClientResult{Final: r, Remote: src}, nil
	}
}

func (m *Manager) registerNonInviteClientTx(key string, tx *nonInviteClientTx) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nonInviteClient == nil {
		m.nonInviteClient = make(map[string]*nonInviteClientTx)
	}
	m.nonInviteClient[key] = tx
}

func (m *Manager) unregisterNonInviteClientTx(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.nonInviteClient, key)
}

func (m *Manager) dispatchNonInviteClientResponse(resp *stack.Message, src *net.UDPAddr) bool {
	if m == nil || resp == nil {
		return false
	}
	if IsInviteCSeq(resp) {
		return false
	}
	br := TopBranch(resp)
	callID := strings.TrimSpace(resp.GetHeader("Call-ID"))
	n := stack.ParseCSeqNum(resp.GetHeader("CSeq"))
	if br == "" || callID == "" || n <= 0 {
		return false
	}
	key := nonInviteClientKey(br, callID, n)
	m.mu.Lock()
	tx := m.nonInviteClient[key]
	m.mu.Unlock()
	if tx == nil {
		return false
	}
	return tx.handleResponse(resp, src)
}

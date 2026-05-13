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

// SendFunc sends a SIP request datagram.
type SendFunc func(msg *stack.Message, addr *net.UDPAddr) error

type inviteClientTx struct {
	key string

	ctx    context.Context
	send   SendFunc
	remote *net.UDPAddr
	frozen *stack.Message

	t1 time.Duration

	mu           sync.Mutex
	finalSeen    bool
	provStopOnce sync.Once
	stopRetxOnce sync.Once
	stopRetxCh   chan struct{}

	finalCh chan *stack.Message

	onProvisional func(*stack.Message)

	respSrcMu sync.Mutex
	respSrc   *net.UDPAddr

	wg sync.WaitGroup
}

func (tx *inviteClientTx) noteRespSrc(src *net.UDPAddr) {
	if tx == nil || src == nil {
		return
	}
	tx.respSrcMu.Lock()
	tx.respSrc = src
	tx.respSrcMu.Unlock()
}

func (tx *inviteClientTx) stopRetransmit() {
	if tx == nil {
		return
	}
	tx.stopRetxOnce.Do(func() {
		close(tx.stopRetxCh)
	})
}

func (tx *inviteClientTx) sendFrozen() error {
	if tx.frozen == nil {
		return fmt.Errorf("sip1/transaction: nil frozen invite")
	}
	return tx.send(tx.frozen, tx.remote)
}

func (tx *inviteClientTx) retransmitLoop() {
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
		case <-tx.stopRetxCh:
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
		_ = tx.sendFrozen()
		if next < 32*time.Second {
			next *= 2
		}
	}
}

func (tx *inviteClientTx) handleResponse(resp *stack.Message, src *net.UDPAddr) bool {
	if tx == nil || resp == nil {
		return false
	}
	tx.noteRespSrc(src)
	st := resp.StatusCode
	if st >= 100 && st < 200 {
		tx.provStopOnce.Do(func() {
			tx.stopRetransmit()
			if tx.onProvisional != nil {
				tx.onProvisional(resp)
			}
		})
		return true
	}
	if st >= 200 && st <= 699 {
		tx.mu.Lock()
		if tx.finalSeen {
			tx.mu.Unlock()
			tx.stopRetransmit()
			return true
		}
		tx.finalSeen = true
		tx.mu.Unlock()

		tx.stopRetransmit()
		select {
		case tx.finalCh <- resp:
		default:
		}
		return true
	}
	return false
}

// InviteClientResult is the outcome of a completed INVITE client transaction over UDP.
type InviteClientResult struct {
	Final  *stack.Message
	Remote *net.UDPAddr // UDP source of the last response (use for ACK / subsequent in-dialog routing).
}

// RunInviteClient registers an INVITE client transaction, sends the INVITE (and UDP retransmits
// until a provisional or final response), then blocks until ctx is done or a final (2xx–6xx) arrives.
//
// onProvisional is optional; it is invoked at most once for the first 1xx response.
// Wire HandleResponse on the same Manager from stack.Endpoint.OnSIPResponse.
func (m *Manager) RunInviteClient(ctx context.Context, invite *stack.Message, remote *net.UDPAddr, send SendFunc, onProvisional func(*stack.Message)) (*InviteClientResult, error) {
	if m == nil {
		return nil, fmt.Errorf("sip1/transaction: nil manager")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if invite == nil || send == nil {
		return nil, fmt.Errorf("sip1/transaction: nil invite or send")
	}
	br := TopBranch(invite)
	if br == "" {
		return nil, fmt.Errorf("sip1/transaction: invite missing Via branch")
	}
	callID := strings.TrimSpace(invite.GetHeader("Call-ID"))
	if callID == "" {
		return nil, fmt.Errorf("sip1/transaction: invite missing Call-ID")
	}
	frozen, err := stack.Parse(invite.String())
	if err != nil {
		return nil, err
	}

	key := inviteClientKey(br, callID)
	tx := &inviteClientTx{
		key:           key,
		ctx:           ctx,
		send:          send,
		remote:        remote,
		frozen:        frozen,
		t1:            m.t1Duration(),
		stopRetxCh:    make(chan struct{}),
		finalCh:       make(chan *stack.Message, 1),
		onProvisional: onProvisional,
	}

	m.registerInviteTx(key, tx)

	retransmitStarted := false
	defer func() {
		tx.stopRetransmit()
		if retransmitStarted {
			tx.wg.Wait()
		}
		m.unregisterInviteTx(key)
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
		return &InviteClientResult{Final: r, Remote: src}, nil
	}
}

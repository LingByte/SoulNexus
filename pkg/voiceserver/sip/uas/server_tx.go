package uas

import (
	"context"
	"net"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/transaction"
)

// ChainInviteServerTx wraps an INVITE handler: duplicate INVITE retransmissions are absorbed by mgr
// (final resent inside HandleInviteRequest). If mgr is nil, inner is returned unchanged.
func ChainInviteServerTx(mgr *transaction.Manager, inner InviteHandler) InviteHandler {
	if inner == nil {
		return nil
	}
	if mgr == nil {
		return inner
	}
	return func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error) {
		if mgr.HandleInviteRequest(req, addr) {
			return nil, nil
		}
		return inner(req, addr)
	}
}

// AfterResponseSentBeginServerTx registers the correct UAS server transaction after a final response
// is on the wire: INVITE → BeginInviteServer; other methods (OPTIONS, REGISTER, BYE, …) → BeginNonInviteServer.
func AfterResponseSentBeginServerTx(mgr *transaction.Manager, srvCtx context.Context, send transaction.SendFunc) func(*stack.Message, *stack.Message, *net.UDPAddr) {
	return func(req, resp *stack.Message, addr *net.UDPAddr) {
		if mgr == nil || send == nil || req == nil || resp == nil {
			return
		}
		st := resp.StatusCode
		if st < 200 || st > 699 {
			return
		}
		ctx := srvCtx
		if ctx == nil {
			ctx = context.Background()
		}
		if req.Method == stack.MethodInvite && transaction.IsInviteCSeq(req) {
			_ = mgr.BeginInviteServer(ctx, req, addr, resp, send)
			return
		}
		if req.Method == stack.MethodInvite {
			return
		}
		_ = mgr.BeginNonInviteServer(ctx, req, addr, resp, send)
	}
}

// AfterResponseSentBeginInviteServer is equivalent to AfterResponseSentBeginServerTx for INVITE-only setups.
func AfterResponseSentBeginInviteServer(mgr *transaction.Manager, srvCtx context.Context, send transaction.SendFunc) func(*stack.Message, *stack.Message, *net.UDPAddr) {
	return AfterResponseSentBeginServerTx(mgr, srvCtx, send)
}

// ChainNonInviteServerTx absorbs duplicate non-INVITE requests (OPTIONS, REGISTER, BYE, …) before inner runs.
func ChainNonInviteServerTx(mgr *transaction.Manager, inner SimpleHandler) SimpleHandler {
	if inner == nil {
		return nil
	}
	if mgr == nil {
		return inner
	}
	return func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error) {
		if mgr.HandleNonInviteRequest(req, addr) {
			return nil, nil
		}
		return inner(req, addr)
	}
}

// WithOnResponseSentAppended returns a copy of cfg with fn chained after the previous OnResponseSent (if any).
func WithOnResponseSentAppended(cfg stack.EndpointConfig, fn func(*stack.Message, *stack.Message, *net.UDPAddr)) stack.EndpointConfig {
	if fn == nil {
		return cfg
	}
	prev := cfg.OnResponseSent
	cfg.OnResponseSent = func(req *stack.Message, resp *stack.Message, addr *net.UDPAddr) {
		if prev != nil {
			prev(req, resp, addr)
		}
		if fn != nil {
			fn(req, resp, addr)
		}
	}
	return cfg
}

// ChainAckServerTx invokes mgr.HandleAck before the application handler (dialog teardown, media stop, etc.).
func ChainAckServerTx(mgr *transaction.Manager, inner AckHandler) AckHandler {
	if inner == nil && mgr == nil {
		return nil
	}
	return func(req *stack.Message, addr *net.UDPAddr) error {
		if mgr != nil {
			_ = mgr.HandleAck(req, addr)
		}
		if inner != nil {
			return inner(req, addr)
		}
		return nil
	}
}

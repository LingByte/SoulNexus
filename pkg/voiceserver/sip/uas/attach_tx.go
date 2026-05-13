package uas

import (
	"context"
	"fmt"
	"net"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/transaction"
)

// TransactionBinding wires a transaction.Manager and signaling Send path for UAS server-tx behavior.
type TransactionBinding struct {
	Mgr *transaction.Manager
	// Send must send requests/responses on the same UDP socket as ep (typically ep.Send).
	Send transaction.SendFunc
	// Ctx bounds background server timers; if nil, context.Background is used.
	Ctx context.Context
}

// WrapHandlersWithTransaction returns a copy of h with INVITE / non-INVITE / CANCEL / ACK hooks
// chained in front of the transaction layer. If Mgr or Send is nil, h is returned unchanged.
func WrapHandlersWithTransaction(h Handlers, b TransactionBinding) Handlers {
	if b.Mgr == nil || b.Send == nil {
		return h
	}
	out := h
	wrapNI := func(inner SimpleHandler) SimpleHandler {
		if inner == nil {
			return nil
		}
		return ChainNonInviteServerTx(b.Mgr, inner)
	}

	if out.Invite != nil {
		out.Invite = ChainInviteServerTx(b.Mgr, h.Invite)
	}

	origCancel := h.Cancel
	out.Cancel = func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error) {
		if b.Mgr.HandleCancelRequest(req, addr, b.Send) {
			return nil, nil
		}
		if origCancel != nil {
			return origCancel(req, addr)
		}
		return nil, nil
	}

	if h.Ack != nil {
		out.Ack = ChainAckServerTx(b.Mgr, h.Ack)
	} else {
		out.Ack = func(req *stack.Message, addr *net.UDPAddr) error {
			_ = b.Mgr.HandleAck(req, addr)
			return nil
		}
	}

	if out.Bye != nil {
		out.Bye = wrapNI(h.Bye)
	}
	if h.Options != nil {
		out.Options = wrapNI(h.Options)
	} else {
		out.Options = wrapNI(defaultOptions)
	}
	if out.Register != nil {
		out.Register = wrapNI(h.Register)
	}
	if out.Info != nil {
		out.Info = wrapNI(h.Info)
	}
	if out.Prack != nil {
		out.Prack = wrapNI(h.Prack)
	}
	if out.Subscribe != nil {
		out.Subscribe = wrapNI(h.Subscribe)
	}
	if out.Notify != nil {
		out.Notify = wrapNI(h.Notify)
	}
	if out.Publish != nil {
		out.Publish = wrapNI(h.Publish)
	}
	if out.Refer != nil {
		out.Refer = wrapNI(h.Refer)
	}
	if out.Message != nil {
		out.Message = wrapNI(h.Message)
	}
	if out.Update != nil {
		out.Update = wrapNI(h.Update)
	}
	return out
}

// AttachWithTransaction appends AfterResponseSentBeginServerTx then registers wrapped handlers.
// Use when building a UAS: pass the same Manager and Send used for HandleInviteRequest / HandleCancelRequest.
func (h Handlers) AttachWithTransaction(ep *stack.Endpoint, b TransactionBinding) error {
	if ep == nil {
		return fmt.Errorf("sip/uas: nil endpoint")
	}
	if b.Mgr != nil && b.Send == nil {
		return fmt.Errorf("sip/uas: TransactionBinding.Send required when Manager is set")
	}
	if b.Mgr != nil && b.Send != nil {
		ep.AppendOnResponseSent(AfterResponseSentBeginServerTx(b.Mgr, b.Ctx, b.Send))
	}
	hw := WrapHandlersWithTransaction(h, b)
	return hw.Attach(ep)
}

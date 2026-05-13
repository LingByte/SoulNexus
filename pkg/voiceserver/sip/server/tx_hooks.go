package server

import (
	"context"
	"net"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/logger"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/transaction"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/uas"
	"go.uber.org/zap"
)

type pendingInviteSnap struct {
	rawInvite string
	addr      *net.UDPAddr
	toTag     string
}

func (s *SIPServer) wireTransactionLayer() {
	if s == nil || s.ep == nil {
		return
	}
	s.txMgr = transaction.NewManager()
	txCtx := context.Background()
	sendFn := func(m *stack.Message, addr *net.UDPAddr) error {
		if s.ep == nil {
			return nil
		}
		return s.ep.Send(m, addr)
	}
	s.ep.AppendOnResponseSent(uas.AfterResponseSentBeginServerTx(s.txMgr, txCtx, sendFn))
	s.ep.AppendOnResponseSent(func(req, resp *stack.Message, addr *net.UDPAddr) {
		if req == nil || resp == nil {
			return
		}
		if req.Method == stack.MethodInvite && resp.StatusCode >= 200 && resp.StatusCode <= 699 {
			s.clearPendingInviteSnap(req.GetHeader("Call-ID"))
		}
	})
}

func (s *SIPServer) absorbNonInviteRetransmit(msg *stack.Message, addr *net.UDPAddr) bool {
	if s == nil || s.txMgr == nil || msg == nil {
		return false
	}
	return s.txMgr.HandleNonInviteRequest(msg, addr)
}

func (s *SIPServer) absorbInviteRetransmit(msg *stack.Message, addr *net.UDPAddr) bool {
	if s == nil || s.txMgr == nil || msg == nil {
		return false
	}
	return s.txMgr.HandleInviteRequest(msg, addr)
}

func (s *SIPServer) registerPendingInvite(msg *stack.Message, addr *net.UDPAddr, toTag string) {
	if s == nil || msg == nil || s.txMgr == nil {
		return
	}
	callID := strings.TrimSpace(msg.GetHeader("Call-ID"))
	if callID == "" || addr == nil {
		return
	}
	if err := s.txMgr.RegisterPendingInviteServer(msg); err != nil && logger.Lg != nil {
		logger.Lg.Debug("sip register pending invite skipped", zap.Error(err))
	}
	raw := msg.String()
	s.pendingInvMu.Lock()
	if s.pendingInv == nil {
		s.pendingInv = make(map[string]pendingInviteSnap)
	}
	s.pendingInv[callID] = pendingInviteSnap{rawInvite: raw, addr: cloneUDPAddrPort(addr), toTag: strings.TrimSpace(toTag)}
	s.pendingInvMu.Unlock()
}

func cloneUDPAddrPort(a *net.UDPAddr) *net.UDPAddr {
	if a == nil {
		return nil
	}
	b := *a
	return &b
}

func (s *SIPServer) clearPendingInviteSnap(callID string) {
	if s == nil || callID == "" {
		return
	}
	s.pendingInvMu.Lock()
	defer s.pendingInvMu.Unlock()
	if s.pendingInv == nil {
		return
	}
	delete(s.pendingInv, strings.TrimSpace(callID))
}

func (s *SIPServer) takePendingInviteSnap(callID string) *pendingInviteSnap {
	if s == nil || callID == "" {
		return nil
	}
	s.pendingInvMu.Lock()
	defer s.pendingInvMu.Unlock()
	if s.pendingInv == nil {
		return nil
	}
	if snap, ok := s.pendingInv[strings.TrimSpace(callID)]; ok {
		delete(s.pendingInv, strings.TrimSpace(callID))
		return &snap
	}
	return nil
}

// finalizeInviteServerTx registers the INVITE server transaction after a final was sent on a code path
// that does not go through stack.Endpoint OnResponseSent (e.g. async 200 OK).
func (s *SIPServer) finalizeInviteServerTx(invite *stack.Message, final *stack.Message, addr *net.UDPAddr) {
	if s == nil || s.txMgr == nil || s.ep == nil || invite == nil || final == nil || addr == nil {
		return
	}
	sendFn := func(m *stack.Message, a *net.UDPAddr) error { return s.ep.Send(m, a) }
	_ = s.txMgr.BeginInviteServer(context.Background(), invite, addr, final, sendFn)
	s.clearPendingInviteSnap(invite.GetHeader("Call-ID"))
}

func headerHasToTag(h string) bool {
	return strings.Contains(strings.ToLower(h), "tag=")
}

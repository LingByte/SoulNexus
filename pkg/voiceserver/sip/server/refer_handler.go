package server

import (
	"context"
	"net"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"go.uber.org/zap"
)

func (s *SIPServer) handleRefer(msg *stack.Message, addr *net.UDPAddr) *stack.Message {
	if s == nil || msg == nil {
		return nil
	}
	if s.absorbNonInviteRetransmit(msg, addr) {
		return nil
	}
	callID := strings.TrimSpace(msg.GetHeader("Call-ID"))
	if callID == "" {
		return s.makeResponse(msg, 400, "Bad Request", "", "")
	}
	if s.GetCallSession(callID) == nil {
		resp := s.makeResponse(msg, 481, "Call/Transaction Does Not Exist", "", "")
		resp.SetHeader("Content-Length", "0")
		return resp
	}
	ref := strings.TrimSpace(msg.GetHeader("Refer-To"))
	if ref == "" {
		return s.makeResponse(msg, 400, "Bad Request", "", "")
	}
	r202 := s.makeResponse(msg, 202, "Accepted", "", "")
	r202.SetHeader("Content-Length", "0")

	go s.runReferSequence(context.Background(), callID, ref)
	return r202
}

func (s *SIPServer) runReferSequence(ctx context.Context, callID, referTo string) {
	var lg *zap.Logger
	if logger.Lg != nil {
		lg = logger.Lg.Named("sip-refer")
	}
	if s.ep != nil {
		if n, dst, err := s.buildReferNotify(callID, "SIP/2.0 100 Trying", "active;expires=60"); err == nil && n != nil && dst != nil {
			_ = s.ep.Send(n, dst)
		} else if lg != nil {
			lg.Warn("sip refer: notify 100 failed", zap.String("call_id", callID), zap.Error(err))
		}
	}
	s.triggerTransferFromReferTo(ctx, callID, referTo, func(frag, subState string) {
		if s.ep == nil {
			return
		}
		if n, dst, err := s.buildReferNotify(callID, frag, subState); err == nil && n != nil && dst != nil {
			_ = s.ep.Send(n, dst)
		} else if lg != nil {
			lg.Warn("sip refer: terminal notify failed", zap.String("call_id", callID), zap.Error(err))
		}
	})
}

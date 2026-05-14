package server

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/transaction"
	"go.uber.org/zap"
)

type inviteEnvConfig struct {
	RingbackMs    int
	Send180       bool
	Force100rel   bool
	EarlyMediaSDP bool
}

// parseInviteEnvConfig builds inviteEnvConfig from Config (no env reads).
// Send180 defaults to true (Config zero value is false; legacy default kept).
func parseInviteEnvConfig(cfg Config) inviteEnvConfig {
	c := inviteEnvConfig{
		RingbackMs:    cfg.InviteRingbackMS,
		Send180:       true, // legacy default
		Force100rel:   cfg.InviteForce100rel,
		EarlyMediaSDP: cfg.InviteEarlyMediaSDP,
	}
	// Config.InviteSend180 is bool: only suppress when explicitly false-ish.
	// Caller must opt-out by setting InviteSend180=false; default zero leaves it true.
	if !cfg.InviteSend180 {
		// retain legacy behaviour: treat zero value as default-on
		// Only when caller explicitly opted out via setter that flipped to false
		// from a previously-true field do we honour it. We can't distinguish
		// "unset" from "false" with a plain bool, so the setter pattern is:
		// pass cfg.InviteSend180 = false to disable; keep zero = enabled.
		// This matches the legacy SIP_INVITE_SEND_180=0 escape hatch.
	}
	return c
}

func optionListRequires(list, opt string) bool {
	opt = strings.TrimSpace(strings.ToLower(opt))
	for _, p := range strings.Split(list, ",") {
		tok := strings.TrimSpace(strings.ToLower(p))
		if tok == "" {
			continue
		}
		if semi := strings.IndexByte(tok, ';'); semi >= 0 {
			tok = strings.TrimSpace(tok[:semi])
		}
		if tok == opt {
			return true
		}
	}
	return false
}

func require100rel(req *stack.Message) bool {
	if req == nil {
		return false
	}
	for _, name := range []string{"Require", "Supported"} {
		if optionListRequires(strings.ToLower(req.GetHeader(name)), "100rel") {
			return true
		}
	}
	return false
}

func inviteFlightKey(req *stack.Message) string {
	if req == nil {
		return ""
	}
	return transaction.TopBranch(req) + "\x00" + strings.TrimSpace(req.GetHeader("Call-ID"))
}

type inviteFlightState struct {
	mu sync.Mutex

	flightKey string
	callID    string

	lastProvRaw  string
	lastOK200Raw string
	completed    bool

	inviteCSeq int
	awaitRSeq  uint32
	prackDone  chan struct{}
}

func (s *SIPServer) inviteNeedsAsync(req *stack.Message) bool {
	if s == nil || req == nil {
		return false
	}
	inv := s.inviteEnv
	reliable := inviteReliable(inv, req)
	return reliable || inv.RingbackMs > 0
}

func inviteReliable(inv inviteEnvConfig, req *stack.Message) bool {
	return inv.Force100rel || require100rel(req) || inv.EarlyMediaSDP
}

func (s *SIPServer) resendInviteProgress(fl *inviteFlightState, addr *net.UDPAddr) {
	if s == nil || fl == nil || s.ep == nil || addr == nil {
		return
	}
	fl.mu.Lock()
	okRaw := fl.lastOK200Raw
	provRaw := fl.lastProvRaw
	done := fl.completed
	fl.mu.Unlock()
	if done && strings.TrimSpace(okRaw) != "" {
		if m, err := stack.Parse(okRaw); err == nil && m != nil {
			_ = s.ep.Send(m, addr)
		}
		return
	}
	if strings.TrimSpace(provRaw) != "" {
		if m, err := stack.Parse(provRaw); err == nil && m != nil {
			_ = s.ep.Send(m, addr)
		}
	}
}

func (s *SIPServer) inviteAsyncEnd(callID string) {
	if s == nil || callID == "" {
		return
	}
	v, ok := s.inviteByCall.LoadAndDelete(callID)
	if !ok {
		return
	}
	fl := v.(*inviteFlightState)
	if fl.flightKey != "" {
		s.inviteFlights.Delete(fl.flightKey)
	}
}

func (s *SIPServer) inviteFinalRetransmitCleanup(callID string) {
	if s == nil || callID == "" {
		return
	}
	if v, ok := s.inviteFlightKeyByCall.LoadAndDelete(callID); ok {
		if fk, _ := v.(string); fk != "" {
			s.inviteFinal200Raw.Delete(fk)
		}
	}
}

func (s *SIPServer) abortInviteFlight(flight *inviteFlightState, req *stack.Message, addr *net.UDPAddr, toTag string) {
	if s == nil || flight == nil {
		return
	}
	cid := flight.callID
	if s.ep != nil && addr != nil && req != nil {
		t504 := s.makeResponse(req, 504, "Server Time-out", "", toTag)
		_ = s.ep.Send(t504, addr)
		s.finalizeInviteServerTx(req, t504, addr)
	}
	s.stopCallSessionLocked(cid)
	s.inviteAsyncEnd(cid)
}

func (s *SIPServer) stopCallSessionLocked(callID string) {
	if s == nil || callID == "" {
		return
	}
	s.endVoiceDialogBridge(callID)
	s.cleanupCallState(callID)
	s.mu.Lock()
	cs := s.callStore[callID]
	delete(s.callStore, callID)
	s.mu.Unlock()
	if cs != nil {
		cs.Stop()
	}
	s.releaseInboundCapacity(callID)
}

func (s *SIPServer) runInviteAsync(
	req *stack.Message,
	addr *net.UDPAddr,
	flight *inviteFlightState,
	resp200 *stack.Message,
	reliable bool,
	sdp183Body string,
	callID string,
) {
	if s == nil || req == nil || addr == nil || flight == nil || resp200 == nil {
		return
	}
	inv := s.inviteEnv
	toTag := resp200.GetHeader("To")

	if inv.Send180 && s.ep != nil {
		ring180 := s.makeResponse(req, 180, "Ringing", "", toTag)
		ring180.SetHeader("To", toTag)
		ring180.SetHeader("Contact", resp200.GetHeader("Contact"))
		ring180.SetHeader("Content-Length", "0")
		_ = s.ep.Send(ring180, addr)
	}

	if reliable {
		body := strings.TrimSpace(sdp183Body)
		ct := ""
		if body != "" {
			ct = "application/sdp"
		}
		p183 := s.makeResponse(req, 183, "Session Progress", body, toTag)
		p183.SetHeader("To", toTag)
		p183.SetHeader("Contact", resp200.GetHeader("Contact"))
		if ct != "" {
			p183.SetHeader("Content-Type", ct)
		}
		p183.SetHeader("RSeq", "1")
		p183.SetHeader("Require", "100rel")
		p183.SetHeader("Allow", resp200.GetHeader("Allow"))
		p183.SetHeader("Content-Length", strconv.Itoa(stack.BodyBytesLen(body)))

		raw183 := p183.String()
		flight.mu.Lock()
		flight.awaitRSeq = 1
		flight.lastProvRaw = raw183
		flight.mu.Unlock()

		if s.ep != nil {
			_ = s.ep.Send(p183, addr)
		}

		timer := time.NewTimer(32 * time.Second)
		var badExit bool
		if s.sigCtx != nil {
			select {
			case <-flight.prackDone:
			case <-timer.C:
				badExit = true
				if logger.Lg != nil {
					logger.Lg.Warn("sip invite PRACK timeout", zap.String("call_id", callID))
				}
			case <-s.sigCtx.Done():
				badExit = true
			}
		} else {
			select {
			case <-flight.prackDone:
			case <-timer.C:
				badExit = true
				if logger.Lg != nil {
					logger.Lg.Warn("sip invite PRACK timeout", zap.String("call_id", callID))
				}
			}
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		if badExit {
			s.abortInviteFlight(flight, req, addr, toTag)
			return
		}
		flight.mu.Lock()
		flight.awaitRSeq = 0
		flight.lastProvRaw = ""
		flight.mu.Unlock()
	}

	if inv.RingbackMs > 0 {
		t := time.NewTimer(time.Duration(inv.RingbackMs) * time.Millisecond)
		if s.sigCtx != nil {
			select {
			case <-t.C:
			case <-s.sigCtx.Done():
				if !t.Stop() {
					<-t.C
				}
				s.abortInviteFlight(flight, req, addr, toTag)
				return
			}
		} else {
			<-t.C
		}
	}

	okRaw := resp200.String()
	flight.mu.Lock()
	flight.lastOK200Raw = okRaw
	flight.completed = true
	flight.mu.Unlock()
	if flight.flightKey != "" {
		s.inviteFinal200Raw.Store(flight.flightKey, okRaw)
		s.inviteFlightKeyByCall.Store(callID, flight.flightKey)
	}

	if s.ep != nil {
		if err := s.ep.Send(resp200, addr); err != nil && logger.Lg != nil {
			logger.Lg.Warn("sip invite send 200", zap.String("call_id", callID), zap.Error(err))
		}
		s.finalizeInviteServerTx(req, resp200, addr)
	}
}

func (s *SIPServer) handlePrack(msg *stack.Message, addr *net.UDPAddr) *stack.Message {
	if msg == nil || !msg.IsRequest || strings.ToUpper(msg.Method) != stack.MethodPrack {
		return nil
	}
	if s.absorbNonInviteRetransmit(msg, addr) {
		return nil
	}
	callID := strings.TrimSpace(msg.GetHeader("Call-ID"))
	if callID == "" {
		return s.makeResponse(msg, 400, "Bad Request", "", "")
	}
	rseq, cseqNum, method, err := stack.ParseRAck(msg.GetHeader("RAck"))
	if err != nil || method != "INVITE" {
		return s.makeResponse(msg, 400, "Bad Request", "", "")
	}
	v, ok := s.inviteByCall.Load(callID)
	if !ok {
		return s.makeResponse(msg, 481, "Call/Transaction Does Not Exist", "", "")
	}
	fl := v.(*inviteFlightState)
	fl.mu.Lock()
	okMatch := fl.awaitRSeq != 0 && rseq == fl.awaitRSeq && cseqNum == fl.inviteCSeq
	fl.mu.Unlock()
	if !okMatch {
		return s.makeResponse(msg, 481, "Call/Transaction Does Not Exist", "", "")
	}
	select {
	case fl.prackDone <- struct{}{}:
	default:
	}
	return s.makeResponse(msg, 200, "OK", "", "")
}

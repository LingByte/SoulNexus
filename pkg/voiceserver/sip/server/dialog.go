package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

type uasDialogState struct {
	remote    *net.UDPAddr
	byeFrom   string // our To header from 200 OK (local tag)
	byeTo     string // remote From from INVITE
	byeReqURI string
	nextCSeq  int
}

func cloneUDP(a *net.UDPAddr) *net.UDPAddr {
	if a == nil {
		return nil
	}
	b := *a
	return &b
}

func parseInviteCSeqNext(cseq string) int {
	cseq = strings.TrimSpace(cseq)
	if cseq == "" {
		return 2
	}
	parts := strings.Fields(cseq)
	if len(parts) == 0 {
		return 2
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil || n < 1 {
		return 2
	}
	return n + 1
}

func requestURIFromContact(contact string) string {
	contact = strings.TrimSpace(contact)
	if contact == "" {
		return ""
	}
	c := contact
	if i := strings.Index(c, "<"); i >= 0 {
		c = c[i+1:]
	}
	if i := strings.Index(c, ">"); i >= 0 {
		c = c[:i]
	}
	c = strings.TrimSpace(c)
	if i := strings.Index(c, ";"); i > 0 {
		c = c[:i]
	}
	c = strings.TrimSpace(c)
	if c == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(c), "sip:") {
		c = "sip:" + c
	}
	return c
}

func (s *SIPServer) rememberUASDialog(callID string, remote *net.UDPAddr, inv *stack.Message, ourToWithTag string) {
	if s == nil || callID == "" || inv == nil || remote == nil {
		return
	}
	reqURI := requestURIFromContact(inv.GetHeader("Contact"))
	if reqURI == "" {
		reqURI = strings.TrimSpace(inv.RequestURI)
	}
	st := &uasDialogState{
		remote:    cloneUDP(remote),
		byeFrom:   strings.TrimSpace(ourToWithTag),
		byeTo:     inv.GetHeader("From"),
		byeReqURI: reqURI,
		nextCSeq:  parseInviteCSeqNext(inv.GetHeader("CSeq")),
	}
	s.dlgMu.Lock()
	if s.uasDlg == nil {
		s.uasDlg = make(map[string]*uasDialogState)
	}
	s.uasDlg[callID] = st
	s.dlgMu.Unlock()
}

func (s *SIPServer) forgetUASDialog(callID string) {
	if s == nil || callID == "" {
		return
	}
	s.dlgMu.Lock()
	delete(s.uasDlg, callID)
	s.dlgMu.Unlock()
}

// ForgetUASDialog clears stored UAS dialog state for Call-ID (e.g. after Web seat teardown).
func (s *SIPServer) ForgetUASDialog(callID string) {
	s.forgetUASDialog(callID)
}

func (s *SIPServer) buildUASBye(callID string) (*stack.Message, *net.UDPAddr, error) {
	if s == nil || callID == "" {
		return nil, nil, fmt.Errorf("sip: invalid hangup")
	}
	s.dlgMu.Lock()
	defer s.dlgMu.Unlock()
	d := s.uasDlg[callID]
	if d == nil || d.remote == nil {
		return nil, nil, fmt.Errorf("sip: no dialog for call-id")
	}
	if strings.TrimSpace(d.byeFrom) == "" || strings.TrimSpace(d.byeTo) == "" {
		return nil, nil, fmt.Errorf("sip: incomplete dialog headers")
	}
	cseq := d.nextCSeq
	d.nextCSeq++
	branch := randomHexBranch()
	via := fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=z9hG4bK%s;rport",
		strings.TrimSpace(s.localIP), s.listenPort, branch)
	msg := &stack.Message{
		IsRequest:  true,
		Method:     stack.MethodBye,
		RequestURI: d.byeReqURI,
		Version:    "SIP/2.0",
	}
	msg.SetHeader("Via", via)
	msg.SetHeader("Max-Forwards", "70")
	msg.SetHeader("From", d.byeFrom)
	msg.SetHeader("To", d.byeTo)
	msg.SetHeader("Call-ID", callID)
	msg.SetHeader("CSeq", fmt.Sprintf("%d BYE", cseq))
	msg.SetHeader("Content-Length", "0")
	return msg, cloneUDP(d.remote), nil
}

// buildReferNotify builds an in-dialog NOTIFY (Event: refer) with a message/sipfrag body.
func (s *SIPServer) buildReferNotify(callID string, sipfragBody string, subscriptionState string) (*stack.Message, *net.UDPAddr, error) {
	if s == nil || callID == "" {
		return nil, nil, fmt.Errorf("sip: invalid refer notify")
	}
	s.dlgMu.Lock()
	defer s.dlgMu.Unlock()
	d := s.uasDlg[callID]
	if d == nil || d.remote == nil {
		return nil, nil, fmt.Errorf("sip: no dialog for call-id")
	}
	if strings.TrimSpace(d.byeFrom) == "" || strings.TrimSpace(d.byeTo) == "" {
		return nil, nil, fmt.Errorf("sip: incomplete dialog headers")
	}
	cseq := d.nextCSeq
	d.nextCSeq++
	branch := randomHexBranch()
	via := fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=z9hG4bK%s;rport",
		strings.TrimSpace(s.localIP), s.listenPort, branch)
	msg := &stack.Message{
		IsRequest:  true,
		Method:     stack.MethodNotify,
		RequestURI: d.byeReqURI,
		Version:    "SIP/2.0",
	}
	msg.SetHeader("Via", via)
	msg.SetHeader("Max-Forwards", "70")
	msg.SetHeader("From", d.byeFrom)
	msg.SetHeader("To", d.byeTo)
	msg.SetHeader("Call-ID", callID)
	msg.SetHeader("CSeq", fmt.Sprintf("%d NOTIFY", cseq))
	msg.SetHeader("Event", "refer")
	if strings.TrimSpace(subscriptionState) == "" {
		subscriptionState = "active;expires=60"
	}
	msg.SetHeader("Subscription-State", subscriptionState)
	msg.SetHeader("Content-Type", "message/sipfrag;version=2.0")
	msg.Body = strings.TrimRight(strings.TrimSpace(sipfragBody), "\r\n") + "\r\n"
	msg.SetHeader("Content-Length", strconv.Itoa(stack.BodyBytesLen(msg.Body)))
	return msg, cloneUDP(d.remote), nil
}

func randomHexBranch() string {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "deadbeef01"
	}
	return hex.EncodeToString(b)
}

// SendUASBye sends BYE to the remote party for an inbound UAS dialog (no local RTP teardown).
func (s *SIPServer) SendUASBye(callID string) error {
	msg, dst, err := s.buildUASBye(callID)
	if err != nil {
		return err
	}
	if err := s.SendSIP(msg, dst); err != nil {
		return err
	}
	s.forgetUASDialog(callID)
	return nil
}

// HangupInboundCall ends an inbound leg: transfer bridge (BYE both sides), or AI call (BYE + teardown).
func (s *SIPServer) HangupInboundCall(callID string) {
	if s == nil || callID == "" {
		return
	}
	s.releaseInboundCapacity(callID)
	defer s.endVoiceDialogBridge(callID)
	defer s.cleanupCallState(callID)
	if s.preHangupCallState(callID) {
		// Business observer claimed the hangup (transfer bridge / WebSeat / etc).
		return
	}
	_ = s.SendUASBye(callID)

	s.mu.Lock()
	cs := s.callStore[callID]
	delete(s.callStore, callID)
	s.mu.Unlock()

	var codec string
	var recSR, recOpusCh int
	if cs != nil {
		codec = cs.NegotiatedSDP().Name
		src := cs.SourceCodec()
		recSR = src.SampleRate
		recOpusCh = src.OpusDecodeChannels
		if recOpusCh < 1 {
			recOpusCh = src.Channels
		}
		cs.Stop()
	}
	s.forgetUASDialog(callID)
	if p := s.callPersistStore(); p != nil {
		go p.OnBye(context.Background(), ByePersistParams{
			CallID:             callID,
			RawPayload:         nil, // recording is now a business-side responsibility
			CodecName:          codec,
			Initiator:          "local",
			RecordSampleRate:   recSR,
			RecordOpusChannels: recOpusCh,
		})
	}
}

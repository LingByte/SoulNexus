package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/sippersist"
	"github.com/LingByte/SoulNexus/pkg/sip/conversation"
	"github.com/LingByte/SoulNexus/pkg/sip/protocol"
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

func (s *SIPServer) rememberUASDialog(callID string, remote *net.UDPAddr, inv *protocol.Message, ourToWithTag string) {
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

func (s *SIPServer) buildUASBye(callID string) (*protocol.Message, *net.UDPAddr, error) {
	if s == nil || callID == "" {
		return nil, nil, fmt.Errorf("sip: invalid hangup")
	}
	s.dlgMu.RLock()
	d := s.uasDlg[callID]
	s.dlgMu.RUnlock()
	if d == nil || d.remote == nil {
		return nil, nil, fmt.Errorf("sip: no dialog for call-id")
	}
	if strings.TrimSpace(d.byeFrom) == "" || strings.TrimSpace(d.byeTo) == "" {
		return nil, nil, fmt.Errorf("sip: incomplete dialog headers")
	}
	branch := randomHexBranch()
	via := fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=z9hG4bK%s;rport",
		strings.TrimSpace(s.localIP), s.listenPort, branch)
	msg := &protocol.Message{
		IsRequest:  true,
		Method:     protocol.MethodBye,
		RequestURI: d.byeReqURI,
		Version:    "SIP/2.0",
	}
	msg.SetHeader("Via", via)
	msg.SetHeader("Max-Forwards", "70")
	msg.SetHeader("From", d.byeFrom)
	msg.SetHeader("To", d.byeTo)
	msg.SetHeader("Call-ID", callID)
	msg.SetHeader("CSeq", fmt.Sprintf("%d BYE", d.nextCSeq))
	msg.SetHeader("Content-Length", "0")
	return msg, d.remote, nil
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
	if conversation.HangupWebSeatBridgeFull(callID) {
		return
	}
	if tb := conversation.HangupTransferBridgeFull(callID); tb != nil {
		if p := s.callPersistStore(); p != nil {
			go p.OnBye(context.Background(), sippersist.ByeParams{
				CallID:             tb.InboundCallID,
				RawPayload:         tb.RawPayload,
				CodecName:          tb.CodecName,
				Initiator:          tb.Initiator,
				RecordSampleRate:   tb.RecordSampleRate,
				RecordOpusChannels: tb.RecordOpusChannels,
			})
		}
		return
	}
	_ = s.SendUASBye(callID)

	s.mu.Lock()
	cs := s.callStore[callID]
	delete(s.callStore, callID)
	s.mu.Unlock()

	var raw []byte
	var codec string
	var recSR, recOpusCh int
	if cs != nil {
		raw = cs.TakeRecording()
		codec = cs.NegotiatedCodec().Name
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
		go p.OnBye(context.Background(), sippersist.ByeParams{
			CallID:             callID,
			RawPayload:         raw,
			CodecName:          codec,
			Initiator:          "local",
			RecordSampleRate:   recSR,
			RecordOpusChannels: recOpusCh,
		})
	}
}

package server

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// Minimal in-memory presence broker: SUBSCRIBE (Event: presence) + PUBLISH (application/pidf+xml)
// with NOTIFY fan-out. Suitable for single-process / lab deployments.

type presenceSub struct {
	presentity string
	remote     *net.UDPAddr
	req        *stack.Message // SUBSCRIBE for header echo
	expiresSec int
	deadline   time.Time
	notifyCSeq int
}

type presenceBroker struct {
	mu      sync.Mutex
	publish map[string]string // presentity URI (lowercased) -> PIDF/XML body
	subs    []*presenceSub
}

var globalPresence = &presenceBroker{
	publish: make(map[string]string),
}

func presentityKeyFromSubscribe(msg *stack.Message) string {
	if msg == nil {
		return ""
	}
	u := strings.TrimSpace(msg.RequestURI)
	if u == "" {
		u = strings.TrimSpace(msg.GetHeader("To"))
		if i := strings.Index(u, "<"); i >= 0 {
			if j := strings.Index(u[i+1:], ">"); j >= 0 {
				u = strings.TrimSpace(u[i+1 : i+1+j])
			}
		}
	}
	return strings.ToLower(strings.TrimSpace(u))
}

func parseSubscribeExpires(msg *stack.Message) int {
	if msg == nil {
		return 3600
	}
	if e := strings.TrimSpace(msg.GetHeader("Expires")); e != "" {
		if n, err := strconv.Atoi(e); err == nil && n >= 0 {
			if n == 0 {
				return 0
			}
			if n > 3600 {
				return 3600
			}
			return n
		}
	}
	return 3600
}

func (s *SIPServer) handleSubscribe(msg *stack.Message, addr *net.UDPAddr) *stack.Message {
	if s == nil || msg == nil {
		return nil
	}
	if s.absorbNonInviteRetransmit(msg, addr) {
		return nil
	}
	ev := strings.ToLower(strings.TrimSpace(msg.GetHeader("Event")))
	if ev == "" {
		ev = "presence"
	}
	if !strings.HasPrefix(ev, "presence") {
		resp := s.makeResponse(msg, 489, "Bad Event", "", "")
		resp.SetHeader("Content-Length", "0")
		return resp
	}
	key := presentityKeyFromSubscribe(msg)
	if key == "" {
		return s.makeResponse(msg, 400, "Bad Request", "", "")
	}
	exp := parseSubscribeExpires(msg)
	if exp == 0 {
		globalPresence.mu.Lock()
		globalPresence.pruneSubByCallID(strings.TrimSpace(msg.GetHeader("Call-ID")))
		globalPresence.mu.Unlock()
		r := s.makeResponse(msg, 200, "OK", "", "")
		r.SetHeader("Expires", "0")
		r.SetHeader("Subscription-State", "terminated;reason=timeout")
		r.SetHeader("Content-Length", "0")
		return r
	}

	nC := stack.ParseCSeqNum(msg.GetHeader("CSeq"))
	if nC < 1 {
		nC = 1
	}
	globalPresence.mu.Lock()
	globalPresence.subs = append(globalPresence.subs, &presenceSub{
		presentity: key,
		remote:     cloneUDPAddr(addr),
		req:        cloneMsgShallow(msg),
		expiresSec: exp,
		deadline:   time.Now().Add(time.Duration(exp) * time.Second),
		notifyCSeq: nC + 1,
	})
	body := globalPresence.publish[key]
	globalPresence.mu.Unlock()

	r := s.makeResponse(msg, 200, "OK", body, "")
	r.SetHeader("Expires", strconv.Itoa(exp))
	r.SetHeader("Subscription-State", "active;expires="+strconv.Itoa(exp))
	if strings.TrimSpace(body) != "" {
		r.SetHeader("Content-Type", "application/pidf+xml")
	}
	return r
}

func (s *SIPServer) handleNotifyPresence(msg *stack.Message, addr *net.UDPAddr) *stack.Message {
	if s == nil || msg == nil {
		return nil
	}
	if s.absorbNonInviteRetransmit(msg, addr) {
		return nil
	}
	// In-dialog NOTIFY as subscriber UA: accept.
	r := s.makeResponse(msg, 200, "OK", "", "")
	r.SetHeader("Content-Length", "0")
	return r
}

func (b *presenceBroker) pruneSubByCallID(callID string) {
	if b == nil || callID == "" {
		return
	}
	var out []*presenceSub
	for _, s := range b.subs {
		if s == nil || s.req == nil {
			continue
		}
		if strings.TrimSpace(s.req.GetHeader("Call-ID")) == callID {
			continue
		}
		out = append(out, s)
	}
	b.subs = out
}

func (b *presenceBroker) notifyPresentity(srv *SIPServer, presentity string, body string, contentType string) {
	if b == nil || srv == nil || srv.ep == nil {
		return
	}
	key := strings.ToLower(strings.TrimSpace(presentity))
	if key == "" {
		return
	}
	type sendPair struct {
		m    *stack.Message
		addr *net.UDPAddr
	}
	var batch []sendPair
	b.mu.Lock()
	now := time.Now()
	for _, sub := range b.subs {
		if sub == nil || sub.remote == nil || sub.req == nil {
			continue
		}
		if sub.presentity != key {
			continue
		}
		if now.After(sub.deadline) {
			continue
		}
		cseq := sub.notifyCSeq
		if cseq < 1 {
			cseq = 2
		}
		sub.notifyCSeq = cseq + 1
		n := srv.buildNotifyForSubscription(sub, cseq, body, contentType)
		if n != nil {
			batch = append(batch, sendPair{m: n, addr: cloneUDPAddr(sub.remote)})
		}
	}
	b.mu.Unlock()
	for _, p := range batch {
		_ = srv.ep.Send(p.m, p.addr)
	}
}

func (s *SIPServer) buildNotifyForSubscription(sub *presenceSub, cseq int, body string, contentType string) *stack.Message {
	if s == nil || sub == nil || sub.req == nil {
		return nil
	}
	req := sub.req
	n := &stack.Message{
		IsRequest:  true,
		Method:     stack.MethodNotify,
		RequestURI: strings.TrimSpace(req.RequestURI),
		Version:    "SIP/2.0",
	}
	if n.RequestURI == "" {
		n.RequestURI = "sip:presence"
	}
	branch := randomHexBranch()
	via := "SIP/2.0/UDP " + strings.TrimSpace(s.localIP) + ":" + strconv.Itoa(s.listenPort) + ";branch=z9hG4bK" + branch + ";rport"
	n.SetHeader("Via", via)
	n.SetHeader("Max-Forwards", "70")
	n.SetHeader("From", req.GetHeader("To"))
	n.SetHeader("To", req.GetHeader("From"))
	n.SetHeader("Call-ID", req.GetHeader("Call-ID"))
	exp := sub.expiresSec
	if exp <= 0 {
		exp = 3600
	}
	n.SetHeader("CSeq", fmt.Sprintf("%d NOTIFY", cseq))
	n.SetHeader("Event", "presence")
	n.SetHeader("Subscription-State", "active;expires="+strconv.Itoa(exp))
	if strings.TrimSpace(body) != "" {
		if contentType == "" {
			contentType = "application/pidf+xml"
		}
		n.SetHeader("Content-Type", contentType)
		n.Body = body
	}
	n.SetHeader("Content-Length", strconv.Itoa(stack.BodyBytesLen(n.Body)))
	return n
}

func cloneUDPAddr(a *net.UDPAddr) *net.UDPAddr {
	if a == nil {
		return nil
	}
	b := *a
	return &b
}

func (s *SIPServer) handlePublishPresence(msg *stack.Message, addr *net.UDPAddr) *stack.Message {
	if s == nil || msg == nil {
		return nil
	}
	if s.absorbNonInviteRetransmit(msg, addr) {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(msg.RequestURI))
	if key == "" {
		return s.makeResponse(msg, 400, "Bad Request", "", "")
	}
	body := msg.Body
	ct := strings.TrimSpace(msg.GetHeader("Content-Type"))
	globalPresence.mu.Lock()
	if globalPresence.publish == nil {
		globalPresence.publish = make(map[string]string)
	}
	globalPresence.publish[key] = body
	globalPresence.mu.Unlock()
	globalPresence.notifyPresentity(s, key, body, ct)
	resp := s.makeResponse(msg, 200, "OK", "", "")
	resp.SetHeader("Content-Length", "0")
	return resp
}

func cloneMsgShallow(m *stack.Message) *stack.Message {
	if m == nil {
		return nil
	}
	cp := *m
	cp.Headers = make(map[string]string, len(m.Headers))
	for k, v := range m.Headers {
		cp.Headers[k] = v
	}
	cp.HeadersMulti = make(map[string][]string, len(m.HeadersMulti))
	for k, v := range m.HeadersMulti {
		cp.HeadersMulti[k] = append([]string(nil), v...)
	}
	cp.Body = m.Body
	return &cp
}

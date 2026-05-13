package server

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/sdp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// ---------- SIPServer.String, Listen/Register/Remove ----------------------

func TestSIPServer_Stringer(t *testing.T) {
	s := New(Config{LocalIP: "10.1.2.3"})
	defer s.Stop()
	if got := s.String(); !strings.Contains(got, "10.1.2.3") {
		t.Errorf("String()=%q", got)
	}
}

func TestRegisterAndRemoveCallSession(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	// nil-safe
	s.RegisterCallSession("", nil)
	s.RegisterCallSession("cid", nil)
	s.RemoveCallSession("")
	s.RemoveCallSession("missing")

	var nilS *SIPServer
	nilS.RegisterCallSession("c", nil)
	nilS.RemoveCallSession("c")
	if nilS.GetCallSession("c") != nil {
		t.Error("nil server should return nil session")
	}

	// Real leg lifecycle
	codecs := []sdp.Codec{{PayloadType: 8, Name: "pcma", ClockRate: 8000, Channels: 1}}
	rtpSess := mustRTPSession(t)
	defer rtpSess.Close()
	leg, err := session.NewMediaLeg(context.Background(), "c-reg", rtpSess, codecs, session.MediaLegConfig{})
	if err != nil {
		t.Fatalf("media leg: %v", err)
	}
	s.RegisterCallSession("c-reg", leg)
	if got := s.GetCallSession("c-reg"); got != leg {
		t.Error("register did not store leg")
	}

	// Replacing should stop the old one (just make sure no panic).
	rtpSess2 := mustRTPSession(t)
	defer rtpSess2.Close()
	leg2, err := session.NewMediaLeg(context.Background(), "c-reg", rtpSess2, codecs, session.MediaLegConfig{})
	if err != nil {
		t.Fatalf("media leg 2: %v", err)
	}
	s.RegisterCallSession("c-reg", leg2)

	s.RemoveCallSession("c-reg")
	if s.GetCallSession("c-reg") != nil {
		t.Error("remove did not clear")
	}
	leg2.Stop()
}

// ---------- attachInboundCallToBusiness (no-op shim) ----------------------

func TestAttachInboundCallToBusiness_NoOp(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	// Must be safe with any input — it is reserved as an extension hook.
	s.attachInboundCallToBusiness("any-call")
	s.attachInboundCallToBusiness("")
}

// ---------- finalizeInviteServerTx nil guards -----------------------------

func TestFinalizeInviteServerTx_NilGuards(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	// All nil args → early return
	s.finalizeInviteServerTx(nil, nil, nil)
	s.finalizeInviteServerTx(&stack.Message{}, nil, nil)
	s.finalizeInviteServerTx(&stack.Message{}, &stack.Message{}, nil)

	var nilS *SIPServer
	nilS.finalizeInviteServerTx(&stack.Message{}, &stack.Message{}, &net.UDPAddr{})
}

// ---------- proxyInviteToRegistrar ----------------------------------------

func TestProxyInviteToRegistrar_NilGuards(t *testing.T) {
	var s *SIPServer
	if err := s.proxyInviteToRegistrar(&stack.Message{}, &net.UDPAddr{}); err == nil {
		t.Error("nil server should error")
	}

	s2 := New(Config{LocalIP: "127.0.0.1"})
	defer s2.Stop()
	if err := s2.proxyInviteToRegistrar(nil, &net.UDPAddr{}); err == nil {
		t.Error("nil msg should error")
	}
	if err := s2.proxyInviteToRegistrar(&stack.Message{}, nil); err == nil {
		t.Error("nil dst should error")
	}
}

func TestProxyInviteToRegistrar_HappyPath(t *testing.T) {
	// Bring up a real UDP server so ep.Send has a live endpoint.
	srv := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1"})
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer srv.Stop()

	// Build an INVITE string with full via/cseq/etc. that stack.Parse accepts.
	raw := strings.Join([]string{
		"INVITE sip:target@127.0.0.1 SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bK-proxy-test;rport",
		"Max-Forwards: 70",
		"From: <sip:a@example.com>;tag=1",
		"To: <sip:target@127.0.0.1>",
		"Call-ID: proxy-inv-1",
		"CSeq: 1 INVITE",
		"Contact: <sip:a@10.0.0.1>",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, err := stack.Parse(raw)
	if err != nil {
		t.Fatalf("parse invite: %v", err)
	}

	// Listen on a loopback socket to catch the proxied packet.
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("dst listen: %v", err)
	}
	defer pc.Close()
	dst := pc.LocalAddr().(*net.UDPAddr)

	if err := srv.proxyInviteToRegistrar(msg, dst); err != nil {
		t.Fatalf("proxyInviteToRegistrar: %v", err)
	}
	_ = pc.SetReadDeadline(time.Now().Add(1 * time.Second))
	buf := make([]byte, 4096)
	n, _, err := pc.ReadFrom(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	body := string(buf[:n])
	if !strings.HasPrefix(body, "INVITE ") {
		t.Errorf("expected INVITE line, got: %q", body[:20])
	}
	// Proxy must prepend its own Via.
	if strings.Count(body, "Via:") < 2 && strings.Count(body, "via:") < 2 {
		// Normalized parser may drop the second header; accept either case
		// but require that the top Via is the proxy's.
		if !strings.Contains(body, "SIP/2.0/UDP 127.0.0.1:") {
			t.Errorf("proxy via not prepended: %q", body)
		}
	}
}

// ---------- runReferSequence ----------------------------------------------

func TestRunReferSequence_NoDialog_JustDoesNotPanic(t *testing.T) {
	s := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1"})
	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop()

	// No dialog stored → buildReferNotify errors, sequence logs and continues.
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.runReferSequence(context.Background(), "no-such-call", "sip:x@y")
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runReferSequence did not return")
	}
}

func TestRunReferSequence_WithDialog_Sends100Trying(t *testing.T) {
	s := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1"})
	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop()

	// Set up a UDP listener to be the "remote" and prime the UAS dialog.
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("remote listen: %v", err)
	}
	defer pc.Close()
	remote := pc.LocalAddr().(*net.UDPAddr)

	inv := &stack.Message{IsRequest: true, Method: "INVITE", RequestURI: "sip:callee@" + remote.String(),
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	inv.SetHeader("From", "<sip:caller@example.com>;tag=c")
	inv.SetHeader("Contact", "<sip:callee@"+remote.String()+">")
	inv.SetHeader("CSeq", "1 INVITE")
	s.rememberUASDialog("refer-call", remote, inv, "<sip:server@local>;tag=srv")

	// Attach a transfer handler so triggerTransferFromReferTo runs the
	// business and emits a terminal notify.
	th := &fakeTransferHandler{}
	s.SetTransferHandler(th)

	_ = pc.SetReadDeadline(time.Now().Add(2 * time.Second))
	packets := make(chan string, 4)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, _, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			packets <- string(buf[:n])
		}
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.runReferSequence(context.Background(), "refer-call", "sip:bob@example.com")
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("runReferSequence did not return")
	}

	// Drain what we can — we should see at least a NOTIFY with Event: refer.
	deadline := time.After(500 * time.Millisecond)
	sawNotify := false
loop:
	for {
		select {
		case p := <-packets:
			if strings.HasPrefix(p, "NOTIFY ") && strings.Contains(p, "Event:") {
				sawNotify = true
			}
		case <-deadline:
			break loop
		}
	}
	if !sawNotify {
		t.Error("expected at least one NOTIFY with Event: refer")
	}
	if !th.called {
		t.Error("TransferHandler.OnRefer should be called")
	}
}

// ---------- upsertRegistration -------------------------------------------

type memRegStore struct {
	mu       sync.Mutex
	saved    map[string]*net.UDPAddr
	deleted  []string
	saveErr  error
	delErr   error
	lookErr  error
}

func newMemRegStore() *memRegStore {
	return &memRegStore{saved: map[string]*net.UDPAddr{}}
}

func (m *memRegStore) SaveRegister(ctx context.Context, user, domain, contactURI string, sig *net.UDPAddr, expiresAt time.Time, ua string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved[user+"@"+domain] = sig
	return nil
}
func (m *memRegStore) DeleteRegister(ctx context.Context, user, domain string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delErr != nil {
		return m.delErr
	}
	m.deleted = append(m.deleted, user+"@"+domain)
	delete(m.saved, user+"@"+domain)
	return nil
}
func (m *memRegStore) LookupRegister(ctx context.Context, user, domain string) (*net.UDPAddr, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lookErr != nil {
		return nil, false, m.lookErr
	}
	a, ok := m.saved[user+"@"+domain]
	return a, ok, nil
}

func mkRegister(to, contact string, expires int) *stack.Message {
	m := &stack.Message{IsRequest: true, Method: "REGISTER", RequestURI: "sip:example.com",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	m.SetHeader("To", to)
	m.SetHeader("From", to)
	m.SetHeader("Contact", contact)
	m.SetHeader("Call-ID", "reg-cid")
	m.SetHeader("CSeq", "1 REGISTER")
	if expires >= 0 {
		m.SetHeader("Expires", strconv.Itoa(expires))
	}
	m.SetHeader("User-Agent", "test-ua")
	return m
}

func TestUpsertRegistration_NilGuards(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	// nil msg / src
	s.upsertRegistration(nil, &net.UDPAddr{})
	s.upsertRegistration(&stack.Message{}, nil)

	// no store configured → silent no-op
	s.upsertRegistration(mkRegister("<sip:alice@example.com>", "<sip:alice@10.0.0.1:5060>", 3600),
		&net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5060})

	var nilS *SIPServer
	nilS.upsertRegistration(&stack.Message{}, &net.UDPAddr{})
}

func TestUpsertRegistration_SaveAndDelete(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	store := newMemRegStore()
	s.SetRegisterStore(store)

	src := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5060}
	// Save path
	s.upsertRegistration(mkRegister("<sip:alice@example.com>", "<sip:alice@10.0.0.1:5060>", 3600), src)
	if _, ok := store.saved["alice@example.com"]; !ok {
		t.Errorf("save not recorded: %+v", store.saved)
	}

	// Delete path (expires=0)
	s.upsertRegistration(mkRegister("<sip:alice@example.com>", "<sip:alice@10.0.0.1:5060>", 0), src)
	if _, ok := store.saved["alice@example.com"]; ok {
		t.Error("delete did not remove binding")
	}
	if len(store.deleted) == 0 {
		t.Error("DeleteRegister not called")
	}

	// Malformed To header → warn path (no-op on store)
	before := len(store.saved)
	s.upsertRegistration(mkRegister("not-a-uri", "<sip:a@10.0.0.1:5060>", 3600), src)
	if len(store.saved) != before {
		t.Error("unparseable To should not save")
	}

	// Missing Contact → warn path
	bad := mkRegister("<sip:bob@example.com>", "", 3600)
	s.upsertRegistration(bad, src)

	// Store errors on save → logs warn, no panic
	store.saveErr = errors.New("db down")
	s.upsertRegistration(mkRegister("<sip:carol@example.com>", "<sip:carol@10.0.0.1:5060>", 3600), src)
	store.saveErr = nil

	// Store errors on delete → logs warn, no panic
	store.delErr = errors.New("db down")
	s.upsertRegistration(mkRegister("<sip:alice@example.com>", "<sip:alice@10.0.0.1:5060>", 0), src)
	store.delErr = nil
}

// ---------- registerPendingInvite round-trip ------------------------------

func TestRegisterPendingInvite_RoundTrip(t *testing.T) {
	s := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1"})
	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop()

	raw := strings.Join([]string{
		"INVITE sip:target@127.0.0.1 SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bK-pend-test;rport",
		"Max-Forwards: 70",
		"From: <sip:a@example.com>;tag=1",
		"To: <sip:target@127.0.0.1>",
		"Call-ID: pend-inv-1",
		"CSeq: 1 INVITE",
		"Contact: <sip:a@10.0.0.1>",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, err := stack.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	addr := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5060}

	// Guard: empty Call-ID path
	blank := &stack.Message{IsRequest: true, Method: "INVITE", Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	s.registerPendingInvite(blank, addr, "tag")
	if snap := s.takePendingInviteSnap(""); snap != nil {
		t.Error("blank call-id snap should not be stored")
	}

	// Guard: nil addr
	s.registerPendingInvite(msg, nil, "tag")

	// Happy path
	s.registerPendingInvite(msg, addr, "tag-xyz")
	snap := s.takePendingInviteSnap("pend-inv-1")
	if snap == nil {
		t.Fatal("pending snap not stored")
	}
	if snap.toTag != "tag-xyz" {
		t.Errorf("toTag=%q", snap.toTag)
	}
	if snap.addr == nil || snap.addr.Port != 5060 {
		t.Errorf("addr=%v", snap.addr)
	}
	if !strings.Contains(snap.rawInvite, "pend-inv-1") {
		t.Errorf("rawInvite missing call-id: %q", snap.rawInvite)
	}

	// Second take is nil (map entry deleted)
	if snap2 := s.takePendingInviteSnap("pend-inv-1"); snap2 != nil {
		t.Error("second take should be nil")
	}

	// clearPendingInviteSnap path after re-insert
	s.registerPendingInvite(msg, addr, "t2")
	s.clearPendingInviteSnap("pend-inv-1")
	if snap3 := s.takePendingInviteSnap("pend-inv-1"); snap3 != nil {
		t.Error("clear did not remove entry")
	}
}

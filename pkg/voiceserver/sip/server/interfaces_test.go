package server

import (
	"context"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/rtp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/sdp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// ---------- fake business implementations ----------------------------------

type fakeInviteHandler struct {
	mu       sync.Mutex
	calls    int
	decision Decision
	err      error
	last     *IncomingCall
}

func (f *fakeInviteHandler) OnIncomingCall(ctx context.Context, in *IncomingCall) (Decision, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.last = in
	return f.decision, f.err
}

func (f *fakeInviteHandler) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type fakeDTMFSink struct {
	mu     sync.Mutex
	events []string
}

func (f *fakeDTMFSink) OnDTMF(callID, digit string, end bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, callID+":"+digit)
}

func (f *fakeDTMFSink) all() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.events))
	copy(out, f.events)
	return out
}

type fakeTransferHandler struct {
	mu     sync.Mutex
	called bool
	last   string
}

func (f *fakeTransferHandler) OnRefer(ctx context.Context, callID, referTo string, notify func(frag, subState string)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.called = true
	f.last = referTo
	if notify != nil {
		notify("SIP/2.0 200 OK", "terminated;reason=noresource")
	}
}

type fakeObserver struct {
	mu          sync.Mutex
	cleanupHits int
	preHangup   bool
	preHits     int
}

func (f *fakeObserver) OnCallPreHangup(callID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.preHits++
	return f.preHangup
}

func (f *fakeObserver) OnCallCleanup(callID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cleanupHits++
}

// ---------- Setter nil-safety + round-trip ------------------------------

func TestSetters_NilServerNoPanic(t *testing.T) {
	var s *SIPServer
	s.SetInviteHandler(nil)
	s.SetDTMFSink(nil)
	s.SetTransferHandler(nil)
	s.SetCallLifecycleObserver(nil)
	if s.inviteHandlerImpl() != nil {
		t.Error("nil server inviteHandlerImpl must be nil")
	}
	if s.dtmfSinkImpl() != nil {
		t.Error("nil server dtmfSinkImpl must be nil")
	}
	if s.transferHandlerImpl() != nil {
		t.Error("nil server transferHandlerImpl must be nil")
	}
	if s.callObserverImpl() != nil {
		t.Error("nil server callObserverImpl must be nil")
	}
}

func TestSetters_RoundTrip(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	ih := &fakeInviteHandler{}
	ds := &fakeDTMFSink{}
	th := &fakeTransferHandler{}
	obs := &fakeObserver{}

	s.SetInviteHandler(ih)
	s.SetDTMFSink(ds)
	s.SetTransferHandler(th)
	s.SetCallLifecycleObserver(obs)

	if s.inviteHandlerImpl() != ih {
		t.Error("invite handler not stored")
	}
	if s.dtmfSinkImpl() != ds {
		t.Error("dtmf sink not stored")
	}
	if s.transferHandlerImpl() != th {
		t.Error("transfer handler not stored")
	}
	if s.callObserverImpl() != obs {
		t.Error("observer not stored")
	}
}

// ---------- compat dispatch behaviour ------------------------------------

func TestCompat_CleanupCallStateFiresObserverAndTerminate(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	obs := &fakeObserver{}
	s.SetCallLifecycleObserver(obs)

	terminated := int32(0)
	s.rememberTerminateHook("c1", func(reason string) {
		atomic.AddInt32(&terminated, 1)
	})

	s.cleanupCallState("c1")
	if obs.cleanupHits != 1 {
		t.Errorf("observer cleanup hits = %d, want 1", obs.cleanupHits)
	}
	if atomic.LoadInt32(&terminated) != 1 {
		t.Error("OnTerminate not fired")
	}
	// Empty / whitespace callID is ignored
	s.cleanupCallState("")
	s.cleanupCallState("   ")
}

func TestCompat_CleanupNoObserverIsSafe(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	s.cleanupCallState("missing-call") // must not panic
}

func TestCompat_PreHangupRoutesToObserver(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	// no observer → false
	if s.preHangupCallState("c1") {
		t.Error("no observer → false expected")
	}
	// empty callID → false
	if s.preHangupCallState("") {
		t.Error("empty callID → false")
	}

	obs := &fakeObserver{preHangup: true}
	s.SetCallLifecycleObserver(obs)
	if !s.preHangupCallState("c1") {
		t.Error("observer.preHangup=true should propagate")
	}
	if obs.preHits != 1 {
		t.Errorf("preHits = %d", obs.preHits)
	}
}

func TestCompat_TriggerTransferDispatch(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	// without handler, notify should still be called once with terminated;…
	notifyCalls := []string{}
	s.triggerTransferFromReferTo(context.Background(), "c1", "sip:x@y", func(frag, subState string) {
		notifyCalls = append(notifyCalls, subState)
	})
	if len(notifyCalls) == 0 {
		t.Error("notify should be called even without TransferHandler")
	}

	// with handler, the handler decides
	th := &fakeTransferHandler{}
	s.SetTransferHandler(th)
	s.triggerTransferFromReferTo(context.Background(), "c2", "sip:y@z", func(frag, subState string) {})
	if !th.called || th.last != "sip:y@z" {
		t.Errorf("handler not invoked: called=%v last=%q", th.called, th.last)
	}
}

func TestCompat_HandleSIPINFODTMF(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	// no sink → no-op
	s.handleSIPINFODTMF("c1", "application/dtmf-relay", "Signal=5\r\n")

	ds := &fakeDTMFSink{}
	s.SetDTMFSink(ds)
	s.handleSIPINFODTMF("c2", "application/dtmf-relay", "Signal=7\r\nDuration=160")
	s.handleSIPINFODTMF("c3", "application/dtmf-relay", "no signal here") // unparseable → drop

	got := ds.all()
	if len(got) != 1 || got[0] != "c2:7" {
		t.Errorf("got %v", got)
	}
}

func TestCompat_OnTerminateFiresOnce(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	count := int32(0)
	s.rememberTerminateHook("c1", func(reason string) {
		atomic.AddInt32(&count, 1)
	})
	s.fireOnTerminate("c1", "first")
	s.fireOnTerminate("c1", "second") // already fired+removed → no-op
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("OnTerminate fired %d times, want 1", count)
	}
	// empty/missing IDs safe
	s.fireOnTerminate("", "x")
	s.fireOnTerminate("nope", "x")
	s.rememberTerminateHook("", nil) // ignored
	s.rememberTerminateHook("c2", nil) // ignored
}

func TestCompat_EndVoiceDialogBridgeFiresTerminateAndClearsBrief(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	s.storeInviteBrief("c1", "from", "to", "1.2.3.4:5060")
	count := int32(0)
	s.rememberTerminateHook("c1", func(reason string) { atomic.AddInt32(&count, 1) })

	s.endVoiceDialogBridge("c1")
	if atomic.LoadInt32(&count) != 1 {
		t.Error("terminate not fired")
	}
	if from, _, _ := s.peekInviteBrief("c1"); from != "" {
		t.Error("invite brief should be cleared")
	}
	// Empty ID is no-op
	s.endVoiceDialogBridge("")
}

// ---------- buildIncomingCallLeg dispatch -------------------------------

func TestIncoming_DefaultLegWhenNoHandler(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	rtpSess := mustRTPSession(t)
	defer rtpSess.Close()

	offer := &sdp.Info{IP: "127.0.0.1", Port: 6000, Codecs: []sdp.Codec{
		{PayloadType: 8, Name: "pcma", ClockRate: 8000, Channels: 1},
	}}
	leg, err := s.buildIncomingCallLeg(context.Background(), "c1", &stack.Message{IsRequest: true, Method: "INVITE"}, nil, offer, rtpSess)
	if err != nil {
		t.Fatalf("buildIncomingCallLeg default: %v", err)
	}
	if leg == nil {
		t.Fatal("leg nil with no handler")
	}
	leg.Stop()
}

func TestIncoming_HandlerAcceptWithMediaLeg(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	rtpSess := mustRTPSession(t)
	defer rtpSess.Close()

	offer := &sdp.Info{IP: "127.0.0.1", Port: 6000, Codecs: []sdp.Codec{
		{PayloadType: 0, Name: "pcmu", ClockRate: 8000, Channels: 1},
	}}

	// Build a leg the business would have crafted itself.
	preBuilt, err := session.NewMediaLeg(context.Background(), "c1", rtpSess, offer.Codecs, session.MediaLegConfig{})
	if err != nil {
		t.Fatalf("preBuilt MediaLeg: %v", err)
	}
	defer preBuilt.Stop()

	terminated := int32(0)
	h := &fakeInviteHandler{
		decision: Decision{
			Accept:      true,
			MediaLeg:    preBuilt,
			OnTerminate: func(reason string) { atomic.AddInt32(&terminated, 1) },
		},
	}
	s.SetInviteHandler(h)

	inviteMsg := &stack.Message{IsRequest: true, Method: "INVITE",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	inviteMsg.SetHeader("From", "f")
	inviteMsg.SetHeader("To", "t")
	leg, err := s.buildIncomingCallLeg(context.Background(), "c1", inviteMsg,
		&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5060},
		offer, rtpSess)
	if err != nil {
		t.Fatalf("buildIncomingCallLeg accept: %v", err)
	}
	if leg != preBuilt {
		t.Error("server should reuse business-supplied MediaLeg")
	}
	if h.callCount() != 1 {
		t.Errorf("handler called %d, want 1", h.callCount())
	}
	if h.last == nil || h.last.FromURI != "f" {
		t.Errorf("IncomingCall snapshot wrong: %+v", h.last)
	}

	// OnTerminate hook should be remembered for cleanup
	s.fireOnTerminate("c1", "test")
	if atomic.LoadInt32(&terminated) != 1 {
		t.Error("OnTerminate not stored or fired")
	}
}

func TestIncoming_HandlerRejectWithCustomStatus(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	rtpSess := mustRTPSession(t)
	defer rtpSess.Close()

	h := &fakeInviteHandler{
		decision: Decision{Accept: false, StatusCode: 486, ReasonPhrase: "Busy Here"},
	}
	s.SetInviteHandler(h)

	offer := &sdp.Info{Codecs: []sdp.Codec{{PayloadType: 0, Name: "pcmu", ClockRate: 8000, Channels: 1}}}
	_, err := s.buildIncomingCallLeg(context.Background(), "c1", &stack.Message{IsRequest: true, Method: "INVITE"}, nil, offer, rtpSess)
	if err == nil {
		t.Fatal("expected rejection")
	}
	rej, ok := err.(*businessRejection)
	if !ok {
		t.Fatalf("error type = %T, want *businessRejection", err)
	}
	if rej.status != 486 || rej.reason != "Busy Here" {
		t.Errorf("rejection = %+v", rej)
	}
}

func TestIncoming_HandlerRejectDefaultStatus(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	rtpSess := mustRTPSession(t)
	defer rtpSess.Close()

	h := &fakeInviteHandler{decision: Decision{Accept: false}}
	s.SetInviteHandler(h)
	offer := &sdp.Info{Codecs: []sdp.Codec{{PayloadType: 0, Name: "pcmu", ClockRate: 8000, Channels: 1}}}
	_, err := s.buildIncomingCallLeg(context.Background(), "c1", &stack.Message{IsRequest: true, Method: "INVITE"}, nil, offer, rtpSess)
	rej, ok := err.(*businessRejection)
	if !ok || rej.status != 480 {
		t.Errorf("default reject status = %v, want 480", rej)
	}
	// Error stringification must mention the call
	if !strings.Contains(err.Error(), "480") {
		t.Errorf("error message: %q", err.Error())
	}
}

func TestIncoming_HandlerAcceptButNilLegIsError(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	rtpSess := mustRTPSession(t)
	defer rtpSess.Close()

	h := &fakeInviteHandler{decision: Decision{Accept: true, MediaLeg: nil}}
	s.SetInviteHandler(h)
	offer := &sdp.Info{Codecs: []sdp.Codec{{PayloadType: 0, Name: "pcmu", ClockRate: 8000, Channels: 1}}}
	_, err := s.buildIncomingCallLeg(context.Background(), "c1", &stack.Message{IsRequest: true, Method: "INVITE"}, nil, offer, rtpSess)
	if err == nil || strings.Contains(err.Error(), "480") {
		t.Errorf("expected nil-leg error, got %v", err)
	}
}

func TestIncoming_GuardClauses(t *testing.T) {
	var s *SIPServer
	rtp := mustRTPSession(t)
	defer rtp.Close()
	if _, err := s.buildIncomingCallLeg(context.Background(), "c", nil, nil, &sdp.Info{}, rtp); err == nil {
		t.Error("nil server must error")
	}
	good := New(Config{LocalIP: "127.0.0.1"})
	defer good.Stop()
	if _, err := good.buildIncomingCallLeg(context.Background(), "c", nil, nil, &sdp.Info{}, nil); err == nil {
		t.Error("nil rtp must error")
	}
	if _, err := good.buildIncomingCallLeg(context.Background(), "c", nil, nil, nil, rtp); err == nil {
		t.Error("nil offer must error")
	}
}

// ---------- Config / RTP allocator ---------------------------------------

func TestNewServer_DefaultLocalIP(t *testing.T) {
	s := New(Config{})
	defer s.Stop()
	if s.localIP != "127.0.0.1" {
		t.Errorf("default localIP = %q, want 127.0.0.1", s.localIP)
	}
}

func TestNewInboundRTPSession_FixedPort(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1", RTPFixedPort: 0}) // 0 means "ephemeral" not "fixed"
	defer s.Stop()
	sess, err := s.newInboundRTPSession()
	if err != nil {
		t.Fatalf("ephemeral: %v", err)
	}
	defer sess.Close()
	if sess.LocalAddr.Port == 0 {
		t.Error("ephemeral port should be allocated")
	}
}

func TestNewInboundRTPSession_Range(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1", RTPPortStart: 30000, RTPPortEnd: 30005})
	defer s.Stop()
	sess, err := s.newInboundRTPSession()
	if err != nil {
		// Ports may be in use on test host; tolerate err but don't fail.
		t.Skipf("range allocator: %v", err)
	}
	defer sess.Close()
}

func TestNewInboundRTPSession_NilServer(t *testing.T) {
	var s *SIPServer
	sess, err := s.newInboundRTPSession()
	if err != nil {
		t.Errorf("nil server should still produce a session: %v", err)
	}
	if sess != nil {
		sess.Close()
	}
}

func TestNewRTPSessionFromRange_InvalidRange(t *testing.T) {
	if _, err := newRTPSessionFromRange(100, 50); err == nil {
		t.Error("inverted range must error")
	}
}

// ---------- registerPasswordOK ------------------------------------------

func TestRegisterPasswordOK_NoConfig(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	if !s.registerPasswordOK(nil) {
		t.Error("no config → allow")
	}
	if !s.registerPasswordOK(&stack.Message{}) {
		t.Error("no config → allow even with empty msg")
	}
}

func TestRegisterPasswordOK_StaticMatch(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1", RegisterStaticPassword: "secret123"})
	defer s.Stop()

	if s.registerPasswordOK(nil) {
		t.Error("required + nil msg → reject")
	}

	mNo := &stack.Message{Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	if s.registerPasswordOK(mNo) {
		t.Error("missing header → reject")
	}

	mWrong := &stack.Message{Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	mWrong.SetHeader(sipRegisterPasswordHeader, "wrong")
	if s.registerPasswordOK(mWrong) {
		t.Error("wrong password → reject")
	}

	mGood := &stack.Message{Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	mGood.SetHeader(sipRegisterPasswordHeader, "secret123")
	if !s.registerPasswordOK(mGood) {
		t.Error("matching password → allow")
	}
}

func TestRegisterPasswordOK_NilServer(t *testing.T) {
	var s *SIPServer
	if !s.registerPasswordOK(nil) {
		t.Error("nil server → allow")
	}
}

// ---------- helpers ------------------------------------------------------

func mustRTPSession(t *testing.T) *rtp.Session {
	t.Helper()
	sess, err := rtp.NewSession(0)
	if err != nil {
		t.Fatalf("rtp session: %v", err)
	}
	return sess
}

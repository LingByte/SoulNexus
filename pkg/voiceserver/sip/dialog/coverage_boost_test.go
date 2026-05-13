package dialog

import (
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// helper: parse a SIP message string with CRLFs and proper trailing blank lines.
func mustParse(t *testing.T, raw string) *stack.Message {
	t.Helper()
	m, err := stack.Parse(raw)
	if err != nil {
		t.Fatalf("stack.Parse: %v", err)
	}
	return m
}

func inviteRaw() string {
	return strings.Join([]string{
		"INVITE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKd2",
		"From: <sip:a@b>;tag=rem-cov",
		"To: <sip:x@y>",
		"Call-ID: cid-cov",
		"CSeq: 5 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
}

// ---------- NewUASFromINVITE error branches ------------------------------

func TestNewUASFromINVITE_Errors(t *testing.T) {
	if _, err := NewUASFromINVITE(nil); err == nil {
		t.Error("nil INVITE must error")
	}
	// non-INVITE method
	bye := mustParse(t, strings.Join([]string{
		"BYE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKx",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>;tag=2",
		"Call-ID: c1",
		"CSeq: 1 BYE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if _, err := NewUASFromINVITE(bye); err == nil {
		t.Error("BYE input must error")
	}
}

func TestNewUASFromINVITE_MissingCallIDOrBranch(t *testing.T) {
	// drop Call-ID
	raw := strings.Join([]string{
		"INVITE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKd",
		"From: <sip:a@b>;tag=r",
		"To: <sip:x@y>",
		"CSeq: 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	inv := mustParse(t, raw)
	if _, err := NewUASFromINVITE(inv); err == nil {
		t.Error("missing Call-ID must error")
	}
}

func TestNewUASFromINVITE_InvalidCSeq(t *testing.T) {
	raw := strings.Join([]string{
		"INVITE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKx",
		"From: <sip:a@b>;tag=r",
		"To: <sip:x@y>",
		"Call-ID: c1",
		"CSeq: 0 INVITE", // 0 is rejected (n <= 0)
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	inv := mustParse(t, raw)
	if _, err := NewUASFromINVITE(inv); err == nil {
		t.Error("CSeq 0 must error")
	}
}

// ---------- Dialog state methods + nil receivers --------------------------

func TestDialog_NilReceiverSafety(t *testing.T) {
	var d *Dialog
	if d.InviteTransactionKey() != "" {
		t.Error("nil InviteTransactionKey must be empty")
	}
	d.SetLocalTag("x")          // no panic
	d.SetLocalTagFromToHeader("") // no panic
	d.Confirm()                 // no panic
	d.Terminate()               // no panic
	if d.State() != StateNone {
		t.Error("nil State must be StateNone")
	}
	if d.GetLocalTag() != "" {
		t.Error("nil GetLocalTag must be empty")
	}
	if d.GetRemoteTag() != "" {
		t.Error("nil GetRemoteTag must be empty")
	}
	if d.InviteCSeqNum() != 0 {
		t.Error("nil InviteCSeqNum must be 0")
	}
	if d.MatchACK(&stack.Message{}) {
		t.Error("nil MatchACK must be false")
	}
}

func TestDialog_StateTransitions(t *testing.T) {
	inv := mustParse(t, inviteRaw())
	d, err := NewUASFromINVITE(inv)
	if err != nil {
		t.Fatalf("NewUASFromINVITE: %v", err)
	}

	if d.State() != StateEarly {
		t.Errorf("initial state %v", d.State())
	}
	if d.GetRemoteTag() != "rem-cov" {
		t.Errorf("remote tag = %q", d.GetRemoteTag())
	}
	if d.InviteCSeqNum() != 5 {
		t.Errorf("CSeq = %d, want 5", d.InviteCSeqNum())
	}

	d.SetLocalTagFromToHeader("<sip:x@y>;tag=loc-cov")
	if d.GetLocalTag() != "loc-cov" {
		t.Errorf("local tag after SetLocalTagFromToHeader = %q", d.GetLocalTag())
	}

	d.Confirm()
	if d.State() != StateConfirmed {
		t.Errorf("after Confirm state = %v", d.State())
	}

	d.Terminate()
	if d.State() != StateTerminated {
		t.Errorf("after Terminate state = %v", d.State())
	}

	// Confirm after Terminate must NOT bring back to confirmed
	d.Confirm()
	if d.State() != StateTerminated {
		t.Errorf("Confirm after Terminate should stay terminated, got %v", d.State())
	}
}

// ---------- MatchACK negative branches -----------------------------------

func TestMatchACK_NegativeBranches(t *testing.T) {
	d, err := NewUASFromINVITE(mustParse(t, inviteRaw()))
	if err != nil {
		t.Fatal(err)
	}
	d.SetLocalTag("loc-cov")

	// nil
	if d.MatchACK(nil) {
		t.Error("nil ack must be false")
	}
	// non-ACK
	bye := mustParse(t, strings.Join([]string{
		"BYE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKbye",
		"From: <sip:a@b>;tag=rem-cov",
		"To: <sip:x@y>;tag=loc-cov",
		"Call-ID: cid-cov",
		"CSeq: 5 BYE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if d.MatchACK(bye) {
		t.Error("BYE must not match ACK")
	}
	// wrong Call-ID
	wrongCID := mustParse(t, strings.Join([]string{
		"ACK sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKbad",
		"From: <sip:a@b>;tag=rem-cov",
		"To: <sip:x@y>;tag=loc-cov",
		"Call-ID: OTHER",
		"CSeq: 5 ACK",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if d.MatchACK(wrongCID) {
		t.Error("wrong Call-ID must not match")
	}
	// wrong CSeq number
	wrongCSeq := mustParse(t, strings.Join([]string{
		"ACK sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKbad2",
		"From: <sip:a@b>;tag=rem-cov",
		"To: <sip:x@y>;tag=loc-cov",
		"Call-ID: cid-cov",
		"CSeq: 99 ACK",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if d.MatchACK(wrongCSeq) {
		t.Error("wrong CSeq must not match")
	}
	// wrong from-tag
	wrongFromTag := mustParse(t, strings.Join([]string{
		"ACK sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKbad3",
		"From: <sip:a@b>;tag=DIFFERENT",
		"To: <sip:x@y>;tag=loc-cov",
		"Call-ID: cid-cov",
		"CSeq: 5 ACK",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if d.MatchACK(wrongFromTag) {
		t.Error("wrong from-tag must not match")
	}
	// wrong to-tag
	wrongToTag := mustParse(t, strings.Join([]string{
		"ACK sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKbad4",
		"From: <sip:a@b>;tag=rem-cov",
		"To: <sip:x@y>;tag=DIFFERENT",
		"Call-ID: cid-cov",
		"CSeq: 5 ACK",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if d.MatchACK(wrongToTag) {
		t.Error("wrong to-tag must not match")
	}
}

// ---------- Registry ----------------------------------------------------

func TestRegistry_PutGetDelete(t *testing.T) {
	r := NewRegistry()
	d, err := NewUASFromINVITE(mustParse(t, inviteRaw()))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Put(d); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if got := r.Get("cid-cov"); got != d {
		t.Errorf("Get returned %v", got)
	}
	if got := r.Get("nope"); got != nil {
		t.Errorf("missing key returned %v", got)
	}
	r.Delete("cid-cov")
	if got := r.Get("cid-cov"); got != nil {
		t.Errorf("after Delete still present: %v", got)
	}
}

func TestRegistry_NilSafety(t *testing.T) {
	var r *Registry
	if err := r.Put(&Dialog{}); err == nil {
		t.Error("nil registry Put must error")
	}
	if got := r.Get("x"); got != nil {
		t.Error("nil registry Get must be nil")
	}
	r.Delete("x") // no panic

	// nil dialog
	r2 := NewRegistry()
	if err := r2.Put(nil); err == nil {
		t.Error("nil dialog Put must error")
	}
	// empty Call-ID
	if err := r2.Put(&Dialog{}); err == nil {
		t.Error("empty Call-ID Put must error")
	}
}

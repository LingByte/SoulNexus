package outbound

import (
	"context"
	"strings"
	"testing"
)

func TestManager_Dial_RequiresBind(t *testing.T) {
	m := NewManager(ManagerConfig{LocalIP: "127.0.0.1", SIPHost: "127.0.0.1", SIPPort: 5060})
	_, err := m.Dial(context.Background(), DialRequest{
		Scenario: ScenarioCampaign,
		Target: DialTarget{
			RequestURI:    "sip:bob@example.com",
			SignalingAddr: "127.0.0.1:5060",
		},
		MediaProfile: MediaProfileNone,
	})
	if err != ErrNoSignalingSender {
		t.Fatalf("expected ErrNoSignalingSender, got %v", err)
	}
}

func TestFormatOutboundFromHeader_CallerID(t *testing.T) {
	const tag = "t1"
	noDisp := formatOutboundFromHeader("", "4001880771", "192.0.2.1", 5060, tag)
	wantNo := "<sip:4001880771@192.0.2.1:5060>;tag=t1"
	if noDisp != wantNo {
		t.Fatalf("no display: got %q want %q", noDisp, wantNo)
	}
	withDisp := formatOutboundFromHeader("客服热线", "4001880771", "192.0.2.1", 5060, tag)
	if !strings.HasPrefix(withDisp, `"客服热线" <sip:4001880771@192.0.2.1:5060>;tag=`+tag) {
		t.Fatalf("with display: %q", withDisp)
	}
}

func TestBuildINVITE_ContainsMethod(t *testing.T) {
	p := inviteParams{
		LocalIP:      "127.0.0.1",
		SIPHost:      "127.0.0.1",
		SIPPort:      5060,
		RequestURI:   "sip:bob@example.com",
		CallID:       "test@127.0.0.1",
		FromTag:      "abc",
		Branch:       "branch1",
		CSeq:         1,
		LocalRTPPort: 10000,
		SDPBody:      "v=0\r\n",
		FromUser:     "alice",
	}
	msg := buildINVITE(p)
	if !msg.IsRequest || msg.Method != "INVITE" {
		t.Fatalf("expected INVITE request")
	}
	if msg.GetHeader("Call-ID") != p.CallID {
		t.Fatalf("call-id")
	}
}

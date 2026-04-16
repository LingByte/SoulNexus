package server

import (
	"net"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/sip/protocol"
	"github.com/LingByte/SoulNexus/pkg/logger"
)

func TestSIPServer_HandleInvite_Builds200OKWithSDP(t *testing.T) {
	tmp := t.TempDir()
	if err := logger.Init(&logger.LogConfig{
		Level:      "debug",
		Filename:   tmp + "/test.log",
		MaxSize:    1,
		MaxAge:     1,
		MaxBackups: 1,
		Daily:      false,
	}, "dev"); err != nil {
		t.Fatalf("logger.Init failed: %v", err)
	}

	srv := New(Config{
		Host:    "127.0.0.1",
		Port:    0,
		LocalIP: "192.0.2.10",
	})

	// A minimal INVITE with SDP body.
	sdp := strings.Join([]string{
		"v=0",
		"o=- 123456 123456 IN IP4 198.51.100.1",
		"s=Session",
		"c=IN IP4 198.51.100.1",
		"t=0 0",
		"m=audio 49170 RTP/AVP 0",
		"a=rtpmap:0 PCMU/8000",
	}, "\r\n")

	raw := strings.Join([]string{
		"INVITE sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP client.com:5060;branch=z9hG4bK776asdhds",
		"To: <sip:user@domain.com>",
		"From: <sip:caller@client.com>;tag=1928301774",
		"Call-ID: a84b4c76e66710@client.com",
		"CSeq: 314159 INVITE",
		"Contact: <sip:caller@client.com>",
		"Content-Type: application/sdp",
		"",
		sdp,
	}, "\r\n")

	msg, err := protocol.Parse(raw)
	if err != nil {
		t.Fatalf("Parse invite failed: %v", err)
	}

	resp := srv.StartInviteHandler(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5060})
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d %s", resp.StatusCode, resp.StatusText)
	}
	if resp.GetHeader("Content-Type") != "application/sdp" {
		t.Fatalf("missing Content-Type")
	}
	if !strings.Contains(resp.Body, "m=audio") {
		t.Fatalf("missing SDP in response body")
	}

	info, err := protocol.ParseSDP(resp.Body)
	if err != nil {
		t.Fatalf("ParseSDP response failed: %v", err)
	}
	if info.IP != "192.0.2.10" {
		t.Fatalf("SDP c= IP mismatch: got=%s", info.IP)
	}
	if info.Port <= 0 {
		t.Fatalf("invalid RTP port: %d", info.Port)
	}
	if len(info.Codecs) == 0 {
		t.Fatalf("expected codecs in response SDP")
	}
}

func TestSIPServer_HandleBye_ClosesSession(t *testing.T) {
	tmp := t.TempDir()
	if err := logger.Init(&logger.LogConfig{
		Level:      "debug",
		Filename:   tmp + "/test.log",
		MaxSize:    1,
		MaxAge:     1,
		MaxBackups: 1,
		Daily:      false,
	}, "dev"); err != nil {
		t.Fatalf("logger.Init failed: %v", err)
	}

	srv := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1"})

	// First create a session via INVITE.
	sdp := strings.Join([]string{
		"v=0",
		"o=- 123456 123456 IN IP4 198.51.100.1",
		"s=Session",
		"c=IN IP4 198.51.100.1",
		"t=0 0",
		"m=audio 49170 RTP/AVP 0",
		"a=rtpmap:0 PCMU/8000",
	}, "\r\n")

	rawInvite := strings.Join([]string{
		"INVITE sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP client.com:5060;branch=z9hG4bK776asdhds",
		"To: <sip:user@domain.com>",
		"From: <sip:caller@client.com>;tag=1928301774",
		"Call-ID: callbye-1",
		"CSeq: 314159 INVITE",
		"Contact: <sip:caller@client.com>",
		"Content-Type: application/sdp",
		"",
		sdp,
	}, "\r\n")

	inviteMsg, err := protocol.Parse(rawInvite)
	if err != nil {
		t.Fatalf("Parse invite failed: %v", err)
	}

	_ = srv.StartInviteHandler(inviteMsg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5060})

	byeRaw := strings.Join([]string{
		"BYE sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP client.com:5060;branch=z9hG4bK776asdhds",
		"To: <sip:user@domain.com>;tag=a6c85cf",
		"From: <sip:caller@client.com>;tag=1928301774",
		"Call-ID: callbye-1",
		"CSeq: 314160 BYE",
		"",
		"",
	}, "\r\n")

	byeMsg, err := protocol.Parse(byeRaw)
	if err != nil {
		t.Fatalf("Parse bye failed: %v", err)
	}

	resp := srv.StartByeHandler(byeMsg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5060})
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Second BYE should not panic; it should return 200 anyway.
	resp2 := srv.StartByeHandler(byeMsg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5060})
	if resp2 == nil || resp2.StatusCode != 200 {
		t.Fatalf("expected 200 on second BYE")
	}
}


package server

import (
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/sip/protocol"
	"github.com/LingByte/SoulNexus/pkg/sip/rtp"
)

// Verifies INVITE/ACK brings media up; uplink RTP is not echoed (AI voice path suppresses loopback).
func TestSIPServer_InviteStartsMedia_NoUplinkEcho(t *testing.T) {
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

	// Create a client RTP socket.
	clientRtp, err := rtp.NewSession(0)
	if err != nil {
		t.Fatalf("client rtp NewSession failed: %v", err)
	}
	defer clientRtp.Close()

	clientIP := "127.0.0.1"
	clientPort := clientRtp.LocalAddr.Port

	srv := New(Config{
		Host:    "127.0.0.1",
		Port:    0,
		LocalIP: "127.0.0.1",
	})

	// INVITE SDP describes where the server should send RTP (to the caller).
	// We use the client's RTP local port.
	inviteSDP := strings.Join([]string{
		"v=0",
		"o=- 123456 123456 IN IP4 " + clientIP,
		"s=Session",
		"c=IN IP4 " + clientIP,
		"t=0 0",
		"m=audio " + strconv.Itoa(clientPort) + " RTP/AVP 0",
		"a=rtpmap:0 PCMU/8000",
	}, "\r\n")

	rawInvite := strings.Join([]string{
		"INVITE sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP client.com:5060;branch=z9hG4bK776asdhds",
		"To: <sip:user@domain.com>",
		"From: <sip:caller@client.com>;tag=1928301774",
		"Call-ID: invite-echo-1",
		"CSeq: 314159 INVITE",
		"Contact: <sip:caller@client.com>",
		"Content-Type: application/sdp",
		"",
		inviteSDP,
	}, "\r\n")

	msg, err := protocol.Parse(rawInvite)
	if err != nil {
		t.Fatalf("Parse invite failed: %v", err)
	}

	// Handle INVITE (no UDP server, directly invoke handler).
	resp := srv.StartInviteHandler(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5060})
	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("expected 200 OK response, got %#v", resp)
	}

	// Parse server's SDP to learn where to send RTP.
	sdpInfo, err := protocol.ParseSDP(resp.Body)
	if err != nil {
		t.Fatalf("ParseSDP response failed: %v", err)
	}
	if sdpInfo.Port <= 0 {
		t.Fatalf("invalid server RTP port: %d", sdpInfo.Port)
	}

	serverAddr := &net.UDPAddr{IP: net.ParseIP(sdpInfo.IP), Port: sdpInfo.Port}
	clientRtp.SetRemoteAddr(serverAddr)

	// Start receiving on client (expect no uplink echo after we send mic RTP).
	recvCh := make(chan []byte, 1)
	errCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 2048)
		_, _, pkt, err := clientRtp.ReceiveRTP(buf)
		if err != nil {
			errCh <- err
			return
		}
		recvCh <- pkt.Payload
	}()

	payload := make([]byte, 160)
	for i := 0; i < len(payload); i++ {
		payload[i] = byte(i)
	}

	// Start media on ACK.
	ackRaw := strings.Join([]string{
		"ACK sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP client.com:5060;branch=z9hG4bK776asdhds",
		"To: <sip:user@domain.com>",
		"From: <sip:caller@client.com>;tag=1928301774",
		"Call-ID: invite-echo-1",
		"CSeq: 314159 ACK",
		"",
		"",
	}, "\r\n")
	ackMsg, err := protocol.Parse(ackRaw)
	if err != nil {
		t.Fatalf("Parse ACK failed: %v", err)
	}
	_ = srv.StartAckHandler(ackMsg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5060})
	time.Sleep(50 * time.Millisecond)

	if err := clientRtp.SendRTP(payload, 0, 160); err != nil {
		t.Fatalf("client SendRTP failed: %v", err)
	}

	select {
	case got := <-recvCh:
		t.Fatalf("unexpected RTP from server (uplink must not be echoed): len=%d", len(got))
	case err := <-errCh:
		t.Fatalf("client ReceiveRTP failed: %v", err)
	case <-time.After(800 * time.Millisecond):
		// No downlink until TTS; uplink must not loop back.
	}
}

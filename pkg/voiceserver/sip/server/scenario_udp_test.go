package server

import (
	"encoding/binary"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/logger"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/sdp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

func TestSIPServer_UDP_RegisterInviteAckRTPBye(t *testing.T) {
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
		LocalIP: "127.0.0.1",
	})
	srv.SetInboundAllowUnknownDID(true)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	host, sigPort := srv.ListenAddr()
	serverSig, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, strconv.Itoa(sigPort)))
	if err != nil {
		t.Fatal(err)
	}

	client, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()
	cPort := client.LocalAddr().(*net.UDPAddr).Port

	callIDReg := "scenario-reg-1"
	regRaw := strings.Join([]string{
		"REGISTER sip:" + host + " SIP/2.0",
		"Via: SIP/2.0/UDP 127.0.0.1:" + strconv.Itoa(cPort) + ";branch=z9hG4bKreg",
		"Max-Forwards: 70",
		"From: <sip:user@" + host + ">;tag=regtag",
		"To: <sip:user@" + host + ">",
		"Call-ID: " + callIDReg,
		"CSeq: 1 REGISTER",
		"Contact: <sip:user@127.0.0.1:" + strconv.Itoa(cPort) + ">",
		"Expires: 3600",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	if _, err := client.WriteToUDP([]byte(regRaw), serverSig); err != nil {
		t.Fatal(err)
	}
	if !waitFinalResponse(t, client, callIDReg, "REGISTER") {
		t.Fatal("REGISTER failed")
	}

	callID := "scenario-inv-1"
	sdpBody := strings.Join([]string{
		"v=0",
		"o=- 123456 123456 IN IP4 127.0.0.1",
		"s=Session",
		"c=IN IP4 127.0.0.1",
		"t=0 0",
		"m=audio 49170 RTP/AVP 0",
		"a=rtpmap:0 PCMU/8000",
	}, "\r\n")

	inviteRaw := strings.Join([]string{
		"INVITE sip:user@" + host + " SIP/2.0",
		"Via: SIP/2.0/UDP 127.0.0.1:" + strconv.Itoa(cPort) + ";branch=z9hG4bKinvi",
		"Max-Forwards: 70",
		"From: <sip:caller@127.0.0.1>;tag=cli",
		"To: <sip:user@" + host + ">",
		"Call-ID: " + callID,
		"CSeq: 1 INVITE",
		"Contact: <sip:caller@127.0.0.1:" + strconv.Itoa(cPort) + ">",
		"Content-Type: application/sdp",
		"Content-Length: " + strconv.Itoa(len(sdpBody)),
		"",
		sdpBody,
	}, "\r\n")

	if _, err := client.WriteToUDP([]byte(inviteRaw), serverSig); err != nil {
		t.Fatal(err)
	}
	ok200 := waitInvite200(t, client, callID)
	if ok200 == nil {
		t.Fatal("INVITE 200 missing")
	}
	if srv.GetCallSession(callID) == nil {
		t.Fatal("expected CallSession after INVITE 200")
	}
	info, err := sdp.Parse(ok200.Body)
	if err != nil {
		t.Fatal(err)
	}
	if info.Port <= 0 || info.IP == "" {
		t.Fatalf("answer SDP: %+v", info)
	}

	toHdr := ok200.GetHeader("To")
	fromHdr := ok200.GetHeader("From")
	reqURI := "sip:user@" + host

	ackRaw := strings.Join([]string{
		"ACK " + reqURI + " SIP/2.0",
		"Via: SIP/2.0/UDP 127.0.0.1:" + strconv.Itoa(cPort) + ";branch=z9hG4bKack",
		"Max-Forwards: 70",
		"From: " + fromHdr,
		"To: " + toHdr,
		"Call-ID: " + callID,
		"CSeq: 1 ACK",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	if _, err := client.WriteToUDP([]byte(ackRaw), serverSig); err != nil {
		t.Fatal(err)
	}

	rtpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(info.IP, strconv.Itoa(info.Port)))
	if err != nil {
		t.Fatal(err)
	}
	rtpPkt := make([]byte, 172)
	rtpPkt[0] = 0x80
	rtpPkt[1] = 0x00
	binary.BigEndian.PutUint16(rtpPkt[2:4], 42)
	binary.BigEndian.PutUint32(rtpPkt[4:8], 160)
	binary.BigEndian.PutUint32(rtpPkt[8:12], 0xCAFEBABE)
	for i := 12; i < len(rtpPkt); i++ {
		rtpPkt[i] = 0xff
	}
	if _, err := client.WriteToUDP(rtpPkt, rtpAddr); err != nil {
		t.Fatal(err)
	}

	byeRaw := strings.Join([]string{
		"BYE " + reqURI + " SIP/2.0",
		"Via: SIP/2.0/UDP 127.0.0.1:" + strconv.Itoa(cPort) + ";branch=z9hG4bKbye",
		"Max-Forwards: 70",
		"From: " + fromHdr,
		"To: " + toHdr,
		"Call-ID: " + callID,
		"CSeq: 2 BYE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	if _, err := client.WriteToUDP([]byte(byeRaw), serverSig); err != nil {
		t.Fatal(err)
	}
	if !waitFinalResponse(t, client, callID, "BYE") {
		t.Fatal("BYE failed")
	}
	if srv.GetCallSession(callID) != nil {
		t.Fatal("expected CallSession cleared after BYE")
	}
}

func waitFinalResponse(t *testing.T, c *net.UDPConn, callID, cseqMethod string) bool {
	t.Helper()
	buf := make([]byte, 8192)
	for i := 0; i < 30; i++ {
		_ = c.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
		n, _, err := c.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		m, err := stack.Parse(string(buf[:n]))
		if err != nil || m == nil || m.IsRequest {
			continue
		}
		if m.GetHeader("Call-ID") != callID {
			continue
		}
		cs := m.GetHeader("CSeq")
		if !strings.Contains(strings.ToUpper(cs), strings.ToUpper(cseqMethod)) {
			continue
		}
		return m.StatusCode == 200
	}
	return false
}

func waitInvite200(t *testing.T, c *net.UDPConn, callID string) *stack.Message {
	t.Helper()
	buf := make([]byte, 16384)
	for i := 0; i < 40; i++ {
		_ = c.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
		n, _, err := c.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		m, err := stack.Parse(string(buf[:n]))
		if err != nil || m == nil || m.IsRequest {
			continue
		}
		if m.GetHeader("Call-ID") != callID {
			continue
		}
		if !strings.Contains(strings.ToUpper(m.GetHeader("CSeq")), "INVITE") {
			continue
		}
		if m.StatusCode == 200 {
			return m
		}
	}
	return nil
}

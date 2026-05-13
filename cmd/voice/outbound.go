// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	siprtp "github.com/LingByte/SoulNexus/pkg/voiceserver/sip/rtp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/sdp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/transaction"
)

// outboundConfig configures a one-shot UAC call from the CLI.
type outboundConfig struct {
	TargetURI  string        // e.g. sip:bob@192.168.1.20:5060
	LocalIP    string        // IP advertised in Contact / SDP c=
	HoldFor    time.Duration // how long to keep the call up after 200 OK
	Recorder   *pcmRecorder  // optional: attached to the MediaLeg
	AsrEcho    bool          // optional: attach QCloud ASR + fixed TTS reply
	ReplyText  string        // TTS reply text when AsrEcho is true
	RecordOnly bool
}

// runOutbound places an INVITE to cfg.TargetURI, completes the dialog, holds
// the call for cfg.HoldFor, then terminates with BYE. Blocks until the flow
// finishes (or fails). Designed to be called from main() as a single-shot
// client-mode run — no SIP UAS is started in this mode.
func runOutbound(ctx context.Context, cfg outboundConfig) error {
	target, err := parseSIPTarget(cfg.TargetURI)
	if err != nil {
		return fmt.Errorf("parse target: %w", err)
	}

	// 1) Open a signalling endpoint on an ephemeral port and wire the
	//    transaction manager to its OnSIPResponse dispatch.
	mgr := transaction.NewManager()
	ep := stack.NewEndpoint(stack.EndpointConfig{
		Host: "0.0.0.0",
		Port: 0,
		OnSIPResponse: func(resp *stack.Message, src *net.UDPAddr) {
			mgr.HandleResponse(resp, src)
		},
	})
	if err := ep.Open(); err != nil {
		return fmt.Errorf("endpoint open: %w", err)
	}
	go func() { _ = ep.Serve(ctx) }()
	defer ep.Close()

	localSigAddr := ep.ListenAddr()
	localSigPort := 0
	if ua, ok := localSigAddr.(*net.UDPAddr); ok {
		localSigPort = ua.Port
	}
	log.Printf("[out] signalling bound: %s", localSigAddr)

	// 2) Allocate RTP session and build offer SDP.
	rtpSess, err := siprtp.NewSession(0)
	if err != nil {
		return fmt.Errorf("rtp alloc: %w", err)
	}
	defer rtpSess.Close()

	offerCodecs := []sdp.Codec{
		{PayloadType: 8, Name: "PCMA", ClockRate: 8000, Channels: 1},
		{PayloadType: 0, Name: "PCMU", ClockRate: 8000, Channels: 1},
	}
	offerBody := sdp.Generate(cfg.LocalIP, rtpSess.LocalAddr.Port, offerCodecs)

	// 3) Build and run the INVITE client transaction.
	callID := newCallID()
	fromTag := newTag()
	branch := "z9hG4bK-" + newTag()

	invite := buildOutboundInvite(buildInviteArgs{
		RequestURI:  target.uri,
		Branch:      branch,
		LocalIP:     cfg.LocalIP,
		LocalSigPort: localSigPort,
		FromURI:     fmt.Sprintf("sip:voiceserver@%s", cfg.LocalIP),
		FromTag:     fromTag,
		ToURI:       target.uri,
		CallID:      callID,
		CSeq:        1,
		Body:        offerBody,
	})

	log.Printf("[out] INVITE → %s (call-id=%s)", target.addr, callID)
	sendFn := func(m *stack.Message, a *net.UDPAddr) error { return ep.Send(m, a) }

	inviteCtx, cancelInvite := context.WithTimeout(ctx, 30*time.Second)
	defer cancelInvite()
	result, err := mgr.RunInviteClient(inviteCtx, invite, target.addr, sendFn, func(prov *stack.Message) {
		log.Printf("[out] %d %s", prov.StatusCode, prov.StatusText)
	})
	if err != nil {
		return fmt.Errorf("run invite: %w", err)
	}
	final := result.Final
	log.Printf("[out] final: %d %s", final.StatusCode, final.StatusText)

	if final.StatusCode < 200 || final.StatusCode >= 300 {
		// Non-2xx final: still owes an ACK (absorbed by server tx, no BYE).
		ack, berr := transaction.BuildAckForInvite(invite, final, invite.RequestURI)
		if berr == nil {
			_ = ep.Send(ack, result.Remote)
		}
		return fmt.Errorf("call rejected: %d %s", final.StatusCode, final.StatusText)
	}

	// 4) Parse 200 OK SDP and attach a MediaLeg so inbound RTP works.
	ackURI := transaction.AckRequestURIFor2xx(final, invite.RequestURI)
	ack, err := transaction.BuildAckForInvite(invite, final, ackURI)
	if err != nil {
		return fmt.Errorf("build ack: %w", err)
	}
	if err := ep.Send(ack, result.Remote); err != nil {
		return fmt.Errorf("send ack: %w", err)
	}
	log.Printf("[out] ACK → %s", result.Remote)

	remoteSDP, err := sdp.Parse(final.Body)
	if err != nil {
		log.Printf("[out] WARN parse 200 sdp: %v", err)
	} else if err := session.ApplyRemoteSDP(rtpSess, remoteSDP); err != nil {
		log.Printf("[out] WARN apply remote sdp: %v", err)
	}

	var leg *session.MediaLeg
	if remoteSDP != nil && len(remoteSDP.Codecs) > 0 {
		legCfg := session.MediaLegConfig{}
		if cfg.Recorder != nil {
			legCfg.InputFilters = append(legCfg.InputFilters, cfg.Recorder.inFilter)
			legCfg.OutputFilters = append(legCfg.OutputFilters, cfg.Recorder.outFilter)
		}
		leg, err = session.NewMediaLeg(ctx, callID, rtpSess, remoteSDP.Codecs, legCfg)
		if err != nil {
			log.Printf("[out] WARN media leg: %v", err)
		} else {
			neg := leg.NegotiatedSDP()
			log.Printf("[out] leg up: codec=%s rtp_local=%s rtp_remote=%s:%d",
				neg.Name, rtpSess.LocalAddr.String(), remoteSDP.IP, remoteSDP.Port)
			if cfg.Recorder != nil {
				cfg.Recorder.setCodec(neg.Name, neg.ClockRate)
				cfg.Recorder.setPCMRate(leg.PCMSampleRate())
			}
			if cfg.AsrEcho {
				if va, err := attachVoiceEcho(ctx, leg, callID, 0, cfg.ReplyText); err != nil {
					log.Printf("[out] asr-echo disabled: %v", err)
				} else {
					defer va.Close()
				}
			}
			leg.Start()
		}
	}

	// 5) Hold the call.
	log.Printf("[out] holding for %s", cfg.HoldFor)
	select {
	case <-ctx.Done():
	case <-time.After(cfg.HoldFor):
	}

	// 6) Teardown: MediaLeg first, then BYE, then flush recorder.
	if leg != nil {
		leg.Stop()
	}

	bye := buildOutboundBye(buildByeArgs{
		RequestURI: ackURI,
		Branch:     "z9hG4bK-" + newTag(),
		LocalIP:    cfg.LocalIP,
		LocalSigPort: localSigPort,
		FromURI:    fmt.Sprintf("sip:voiceserver@%s", cfg.LocalIP),
		FromTag:    fromTag,
		ToRaw:      final.GetHeader("To"),
		CallID:     callID,
		CSeq:       2,
	})
	log.Printf("[out] BYE → %s", result.Remote)
	byeCtx, cancelBye := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelBye()
	byeRes, err := mgr.RunNonInviteClient(byeCtx, bye, result.Remote, sendFn)
	if err != nil {
		log.Printf("[out] BYE error: %v", err)
	} else {
		log.Printf("[out] BYE final: %d %s", byeRes.Final.StatusCode, byeRes.Final.StatusText)
	}

	if cfg.Recorder != nil {
		cfg.Recorder.flush()
	}
	return nil
}

// ---------- helpers --------------------------------------------------------

type sipTarget struct {
	uri  string      // canonicalised "sip:user@host:port"
	addr *net.UDPAddr
}

// parseSIPTarget accepts "sip:user@host:port" or "sip:user@host" (default port 5060).
func parseSIPTarget(raw string) (sipTarget, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return sipTarget{}, fmt.Errorf("empty target")
	}
	s = strings.TrimPrefix(s, "<")
	s = strings.TrimSuffix(s, ">")
	if !strings.HasPrefix(strings.ToLower(s), "sip:") {
		return sipTarget{}, fmt.Errorf("target must start with sip:")
	}
	rest := s[4:]
	at := strings.Index(rest, "@")
	if at < 0 {
		return sipTarget{}, fmt.Errorf("target missing user@host")
	}
	hostport := rest[at+1:]
	if semi := strings.Index(hostport, ";"); semi >= 0 {
		hostport = hostport[:semi]
	}
	host, portStr, err := net.SplitHostPort(hostport)
	port := 5060
	if err != nil {
		host = hostport
	} else if portStr != "" {
		p, perr := strconv.Atoi(portStr)
		if perr != nil || p <= 0 || p > 65535 {
			return sipTarget{}, fmt.Errorf("bad port in target: %q", portStr)
		}
		port = p
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return sipTarget{}, fmt.Errorf("resolve %q: %w", host, err)
		}
		for _, candidate := range ips {
			if v4 := candidate.To4(); v4 != nil {
				ip = v4
				break
			}
		}
		if ip == nil {
			ip = ips[0]
		}
	}
	canonical := fmt.Sprintf("sip:%s@%s:%d", rest[:at], host, port)
	return sipTarget{uri: canonical, addr: &net.UDPAddr{IP: ip, Port: port}}, nil
}

type buildInviteArgs struct {
	RequestURI   string
	Branch       string
	LocalIP      string
	LocalSigPort int
	FromURI      string
	FromTag      string
	ToURI        string
	CallID       string
	CSeq         int
	Body         string
}

func buildOutboundInvite(a buildInviteArgs) *stack.Message {
	m := &stack.Message{
		IsRequest:    true,
		Method:       stack.MethodInvite,
		RequestURI:   a.RequestURI,
		Version:      "SIP/2.0",
		Headers:      map[string]string{},
		HeadersMulti: map[string][]string{},
	}
	m.SetHeader("Via", fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=%s;rport", a.LocalIP, a.LocalSigPort, a.Branch))
	m.SetHeader("Max-Forwards", "70")
	m.SetHeader("From", fmt.Sprintf("<%s>;tag=%s", a.FromURI, a.FromTag))
	m.SetHeader("To", fmt.Sprintf("<%s>", a.ToURI))
	m.SetHeader("Call-ID", a.CallID)
	m.SetHeader("CSeq", fmt.Sprintf("%d INVITE", a.CSeq))
	m.SetHeader("Contact", fmt.Sprintf("<sip:voiceserver@%s:%d>", a.LocalIP, a.LocalSigPort))
	m.SetHeader("Allow", "INVITE, ACK, BYE, CANCEL, OPTIONS, INFO, PRACK, UPDATE")
	m.SetHeader("User-Agent", "VoiceServer-Dialer/1.0")
	if a.Body != "" {
		m.SetHeader("Content-Type", "application/sdp")
		m.Body = a.Body
	}
	m.SetHeader("Content-Length", strconv.Itoa(stack.BodyBytesLen(a.Body)))
	return m
}

type buildByeArgs struct {
	RequestURI   string
	Branch       string
	LocalIP      string
	LocalSigPort int
	FromURI      string
	FromTag      string
	ToRaw        string // full To header from 200 OK (includes tag)
	CallID       string
	CSeq         int
}

func buildOutboundBye(a buildByeArgs) *stack.Message {
	m := &stack.Message{
		IsRequest:    true,
		Method:       stack.MethodBye,
		RequestURI:   a.RequestURI,
		Version:      "SIP/2.0",
		Headers:      map[string]string{},
		HeadersMulti: map[string][]string{},
	}
	m.SetHeader("Via", fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=%s;rport", a.LocalIP, a.LocalSigPort, a.Branch))
	m.SetHeader("Max-Forwards", "70")
	m.SetHeader("From", fmt.Sprintf("<%s>;tag=%s", a.FromURI, a.FromTag))
	m.SetHeader("To", a.ToRaw)
	m.SetHeader("Call-ID", a.CallID)
	m.SetHeader("CSeq", fmt.Sprintf("%d BYE", a.CSeq))
	m.SetHeader("User-Agent", "VoiceServer-Dialer/1.0")
	m.SetHeader("Content-Length", "0")
	return m
}

func newTag() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func newCallID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:]) + "@voiceserver"
}

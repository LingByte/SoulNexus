package server

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// incoming.go bridges the inbound INVITE handler in sip_server.go to the
// business InviteHandler interface. It builds an IncomingCall snapshot,
// dispatches to the business layer (when wired), and returns the chosen
// MediaLeg. Without an InviteHandler, a default echo-style MediaLeg is
// constructed so a bare VoiceServer is still useful for SIP-protocol tests.

import (
	"context"
	"fmt"
	"net"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/rtp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/sdp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// businessRejection is returned by buildIncomingCallLeg when the business
// InviteHandler explicitly rejected the INVITE with a chosen status. Allows
// the caller to use the requested SIP status instead of a generic 488.
type businessRejection struct {
	status int
	reason string
}

func (b *businessRejection) Error() string {
	return fmt.Sprintf("business rejected invite: %d %s", b.status, b.reason)
}

// buildIncomingCallLeg dispatches to the InviteHandler (if registered) and
// returns the MediaLeg to attach to the call. The caller is responsible for
// stopping the leg on failure.
func (s *SIPServer) buildIncomingCallLeg(
	ctx context.Context,
	callID string,
	msg *stack.Message,
	from *net.UDPAddr,
	offer *sdp.Info,
	rtpSess *rtp.Session,
) (*session.MediaLeg, error) {
	if s == nil {
		return nil, fmt.Errorf("server: nil SIPServer")
	}
	if rtpSess == nil {
		return nil, fmt.Errorf("server: nil rtp session")
	}
	if offer == nil {
		return nil, fmt.Errorf("server: nil sdp offer")
	}

	h := s.inviteHandlerImpl()
	if h == nil {
		// No business layer wired: build a default mono-echo MediaLeg so the
		// SIP stack is still functional for protocol-level tests / smoke runs.
		return session.NewMediaLeg(ctx, callID, rtpSess, offer.Codecs, session.MediaLegConfig{})
	}

	in := &IncomingCall{
		CallID:              callID,
		FromURI:             headerOrEmpty(msg, "From"),
		ToURI:               headerOrEmpty(msg, "To"),
		RemoteSignalingAddr: from,
		SDP:                 offer,
		RawMessage:          msg,
		RTPSession:          rtpSess,
	}
	dec, err := h.OnIncomingCall(ctx, in)
	if err != nil {
		return nil, err
	}
	if !dec.Accept {
		status := dec.StatusCode
		if status == 0 {
			status = 480
		}
		reason := dec.ReasonPhrase
		if reason == "" {
			reason = "Temporarily Unavailable"
		}
		return nil, &businessRejection{status: status, reason: reason}
	}
	if dec.MediaLeg == nil {
		return nil, fmt.Errorf("server: InviteHandler accepted but returned nil MediaLeg")
	}
	if dec.OnTerminate != nil {
		s.rememberTerminateHook(callID, dec.OnTerminate)
	}
	return dec.MediaLeg, nil
}

func headerOrEmpty(m *stack.Message, name string) string {
	if m == nil {
		return ""
	}
	return m.GetHeader(name)
}

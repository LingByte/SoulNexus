package server

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// compat.go provides server-internal substitutes for the previously-imported
// pkg/sip/conversation and pkg/sip/voicedialog packages. Each function here
// dispatches to a (nil-safe) interface registered via the SIPServer.Set*
// methods, or is a no-op when the business layer hasn't wired anything in.
//
// The goal is to make sip_server.go (and its sibling handlers) compile with
// minimal textual changes from the LingEchoX original, while keeping the
// VoiceServer package free of any business imports.

import (
	"context"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/dtmf"
)

// ---------- Setters for the new interfaces (mirror the existing Set* style) ----

// SetInviteHandler wires the business INVITE handler.
func (s *SIPServer) SetInviteHandler(h InviteHandler) {
	if s == nil {
		return
	}
	s.inviteHandlerMu.Lock()
	defer s.inviteHandlerMu.Unlock()
	s.inviteHandler = h
}

func (s *SIPServer) inviteHandlerImpl() InviteHandler {
	if s == nil {
		return nil
	}
	s.inviteHandlerMu.RLock()
	defer s.inviteHandlerMu.RUnlock()
	return s.inviteHandler
}

// SetDTMFSink wires DTMF (RFC 2833 + SIP INFO) delivery.
func (s *SIPServer) SetDTMFSink(d DTMFSink) {
	if s == nil {
		return
	}
	s.dtmfSinkMu.Lock()
	defer s.dtmfSinkMu.Unlock()
	s.dtmfSink = d
}

func (s *SIPServer) dtmfSinkImpl() DTMFSink {
	if s == nil {
		return nil
	}
	s.dtmfSinkMu.RLock()
	defer s.dtmfSinkMu.RUnlock()
	return s.dtmfSink
}

// SetTransferHandler wires SIP REFER (call transfer) handling.
func (s *SIPServer) SetTransferHandler(h TransferHandler) {
	if s == nil {
		return
	}
	s.transferHandlerMu.Lock()
	defer s.transferHandlerMu.Unlock()
	s.transferHandler = h
}

func (s *SIPServer) transferHandlerImpl() TransferHandler {
	if s == nil {
		return nil
	}
	s.transferHandlerMu.RLock()
	defer s.transferHandlerMu.RUnlock()
	return s.transferHandler
}

// SetCallLifecycleObserver wires teardown observation.
func (s *SIPServer) SetCallLifecycleObserver(o CallLifecycleObserver) {
	if s == nil {
		return
	}
	s.callObserverMu.Lock()
	defer s.callObserverMu.Unlock()
	s.callObserver = o
}

func (s *SIPServer) callObserverImpl() CallLifecycleObserver {
	if s == nil {
		return nil
	}
	s.callObserverMu.RLock()
	defer s.callObserverMu.RUnlock()
	return s.callObserver
}

// ---------- Substitutes for conversation.* / voicedialog.* calls --------

// cleanupCallState is the in-server equivalent of conversation.CleanupCallState.
// It dispatches to the business observer and to call-specific OnTerminate hooks.
func (s *SIPServer) cleanupCallState(callID string) {
	if s == nil || strings.TrimSpace(callID) == "" {
		return
	}
	if obs := s.callObserverImpl(); obs != nil {
		obs.OnCallCleanup(callID)
	}
	s.fireOnTerminate(callID, "cleanup")
}

// preHangupCallState is the equivalent of conversation.HangupWebSeatBridge*/
// HangupTransferBridge*: lets the business observer claim teardown ("I have
// already hung up the bridge, don't send your own BYE"). Returns true when
// the observer handled the hangup.
func (s *SIPServer) preHangupCallState(callID string) bool {
	if s == nil || strings.TrimSpace(callID) == "" {
		return false
	}
	if obs := s.callObserverImpl(); obs != nil {
		return obs.OnCallPreHangup(callID)
	}
	return false
}

// callOwnedByBusinessRouting reports whether the business layer has signalled
// that media for this call is already owned by an external bridge (transfer,
// WebSeat, etc). Replaces conversation.ActiveTransferBridgeForCallID +
// ActiveWebSeatSession. The default implementation always returns false; a
// CallLifecycleObserver implementation that needs this can answer through
// its OnCallPreHangup decision pattern.
func (s *SIPServer) callOwnedByBusinessRouting(callID string) bool {
	// In the cleaner architecture, this is no longer needed: if the business
	// layer wants to take over media routing, its InviteHandler.OnIncomingCall
	// returns a Decision with no MediaLeg (or returns a custom one that already
	// has the bridge wired). The server only auto-attaches the leg when the
	// decision asked it to.
	return false
}

// triggerTransferFromReferTo is the equivalent of
// conversation.TriggerTransferFromReferTo. Dispatches to TransferHandler.OnRefer.
func (s *SIPServer) triggerTransferFromReferTo(ctx context.Context, callID, referTo string, notify func(frag, subState string)) {
	if s == nil {
		return
	}
	if h := s.transferHandlerImpl(); h != nil {
		h.OnRefer(ctx, callID, referTo, notify)
		return
	}
	// Without a TransferHandler, terminate the subscription immediately so the
	// peer doesn't hang waiting for NOTIFY.
	if notify != nil {
		notify("SIP/2.0 501 Not Implemented", "terminated;reason=noresource")
	}
}

// handleSIPINFODTMF replaces conversation.HandleSIPINFODTMF. It parses the
// SIP INFO DTMF body locally (using pkg/sip/dtmf) and forwards to DTMFSink.
func (s *SIPServer) handleSIPINFODTMF(callID, contentType, body string) {
	if s == nil {
		return
	}
	sink := s.dtmfSinkImpl()
	if sink == nil {
		return
	}
	digit, ok := dtmf.DigitFromSIPINFO(contentType, body)
	if !ok {
		return
	}
	sink.OnDTMF(callID, digit, true)
}

// endVoiceDialogBridge replaces server.endVoiceDialogBridge → voicedialog.EndCall.
// Now: just clear the INVITE brief snapshot and fire the call's OnTerminate.
func (s *SIPServer) endVoiceDialogBridge(callID string) {
	if s == nil || strings.TrimSpace(callID) == "" {
		return
	}
	s.fireOnTerminate(callID, "ended")
	s.clearInviteBrief(callID)
}

// attachInboundCallToBusiness replaces voicedialog.AttachInboundVoiceDialog.
// In the new design, the business layer has already configured its MediaLeg
// (with processors / RTP taps / OnTerminate hook) before NewMediaLeg returned.
// On ACK, the server simply Start()s the MediaLeg; nothing else is needed.
//
// The function name is kept as a hook-point for future plug-ins that want to
// observe ACK without re-implementing InviteHandler.OnIncomingCall.
func (s *SIPServer) attachInboundCallToBusiness(callID string) {
	// Currently a no-op: starting media is the caller's responsibility (see
	// handleAck which calls leg.Start()). Reserved for future extension.
}

// ---------- OnTerminate hook bookkeeping --------------------------------

// rememberTerminateHook stores a per-call hook from a Decision so BYE/cleanup
// paths can fire it later.
func (s *SIPServer) rememberTerminateHook(callID string, fn func(reason string)) {
	if s == nil || strings.TrimSpace(callID) == "" || fn == nil {
		return
	}
	s.terminateMu.Lock()
	defer s.terminateMu.Unlock()
	if s.terminateHooks == nil {
		s.terminateHooks = make(map[string]func(reason string))
	}
	s.terminateHooks[callID] = fn
}

// fireOnTerminate runs the per-call OnTerminate hook exactly once.
func (s *SIPServer) fireOnTerminate(callID, reason string) {
	if s == nil || strings.TrimSpace(callID) == "" {
		return
	}
	s.terminateMu.Lock()
	fn := s.terminateHooks[callID]
	delete(s.terminateHooks, callID)
	s.terminateMu.Unlock()
	if fn != nil {
		fn(reason)
	}
}

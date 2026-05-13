// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package webseat

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"go.uber.org/zap"
)

// Errors surfaced by Hub methods. Stable across versions so callers
// can `errors.Is` them.
var (
	ErrAlreadyAwaiting = errors.New("webseat: call already awaiting handoff")
	ErrAlreadyBridged  = errors.New("webseat: call already bridged to an agent")
	ErrNotAwaiting     = errors.New("webseat: no such awaiting call")
	ErrNotBridged      = errors.New("webseat: no such active bridge")
	ErrTokenInvalid    = errors.New("webseat: token does not match")
)

// Bridge is the audio splice between an inbound SIP MediaLeg and a
// browser WebRTC peer. The hub holds a single Bridge instance for the
// process; per-call state lives in the bridge's own implementation.
//
// Connect is called when a browser accepts an awaiting call. It
// receives the SIP leg + the peer-supplied SDP offer, returns the
// answer SDP, and runs until the call ends. Implementations should
// return promptly after publishing the answer; long-running splice
// goroutines are owned by the bridge.
//
// Disconnect is called on Hub.Hangup to tear down a previously
// established bridge. Idempotent — a Disconnect on an unknown callID
// must be a no-op so racy "browser hangs up at the same moment SIP
// peer sends BYE" doesn't double-fault.
type Bridge interface {
	Connect(ctx context.Context, callID string, leg *session.MediaLeg, offerSDP string) (answerSDP string, err error)
	Disconnect(callID string) error
}

// nopBridge satisfies Bridge with no-ops, useful for unit tests and
// for "register the routes but don't actually splice yet" deployments.
// Production must wire a real implementation (pion-backed, see
// pkg/voice/webrtc) — calling Connect on the nopBridge surfaces a
// clear error so misconfiguration is caught early.
type nopBridge struct{}

// NopBridge returns the no-op default. Connect always errors.
func NopBridge() Bridge { return nopBridge{} }

func (nopBridge) Connect(_ context.Context, _ string, _ *session.MediaLeg, _ string) (string, error) {
	return "", errors.New("webseat: no Bridge configured (default nopBridge)")
}
func (nopBridge) Disconnect(_ string) error { return nil }

// Config wires the Hub to the rest of VoiceServer. All fields are
// optional except Bridge — the Hub falls back to safe defaults
// otherwise.
type Config struct {
	// Bridge performs the SIP↔WebRTC audio splice. Required for a
	// production deploy; pass NopBridge() during integration tests.
	Bridge Bridge

	// JoinTimeout caps how long an awaiting call sits before we give
	// up and surface the timeout to the caller. 0 → 30 s.
	JoinTimeout time.Duration

	// Token, if non-empty, is the shared secret browsers must present
	// in the `?token=` query param of the WebSocket / HTTP endpoints.
	// Empty disables auth — only acceptable for development.
	Token string

	// Logger is optional; nil → zap.NewNop().
	Logger *zap.Logger

	// OnAwaiting / OnBridged / OnEnded fire on lifecycle transitions.
	// Useful for metrics + dashboards; runs on the calling goroutine.
	OnAwaiting func(callID string)
	OnBridged  func(callID string)
	OnEnded    func(callID, reason string)
}

// Hub coordinates inbound SIP calls awaiting browser agents.
type Hub struct {
	cfg Config
	log *zap.Logger

	mu       sync.Mutex
	awaiting map[string]*awaitingCall
	active   map[string]*activeCall
	nextGen  uint64
}

type awaitingCall struct {
	callID    string
	leg       *session.MediaLeg
	at        time.Time
	cancelTimeout context.CancelFunc

	// gen distinguishes successive awaiting registrations for the
	// same callID. The watchdog captures the gen it was started for
	// and only deletes if the live entry still has the same gen —
	// this avoids a race where Pickup fails, restores an awaiting
	// entry, and then the OLD watchdog (woken by ctx cancel during
	// the pickup attempt) deletes the just-restored entry.
	gen uint64
}

type activeCall struct {
	callID  string
	leg     *session.MediaLeg
	startedAt time.Time
}

// New validates cfg and returns a Hub. Bridge defaults to NopBridge
// when nil so cmd code can mount the HTTP routes incrementally; that
// path will fail loudly on the first /join attempt.
func New(cfg Config) *Hub {
	if cfg.Bridge == nil {
		cfg.Bridge = NopBridge()
	}
	if cfg.JoinTimeout <= 0 {
		cfg.JoinTimeout = 30 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	return &Hub{
		cfg:      cfg,
		log:      cfg.Logger,
		awaiting: make(map[string]*awaitingCall),
		active:   make(map[string]*activeCall),
	}
}

// RegisterAwaiting marks a call as ready for browser handoff. The SIP
// MediaLeg must already be in a state where its AI pipeline (if any)
// has detached — typically called from the dialog plane right after
// it received a transfer.request and decided "send to webseat".
//
// The hub starts a watchdog: if no browser accepts within
// JoinTimeout, the awaiting entry is dropped and OnEnded fires with
// reason="timeout". Callers wanting to react to that should register
// OnEnded; the SIP server's BYE flow is unchanged either way.
func (h *Hub) RegisterAwaiting(callID string, leg *session.MediaLeg) error {
	if h == nil {
		return errors.New("webseat: nil hub")
	}
	if callID == "" {
		return errors.New("webseat: empty callID")
	}
	if leg == nil {
		return errors.New("webseat: nil MediaLeg")
	}
	h.mu.Lock()
	if _, dup := h.awaiting[callID]; dup {
		h.mu.Unlock()
		return ErrAlreadyAwaiting
	}
	if _, dup := h.active[callID]; dup {
		h.mu.Unlock()
		return ErrAlreadyBridged
	}
	timeoutCtx, cancel := context.WithTimeout(context.Background(), h.cfg.JoinTimeout)
	h.nextGen++
	gen := h.nextGen
	h.awaiting[callID] = &awaitingCall{
		callID:        callID,
		leg:           leg,
		at:            time.Now(),
		cancelTimeout: cancel,
		gen:           gen,
	}
	h.mu.Unlock()
	if h.cfg.OnAwaiting != nil {
		h.cfg.OnAwaiting(callID)
	}
	go h.watchdog(timeoutCtx, callID, gen)
	h.log.Info("webseat: awaiting", zap.String("call_id", callID))
	return nil
}

// Awaiting returns a snapshot of currently-awaiting Call-IDs. Order
// is undefined — sort at the caller if needed.
func (h *Hub) Awaiting() []string {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, 0, len(h.awaiting))
	for k := range h.awaiting {
		out = append(out, k)
	}
	return out
}

// Pickup atomically transitions a call from awaiting → bridged by
// invoking the configured Bridge. The browser-supplied offerSDP is
// passed straight through; the answer is returned to the caller.
//
// On Bridge.Connect failure the awaiting entry is restored (so
// another browser can retry) and the original error is returned.
func (h *Hub) Pickup(ctx context.Context, callID, offerSDP string) (string, error) {
	if h == nil {
		return "", errors.New("webseat: nil hub")
	}
	h.mu.Lock()
	aw, ok := h.awaiting[callID]
	if !ok {
		h.mu.Unlock()
		return "", ErrNotAwaiting
	}
	delete(h.awaiting, callID)
	if aw.cancelTimeout != nil {
		aw.cancelTimeout()
	}
	h.mu.Unlock()

	answer, err := h.cfg.Bridge.Connect(ctx, callID, aw.leg, offerSDP)
	if err != nil {
		// Restore awaiting state so a different browser can try.
		// Re-arm the watchdog with a fresh JoinTimeout and a new
		// generation tag — the OLD watchdog may be racing to fire
		// after its ctx was cancelled by the failed pickup; the
		// gen check in watchdog() ensures it observes a mismatch
		// against the fresh entry and exits cleanly.
		newCtx, cancel := context.WithTimeout(context.Background(), h.cfg.JoinTimeout)
		h.mu.Lock()
		h.nextGen++
		gen := h.nextGen
		h.awaiting[callID] = &awaitingCall{
			callID:        callID,
			leg:           aw.leg,
			at:            aw.at, // preserve original timestamp for fairness
			cancelTimeout: cancel,
			gen:           gen,
		}
		h.mu.Unlock()
		go h.watchdog(newCtx, callID, gen)
		return "", fmt.Errorf("bridge connect: %w", err)
	}

	h.mu.Lock()
	h.active[callID] = &activeCall{
		callID:    callID,
		leg:       aw.leg,
		startedAt: time.Now(),
	}
	h.mu.Unlock()
	if h.cfg.OnBridged != nil {
		h.cfg.OnBridged(callID)
	}
	h.log.Info("webseat: bridged", zap.String("call_id", callID))
	return answer, nil
}

// Hangup tears down an active bridge. Returns ErrNotBridged if the
// call was awaiting (use ReleaseAwaiting for that) or unknown.
// Always tells the Bridge to disconnect, even if our internal map
// is empty — that handles "browser hangs up after we removed the
// entry" races without surfacing a confusing error.
func (h *Hub) Hangup(callID, reason string) error {
	if h == nil {
		return errors.New("webseat: nil hub")
	}
	h.mu.Lock()
	ac, ok := h.active[callID]
	if ok {
		delete(h.active, callID)
	}
	h.mu.Unlock()

	// Bridge.Disconnect is idempotent per its interface contract, so
	// we always call it — even when ok=false there might be a stale
	// peer the Bridge knows about that our hub forgot about.
	_ = h.cfg.Bridge.Disconnect(callID)

	if !ok {
		return ErrNotBridged
	}
	if h.cfg.OnEnded != nil {
		h.cfg.OnEnded(callID, reason)
	}
	h.log.Info("webseat: bridge ended",
		zap.String("call_id", callID),
		zap.String("reason", reason),
		zap.Duration("duration", time.Since(ac.startedAt)))
	return nil
}

// ReleaseAwaiting cancels a not-yet-picked-up call. Used when the
// SIP peer hangs up before any browser joins, or when the dialog
// plane changes its mind and wants the call back.
func (h *Hub) ReleaseAwaiting(callID, reason string) error {
	if h == nil {
		return errors.New("webseat: nil hub")
	}
	h.mu.Lock()
	aw, ok := h.awaiting[callID]
	if ok {
		delete(h.awaiting, callID)
		if aw.cancelTimeout != nil {
			aw.cancelTimeout()
		}
	}
	h.mu.Unlock()
	if !ok {
		return ErrNotAwaiting
	}
	if h.cfg.OnEnded != nil {
		h.cfg.OnEnded(callID, reason)
	}
	h.log.Info("webseat: awaiting released",
		zap.String("call_id", callID),
		zap.String("reason", reason))
	return nil
}

// watchdog runs per-call. When ctx fires (timeout or explicit
// cancel) it removes the awaiting entry IF it still exists AND its
// generation matches the one this watchdog was launched for. The gen
// check distinguishes "I'm the live watchdog and the timeout
// expired" (delete + fire OnEnded) from "I was cancelled by a Pickup
// that succeeded or by a Pickup that failed and re-armed a NEW
// watchdog" (no-op, exit silently).
func (h *Hub) watchdog(ctx context.Context, callID string, gen uint64) {
	<-ctx.Done()
	h.mu.Lock()
	aw, stillAwaiting := h.awaiting[callID]
	expired := stillAwaiting && aw.gen == gen
	if expired {
		delete(h.awaiting, callID)
	}
	h.mu.Unlock()
	if !expired {
		return
	}
	if h.cfg.OnEnded != nil {
		h.cfg.OnEnded(callID, "timeout")
	}
	h.log.Info("webseat: awaiting timed out", zap.String("call_id", callID))
}

// ---- helpers ----

// tokenOK reports whether got matches the configured Token using a
// constant-time compare. Empty configured token = auth disabled.
func (h *Hub) tokenOK(got string) bool {
	expected := h.cfg.Token
	if expected == "" {
		return true
	}
	if len(got) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

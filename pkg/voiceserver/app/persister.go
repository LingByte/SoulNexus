// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package app

import (
	"fmt"
	"context"
	"encoding/json"
	"errors"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/persist"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/server"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/metrics"
	"gorm.io/gorm"
)

// CallPersister captures the lifecycle of one call into the SIPCall
// table. It is concurrency-safe: ASR / TTS / SIP teardown can fire from
// different goroutines without the persister losing rows.
//
// The struct is exported so cmd/voice (SIP inbound/outbound) can hold
// it directly; transport adapters get a SessionPersister wrapper via
// AsSessionPersister.
type CallPersister struct {
	db        *gorm.DB
	callID    string
	transport string // sip / xiaozhi / webrtc — for downstream tables

	mu        sync.Mutex
	closed    bool
	lastFinal string // most recent ASR final, paired with the next OnTurn
	finalAt   time.Time
}

// CallID returns the persister's call identifier (used by the recorder
// flush path to stamp the URL on the same row).
func (p *CallPersister) CallID() string {
	if p == nil {
		return ""
	}
	return p.callID
}

// Transport returns the call transport classification ("sip"/"xiaozhi"/
// "webrtc"). Used by recorder/metric code that wants to label rows.
func (p *CallPersister) Transport() string {
	if p == nil {
		return ""
	}
	return p.transport
}

// NewCallPersister creates the SIPCall row in state=ringing for an
// inbound INVITE. Returns nil (logged) on DB failure so the call can
// proceed.
func NewCallPersister(ctx context.Context, db *gorm.DB, inv *server.IncomingCall, direction string) *CallPersister {
	if inv == nil {
		return nil
	}
	remoteSig := ""
	if inv.RemoteSignalingAddr != nil {
		remoteSig = inv.RemoteSignalingAddr.String()
	}
	return NewGenericCallPersister(ctx, db, inv.CallID, "sip", direction,
		inv.FromURI, inv.ToURI, remoteSig, "")
}

// NewGenericCallPersister is the transport-agnostic constructor used by
// every adapter (SIP, xiaozhi, WebRTC). It writes a row in state=ringing
// with whatever caller / callee identity the transport can supply.
//
// transport classifies the call ("sip" / "xiaozhi" / "webrtc") so
// dashboards can split metrics; userAgent is the device or browser
// identifier (free-form, e.g. "ESP32-LingByte-A0:B1:..", "Mozilla/5.0
// ...", "Asterisk").
//
// Returns nil when persistence is disabled (db == nil) or when the row
// can't be created — the call still runs without a DB record.
func NewGenericCallPersister(ctx context.Context, db *gorm.DB, callID, transport, direction, fromHeader, toHeader, remoteSignaling, userAgent string) *CallPersister {
	if db == nil || callID == "" {
		return nil
	}
	now := time.Now().UTC()
	row := &persist.SIPCall{
		CallID:          callID,
		Direction:       direction,
		Transport:       strings.ToLower(strings.TrimSpace(transport)),
		RemoteUserAgent: userAgent,
		FromHeader:      fromHeader,
		ToHeader:        toHeader,
		FromNumber:      persist.ExtractSIPUserPart(fromHeader),
		ToNumber:        persist.ExtractSIPUserPart(toHeader),
		RemoteAddr:      remoteSignaling,
		State:           persist.SIPCallStateRinging,
		InviteAt:        &now,
	}
	if err := persist.CreateSIPCall(ctx, db, row); err != nil {
		logger.Info(fmt.Sprintf("[persist] call=%s create row failed: %v", callID, err))
		return nil
	}
	p := &CallPersister{db: db, callID: callID, transport: row.Transport}
	// Stamp a "call.started" event so the per-call timeline starts here.
	p.AppendEvent(ctx, persist.EventKindCallStarted, persist.EventLevelInfo,
		JSONObject(map[string]any{
			"transport":  row.Transport,
			"direction":  direction,
			"from":       fromHeader,
			"to":         toHeader,
			"remote":     remoteSignaling,
			"user_agent": userAgent,
		}))
	return p
}

// OnAccept stamps the negotiated codec and RTP topology and flips state
// to `established`. Called once the MediaLeg is built.
func (p *CallPersister) OnAccept(ctx context.Context, leg *session.MediaLeg, remoteRTP string) {
	if p == nil || leg == nil {
		return
	}
	neg := leg.NegotiatedSDP()
	localRTP := ""
	if rtpSess := leg.RTPSession(); rtpSess != nil && rtpSess.LocalAddr != nil {
		localRTP = rtpSess.LocalAddr.String()
	}
	p.OnAcceptMeta(ctx, strings.ToLower(neg.Name), int(neg.PayloadType), neg.ClockRate, localRTP, remoteRTP)
}

// OnAcceptMeta is the codec-only variant used by transports that don't
// carry an RTP session of their own (xiaozhi WS, WebRTC). codec is the
// wire codec name ("opus", "pcm", "pcma", …); sampleRate is the
// negotiated clock; localRTPAddr / remoteRTPAddr may be empty.
func (p *CallPersister) OnAcceptMeta(ctx context.Context, codec string, payloadType int, sampleRate int, localRTPAddr, remoteRTPAddr string) {
	if p == nil {
		return
	}
	now := time.Now().UTC()
	upd := map[string]any{
		"state":      persist.SIPCallStateEstablished,
		"ack_at":     now,
		"codec":      strings.ToLower(strings.TrimSpace(codec)),
		"clock_rate": sampleRate,
	}
	if payloadType > 0 {
		upd["payload_type"] = payloadType
	}
	if localRTPAddr != "" {
		upd["local_rtp_addr"] = localRTPAddr
	}
	if remoteRTPAddr != "" {
		upd["remote_rtp_addr"] = remoteRTPAddr
	}
	if _, err := persist.UpdateSIPCallStateByCallID(ctx, p.db, p.callID, upd); err != nil {
		logger.Info(fmt.Sprintf("[persist] call=%s onAccept update failed: %v", p.callID, err))
	}
}

// OnTerminate stamps end timestamps, duration and end status. Idempotent.
func (p *CallPersister) OnTerminate(ctx context.Context, reason string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	now := time.Now().UTC()
	row, err := persist.FindSIPCallByCallID(ctx, p.db, p.callID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Info(fmt.Sprintf("[persist] call=%s onTerminate find failed: %v", p.callID, err))
	}
	upd := map[string]any{
		"state":          persist.SIPCallStateEnded,
		"bye_at":         now,
		"ended_at":       now,
		"end_status":     ClassifyEndStatus(reason),
		"failure_reason": reason,
	}
	if row.InviteAt != nil && !row.InviteAt.IsZero() {
		upd["duration_sec"] = int(now.Sub(row.InviteAt.UTC()).Seconds())
	}
	if _, err := persist.UpdateSIPCallStateByCallID(ctx, p.db, p.callID, upd); err != nil {
		logger.Info(fmt.Sprintf("[persist] call=%s onTerminate update failed: %v", p.callID, err))
	}
}

// OnASRFinal remembers the most recent user transcript so the next
// OnTurn can pair it with the assistant text into a complete dialog turn.
func (p *CallPersister) OnASRFinal(text string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.lastFinal = strings.TrimSpace(text)
	p.finalAt = time.Now().UTC()
	p.mu.Unlock()
}

// OnTurn appends a SIPCallDialogTurn after each TTS speak finishes. If
// the dialog app supplied CommandMeta with UserText, that overrides the
// buffered ASR final (e.g. when the dialog app rephrased the input).
func (p *CallPersister) OnTurn(ctx context.Context, ev gateway.TurnEvent) {
	if p == nil {
		return
	}
	p.mu.Lock()
	userText := p.lastFinal
	p.lastFinal = ""
	p.mu.Unlock()
	if ev.Meta != nil && strings.TrimSpace(ev.Meta.UserText) != "" {
		userText = ev.Meta.UserText
	}
	turn := persist.SIPCallDialogTurn{
		ASRText:        userText,
		LLMText:        ev.LLMText,
		At:             time.Now().UTC(),
		TTSMs:          ev.DurationMs,
		TTSFirstByteMs: ev.TTSFirstByteMs,
		E2EFirstByteMs: ev.E2EFirstByteMs,
	}
	if ev.Meta != nil {
		turn.LLMModel = ev.Meta.LLMModel
		turn.LLMFirstMs = ev.Meta.LLMFirstMs
		turn.LLMWallMs = ev.Meta.LLMWallMs
	}
	if _, err := persist.AppendSIPCallTurn(ctx, p.db, p.callID, turn); err != nil {
		logger.Info(fmt.Sprintf("[persist] call=%s onTurn append failed: %v", p.callID, err))
	}
}

// OnRecording stamps the final WAV path and size after flush. Best-effort.
func (p *CallPersister) OnRecording(ctx context.Context, url string, bytes int) {
	if p == nil {
		return
	}
	upd := map[string]any{
		"recording_url":       url,
		"recording_wav_bytes": bytes,
	}
	if _, err := persist.UpdateSIPCallStateByCallID(ctx, p.db, p.callID, upd); err != nil {
		logger.Info(fmt.Sprintf("[persist] call=%s onRecording update failed: %v", p.callID, err))
	}
}

// AppendEvent inserts one row into call_events. nil-safe.
func (p *CallPersister) AppendEvent(ctx context.Context, kind, level string, detail []byte) {
	if p == nil {
		return
	}
	if err := persist.AppendCallEvent(ctx, p.db, p.callID, kind, level, time.Now().UTC(), detail); err != nil {
		logger.Info(fmt.Sprintf("[persist] call=%s event=%s append failed: %v", p.callID, kind, err))
	}
}

// AppendMediaStats inserts one row into call_media_stats. nil-safe.
func (p *CallPersister) AppendMediaStats(ctx context.Context, s gateway.MediaStatsSample) {
	if p == nil {
		return
	}
	row := &persist.CallMediaStats{
		CallID:          p.callID,
		Transport:       p.transport,
		At:              s.At,
		Final:           s.Final,
		Codec:           s.Codec,
		ClockRate:       s.ClockRate,
		Channels:        s.Channels,
		RemoteAddr:      s.RemoteAddr,
		PacketsSent:     s.PacketsSent,
		PacketsReceived: s.PacketsReceived,
		BytesSent:       s.BytesSent,
		BytesReceived:   s.BytesReceived,
		PacketsLost:     s.PacketsLost,
		NACKsSent:       s.NACKsSent,
		NACKsReceived:   s.NACKsReceived,
		RTTMs:           s.RTTMs,
		JitterMs:        s.JitterMs,
		LossRate:        s.LossRate,
		BitrateKbps:     s.BitrateKbps,
		Note:            s.Note,
	}
	if err := persist.AppendCallMediaStats(ctx, p.db, row); err != nil {
		logger.Info(fmt.Sprintf("[persist] call=%s media-stats append failed: %v", p.callID, err))
	}
}

// AppendRecording inserts one row into call_recording. nil-safe.
func (p *CallPersister) AppendRecording(ctx context.Context, r gateway.RecordingInfo) {
	if p == nil {
		return
	}
	row := &persist.CallRecording{
		CallID:     p.callID,
		Transport:  p.transport,
		Key:        r.Key,
		URL:        r.URL,
		Format:     r.Format,
		Layout:     r.Layout,
		SampleRate: r.SampleRate,
		Channels:   r.Channels,
		Bytes:      r.Bytes,
		DurationMs: r.DurationMs,
		Hash:       r.Hash,
		Note:       r.Note,
	}
	if err := persist.AppendCallRecording(ctx, p.db, row); err != nil {
		logger.Info(fmt.Sprintf("[persist] call=%s recording append failed: %v", p.callID, err))
	}
}

// JSONObject marshals a small map to bytes for the call_events.detail
// column. Returns nil on any failure (event still written, just without
// detail) so callers don't have to guard the encoding step.
func JSONObject(m map[string]any) []byte {
	if len(m) == 0 {
		return nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return b
}

// MakePersisterFactory returns a closure suitable for the
// PersisterFactory field on the xiaozhi and WebRTC ServerConfigs. Each
// call mints a fresh CallPersister and wraps it as a SessionPersister
// so the adapter package can drive lifecycle hooks without depending on
// cmd-level types.
//
// db == nil disables persistence (returns nil so the adapter falls
// through to a no-op). direction is the SIPCall.Direction value to
// stamp on every row; transport is one of "sip" / "xiaozhi" / "webrtc".
// userAgent is read from the per-call factory call (not closed over)
// since each session has its own device / browser identity.
func MakePersisterFactory(db *gorm.DB, transport, direction string) func(ctx context.Context, callID, fromHeader, toHeader, remoteAddr string) gateway.SessionPersister {
	if db == nil {
		return nil
	}
	return func(ctx context.Context, callID, fromHeader, toHeader, remoteAddr string) gateway.SessionPersister {
		// userAgent is encoded into fromHeader by some adapters
		// (xiaozhi puts the device id there); WebRTC passes the
		// User-Agent through clientMeta as fromHeader. Either way it
		// ends up on the row.
		p := NewGenericCallPersister(ctx, db, callID, transport, direction,
			fromHeader, toHeader, remoteAddr, fromHeader)
		if p == nil {
			return nil
		}
		return p.AsSessionPersister()
	}
}

// AsSessionPersister adapts *CallPersister to the transport-agnostic
// gateway.SessionPersister interface so the xiaozhi and WebRTC adapters
// can record into voiceserver.db without importing this package's
// concrete type. The returned value is nil-safe: passing it through
// ServerConfig is fine even when persistence is disabled (db == nil) —
// the underlying CallPersister is just nil and every method becomes a
// no-op.
func (p *CallPersister) AsSessionPersister() gateway.SessionPersister {
	return persisterAdapter{p: p}
}

type persisterAdapter struct{ p *CallPersister }

func (a persisterAdapter) OnAccept(ctx context.Context, codec string, sampleRate int, remoteAddr string) {
	a.p.OnAcceptMeta(ctx, codec, 0, sampleRate, "", remoteAddr)
	// Mirror the codec into the timeline so the per-call event view
	// shows what the negotiation settled on.
	a.p.AppendEvent(ctx, persist.EventKindMediaCodec, persist.EventLevelInfo,
		JSONObject(map[string]any{"codec": codec, "sample_rate": sampleRate, "remote_addr": remoteAddr}))
	// Metrics side-effect: OnAccept is the closest analogue to "call
	// became live" across all three transports — we bump the
	// active_calls gauge here so /metrics reflects currently-bridged
	// calls, not just admitted ones.
	if a.p != nil {
		metrics.CallStarted(a.p.transport)
	}
}

func (a persisterAdapter) OnASRFinal(ctx context.Context, text string) {
	a.p.OnASRFinal(text)
	a.p.AppendEvent(ctx, persist.EventKindASRFinal, persist.EventLevelInfo,
		JSONObject(map[string]any{"text": text}))
}

func (a persisterAdapter) OnTurn(ctx context.Context, t gateway.TurnEvent) {
	a.p.OnTurn(ctx, t)
	a.p.AppendEvent(ctx, persist.EventKindTTSEnd, persist.EventLevelInfo,
		JSONObject(map[string]any{
			"utter": t.UtteranceID, "ok": t.OK,
			"dur_ms": t.DurationMs, "text": t.LLMText,
			"tts_ttfb_ms": t.TTSFirstByteMs,
			"e2e_ms":      t.E2EFirstByteMs,
		}))
	if t.E2EFirstByteMs > 0 {
		a.p.AppendEvent(ctx, persist.EventKindE2EFirstByte, persist.EventLevelInfo,
			JSONObject(map[string]any{
				"utter":       t.UtteranceID,
				"e2e_ms":      t.E2EFirstByteMs,
				"tts_ttfb_ms": t.TTSFirstByteMs,
			}))
	}
	metrics.ObserveE2EFirstByte(t.E2EFirstByteMs)
	metrics.ObserveTTSFirstByte(t.TTSFirstByteMs)
	if t.Meta != nil {
		metrics.ObserveLLMFirstByte(t.Meta.LLMFirstMs)
	}
	if !t.OK && a.p != nil {
		metrics.TTSError(a.p.transport)
	}
}

func (a persisterAdapter) OnTerminate(ctx context.Context, reason string) {
	a.p.AppendEvent(ctx, persist.EventKindCallTerminated, persist.EventLevelInfo,
		JSONObject(map[string]any{"reason": reason}))
	a.p.OnTerminate(ctx, reason)
	if a.p != nil {
		metrics.CallEnded(a.p.transport, MetricEndStatus(reason))
	}
}

func (a persisterAdapter) OnEvent(ctx context.Context, kind, level string, detail []byte) {
	a.p.AppendEvent(ctx, kind, level, detail)
}

func (a persisterAdapter) OnMediaStats(ctx context.Context, s gateway.MediaStatsSample) {
	a.p.AppendMediaStats(ctx, s)
}

func (a persisterAdapter) OnRecording(ctx context.Context, r gateway.RecordingInfo) {
	a.p.AppendRecording(ctx, r)
}

// MetricEndStatus collapses free-form teardown reasons into a small,
// fixed-cardinality vocabulary suitable for a Prometheus label.
// Distinct from the DB-side ClassifyEndStatus (which uses SIP-aligned
// status codes for the call row) — metric labels stay coarse so the
// `voiceserver_calls_total` series count never explodes when a new
// reason string is introduced upstream.
func MetricEndStatus(reason string) string {
	switch r := strings.ToLower(reason); {
	case r == "" || r == "bye" || r == "ok" || r == "device closed":
		return "ok"
	case strings.HasPrefix(r, "dialog-hangup"):
		return "dialog-hangup"
	case r == "pipeline-error" || r == "encoder-error" || r == "decoder-error":
		return "media-error"
	case r == "ice-failed" || r == "ice failed":
		return "ice-failed"
	case strings.Contains(r, "timeout"):
		return "timeout"
	default:
		return "other"
	}
}

// ClassifyEndStatus maps a free-form teardown reason to an EndStatus
// value. The reason strings come from pkg/sip/server (BYE / CANCEL /
// cleanup / …); we keep the mapping permissive — unknown reasons still
// land in the row.
func ClassifyEndStatus(reason string) string {
	r := strings.ToLower(strings.TrimSpace(reason))
	switch {
	case r == "":
		return persist.SIPCallEndUnknown
	case strings.Contains(r, "bye"):
		return persist.SIPCallEndCompletedRemote
	case strings.Contains(r, "cancel"):
		return persist.SIPCallEndCancelled
	case strings.Contains(r, "busy"):
		return persist.SIPCallEndBusy
	case strings.Contains(r, "decline"):
		return persist.SIPCallEndDeclined
	case strings.Contains(r, "transport"):
		return persist.SIPCallEndTransportError
	case strings.Contains(r, "cleanup"), strings.Contains(r, "normal"):
		return persist.SIPCallEndNormalClearing
	case strings.Contains(r, "error"), strings.Contains(r, "fail"):
		return persist.SIPCallEndServerError
	default:
		return persist.SIPCallEndCompletedLocal
	}
}

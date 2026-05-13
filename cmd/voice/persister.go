// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package main

// Per-call persistence glue between the SIP/voice control plane and the
// `pkg/persist` package. One callPersister is created at INVITE time, lives
// for the life of a SIP dialog, and is closed (final UPDATE) on teardown.
//
// It is intentionally chatty about errors but never fatal: persistence is a
// side-effect — the call must continue working even if the DB is read-only,
// the disk is full, or `voiceserver.db` was deleted out from under us.

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/persist"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/server"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/utils"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/metrics"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/recorder"
	"gorm.io/gorm"
)

// openVoiceServerDB opens (and migrates) the SQLite database used by the
// voiceserver process. The file is `voiceserver.db` in CWD by default and
// can be overridden with VOICESERVER_DB. Returns (nil, nil) when persistence
// is disabled via VOICESERVER_DB="off".
func openVoiceServerDB() (*gorm.DB, error) {
	dsn := strings.TrimSpace(os.Getenv("VOICESERVER_DB"))
	if dsn == "" {
		dsn = "voiceserver.db"
	}
	if strings.EqualFold(dsn, "off") || strings.EqualFold(dsn, "none") {
		return nil, nil
	}
	db, err := utils.InitDatabase(nil, "", dsn)
	if err != nil {
		return nil, err
	}
	if err := persist.Migrate(db); err != nil {
		_ = closeGormDB(db)
		return nil, err
	}
	return db, nil
}

func closeGormDB(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// callPersister captures the lifecycle of one SIP dialog into the SIPCall
// table. It is concurrency-safe: ASR / TTS / SIP teardown can fire from
// different goroutines without the persister losing rows.
type callPersister struct {
	db        *gorm.DB
	callID    string
	transport string // sip / xiaozhi / webrtc — for downstream tables

	mu        sync.Mutex
	closed    bool
	lastFinal string // most recent ASR final, paired with the next OnTurn
	finalAt   time.Time
}

// newCallPersister creates the SIPCall row in state=ringing for an inbound
// INVITE. Returns nil (logged) on DB failure so the call can proceed.
func newCallPersister(ctx context.Context, db *gorm.DB, inv *server.IncomingCall, direction string) *callPersister {
	if inv == nil {
		return nil
	}
	remoteSig := ""
	if inv.RemoteSignalingAddr != nil {
		remoteSig = inv.RemoteSignalingAddr.String()
	}
	return newGenericCallPersister(ctx, db, inv.CallID, "sip", direction,
		inv.FromURI, inv.ToURI, remoteSig, "")
}

// newGenericCallPersister is the transport-agnostic constructor used by
// every adapter (SIP, xiaozhi, WebRTC). It writes a row in state=ringing
// with whatever caller / callee identity the transport can supply.
//
// transport classifies the call ("sip" / "xiaozhi" / "webrtc") so dashboards
// can split metrics; userAgent is the device or browser identifier (free-
// form, e.g. "ESP32-LingByte-A0:B1:..", "Mozilla/5.0 ...", "Asterisk").
//
// Returns nil when persistence is disabled (db == nil) or when the row
// can't be created — the call still runs without a DB record.
func newGenericCallPersister(ctx context.Context, db *gorm.DB, callID, transport, direction, fromHeader, toHeader, remoteSignaling, userAgent string) *callPersister {
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
		log.Printf("[persist] call=%s create row failed: %v", callID, err)
		return nil
	}
	p := &callPersister{db: db, callID: callID, transport: row.Transport}
	// Stamp a "call.started" event so the per-call timeline starts here.
	p.appendEvent(ctx, persist.EventKindCallStarted, persist.EventLevelInfo,
		jsonObject(map[string]any{
			"transport":  row.Transport,
			"direction":  direction,
			"from":       fromHeader,
			"to":         toHeader,
			"remote":     remoteSignaling,
			"user_agent": userAgent,
		}))
	return p
}

// onAccept stamps the negotiated codec and RTP topology and flips state to
// `established`. Called once the MediaLeg is built.
func (p *callPersister) onAccept(ctx context.Context, leg *session.MediaLeg, remoteRTP string) {
	if p == nil || leg == nil {
		return
	}
	neg := leg.NegotiatedSDP()
	localRTP := ""
	if rtpSess := leg.RTPSession(); rtpSess != nil && rtpSess.LocalAddr != nil {
		localRTP = rtpSess.LocalAddr.String()
	}
	p.onAcceptMeta(ctx, strings.ToLower(neg.Name), int(neg.PayloadType), neg.ClockRate, localRTP, remoteRTP)
}

// onAcceptMeta is the codec-only variant used by transports that don't
// carry an RTP session of their own (xiaozhi WS, WebRTC). codec is the
// wire codec name ("opus", "pcm", "pcma", …); sampleRate is the
// negotiated clock; localRTPAddr / remoteRTPAddr may be empty.
func (p *callPersister) onAcceptMeta(ctx context.Context, codec string, payloadType int, sampleRate int, localRTPAddr, remoteRTPAddr string) {
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
		log.Printf("[persist] call=%s onAccept update failed: %v", p.callID, err)
	}
}

// onTerminate stamps end timestamps, duration and end status. Idempotent.
func (p *callPersister) onTerminate(ctx context.Context, reason string) {
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
		log.Printf("[persist] call=%s onTerminate find failed: %v", p.callID, err)
	}
	upd := map[string]any{
		"state":          persist.SIPCallStateEnded,
		"bye_at":         now,
		"ended_at":       now,
		"end_status":     classifyEndStatus(reason),
		"failure_reason": reason,
	}
	if row.InviteAt != nil && !row.InviteAt.IsZero() {
		upd["duration_sec"] = int(now.Sub(row.InviteAt.UTC()).Seconds())
	}
	if _, err := persist.UpdateSIPCallStateByCallID(ctx, p.db, p.callID, upd); err != nil {
		log.Printf("[persist] call=%s onTerminate update failed: %v", p.callID, err)
	}
}

// onASRFinal remembers the most recent user transcript so the next
// onTurn can pair it with the assistant text into a complete dialog turn.
func (p *callPersister) onASRFinal(text string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.lastFinal = strings.TrimSpace(text)
	p.finalAt = time.Now().UTC()
	p.mu.Unlock()
}

// onTurn appends a SIPCallDialogTurn after each TTS speak finishes. If the
// dialog app supplied CommandMeta with UserText, that overrides the buffered
// ASR final (e.g. when the dialog app rephrased the input).
func (p *callPersister) onTurn(ctx context.Context, ev gateway.TurnEvent) {
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
		log.Printf("[persist] call=%s onTurn append failed: %v", p.callID, err)
	}
}

// onRecording stamps the final WAV path and size after flush. Best-effort.
func (p *callPersister) onRecording(ctx context.Context, url string, bytes int) {
	if p == nil {
		return
	}
	upd := map[string]any{
		"recording_url":       url,
		"recording_wav_bytes": bytes,
	}
	if _, err := persist.UpdateSIPCallStateByCallID(ctx, p.db, p.callID, upd); err != nil {
		log.Printf("[persist] call=%s onRecording update failed: %v", p.callID, err)
	}
}

// appendEvent inserts one row into call_events. nil-safe.
func (p *callPersister) appendEvent(ctx context.Context, kind, level string, detail []byte) {
	if p == nil {
		return
	}
	if err := persist.AppendCallEvent(ctx, p.db, p.callID, kind, level, time.Now().UTC(), detail); err != nil {
		log.Printf("[persist] call=%s event=%s append failed: %v", p.callID, kind, err)
	}
}

// appendMediaStats inserts one row into call_media_stats. nil-safe.
func (p *callPersister) appendMediaStats(ctx context.Context, s gateway.MediaStatsSample) {
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
		log.Printf("[persist] call=%s media-stats append failed: %v", p.callID, err)
	}
}

// appendRecording inserts one row into call_recording. nil-safe.
func (p *callPersister) appendRecording(ctx context.Context, r gateway.RecordingInfo) {
	if p == nil {
		return
	}
	row := &persist.CallRecording{
		CallID:     p.callID,
		Transport:  p.transport,
		Bucket:     r.Bucket,
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
		log.Printf("[persist] call=%s recording append failed: %v", p.callID, err)
	}
}

// makeRecorderFactory returns a closure suitable for the
// RecorderFactory field on the xiaozhi and WebRTC ServerConfigs. When
// record=false (the default) it returns nil so no recorder is built and
// no WAV is written. When record=true every accepted call gets a fresh
// recorder pointing at the supplied bucket; the per-transport session
// pushes PCM into it through the call lifetime and flushes (uploads
// stereo WAV + writes call_recording row) at teardown.
func makeRecorderFactory(record bool, bucket, transport string) func(callID, codec string, sampleRate int) *recorder.Recorder {
	if !record {
		return nil
	}
	if bucket == "" {
		bucket = "voiceserver-recordings"
	}
	return func(callID, codec string, sampleRate int) *recorder.Recorder {
		return recorder.New(recorder.Config{
			CallID:        callID,
			Bucket:        bucket,
			SampleRate:    sampleRate,
			Transport:     transport,
			Codec:         codec,
			ChunkInterval: recordingChunkInterval(),
		})
	}
}

// recordingChunkInterval returns the rolling-chunk upload cadence
// configured via -record-chunk (0 = no rolling uploads). Reading
// through a function rather than the var directly keeps the rest of
// persister.go from depending on flag-defined globals.
func recordingChunkInterval() time.Duration {
	return recordChunkFlag
}

// jsonObject marshals a small map to bytes for the call_events.detail
// column. Returns nil on any failure (event still written, just without
// detail) so callers don't have to guard the encoding step.
func jsonObject(m map[string]any) []byte {
	if len(m) == 0 {
		return nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return b
}

// makePersisterFactory returns a closure suitable for the
// PersisterFactory field on the xiaozhi and WebRTC ServerConfigs. Each
// call mints a fresh callPersister and wraps it as a SessionPersister so
// the adapter package can drive lifecycle hooks without depending on
// cmd-level types.
//
// db == nil disables persistence (returns a factory that yields nil).
// direction is the SIPCall.Direction value to stamp on every row;
// transport is one of "sip" / "xiaozhi" / "webrtc". userAgent is read
// from the per-call factory call (not closed over) since each session
// has its own device / browser identity.
func makePersisterFactory(db *gorm.DB, transport, direction string) func(ctx context.Context, callID, fromHeader, toHeader, remoteAddr string) gateway.SessionPersister {
	if db == nil {
		return nil
	}
	return func(ctx context.Context, callID, fromHeader, toHeader, remoteAddr string) gateway.SessionPersister {
		// userAgent is encoded into fromHeader by some adapters
		// (xiaozhi puts the device id there); WebRTC passes the
		// User-Agent through clientMeta as fromHeader. Either way
		// it ends up on the row.
		p := newGenericCallPersister(ctx, db, callID, transport, direction,
			fromHeader, toHeader, remoteAddr, fromHeader)
		if p == nil {
			return nil
		}
		return p.asSessionPersister()
	}
}

// asSessionPersister adapts *callPersister to the transport-agnostic
// gateway.SessionPersister interface so the xiaozhi and WebRTC adapters
// can record into voiceserver.db without importing cmd/voiceserver. The
// returned value is nil-safe: passing it through ServerConfig is fine
// even when persistence is disabled (db == nil) — the underlying
// callPersister is just nil and every method becomes a no-op.
func (p *callPersister) asSessionPersister() gateway.SessionPersister {
	return persisterAdapter{p: p}
}

type persisterAdapter struct{ p *callPersister }

func (a persisterAdapter) OnAccept(ctx context.Context, codec string, sampleRate int, remoteAddr string) {
	a.p.onAcceptMeta(ctx, codec, 0, sampleRate, "", remoteAddr)
	// Mirror the codec into the timeline so the per-call event view
	// shows what the negotiation settled on.
	a.p.appendEvent(ctx, persist.EventKindMediaCodec, persist.EventLevelInfo,
		jsonObject(map[string]any{"codec": codec, "sample_rate": sampleRate, "remote_addr": remoteAddr}))
	// Metrics side-effect: OnAccept is the closest analogue to
	// "call became live" across all three transports — we bump the
	// active_calls gauge here so /metrics reflects currently-bridged
	// calls, not just admitted ones.
	if a.p != nil {
		metrics.CallStarted(a.p.transport)
	}
}
func (a persisterAdapter) OnASRFinal(ctx context.Context, text string) {
	a.p.onASRFinal(text)
	a.p.appendEvent(ctx, persist.EventKindASRFinal, persist.EventLevelInfo,
		jsonObject(map[string]any{"text": text}))
}
func (a persisterAdapter) OnTurn(ctx context.Context, t gateway.TurnEvent) {
	a.p.onTurn(ctx, t)
	a.p.appendEvent(ctx, persist.EventKindTTSEnd, persist.EventLevelInfo,
		jsonObject(map[string]any{
			"utter": t.UtteranceID, "ok": t.OK,
			"dur_ms": t.DurationMs, "text": t.LLMText,
			"tts_ttfb_ms": t.TTSFirstByteMs,
			"e2e_ms":      t.E2EFirstByteMs,
		}))
	// Emit a dedicated e2e.first_byte event only when we have a real
	// user-perceived number (non-zero means: this Speak followed an
	// ASR final and produced audio). Keeps the timeline uncluttered —
	// intra-turn sentences don't spam this event.
	if t.E2EFirstByteMs > 0 {
		a.p.appendEvent(ctx, persist.EventKindE2EFirstByte, persist.EventLevelInfo,
			jsonObject(map[string]any{
				"utter":       t.UtteranceID,
				"e2e_ms":      t.E2EFirstByteMs,
				"tts_ttfb_ms": t.TTSFirstByteMs,
			}))
	}
	// Feed the Prom histograms. ObserveX calls are no-ops for 0 /
	// missing values so the quantile buffers stay clean.
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
	a.p.appendEvent(ctx, persist.EventKindCallTerminated, persist.EventLevelInfo,
		jsonObject(map[string]any{"reason": reason}))
	a.p.onTerminate(ctx, reason)
	// Decrement active_calls and bump calls_total. We pair CallStarted
	// (OnAccept) with CallEnded (OnTerminate) rather than session
	// start/end — that way a call that never reached media bridging
	// doesn't skew the gauge.
	if a.p != nil {
		metrics.CallEnded(a.p.transport, metricEndStatus(reason))
	}
}

// metricEndStatus collapses free-form teardown reasons into a small,
// fixed-cardinality vocabulary suitable for a Prometheus label.
// Distinct from the DB-side classifyEndStatus (which uses SIP-aligned
// status codes for the call row) — metric labels stay coarse so the
// `voiceserver_calls_total` series count never explodes when a new
// reason string is introduced upstream.
func metricEndStatus(reason string) string {
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
func (a persisterAdapter) OnEvent(ctx context.Context, kind, level string, detail []byte) {
	a.p.appendEvent(ctx, kind, level, detail)
}
func (a persisterAdapter) OnMediaStats(ctx context.Context, s gateway.MediaStatsSample) {
	a.p.appendMediaStats(ctx, s)
}
func (a persisterAdapter) OnRecording(ctx context.Context, r gateway.RecordingInfo) {
	a.p.appendRecording(ctx, r)
}

// classifyEndStatus maps a free-form teardown reason to an EndStatus value.
// The reason strings come from pkg/sip/server (BYE / CANCEL / cleanup / …);
// we keep the mapping permissive — unknown reasons still land in the row.
func classifyEndStatus(reason string) string {
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

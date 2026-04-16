// Package webseat bridges an inbound SIP call to a browser over WebRTC after the caller presses transfer (SIP_TRANSFER_NUMBER=web).
package webseat

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/sip/bridge"
	siprtp "github.com/LingByte/SoulNexus/pkg/sip/rtp"
	sipSession "github.com/LingByte/SoulNexus/pkg/sip/session"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

const (
	// EnvTrackWait is max wait after returning the SDP answer for the browser to connect and send the first audio track (e.g. "90s").
	EnvTrackWait = "SIP_WEBSEAT_TRACK_WAIT"
	// EnvWSToken is the shared secret for GET /webseat/v1/ws?token=... (empty = accept any client; not recommended for production).
	EnvWSToken = "SIP_WEBSEAT_WS_TOKEN"
)

var (
	wsUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(*http.Request) bool {
			return true
		},
	}
	wsTokenMissingOnce sync.Once
)

// Config wires SIP teardown and store updates from cmd/sip (avoid import cycles).
type Config struct {
	RemoveCallSession     func(callID string)
	ForgetUASDialog       func(callID string)
	SendUASBye            func(callID string) error
	ReleaseTransferDedupe func(callID string)
	// FinalizeInboundPersist runs once when a web-seat handoff ends (BYE, hangup, or bridge teardown).
	// callID is the inbound PSTN Call-ID; initiator is "remote" (customer BYE) or "local" (operator / full hangup).
	FinalizeInboundPersist func(ctx context.Context, callID, initiator string, raw []byte, codecName string, recordSampleRate, recordOpusChannels int)
}

// Hub tracks pending joins and active bridges.
type Hub struct {
	cfg Config

	mu       sync.Mutex
	awaiting map[string]*awaitEntry   // inbound Call-ID
	active   map[string]*activeBridge // inbound Call-ID

	wsMu    sync.Mutex
	wsConns map[*websocket.Conn]struct{}
}

type awaitEntry struct {
	cs *sipSession.CallSession
	lg *zap.Logger
	at time.Time
}

type activeBridge struct {
	callID  string
	inbound *sipSession.CallSession
	br      *bridge.TwoLegPCMBridge
	pc      *webrtc.PeerConnection
}

var defaultHub *Hub

// InitDefault configures the process-wide hub (call once from main).
func InitDefault(cfg Config) {
	defaultHub = &Hub{
		cfg:      cfg,
		awaiting: make(map[string]*awaitEntry),
		active:   make(map[string]*activeBridge),
		wsConns:  make(map[*websocket.Conn]struct{}),
	}
}

// JoinHTTP serves POST join (browser WebRTC offer). Mount on Gin as WrapF(JoinHTTP).
func JoinHTTP(w http.ResponseWriter, r *http.Request) {
	if defaultHub == nil {
		http.Error(w, "webseat not initialized", http.StatusServiceUnavailable)
		return
	}
	defaultHub.handleJoin(w, r)
}

// HangupHTTP serves POST hangup (JSON body call_id).
func HangupHTTP(w http.ResponseWriter, r *http.Request) {
	if defaultHub == nil {
		http.Error(w, "webseat not initialized", http.StatusServiceUnavailable)
		return
	}
	defaultHub.handleAgentHangup(w, r)
}

// RejectHTTP serves POST reject (JSON body call_id).
func RejectHTTP(w http.ResponseWriter, r *http.Request) {
	if defaultHub == nil {
		http.Error(w, "webseat not initialized", http.StatusServiceUnavailable)
		return
	}
	defaultHub.handleAgentReject(w, r)
}

// WebSocketHTTP serves GET WebSocket upgrade (?token=...).
func WebSocketHTTP(w http.ResponseWriter, r *http.Request) {
	if defaultHub == nil {
		http.Error(w, "webseat not initialized", http.StatusServiceUnavailable)
		return
	}
	defaultHub.handleWebSocket(w, r)
}

// RegisterAwaiting marks inbound as waiting for browser WebRTC join (after AI media stopped).
func RegisterAwaiting(callID string, cs *sipSession.CallSession, lg *zap.Logger) error {
	if defaultHub == nil {
		return errors.New("webseat: InitDefault not called")
	}
	if strings.TrimSpace(callID) == "" || cs == nil {
		return errors.New("webseat: invalid call or session")
	}
	h := defaultHub
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.active[callID] != nil {
		return fmt.Errorf("webseat: call %q already bridged", callID)
	}
	h.awaiting[callID] = &awaitEntry{cs: cs, lg: lg, at: time.Now()}
	go h.awaitWatchdog(callID)
	if lg != nil {
		lg.Info("webseat: awaiting browser join", zap.String("call_id", callID))
	}
	go h.broadcastIncoming(callID)
	return nil
}

func webseatWSTokenOK(r *http.Request) bool {
	expected := strings.TrimSpace(utils.GetEnv(EnvWSToken))
	got := strings.TrimSpace(r.URL.Query().Get("token"))
	if expected == "" {
		wsTokenMissingOnce.Do(func() {
			if logger.Lg != nil {
				logger.Lg.Warn("webseat: SIP_WEBSEAT_WS_TOKEN is empty; WebSocket accepts any client")
			}
		})
		return true
	}
	if len(got) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

func (h *Hub) wsAdd(c *websocket.Conn) {
	if h == nil || c == nil {
		return
	}
	h.wsMu.Lock()
	h.wsConns[c] = struct{}{}
	n := len(h.wsConns)
	h.wsMu.Unlock()
	if logger.Lg != nil {
		logger.Lg.Info("webseat: websocket client connected", zap.Int("clients", n))
	}
	h.broadcastPresence()
}

func (h *Hub) wsRemove(c *websocket.Conn) {
	if h == nil || c == nil {
		return
	}
	h.wsMu.Lock()
	delete(h.wsConns, c)
	h.wsMu.Unlock()
}

// broadcastIncoming notifies all WS clients that a call is waiting for join (JSON: type=incoming, call_id).
func (h *Hub) broadcastIncoming(callID string) {
	if h == nil || strings.TrimSpace(callID) == "" {
		return
	}
	msg, err := json.Marshal(map[string]any{
		"type":    "incoming",
		"call_id": callID,
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return
	}
	h.wsMu.Lock()
	list := make([]*websocket.Conn, 0, len(h.wsConns))
	for c := range h.wsConns {
		list = append(list, c)
	}
	h.wsMu.Unlock()
	for _, c := range list {
		_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
			_ = c.Close()
			h.wsRemove(c)
		}
	}
}

// broadcastPresence notifies all agent WS clients of current listener count (replaces a separate HTTP presence probe).
func (h *Hub) broadcastPresence() {
	if h == nil {
		return
	}
	h.wsMu.Lock()
	n := len(h.wsConns)
	list := make([]*websocket.Conn, 0, n)
	for c := range h.wsConns {
		list = append(list, c)
	}
	h.wsMu.Unlock()
	msg, err := json.Marshal(map[string]any{
		"type":       "presence",
		"ws_clients": n,
		"online":     n > 0,
		"ts":         time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return
	}
	for _, c := range list {
		_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
			_ = c.Close()
			h.wsRemove(c)
		}
	}
}

// handleWebSocket: GET /webseat/v1/ws?token=... — push incoming call_id to the agent page.
func (h *Hub) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	if !webseatWSTokenOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	h.wsAdd(conn)
	go func(c *websocket.Conn) {
		defer func() {
			_ = c.Close()
			h.wsRemove(c)
			h.broadcastPresence()
		}()
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}(conn)
}

func (h *Hub) awaitWatchdog(callID string) {
	wait := 5 * time.Minute
	if v := strings.TrimSpace(utils.GetEnv("SIP_WEBSEAT_JOIN_TIMEOUT")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			wait = d
		}
	}
	time.Sleep(wait)
	h.mu.Lock()
	defer h.mu.Unlock()
	if e, ok := h.awaiting[callID]; ok {
		delete(h.awaiting, callID)
		if e.lg != nil {
			e.lg.Warn("webseat: join timeout, releasing slot", zap.String("call_id", callID))
		}
		if h.cfg.ReleaseTransferDedupe != nil {
			h.cfg.ReleaseTransferDedupe(callID)
		}
	}
}

// IsPendingOrActive is true while waiting for WebRTC or while a bridge is running (suppress late ACK voice attach).
func IsPendingOrActive(callID string) bool {
	if defaultHub == nil || callID == "" {
		return false
	}
	h := defaultHub
	h.mu.Lock()
	defer h.mu.Unlock()
	_, p := h.awaiting[callID]
	_, a := h.active[callID]
	return p || a
}

// IsActive is true only when WebRTC bridge is active (browser already joined).
func IsActive(callID string) bool {
	if defaultHub == nil || callID == "" {
		return false
	}
	h := defaultHub
	h.mu.Lock()
	defer h.mu.Unlock()
	_, a := h.active[callID]
	return a
}

// HangupIfCustomerBye tears down Web seat when the PSTN side sends BYE. Returns true if handled.
func HangupIfCustomerBye(callID string) bool {
	return teardownWebSeat(callID, false)
}

// HangupFull tears down Web seat and BYE the customer (browser left or operator hangup).
func HangupFull(callID string) bool {
	return teardownWebSeat(callID, true)
}

func persistSnapshotInbound(cs *sipSession.CallSession) (raw []byte, codecName string, recSR, recOpusCh int) {
	if cs == nil {
		return nil, "", 0, 0
	}
	raw = cs.TakeRecording()
	codecName = cs.NegotiatedCodec().Name
	src := cs.SourceCodec()
	recSR = src.SampleRate
	recOpusCh = src.OpusDecodeChannels
	if recOpusCh < 1 {
		recOpusCh = src.Channels
	}
	return raw, codecName, recSR, recOpusCh
}

func (h *Hub) emitFinalizePersist(callID, initiator string, cs *sipSession.CallSession) {
	if h == nil || strings.TrimSpace(callID) == "" || h.cfg.FinalizeInboundPersist == nil {
		return
	}
	init := strings.TrimSpace(initiator)
	if init == "" {
		init = "remote"
	}
	raw, codec, sr, ch := persistSnapshotInbound(cs)
	go h.cfg.FinalizeInboundPersist(context.Background(), callID, init, raw, codec, sr, ch)
}

func teardownWebSeat(callID string, sendByeToCustomer bool) bool {
	if defaultHub == nil || callID == "" {
		return false
	}
	h := defaultHub
	h.mu.Lock()
	ab, ok := h.active[callID]
	if !ok {
		entry, waiting := h.awaiting[callID]
		if waiting {
			delete(h.awaiting, callID)
			h.mu.Unlock()
			initiator := "remote"
			if sendByeToCustomer {
				initiator = "local"
			}
			var cs *sipSession.CallSession
			if entry != nil {
				cs = entry.cs
			}
			h.emitFinalizePersist(callID, initiator, cs)
			if cs != nil {
				cs.Stop()
			}
			if sendByeToCustomer && h.cfg.SendUASBye != nil {
				_ = h.cfg.SendUASBye(callID)
			}
			if h.cfg.RemoveCallSession != nil {
				h.cfg.RemoveCallSession(callID)
			}
			if h.cfg.ForgetUASDialog != nil {
				h.cfg.ForgetUASDialog(callID)
			}
			if h.cfg.ReleaseTransferDedupe != nil {
				h.cfg.ReleaseTransferDedupe(callID)
			}
			return true
		}
		h.mu.Unlock()
		return false
	}
	delete(h.active, callID)
	h.mu.Unlock()

	initiator := "remote"
	if sendByeToCustomer {
		initiator = "local"
	}
	h.emitFinalizePersist(callID, initiator, ab.inbound)

	// br is nil until the browser connects and OnTrack runs (async after join response).
	if ab.br != nil {
		ab.br.Stop()
	}
	if ab.pc != nil {
		_ = ab.pc.Close()
	}
	if ab.inbound != nil {
		ab.inbound.CloseRTPOnly()
	}
	if sendByeToCustomer && h.cfg.SendUASBye != nil {
		_ = h.cfg.SendUASBye(callID)
	}
	if h.cfg.RemoveCallSession != nil {
		h.cfg.RemoveCallSession(callID)
	}
	if h.cfg.ForgetUASDialog != nil {
		h.cfg.ForgetUASDialog(callID)
	}
	if logger.Lg != nil {
		logger.Lg.Info("webseat: session torn down", zap.String("call_id", callID), zap.Bool("bye_customer", sendByeToCustomer))
	}
	return true
}

type joinBody struct {
	CallID     string                    `json:"call_id"`
	SDP        string                    `json:"sdp"`
	Type       string                    `json:"type"`
	Candidates []webrtc.ICECandidateInit `json:"candidates"`
}

func (h *Hub) handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body joinBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "json", http.StatusBadRequest)
		return
	}
	callID := strings.TrimSpace(body.CallID)
	if callID == "" || strings.TrimSpace(body.SDP) == "" {
		http.Error(w, "call_id and sdp required", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	entry, ok := h.awaiting[callID]
	if ok {
		delete(h.awaiting, callID)
	}
	h.mu.Unlock()
	if !ok {
		http.Error(w, "unknown or expired call_id", http.StatusNotFound)
		return
	}

	lg := entry.lg
	if lg == nil && logger.Lg != nil {
		lg = logger.Lg
	}
	if lg == nil {
		lg = zap.NewNop()
	}

	answer, err := h.completeJoin(r.Context(), callID, entry.cs, body, lg)
	if err != nil {
		if h.cfg.ReleaseTransferDedupe != nil {
			h.cfg.ReleaseTransferDedupe(callID)
		}
		lg.Warn("webseat: join failed", zap.String("call_id", callID), zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(answer)
}

// handleAgentHangup ends the web seat leg and sends BYE to the PSTN customer (JSON body: { "call_id": "..." }).
func (h *Hub) handleAgentHangup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		CallID string `json:"call_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "json", http.StatusBadRequest)
		return
	}
	callID := strings.TrimSpace(body.CallID)
	if callID == "" {
		http.Error(w, "call_id required", http.StatusBadRequest)
		return
	}
	if !HangupFull(callID) {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleAgentReject declines an awaiting or active web seat (same effect as hangup: BYE customer, teardown).
func (h *Hub) handleAgentReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		CallID string `json:"call_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "json", http.StatusBadRequest)
		return
	}
	callID := strings.TrimSpace(body.CallID)
	if callID == "" {
		http.Error(w, "call_id required", http.StatusBadRequest)
		return
	}
	if !HangupFull(callID) {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type joinAnswer struct {
	Type string `json:"type"`
	SDP  string `json:"sdp"`
}

func (h *Hub) completeJoin(ctx context.Context, callID string, inbound *sipSession.CallSession, body joinBody, lg *zap.Logger) (*joinAnswer, error) {
	m := newMediaEngine()
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))
	pc, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: defaultICEServers(),
	})
	if err != nil {
		return nil, err
	}

	// OnTrack fires only after the browser applies our answer and RTP flows — never block the HTTP response on this.
	trackCh := make(chan *webrtc.TrackRemote, 1)
	pc.OnTrack(func(tr *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		select {
		case trackCh <- tr:
		default:
		}
	})

	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		if s == webrtc.PeerConnectionStateFailed || s == webrtc.PeerConnectionStateClosed {
			_ = teardownWebSeat(callID, true)
		}
	})

	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: body.SDP}
	if err := pc.SetRemoteDescription(offer); err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("SetRemoteDescription: %w", err)
	}
	for _, c := range body.Candidates {
		_ = pc.AddICECandidate(c)
	}

	opusCap := webrtc.RTPCodecCapability{
		MimeType:    webrtc.MimeTypeOpus,
		ClockRate:   48000,
		Channels:    2,
		SDPFmtpLine: "minptime=10;useinbandfec=1",
	}
	txLocal, err := webrtc.NewTrackLocalStaticSample(opusCap, "audio", "soulnexus")
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	if _, err := pc.AddTrack(txLocal); err != nil {
		_ = pc.Close()
		return nil, err
	}

	ans, err := pc.CreateAnswer(nil)
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	if err := pc.SetLocalDescription(ans); err != nil {
		_ = pc.Close()
		return nil, err
	}

	gatherComplete := webrtc.GatheringCompletePromise(pc)
	select {
	case <-gatherComplete:
	case <-time.After(15 * time.Second):
		_ = pc.Close()
		return nil, errors.New("ICE gather timeout")
	case <-ctx.Done():
		_ = pc.Close()
		return nil, ctx.Err()
	}

	h.mu.Lock()
	h.active[callID] = &activeBridge{
		callID:  callID,
		inbound: inbound,
		br:      nil,
		pc:      pc,
	}
	h.mu.Unlock()

	go h.waitRemoteTrackAndBridge(callID, inbound, pc, txLocal, trackCh, lg)

	ld := pc.LocalDescription()
	lg.Info("webseat: answer sent, waiting for browser RTP / OnTrack", zap.String("call_id", callID))
	return &joinAnswer{Type: ld.Type.String(), SDP: ld.SDP}, nil
}

func (h *Hub) waitRemoteTrackAndBridge(
	callID string,
	inbound *sipSession.CallSession,
	pc *webrtc.PeerConnection,
	txLocal *webrtc.TrackLocalStaticSample,
	trackCh <-chan *webrtc.TrackRemote,
	lg *zap.Logger,
) {
	wait := 90 * time.Second
	if v := strings.TrimSpace(utils.GetEnv(EnvTrackWait)); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			wait = d
		}
	}

	var remoteTrack *webrtc.TrackRemote
	select {
	case remoteTrack = <-trackCh:
	case <-time.After(wait):
		if lg != nil {
			lg.Warn("webseat: no remote audio track after answer (ICE or mic?)", zap.String("call_id", callID), zap.Duration("wait", wait))
		}
		if h.cfg.ReleaseTransferDedupe != nil {
			h.cfg.ReleaseTransferDedupe(callID)
		}
		_ = teardownWebSeat(callID, false)
		return
	}
	if remoteTrack == nil {
		return
	}

	h.mu.Lock()
	ab, ok := h.active[callID]
	if !ok || ab == nil || ab.pc != pc {
		h.mu.Unlock()
		return
	}
	h.mu.Unlock()

	webCodec := mediaFromRemoteTrack(remoteTrack)
	wt := NewTransport(remoteTrack, txLocal, webCodec)

	// Keep caller media alive during "awaiting join" so transfer ringing can be played.
	// Stop MediaSession right before building bridge transports to avoid dual RTP readers.
	inbound.StopMediaPreserveRTP()

	ccIn := inbound.SourceCodec()
	callerRx := siprtp.NewSIPRTPTransport(inbound.RTPSession(), ccIn, media.DirectionInput, inbound.DTMFPayloadType())
	callerTx := siprtp.NewSIPRTPTransport(inbound.RTPSession(), ccIn, media.DirectionOutput, 0)

	br, err := bridge.NewTwoLegPCMBridge(callerRx, callerTx, wt, wt)
	if err != nil {
		if lg != nil {
			lg.Warn("webseat: pcm bridge build failed", zap.String("call_id", callID), zap.Error(err))
		}
		if h.cfg.ReleaseTransferDedupe != nil {
			h.cfg.ReleaseTransferDedupe(callID)
		}
		_ = teardownWebSeat(callID, false)
		return
	}

	h.mu.Lock()
	ab, ok = h.active[callID]
	if !ok || ab == nil || ab.pc != pc {
		h.mu.Unlock()
		_ = pc.Close()
		return
	}
	ab.br = br
	h.mu.Unlock()

	br.Start()
	if lg != nil {
		lg.Info("webseat: bridge started", zap.String("call_id", callID), zap.String("web_codec", webCodec.Codec))
	}
}

func newMediaEngine() *webrtc.MediaEngine {
	me := &webrtc.MediaEngine{}
	_ = me.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   48000,
			Channels:    2,
			SDPFmtpLine: "minptime=10;useinbandfec=1",
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio)
	_ = me.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU, ClockRate: 8000},
		PayloadType:        0,
	}, webrtc.RTPCodecTypeAudio)
	return me
}

func defaultICEServers() []webrtc.ICEServer {
	raw := strings.TrimSpace(utils.GetEnv("SIP_WEBSEAT_ICE_SERVERS"))
	if raw != "" {
		var servers []webrtc.ICEServer
		if err := json.Unmarshal([]byte(raw), &servers); err == nil && len(servers) > 0 {
			return servers
		}
	}
	return []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}}
}

func mediaFromRemoteTrack(tr *webrtc.TrackRemote) media.CodecConfig {
	c := tr.Codec()
	mime := strings.ToLower(c.MimeType)
	ch := int(c.Channels)
	if ch < 1 {
		ch = 1
	}
	switch {
	case strings.Contains(mime, "opus"):
		decodeCh := ch
		if decodeCh > 2 {
			decodeCh = 2
		}
		return media.CodecConfig{
			Codec:              "opus",
			SampleRate:         int(c.ClockRate),
			Channels:           1,
			BitDepth:           16,
			FrameDuration:      "20ms",
			OpusDecodeChannels: decodeCh,
		}
	default:
		return media.CodecConfig{Codec: "opus", SampleRate: 48000, Channels: 1, BitDepth: 16, FrameDuration: "20ms", OpusDecodeChannels: 2}
	}
}

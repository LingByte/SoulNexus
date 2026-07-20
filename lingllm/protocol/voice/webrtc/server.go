package webrtc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/lingllm/protocol/voice/asr"
	"github.com/LingByte/lingllm/protocol/voice/gateway"
	"github.com/LingByte/lingllm/protocol/voice/transport"
	pionwebrtc "github.com/pion/webrtc/v4"
)

// ServerConfig wires the WebRTC adapter to dialog.Session.
type ServerConfig struct {
	SessionFactory  transport.SessionFactory
	DialogWSURL     string
	Engine          EngineConfig
	ICEServers      []pionwebrtc.ICEServer
	CallIDPrefix    string
	AllowedOrigins  []string
	DenoiserFactory func() asr.Denoiser
	ConfigureClient func(*gateway.ClientConfig)
	OnSessionStart  func(ctx context.Context, callID, peerInfo string)
	OnSessionEnd    func(ctx context.Context, callID, reason string)
}

// Server is the HTTP signaling entry point.
type Server struct {
	cfg      ServerConfig
	api      *pionwebrtc.API
	mu       sync.Mutex
	sessions map[string]*session
}

// NewServer validates cfg and builds the shared pion API.
func NewServer(cfg ServerConfig) (*Server, error) {
	if cfg.SessionFactory == nil {
		return nil, errors.New("webrtc: nil SessionFactory")
	}
	if strings.TrimSpace(cfg.DialogWSURL) == "" {
		return nil, errors.New("webrtc: empty DialogWSURL")
	}
	if cfg.CallIDPrefix == "" {
		cfg.CallIDPrefix = "wrtc"
	}
	if len(cfg.ICEServers) > 0 {
		cfg.Engine.ICEServers = append(cfg.Engine.ICEServers, cfg.ICEServers...)
	}
	if len(cfg.Engine.ICEServers) == 0 {
		cfg.Engine.ICEServers = []pionwebrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		}
	}
	api, err := BuildAPI(cfg.Engine)
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:      cfg,
		api:      api,
		sessions: make(map[string]*session),
	}, nil
}

// HandleOffer completes SDP negotiation for one browser call.
func (s *Server) HandleOffer(w http.ResponseWriter, r *http.Request) {
	s.applyCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	var offer OfferRequest
	if err := json.Unmarshal(body, &offer); err != nil {
		http.Error(w, "json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(offer.SDP) == "" || offer.Type != "offer" {
		http.Error(w, "expected {sdp,type:offer}", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	sess, answer, err := newSession(ctx, s.cfg, s.api, offer)
	if err != nil {
		http.Error(w, "handshake: "+err.Error(), http.StatusInternalServerError)
		return
	}
	sess.clientMeta = clientInfo(r)

	s.mu.Lock()
	s.sessions[sess.callID] = sess
	s.mu.Unlock()
	go s.cleanupOnClose(sess)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(answer)
}

// HandleHangup terminates a call by ID.
func (s *Server) HandleHangup(w http.ResponseWriter, r *http.Request) {
	s.applyCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("call_id"))
	if id == "" {
		http.Error(w, "missing call_id", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	sess := s.sessions[id]
	delete(s.sessions, id)
	s.mu.Unlock()
	if sess == nil {
		// Idempotent: session may already have torn down after pipeline/ICE failure.
		w.WriteHeader(http.StatusNoContent)
		return
	}
	sess.teardown("client-hangup")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) cleanupOnClose(sess *session) {
	<-sess.done
	s.mu.Lock()
	delete(s.sessions, sess.callID)
	s.mu.Unlock()
}

func (s *Server) applyCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}
	allow := len(s.cfg.AllowedOrigins) == 0
	if !allow {
		for _, a := range s.cfg.AllowedOrigins {
			if a == origin || a == "*" {
				allow = true
				break
			}
		}
	}
	if !allow {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Vary", "Origin")
}

func clientInfo(r *http.Request) string {
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	ua := strings.TrimSpace(r.Header.Get("User-Agent"))
	addr := r.RemoteAddr
	if xff != "" {
		addr = xff
	}
	if ua == "" {
		return addr
	}
	return fmt.Sprintf("%s ua=%q", addr, ua)
}

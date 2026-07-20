package xiaozhi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/lingllm/protocol/voice/gateway"
	"github.com/LingByte/lingllm/protocol/voice/transport"
	"github.com/gorilla/websocket"
)

// ServerConfig wires the xiaozhi adapter.
type ServerConfig struct {
	SessionFactory       transport.SessionFactory
	RealtimeFactory      RealtimeAgentFactory
	DialogWSURL          string
	CallIDPrefix         string
	ResolveDevicePayload func(ctx context.Context, deviceID string) ([]byte, error)
	ConfigureClient      func(*gateway.ClientConfig)
	OnSessionStart       func(ctx context.Context, callID, deviceID string)
	OnSessionEnd         func(ctx context.Context, callID, reason string)
	// ConfigureSession allows third-party extensions to register custom message handlers
	ConfigureSession func(session *wsSession)
}

// Server accepts xiaozhi WebSocket connections.
type Server struct {
	cfg ServerConfig
	up  websocket.Upgrader
}

// NewServer validates cfg.
func NewServer(cfg ServerConfig) (*Server, error) {
	if cfg.SessionFactory == nil {
		return nil, errors.New("xiaozhi: nil SessionFactory")
	}
	if strings.TrimSpace(cfg.DialogWSURL) == "" {
		return nil, errors.New("xiaozhi: empty DialogWSURL")
	}
	if cfg.RealtimeFactory == nil {
		return nil, errors.New("xiaozhi: nil RealtimeFactory")
	}
	if cfg.CallIDPrefix == "" {
		cfg.CallIDPrefix = "xz"
	}
	return &Server{
		cfg: cfg,
		up: websocket.Upgrader{
			CheckOrigin:     func(_ *http.Request) bool { return true },
			ReadBufferSize:  4 * 1024,
			WriteBufferSize: 16 * 1024,
		},
	}, nil
}

// Handle upgrades the HTTP connection to the xiaozhi protocol.
func (s *Server) Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := s.up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	deviceID := strings.TrimSpace(r.Header.Get("Device-Id"))
	if deviceID == "" {
		deviceID = strings.TrimSpace(r.Header.Get("device-id"))
	}
	if deviceID == "" {
		deviceID = strings.TrimSpace(r.URL.Query().Get("device-id"))
	}
	if deviceID == "" {
		deviceID = strings.TrimSpace(r.URL.Query().Get("device_id"))
	}

	callID := fmt.Sprintf("%s-%d", s.cfg.CallIDPrefix, time.Now().UnixNano())
	cfg := s.cfg

	payloadRaw := strings.TrimSpace(r.URL.Query().Get("payload"))
	if payloadRaw == "" && deviceID != "" && cfg.ResolveDevicePayload != nil {
		resolved, err := cfg.ResolveDevicePayload(r.Context(), deviceID)
		if err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, MakeError("device binding failed", true))
			_ = conn.Close()
			return
		}
		payloadRaw = string(resolved)
	}
	if payloadRaw != "" {
		merged, err := gateway.MergeDialogPayloadQuery(strings.TrimSpace(cfg.DialogWSURL), []byte(payloadRaw))
		if err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, MakeError("invalid payload query", true))
			_ = conn.Close()
			return
		}
		cfg.DialogWSURL = merged
	}

	sess := newSession(cfg, conn, callID, deviceID)

	// Allow third-party extensions to configure the session
	if cfg.ConfigureSession != nil {
		cfg.ConfigureSession(sess)
	}

	sess.run(r.Context())
}

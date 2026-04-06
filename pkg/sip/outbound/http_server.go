package outbound

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// Dialer is the minimal outbound caller interface used by HTTP API.
type Dialer interface {
	Dial(ctx context.Context, req DialRequest) (callID string, err error)
}

// StartDialHTTPServer starts an optional HTTP API for proactive outbound dialing.
// Endpoint: POST /sip/v1/outbound/dial
func StartDialHTTPServer(addr, token string, d Dialer) (*http.Server, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" || d == nil {
		return nil, nil
	}
	api := &dialHTTPAPI{token: strings.TrimSpace(token), dialer: d}
	mux := http.NewServeMux()
	mux.HandleFunc("/sip/v1/outbound/dial", api.handleDial)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("SoulNexus SIP outbound API: POST /sip/v1/outbound/dial\n"))
			return
		}
		http.NotFound(w, r)
	})

	srv := &http.Server{Addr: addr, Handler: dialCORSMiddleware(mux)}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("sip outbound http: listen %s: %w", addr, err)
	}
	go func() {
		if logger.Lg != nil {
			logger.Lg.Info("sip outbound http listening", zap.String("addr", addr))
		}
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) && logger.Lg != nil {
			logger.Lg.Warn("sip outbound http serve", zap.Error(err))
		}
	}()
	return srv, nil
}

type dialHTTPAPI struct {
	token string
	dialer Dialer
}

type dialRequestBody struct {
	Scenario          string `json:"scenario"`
	MediaProfile      string `json:"media_profile"`
	ScriptID          string `json:"script_id"`
	CorrelationID     string `json:"correlation_id"`
	CallerUser        string `json:"caller_user"`
	CallerDisplayName string `json:"caller_display_name"`

	RequestURI    string `json:"request_uri"`
	SignalingAddr string `json:"signaling_addr"`
	TargetNumber  string `json:"target_number"`
	OutboundHost  string `json:"outbound_host"`
	OutboundPort  int    `json:"outbound_port"`
}

func dialCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Token")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *dialHTTPAPI) authorized(r *http.Request) bool {
	expected := strings.TrimSpace(a.token)
	if expected == "" {
		return true
	}
	got := strings.TrimSpace(r.Header.Get("X-API-Token"))
	if got == "" {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			got = strings.TrimSpace(auth[7:])
		}
	}
	if len(got) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

func (a *dialHTTPAPI) handleDial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	if !a.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body dialRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "json", http.StatusBadRequest)
		return
	}
	req, target, err := toDialRequest(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	callID, err := a.dialer.Dial(ctx, req)
	if err != nil {
		http.Error(w, "dial failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"call_id":        callID,
		"scenario":       req.Scenario,
		"media_profile":  req.MediaProfile,
		"target_uri":     target.RequestURI,
		"signaling_addr": target.SignalingAddr,
	})
}

func toDialRequest(body dialRequestBody) (DialRequest, DialTarget, error) {
	target, err := dialTargetFromBody(body)
	if err != nil {
		return DialRequest{}, DialTarget{}, err
	}
	req := DialRequest{
		Scenario:          ScenarioCampaign,
		Target:            target,
		MediaProfile:      MediaProfileAI,
		ScriptID:          strings.TrimSpace(body.ScriptID),
		CorrelationID:     strings.TrimSpace(body.CorrelationID),
		CallerUser:        strings.TrimSpace(body.CallerUser),
		CallerDisplayName: strings.TrimSpace(body.CallerDisplayName),
	}
	if s := strings.TrimSpace(strings.ToLower(body.Scenario)); s != "" {
		req.Scenario = Scenario(s)
	}
	if mp := strings.TrimSpace(strings.ToLower(body.MediaProfile)); mp != "" {
		req.MediaProfile = MediaProfile(mp)
	}
	return req, target, nil
}

func dialTargetFromBody(body dialRequestBody) (DialTarget, error) {
	reqURI := normalizeSIPRequestURI(strings.TrimSpace(body.RequestURI))
	sig := strings.TrimSpace(body.SignalingAddr)
	if reqURI != "" {
		if sig == "" {
			return DialTarget{}, fmt.Errorf("signaling_addr required when request_uri is set")
		}
		return DialTarget{RequestURI: reqURI, SignalingAddr: sig}, nil
	}

	num := strings.TrimSpace(body.TargetNumber)
	host := strings.TrimSpace(body.OutboundHost)
	port := body.OutboundPort
	if port <= 0 {
		port = 5060
	}
	if num != "" && host != "" {
		if sig == "" {
			sig = net.JoinHostPort(host, strconv.Itoa(port))
		}
		return DialTarget{
			RequestURI:    fmt.Sprintf("sip:%s@%s:%d", num, host, port),
			SignalingAddr: sig,
		}, nil
	}

	if t, ok := DialTargetFromEnv(); ok {
		return t, nil
	}
	return DialTarget{}, fmt.Errorf("missing target: use request_uri+signaling_addr, or target_number+outbound_host")
}


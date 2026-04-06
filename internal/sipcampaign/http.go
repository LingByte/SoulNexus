package sipcampaign

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

type HTTPServer struct {
	svc   *Service
	token string
	srv   *http.Server
}

func StartHTTPServer(addr, token string, svc *Service) (*HTTPServer, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" || svc == nil {
		return nil, nil
	}
	h := &HTTPServer{svc: svc, token: strings.TrimSpace(token)}
	mux := http.NewServeMux()
	mux.HandleFunc("/sip/v1/campaigns", h.handleCampaigns)
	mux.HandleFunc("/sip/v1/campaigns/", h.handleCampaignByID)
	mux.HandleFunc("/sip/v1/campaigns/metrics", h.handleMetrics)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("SoulNexus campaign API: /sip/v1/campaigns\n"))
			return
		}
		http.NotFound(w, r)
	})
	h.srv = &http.Server{Addr: addr, Handler: mux}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("campaign http listen %s: %w", addr, err)
	}
	go func() {
		if logger.Lg != nil {
			logger.Lg.Info("campaign http listening", zap.String("addr", addr))
		}
		if err := h.srv.Serve(ln); err != nil && err != http.ErrServerClosed && logger.Lg != nil {
			logger.Lg.Warn("campaign http serve", zap.Error(err))
		}
	}()
	return h, nil
}

func (h *HTTPServer) Shutdown(ctx context.Context) error {
	if h == nil || h.srv == nil {
		return nil
	}
	return h.srv.Shutdown(ctx)
}

func (h *HTTPServer) authorized(r *http.Request) bool {
	expected := strings.TrimSpace(h.token)
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

func (h *HTTPServer) handleCampaigns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	if !h.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body CreateCampaignInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "json", http.StatusBadRequest)
		return
	}
	out, err := h.svc.CreateCampaign(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *HTTPServer) handleCampaignByID(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/sip/v1/campaigns/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	id64, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || id64 == 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	id := uint(id64)
	switch parts[1] {
	case "contacts":
		h.handleCampaignContacts(w, r, id)
	case "start":
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if err := h.svc.StartCampaign(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case "pause":
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if err := h.svc.PauseCampaign(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case "resume":
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if err := h.svc.ResumeCampaign(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

func (h *HTTPServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	if !h.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.svc.SnapshotMetrics())
}

func (h *HTTPServer) handleCampaignContacts(w http.ResponseWriter, r *http.Request, campaignID uint) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var contacts []ContactInput
	if err := json.NewDecoder(r.Body).Decode(&contacts); err != nil {
		http.Error(w, "json", http.StatusBadRequest)
		return
	}
	n, err := h.svc.EnqueueContacts(r.Context(), campaignID, contacts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"accepted": n})
}


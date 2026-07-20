package handlers

import (
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/voiceattach"
	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/session"
	dialogwebrtc "github.com/LingByte/SoulNexus/pkg/dialog/transport/webrtc"
	dialogws "github.com/LingByte/SoulNexus/pkg/dialog/transport/websocket"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type createVoiceSessionRequest struct {
	Transport   string   `json:"transport"`
	AssistantID string   `json:"assistantId"`
	SampleRate  int      `json:"sampleRateHz"`
	DialogMode  string   `json:"dialogMode"`
	Temperature *float64 `json:"temperature"`
	// JsSourceID marks sessions opened from an injected JS template (mini-program / APP / H5).
	JsSourceID string `json:"jsSourceId"`
}

// InitVoiceSessionModule configures the process-wide generic voice session manager.
func InitVoiceSessionModule() {
	session.InitDefault(session.Config{
		WebSocketPath:   constants.LingechoVoiceSessionPathPrefix + "/ws",
		WebRTCOfferPath: constants.LingechoVoiceSessionPathPrefix + "/webrtc/offer",
	})
}

// registerVoiceSessionRoutes mounts the realtime voice session endpoints.
// These are tenant-scoped; no RBAC permission constants are used yet and
// auth is enforced upstream by whatever middleware wraps r (typically the
// same authenticated-tenant guard used by the rest of the console API).
//
// Sub-routes registered:
//   - POST   /sessions             → createVoiceSession
//   - DELETE /sessions/:sessionId  → endVoiceSession
//   - GET    /ws                   → voiceSessionWebSocket
//   - POST   /webrtc/offer         → voiceSessionWebRTCOffer
func (h *Handlers) registerVoiceSessionRoutes(r *humax.Group) {
	g := r.Group(constants.LingechoVoiceSessionPathPrefix)
	{
		g.POST("/sessions", h.createVoiceSession)
		g.DELETE("/sessions/:sessionId", h.endVoiceSession)
		g.GET("/ws", h.voiceSessionWebSocket)
		g.POST("/webrtc/offer", h.voiceSessionWebRTCOffer)
	}
}

// createVoiceSession allocates a realtime voice session on the in-process
// session manager. Clients can then connect via the returned WebSocket
// path or WebRTC offer URL depending on the chosen transport.
//
//   - POST <LingechoVoiceSessionPathPrefix>/sessions
//   - Body: { transport, assistantId, sampleRateHz } — all optional.
//
// Supported transports:
//   - "websocket" (default) → client connects to /ws
//   - "webrtc"              → client negotiates via /webrtc/offer
//
// Text chat uses /lingecho/dialog/v1 (not this API).
//
// Response: session info object.
func (h *Handlers) createVoiceSession(c *gin.Context) {
	mgr := session.Default()
	if mgr == nil {
		response.Render(c, response.NewI18n(response.CodeServiceUnavail, i18n.KeyVoiceSessionUnavailable))
		return
	}
	tenantID, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	var req createVoiceSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid body", err))
		return
	}
	kind := session.TransportWebSocket
	switch req.Transport {
	case "", "websocket":
		kind = session.TransportWebSocket
	case "webrtc":
		kind = session.TransportWebRTC
	case "text":
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyUnsupportedTransport))
		return
	default:
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyUnsupportedTransport))
		return
	}
	assistantID := utils.ParseOptionalID(req.AssistantID)
	dialogMode := strings.TrimSpace(req.DialogMode)
	voiceDialogWsURL := ""
	if assistantID > 0 && h.db != nil {
		if ast, err := models.GetActiveAssistantForTenant(h.db, assistantID, tenantID); err == nil {
			if spec, err := models.ResolveAssistantSpec(h.db, ast); err == nil {
				voiceDialogWsURL = strings.TrimSpace(spec.VoiceDialogWsURL)
			}
		}
	}
	if dialogMode == "" && voiceDialogWsURL != "" && kind == session.TransportWebSocket {
		dialogMode = "gateway"
	}
	credID := middleware.AuthCredentialID(c)
	info, err := mgr.Create(session.CreateParams{
		TenantID:            tenantID,
		Transport:           kind,
		AssistantID:         assistantID,
		SampleRate:          req.SampleRate,
		SkipCallPersist:     true,
		DialogMode:          dialogMode,
		RealtimeTemperature: realtimeTemperatureFromReq(req.Temperature),
		UserID:              middleware.AuthUserID(c),
		CredentialID:        credID,
	})
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "create session failed", err))
		return
	}
	aiSource := "assistant_debug_voice"
	jsSourceID := strings.TrimSpace(req.JsSourceID)
	if jsSourceID != "" {
		aiSource = "js_template"
		callbinding.SetJSSourceID(info.SessionID, jsSourceID)
		if h.db != nil {
			_ = models.RecordJSTemplateUsage(h.db, tenantID, jsSourceID, models.JSTemplateUsageSessionStart, info.SessionID, credID, middleware.AuthUserID(c))
		}
	} else if credID > 0 {
		aiSource = "js_embed"
	}
	callbinding.SetAISource(info.SessionID, aiSource)
	info.DialogMode = dialogMode
	info.VoiceDialogWsURL = voiceDialogWsURL
	response.SuccessI18n(c, i18n.KeySuccess, info)
}

// endVoiceSession terminates a live session on the session manager.
//
//   - DELETE <LingechoVoiceSessionPathPrefix>/sessions/:sessionId
//   - Path: sessionId (string) — the session id returned by /sessions.
//
// Response: success envelope (no body on success).
func (h *Handlers) endVoiceSession(c *gin.Context) {
	mgr := session.Default()
	if mgr == nil {
		response.Render(c, response.NewI18n(response.CodeServiceUnavail, i18n.KeyVoiceSessionUnavailable))
		return
	}
	sessionID := strings.TrimSpace(c.Param("sessionId"))
	if sessionID == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeySessionIDRequired))
		return
	}
	mgr.End(sessionID)
	response.SuccessI18n(c, i18n.KeySuccess, nil)
}

// voiceSessionWebSocket upgrades a GET request into the WebSocket used
// by browser-based realtime voice sessions.
//
//   - GET <LingechoVoiceSessionPathPrefix>/ws
//
// Upgrade + protocol framing are delegated to pkg/dialog/transport/websocket;
// this handler only performs the HTTP upgrade.
func (h *Handlers) voiceSessionWebSocket(c *gin.Context) {
	dialogws.Serve(c.Writer, c.Request, voiceSessionEngineLogger())
}

// voiceSessionWebRTCOffer negotiates a WebRTC session offer. The caller
// posts an SDP offer and receives an SDP answer plus ICE candidates to
// route realtime audio to/from the server-side voice dialog engine.
//
//   - POST <LingechoVoiceSessionPathPrefix>/webrtc/offer
//   - Body: engine-specific offer request (see pkg/dialog/transport/webrtc).
//
// Response: the offer answer (SDP) object from the voice dialog engine.
func (h *Handlers) voiceSessionWebRTCOffer(c *gin.Context) {
	var req dialogwebrtc.OfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid body", err))
		return
	}
	answer, err := dialogwebrtc.NegotiateOffer(c.Request.Context(), req, voiceSessionEngineLogger())
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "webrtc negotiate failed", err))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, answer)
}

func voiceSessionEngineLogger() engine.Logger {
	return voiceattach.NewZapEngineLogger(voiceSessionZapLogger())
}

func voiceSessionZapLogger() *zap.Logger {
	if logger.Lg != nil {
		return logger.Lg.Named("voice-session")
	}
	return nil
}

func realtimeTemperatureFromReq(v *float64) float64 {
	if v == nil {
		return 0
	}
	if *v <= 0 {
		return 0
	}
	return *v
}

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/app"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/xiaozhi"
)

// soulnexusVoiceSecretHeader is the header sent by cmd/voice when calling
// the SoulNexus binding API. The value is preserved (X-Lingecho-Voice-
// Secret) for backward compatibility with deployed firmware.
const soulnexusVoiceSecretHeader = "X-Lingecho-Voice-Secret"

// soulnexusBindingHTTPResp models the SoulNexus /api/voice/lingecho/binding
// envelope.
type soulnexusBindingHTTPResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// mountXiaozhi registers the xiaozhi WebSocket adapter under
// cfg.XiaozhiPath. ESP32 hardware (audio_params.format=opus) and browser
// web clients (format=pcm) both land here; the handshake distinguishes
// them. Errors during factory construction are logged and the function
// returns false — the listener keeps running so SIP traffic isn't
// affected by a misconfigured xiaozhi stack.
func (h *Handlers) mountXiaozhi(r gin.IRoutes) bool {
	dialogWS := strings.TrimSpace(h.cfg.DialogWS)
	if dialogWS == "" {
		log.Printf("[xiaozhi] disabled: VOICE_DIALOG_WS is empty")
		return false
	}
	factory, err := app.NewXiaozhiFactory()
	if err != nil {
		log.Printf("[xiaozhi] disabled: %v", err)
		return false
	}
	srv, err := xiaozhi.NewServer(xiaozhi.ServerConfig{
		SessionFactory:   factory,
		DialogWSURL:      dialogWS,
		CallIDPrefix:     "xz",
		BargeInFactory:   app.NewBargeInDetector,
		ConfigureClient:  app.ApplyDialogReconnect,
		Logger:           app.VoiceLogger("xiaozhi"),
		PersisterFactory: app.MakePersisterFactory(h.cfg.DB, "xiaozhi", "inbound"),
		RecorderFactory:  app.MakeRecorderFactory(h.cfg.Record, "xiaozhi"),
		OnSessionStart: func(_ context.Context, callID, deviceID string) {
			log.Printf("[xiaozhi] session start call=%s device=%s", callID, deviceID)
		},
		OnSessionEnd: func(_ context.Context, callID, reason string) {
			log.Printf("[xiaozhi] session end   call=%s reason=%s", callID, reason)
		},
	})
	if err != nil {
		log.Printf("[xiaozhi] init failed: %v", err)
		return false
	}
	r.GET(h.cfg.XiaozhiPath, gin.WrapF(srv.Handle))
	log.Printf("[xiaozhi] mounted: ws://<http>%s -> dialog=%s (esp32+web)", h.cfg.XiaozhiPath, gateway.RedactDialogDialURL(dialogWS))
	h.xiaozhiSrv = srv
	return true
}

// mountSoulnexusHardware registers the legacy SoulNexus ESP32 firmware
// path. SoulNexus firmware historically opened WebSocket on
// /api/voice/lingecho/v1/ with Device-Id; this wrapper resolves that
// device against the SoulNexus binding endpoint and merges the returned
// dialog credentials into the URL ?payload= before delegating to the
// xiaozhi adapter. URL paths and the X-Lingecho-Voice-Secret header are
// preserved for backward compatibility with deployed firmware.
func (h *Handlers) mountSoulnexusHardware(r gin.IRoutes) {
	srv := h.xiaozhiSrv
	wsPath := "/" + strings.Trim(h.cfg.SoulnexusHardwarePath, "/")
	if !strings.HasSuffix(wsPath, "/") {
		wsPath += "/"
	}
	bindingBaseURL := strings.TrimSpace(h.cfg.SoulnexusHardwareBindingURL)
	bindingSecret := strings.TrimSpace(h.cfg.SoulnexusHardwareBindingSecret)
	if srv == nil || bindingBaseURL == "" {
		return
	}
	client := &http.Client{Timeout: 12 * time.Second}
	r.GET(wsPath, func(c *gin.Context) {
		req := c.Request
		dev := strings.TrimSpace(req.Header.Get("Device-Id"))
		if dev == "" {
			dev = strings.TrimSpace(req.URL.Query().Get("device-id"))
		}
		if dev == "" {
			c.String(http.StatusBadRequest, "missing Device-Id")
			return
		}
		payloadJSON, err := fetchSoulnexusBindingPayload(req.Context(), client, bindingBaseURL, bindingSecret, dev)
		if err != nil {
			log.Printf("[lingecho-hw] binding failed device=%s: %v", dev, err)
			c.String(http.StatusBadGateway, "binding failed")
			return
		}
		u := *req.URL
		q := req.URL.Query()
		q.Set("payload", string(payloadJSON))
		u.RawQuery = q.Encode()
		r2 := req.Clone(req.Context())
		r2.URL = &u
		srv.Handle(c.Writer, r2)
	})
	log.Printf("[lingecho-hw] mounted ws path=%s (binding=%s)", wsPath, redactBindingURL(bindingBaseURL))
}

// fetchSoulnexusBindingPayload calls the SoulNexus binding API and
// returns the inner `payload` JSON, ready to inject as ?payload= on the
// xiaozhi adapter URL. Returns an error when the API is unreachable, an
// HTTP error, or when the response shape doesn't carry a non-empty
// payload.
func fetchSoulnexusBindingPayload(ctx context.Context, client *http.Client, bindingBaseURL, secret, deviceID string) ([]byte, error) {
	base, err := url.Parse(strings.TrimSpace(bindingBaseURL))
	if err != nil {
		return nil, err
	}
	q := base.Query()
	q.Set("device-id", deviceID)
	base.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Device-Id", deviceID)
	if strings.TrimSpace(secret) != "" {
		req.Header.Set(soulnexusVoiceSecretHeader, secret)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binding http %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var env soulnexusBindingHTTPResp
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("binding json: %w", err)
	}
	if env.Code != 200 {
		return nil, fmt.Errorf("binding code=%d msg=%s", env.Code, env.Msg)
	}
	var data struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, fmt.Errorf("binding data: %w", err)
	}
	if len(strings.TrimSpace(string(data.Payload))) == 0 {
		return nil, fmt.Errorf("binding: empty payload")
	}
	return data.Payload, nil
}

// redactBindingURL strips query/credentials from raw, keeping only the
// scheme + host + path for log lines.
func redactBindingURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return raw
	}
	return u.Scheme + "://" + u.Host + u.Path
}

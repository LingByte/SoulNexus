// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package rtcsfu_replica registers a replica SFU process with a primary control plane over HTTP.
package rtcsfu_replica

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/rtcsfu"
	"go.uber.org/zap"
)

const (
	registerPath = "/api/rtcsfu/v1/admin/nodes/register"
	touchPath    = "/api/rtcsfu/v1/admin/nodes/replica/touch"
)

// RunAnnouncer periodically registers (or touches) the replica on the primary, using X-RTCSFU-Key.
// On SIGINT/SIGTERM it attempts DELETE /admin/nodes/replica/:id on the primary (best-effort).
// Optional mTLS: RTCSFU_REPLICA_MTLS_* env (see config).
func RunAnnouncer() {
	cfg := config.GlobalConfig.RTCSFU
	if !cfg.Enabled {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(cfg.ClusterRole), "replica") {
		return
	}
	primary := strings.TrimSpace(cfg.PrimaryBaseURL)
	if primary == "" {
		logger.Info("RTCSFU replica announcer: skipped (RTCSFU_PRIMARY_BASE_URL empty)")
		return
	}
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		logger.Warn("RTCSFU replica announcer: skipped (RTCSFU_API_KEY required to authenticate to primary)")
		return
	}
	raw := bytes.TrimSpace([]byte(cfg.ReplicaSelfJSON))
	if len(raw) == 0 {
		logger.Warn("RTCSFU replica announcer: skipped (RTCSFU_REPLICA_SELF_JSON empty)")
		return
	}
	nodes, err := rtcsfu.ParseNodesJSONFlexible(raw)
	if err != nil || len(nodes) == 0 {
		logger.Warn("RTCSFU replica announcer: invalid RTCSFU_REPLICA_SELF_JSON", zap.Error(err))
		return
	}
	nodeID := string(nodes[0].ID)

	client, err := newReplicaHTTPClient(cfg)
	if err != nil {
		logger.Warn("RTCSFU replica announcer: http client", zap.Error(err))
		return
	}

	interval := cfg.ReplicaHeartbeatSec
	if interval <= 0 {
		interval = 30
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.ReplicaHeartbeatMode))
	if mode == "" {
		mode = "register"
	}

	registerURL := JoinPrimaryAPI(primary, registerPath)
	touchURL := JoinPrimaryAPI(primary, touchPath)
	deregisterURL := JoinPrimaryAPI(primary, "/api/rtcsfu/v1/admin/nodes/replica/"+url.PathEscape(nodeID))

	touchSecret := strings.TrimSpace(cfg.ReplicaTouchHMACSecret)
	touchTTL := cfg.ReplicaTouchTokenTTLSeconds
	if touchTTL <= 0 {
		touchTTL = 120
	}
	runLoop(client, raw, apiKey, nodeID, registerURL, touchURL, deregisterURL, interval, mode, touchSecret, touchTTL)
}

func runLoop(
	client *http.Client,
	raw []byte,
	apiKey, nodeID, registerURL, touchURL, deregisterURL string,
	interval int,
	mode string,
	touchSecret string,
	touchTTL int,
) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		if err := doDeregister(shutCtx, client, deregisterURL, apiKey); err != nil {
			logger.Warn("RTCSFU replica announcer: deregister", zap.Error(err))
		} else {
			logger.Info("RTCSFU replica announcer: deregister ok", zap.String("node_id", nodeID))
		}
	}()

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	logger.Info("RTCSFU replica announcer started",
		zap.String("primary_register_url", registerURL),
		zap.String("node_id", nodeID),
		zap.Int("heartbeat_sec", interval),
		zap.String("heartbeat_mode", mode),
	)

	if mode == "touch" {
		if err := doRegister(ctx, client, registerURL, apiKey, raw); err != nil {
			logger.Warn("RTCSFU replica announcer: touch mode needs initial register", zap.Error(err))
			return
		}
	}

	beat := func() {
		if mode == "touch" {
			if err := doTouch(ctx, client, touchURL, apiKey, nodeID, touchSecret, touchTTL); err != nil {
				logger.Warn("RTCSFU replica announcer: touch failed", zap.Error(err))
			}
			return
		}
		if err := doRegister(ctx, client, registerURL, apiKey, raw); err != nil {
			logger.Warn("RTCSFU replica announcer: register failed", zap.Error(err))
		}
	}

	beat()
	for {
		select {
		case <-ctx.Done():
			logger.Info("RTCSFU replica announcer stopping", zap.Error(ctx.Err()))
			return
		case <-ticker.C:
			beat()
		}
	}
}

func doRegister(ctx context.Context, client *http.Client, url, apiKey string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-RTCSFU-Key", apiKey)
	return doAdminReq2xx(client, req)
}

func doTouch(ctx context.Context, client *http.Client, url, apiKey, nodeID, touchSecret string, touchTTL int) error {
	body := map[string]string{"id": nodeID}
	if touchSecret != "" {
		exp := time.Now().Add(time.Duration(touchTTL) * time.Second).UTC()
		tok, err := rtcsfu.MintReplicaTouchToken(touchSecret, nodeID, exp)
		if err != nil {
			return err
		}
		body["touch_token"] = tok
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-RTCSFU-Key", apiKey)
	return doAdminReq2xx(client, req)
}

func doDeregister(ctx context.Context, client *http.Client, url, apiKey string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-RTCSFU-Key", apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func doAdminReq2xx(client *http.Client, req *http.Request) error {
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

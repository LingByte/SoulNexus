package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/cache"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HandleOTACheck handles OTA version check and device activation status
// POST /ota/
func (h *Handlers) HandleOTACheck(c *gin.Context) {
	// Support both uppercase and lowercase headers (HTTP headers are case-insensitive, but some clients use lowercase)
	deviceID := c.GetHeader("Device-Id")
	if deviceID == "" {
		deviceID = c.GetHeader("device-id")
	}
	logger.Info("deviceID", zap.String("deviceID", deviceID))

	clientID := c.GetHeader("Client-Id")
	if clientID == "" {
		clientID = c.GetHeader("client-id")
	}
	logger.Info("clientID", zap.String("clientID", clientID))

	if deviceID == "" {
		response.Fail(c, "Device ID is required", nil)
		return
	}

	if clientID == "" {
		clientID = deviceID
	}

	// Validate MAC address format
	if !isMacAddressValid(deviceID) {
		logger.Error("Invalid MAC address", zap.String("deviceID", deviceID))
		response.Fail(c, "Invalid device ID", nil)
		return
	}

	// Parse request body - lenient handling, allow connection even if JSON is incomplete
	var req models.DeviceReportReq
	bodyBytes, _ := c.GetRawData()
	if len(bodyBytes) > 0 {
		// Try to parse JSON, but don't require all fields to be correct
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			// JSON parsing failed, log warning but continue processing (can connect as long as deviceID exists)
			logger.Warn("JSON parsing partially failed, but continuing processing",
				zap.Error(err),
				zap.String("deviceID", deviceID),
				zap.String("body", string(bodyBytes)))
			// Create empty req object to ensure subsequent processing doesn't panic
			req = models.DeviceReportReq{}
		}
	} else {
		// No request body, create empty req
		req = models.DeviceReportReq{}
	}

	// Build response - completely consistent with xiaozhi-esp32 flow
	resp := h.buildOTAResponse(deviceID, clientID, &req)
	logger.Info("OTA response",
		zap.String("deviceID", deviceID),
		zap.String("clientID", clientID),
		zap.Any("response", resp))
	c.JSON(http.StatusOK, resp)
}

// HandleOTAActivate handles quick device activation check
// POST /ota/activate or POST /xiaozhi/ota/activate
func (h *Handlers) HandleOTAActivate(c *gin.Context) {
	// Support both uppercase and lowercase headers
	deviceID := c.GetHeader("Device-Id")
	if deviceID == "" {
		deviceID = c.GetHeader("device-id")
	}

	if deviceID == "" {
		c.Status(http.StatusAccepted)
		return
	}

	device, err := models.GetDeviceByMacAddress(h.db, deviceID)
	if err != nil || device == nil {
		c.Status(http.StatusAccepted)
		return
	}

	c.String(http.StatusOK, "success")
}

// HandleOTAGet handles OTA health check
// GET /ota/
func (h *Handlers) HandleOTAGet(c *gin.Context) {
	// Check WebSocket configuration (required)
	wsURL := utils.GetValue(h.db, constants.KEY_SERVER_WEBSOCKET)
	if wsURL == "" || wsURL == "null" {
		c.String(http.StatusOK, "OTA interface abnormal, missing websocket address, please login to control panel, find [server.websocket] configuration in parameter management")
		return
	}

	// MQTT Gateway is optional - if not configured, devices will use WebSocket
	mqttGateway := utils.GetValue(h.db, constants.KEY_SERVER_MQTT_GATEWAY)

	// Count WebSocket clusters (split by semicolon)
	wsCount := len(strings.Split(wsURL, ";"))

	// Build status message
	statusMsg := fmt.Sprintf("OTA interface running normally, websocket cluster count: %d", wsCount)
	if mqttGateway != "" && mqttGateway != "null" {
		statusMsg += ", MQTT gateway configured"
	} else {
		statusMsg += ", using WebSocket connection"
	}

	c.String(http.StatusOK, statusMsg)
}

// buildOTAResponse builds the OTA response based on device status
// Completely consistent with xiaozhi-esp32 activation code flow
func (h *Handlers) buildOTAResponse(deviceID, clientID string, req *models.DeviceReportReq) *models.DeviceReportResp {
	_ = clientID // Reserved for future use
	// Get timezone offset from config or use default
	timezoneOffset := 8 * 60 // Default UTC+8 (in minutes)
	resp := &models.DeviceReportResp{}

	// Build server time (consistent with xiaozhi-esp32 format)
	now := time.Now()
	resp.ServerTime = &models.ServerTime{
		Timestamp:      now.UnixMilli(),
		TimezoneOffset: timezoneOffset,
	}

	// Check if device exists
	device, err := models.GetDeviceByMacAddress(h.db, deviceID)

	if err != nil || device == nil {
		// Device doesn't exist - generate activation code (completely consistent with xiaozhi-esp32)
		activation := h.buildActivation(deviceID, req)
		resp.Activation = activation

		// Device not bound, return current uploaded firmware info (no update) for compatibility with old firmware versions
		appVersion := "1.0.0"
		if req.Application != nil && req.Application.Version != "" {
			appVersion = req.Application.Version
		}
		resp.Firmware = &models.Firmware{
			Version: appVersion,
			URL:     "NOT_ACTIVATED_FIRMWARE_THIS_IS_A_INVALID_URL", // Consistent with xiaozhi-esp32
		}
	} else {
		// Device exists - update connection info and return normal configuration
		now := time.Now()
		device.LastConnected = &now
		if req.Application != nil {
			device.AppVersion = req.Application.Version
		}
		models.UpdateDevice(h.db, device)

		// Only return firmware upgrade info when device is bound and autoUpdate is not 0
		if device.AutoUpdate != 0 {
			boardType := device.Board
			if boardType == "" {
				boardType = "default"
			}
			appVersion := "1.0.0"
			if req.Application != nil && req.Application.Version != "" {
				appVersion = req.Application.Version
			}
			firmware := h.getLatestFirmware(boardType, appVersion)
			resp.Firmware = firmware
		} else {
			appVersion := "1.0.0"
			if req.Application != nil && req.Application.Version != "" {
				appVersion = req.Application.Version
			}
			resp.Firmware = &models.Firmware{
				Version: appVersion,
				URL:     "",
			}
		}
	}

	// Build WebSocket configuration (only return when device is activated)
	if device != nil {
		// WebSocket URL: legacy SoulNexus in-process path /api/voice/lingecho/v1/
		// or VoiceServer SoulNexus mount e.g. ws://<voice-host>:7080/voice/lingecho/v1/ (path kept for compat)
		// (requires cmd/voice -lingecho-hw-ws-path and binding URL on SoulNexus).
		wsURL := utils.GetValue(h.db, constants.KEY_SERVER_WEBSOCKET)
		if wsURL == "" || wsURL == "null" {
			// Default: SoulNexus /api/voice/lingecho/v1/ (in-process pkg/voice) or set
			// server.websocket to VoiceServer e.g. ws://host:7080/voice/lingecho/v1/
			if config.GlobalConfig.Server.URL != "" {
				baseURL := strings.TrimSuffix(config.GlobalConfig.Server.URL, "/")
				// Keep API prefix since route is under /api
				wsURL = strings.Replace(baseURL, "http://", "ws://", 1)
				wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
				// SoulNexus legacy hardware WS path (see registerVoiceTrainingRoutes).
				wsURL = fmt.Sprintf("%s/api/voice/lingecho/v1/", wsURL)
			} else {
				// Default to localhost with correct path
				wsURL = "ws://localhost:7072/api/voice/lingecho/v1/"
			}
		} else {
			// Use configured WebSocket URL directly
			// Split by semicolon and select one randomly if multiple URLs provided
			urls := strings.Split(wsURL, ";")
			if len(urls) > 0 {
				// Simple random selection
				rand.Seed(time.Now().UnixNano())
				wsURL = strings.TrimSpace(urls[rand.Intn(len(urls))])
			}
			// Use the URL as-is, just ensure it doesn't have trailing issues
			wsURL = strings.TrimSpace(wsURL)
		}

		// Build MQTT configuration (if configured)
		// According to xiaozhi-esp32 logic: if MQTT is configured, return MQTT only; otherwise return WebSocket
		mqttGateway := utils.GetValue(h.db, constants.KEY_SERVER_MQTT_GATEWAY)
		if mqttGateway != "" && mqttGateway != "null" {
			// MQTT is configured, return MQTT configuration (xiaozhi-esp32 behavior)
			boardType := device.Board
			if boardType == "" {
				boardType = "default"
			}
			groupId := fmt.Sprintf("GID_%s", strings.ReplaceAll(strings.ReplaceAll(boardType, ":", "_"), " ", "_"))
			mqttConfig := h.buildMQTTConfig(deviceID, groupId)
			if mqttConfig != nil {
				mqttConfig.Endpoint = mqttGateway
				resp.MQTT = mqttConfig
			}
			logger.Info("MQTT gateway configured, returning MQTT config", zap.String("deviceID", deviceID))
		} else {
			// MQTT is not configured, return WebSocket configuration (xiaozhi-esp32 behavior)
			resp.Websocket = &models.Websocket{
				URL:   wsURL,
				Token: "", // Can be generated if auth is enabled
			}
			logger.Info("MQTT gateway not configured, returning WebSocket config",
				zap.String("deviceID", deviceID),
				zap.String("websocketURL", wsURL))
		}
	}

	return resp
}

// buildActivation generates activation code (completely consistent with xiaozhi-esp32)
// Uses local cache to store activation codes (via cache.GetGlobalCache(), defaults to local cache)
func (h *Handlers) buildActivation(deviceID string, req *models.DeviceReportReq) *models.Activation {
	ctx := context.Background()
	// Use global cache (defaults to local cache, configurable via CACHE_TYPE environment variable)
	cacheClient := cache.GetGlobalCache()

	// Check if activation code already exists
	safeDeviceId := strings.ReplaceAll(strings.ToLower(deviceID), ":", "_")
	dataKey := fmt.Sprintf("ota:activation:data:%s", safeDeviceId)

	var cachedCode string
	if data, ok := cacheClient.Get(ctx, dataKey); ok {
		if dataMap, ok := data.(map[string]interface{}); ok {
			if code, ok := dataMap["activation_code"].(string); ok {
				cachedCode = code
			}
		}
	}

	activation := &models.Activation{
		Challenge: deviceID,
	}

	if cachedCode != "" {
		// Use existing activation code
		activation.Code = cachedCode
		frontendURL := utils.GetValue(h.db, constants.KEY_SERVER_FRONTED_URL)
		if frontendURL == "" || frontendURL == "null" {
			frontendURL = "http://xiaozhi.server.com"
		}
		activation.Message = fmt.Sprintf("%s\n%s", frontendURL, cachedCode)
	} else {
		// Generate new 6-digit activation code
		newCode := generateActivationCode()
		activation.Code = newCode
		frontendURL := utils.GetValue(h.db, constants.KEY_SERVER_FRONTED_URL)
		if frontendURL == "" || frontendURL == "null" {
			frontendURL = "http://xiaozhi.server.com"
		}
		activation.Message = fmt.Sprintf("%s\n%s", frontendURL, newCode)

		// Get device information
		boardType := "default"
		if req.Board != nil {
			boardType = req.Board.Type
		} else if req.Device != nil {
			if model, ok := req.Device["model"].(string); ok {
				boardType = model
			}
		} else if req.Model != "" {
			boardType = req.Model
		} else if req.ChipModelName != "" {
			boardType = req.ChipModelName
		}

		appVersion := "1.0.0"
		if req.Application != nil && req.Application.Version != "" {
			appVersion = req.Application.Version
		}

		// Store device data to local cache (consistent with xiaozhi-esp32 Redis storage logic)
		dataMap := map[string]interface{}{
			"id":              deviceID,
			"mac_address":     deviceID,
			"board":           boardType,
			"app_version":     appVersion,
			"deviceId":        deviceID,
			"activation_code": newCode,
		}

		// Write main data key: ota:activation:data:{safeDeviceId}
		// Expiration: 24 hours (consistent with xiaozhi-esp32)
		// Use local cache storage (via cache.GetGlobalCache())
		cacheClient.Set(ctx, dataKey, dataMap, 24*time.Hour)

		// Write reverse lookup activation code key: ota:activation:code:{activationCode} -> deviceId
		codeKey := fmt.Sprintf("ota:activation:code:%s", newCode)
		cacheClient.Set(ctx, codeKey, deviceID, 24*time.Hour)

		logger.Info("Generated new activation code",
			zap.String("deviceID", deviceID),
			zap.String("activationCode", newCode),
			zap.String("boardType", boardType))
	}

	return activation
}

// generateActivationCode generates 6-digit activation code (consistent with xiaozhi-esp32)
func generateActivationCode() string {
	rand.Seed(time.Now().UnixNano())
	code := rand.Intn(900000) + 100000 // Generate 6-digit number between 100000-999999
	return fmt.Sprintf("%06d", code)
}

// buildMQTTConfig builds MQTT configuration (completely consistent with xiaozhi-esp32)
func (h *Handlers) buildMQTTConfig(deviceID string, groupId string) *models.MQTT {
	// Build safe MAC address
	macAddressSafe := strings.ReplaceAll(deviceID, ":", "_")
	mqttClientID := fmt.Sprintf("%s@@@%s@@@%s", groupId, macAddressSafe, macAddressSafe)

	// Build username (base64 encoded user data)
	userData := map[string]interface{}{
		"ip": "unknown",
	}
	userDataJSON, _ := json.Marshal(userData)
	username := base64.StdEncoding.EncodeToString(userDataJSON)

	// Generate password (HMAC-SHA256 signature)
	password := ""
	signatureKey := utils.GetValue(h.db, constants.KEY_SERVER_MQTT_SIGNATURE_KEY)
	if signatureKey != "" {
		content := fmt.Sprintf("%s|%s", mqttClientID, username)
		password = generatePasswordSignature(content, signatureKey)
	}

	return &models.MQTT{
		ClientID:       mqttClientID,
		Username:       username,
		Password:       password,
		PublishTopic:   "device-server",
		SubscribeTopic: fmt.Sprintf("devices/p2p/%s", macAddressSafe),
	}
}

// generatePasswordSignature generates MQTT password signature using HMAC-SHA256
func generatePasswordSignature(content, secretKey string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(content))
	signature := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(signature)
}

// getLatestFirmware gets the latest firmware for a board type
func (h *Handlers) getLatestFirmware(boardType, currentVersion string) *models.Firmware {
	if boardType == "" {
		boardType = "default"
	}

	// Get latest firmware from database
	ota, err := models.GetLatestOTA(h.db, boardType)
	if err != nil || ota == nil {
		// No firmware available, return current version
		return &models.Firmware{
			Version: currentVersion,
			URL:     "",
		}
	}

	// Check if new version is available
	if ota.Version != currentVersion {
		// New version available
		return &models.Firmware{
			Version: ota.Version,
			URL:     ota.FirmwarePath,
		}
	}

	// Same version, no update
	return &models.Firmware{
		Version: currentVersion,
		URL:     "",
	}
}

// isMacAddressValid validates MAC address format
func isMacAddressValid(macAddress string) bool {
	if macAddress == "" {
		return false
	}
	// MAC address pattern: XX:XX:XX:XX:XX:XX or XX-XX-XX-XX-XX-XX
	macPattern := `^([0-9A-Za-z]{2}[:-]){5}([0-9A-Za-z]{2})$`
	matched, _ := regexp.MatchString(macPattern, macAddress)
	return matched
}

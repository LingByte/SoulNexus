package handlers

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

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/cache"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
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

	// Parse request body - 宽松处理，即使JSON不完整也允许连接
	var req models.DeviceReportReq
	bodyBytes, _ := c.GetRawData()
	if len(bodyBytes) > 0 {
		// 尝试解析JSON，但不强制要求所有字段都正确
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			// JSON解析失败，记录警告但继续处理（只要有deviceID就能连接）
			logger.Warn("JSON解析部分失败，但继续处理",
				zap.Error(err),
				zap.String("deviceID", deviceID),
				zap.String("body", string(bodyBytes)))
			// 创建一个空的req对象，确保后续处理不会panic
			req = models.DeviceReportReq{}
		}
	} else {
		// 没有请求体，创建空req
		req = models.DeviceReportReq{}
	}

	// Build response - 与 xiaozhi-esp32 完全一致的流程
	resp := h.buildOTAResponse(deviceID, clientID, &req)
	logger.Info("OTA响应",
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
		c.String(http.StatusOK, "OTA接口不正常，缺少websocket地址，请登录智控台，在参数管理找到【server.websocket】配置")
		return
	}

	// MQTT Gateway is optional - if not configured, devices will use WebSocket
	mqttGateway := utils.GetValue(h.db, constants.KEY_SERVER_MQTT_GATEWAY)

	// Count WebSocket clusters (split by semicolon)
	wsCount := len(strings.Split(wsURL, ";"))

	// Build status message
	statusMsg := fmt.Sprintf("OTA接口运行正常，websocket集群数量：%d", wsCount)
	if mqttGateway != "" && mqttGateway != "null" {
		statusMsg += "，已配置MQTT网关"
	} else {
		statusMsg += "，使用WebSocket连接"
	}

	c.String(http.StatusOK, statusMsg)
}

// buildOTAResponse builds the OTA response based on device status
// 与 xiaozhi-esp32 完全一致的激活码流程
func (h *Handlers) buildOTAResponse(deviceID, clientID string, req *models.DeviceReportReq) *models.DeviceReportResp {
	_ = clientID // Reserved for future use
	// Get timezone offset from config or use default
	timezoneOffset := 8 * 60 // Default UTC+8 (in minutes)
	resp := &models.DeviceReportResp{}

	// Build server time (与 xiaozhi-esp32 格式一致)
	now := time.Now()
	resp.ServerTime = &models.ServerTime{
		Timestamp:      now.UnixMilli(),
		TimezoneOffset: timezoneOffset,
	}

	// Check if device exists
	device, err := models.GetDeviceByMacAddress(h.db, deviceID)

	if err != nil || device == nil {
		// 设备不存在 - 生成激活码（与 xiaozhi-esp32 完全一致）
		activation := h.buildActivation(deviceID, req)
		resp.Activation = activation

		// 设备未绑定，返回当前上传的固件信息（不更新）以此兼容旧固件版本
		appVersion := "1.0.0"
		if req.Application != nil && req.Application.Version != "" {
			appVersion = req.Application.Version
		}
		resp.Firmware = &models.Firmware{
			Version: appVersion,
			URL:     "NOT_ACTIVATED_FIRMWARE_THIS_IS_A_INVALID_URL", // 与 xiaozhi-esp32 一致
		}
	} else {
		// 设备已存在 - 更新连接信息并返回正常配置
		now := time.Now()
		device.LastConnected = &now
		if req.Application != nil {
			device.AppVersion = req.Application.Version
		}
		models.UpdateDevice(h.db, device)

		// 只有在设备已绑定且autoUpdate不为0的情况下才返回固件升级信息
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

	// Build WebSocket configuration (仅在设备已激活时返回)
	if device != nil {
		// 实际路由路径是 /api/voice/lingecho/v1/（在 registerVoiceTrainingRoutes 中注册）
		wsURL := utils.GetValue(h.db, constants.KEY_SERVER_WEBSOCKET)
		if wsURL == "" || wsURL == "null" {
			// Use default WebSocket URL based on config
			// 实际路由路径：/api/voice/lingecho/v1/
			if config.GlobalConfig.ServerUrl != "" {
				baseURL := strings.TrimSuffix(config.GlobalConfig.ServerUrl, "/")
				// 保留 API prefix，因为路由在 /api 下
				wsURL = strings.Replace(baseURL, "http://", "ws://", 1)
				wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
				// 实际路由路径：/api/voice/lingecho/v1/
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
			logger.Info("MQTT网关已配置，返回MQTT配置", zap.String("deviceID", deviceID))
		} else {
			// MQTT is not configured, return WebSocket configuration (xiaozhi-esp32 behavior)
			resp.Websocket = &models.Websocket{
				URL:   wsURL,
				Token: "", // Can be generated if auth is enabled
			}
			logger.Info("未配置MQTT网关，返回WebSocket配置",
				zap.String("deviceID", deviceID),
				zap.String("websocketURL", wsURL))
		}
	}

	return resp
}

// buildActivation 生成激活码（与 xiaozhi-esp32 完全一致）
// 使用本地缓存存储激活码（通过 cache.GetGlobalCache()，默认使用本地缓存）
func (h *Handlers) buildActivation(deviceID string, req *models.DeviceReportReq) *models.Activation {
	ctx := context.Background()
	// 使用全局缓存（默认是本地缓存，可通过 CACHE_TYPE 环境变量配置）
	cacheClient := cache.GetGlobalCache()

	// 检查是否已有激活码
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
		// 使用已存在的激活码
		activation.Code = cachedCode
		frontendURL := utils.GetValue(h.db, constants.KEY_SERVER_FRONTED_URL)
		if frontendURL == "" || frontendURL == "null" {
			frontendURL = "http://xiaozhi.server.com"
		}
		activation.Message = fmt.Sprintf("%s\n%s", frontendURL, cachedCode)
	} else {
		// 生成新的6位数字激活码
		newCode := generateActivationCode()
		activation.Code = newCode
		frontendURL := utils.GetValue(h.db, constants.KEY_SERVER_FRONTED_URL)
		if frontendURL == "" || frontendURL == "null" {
			frontendURL = "http://xiaozhi.server.com"
		}
		activation.Message = fmt.Sprintf("%s\n%s", frontendURL, newCode)

		// 获取设备信息
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

		// 存储设备数据到本地缓存（与 xiaozhi-esp32 的 Redis 存储逻辑一致）
		dataMap := map[string]interface{}{
			"id":              deviceID,
			"mac_address":     deviceID,
			"board":           boardType,
			"app_version":     appVersion,
			"deviceId":        deviceID,
			"activation_code": newCode,
		}

		// 写入主数据 key: ota:activation:data:{safeDeviceId}
		// 过期时间：24小时（与 xiaozhi-esp32 一致）
		// 使用本地缓存存储（通过 cache.GetGlobalCache()）
		cacheClient.Set(ctx, dataKey, dataMap, 24*time.Hour)

		// 写入反查激活码 key: ota:activation:code:{activationCode} -> deviceId
		codeKey := fmt.Sprintf("ota:activation:code:%s", newCode)
		cacheClient.Set(ctx, codeKey, deviceID, 24*time.Hour)

		logger.Info("生成新激活码",
			zap.String("deviceID", deviceID),
			zap.String("activationCode", newCode),
			zap.String("boardType", boardType))
	}

	return activation
}

// generateActivationCode 生成6位数字激活码（与 xiaozhi-esp32 一致）
func generateActivationCode() string {
	rand.Seed(time.Now().UnixNano())
	code := rand.Intn(900000) + 100000 // 生成 100000-999999 之间的6位数字
	return fmt.Sprintf("%06d", code)
}

// buildMQTTConfig builds MQTT configuration (与 xiaozhi-esp32 完全一致)
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

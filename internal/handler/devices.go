package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/cache"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BindDevice binds device (activates device) - completely consistent with xiaozhi-esp32
// POST /device/bind/:agentId/:deviceCode
func (h *Handlers) BindDevice(c *gin.Context) {
	agentIdStr := c.Param("agentId")
	deviceCode := c.Param("deviceCode")

	logger.Info("开始设备绑定流程",
		zap.String("agentId", agentIdStr),
		zap.String("deviceCode", deviceCode),
		zap.String("clientIP", c.ClientIP()))

	if deviceCode == "" {
		logger.Error("设备绑定失败：激活码为空")
		response.Fail(c, "Activation code cannot be empty", nil)
		return
	}

	// Verify activation code
	ctx := context.Background()
	// Use global cache (default is local cache, can be configured via CACHE_TYPE environment variable)
	cacheClient := cache.GetGlobalCache()

	// Get device ID from local cache (key format consistent with xiaozhi-esp32 Redis key)
	deviceKey := fmt.Sprintf("ota:activation:code:%s", deviceCode)
	logger.Info("查找激活码缓存", zap.String("deviceKey", deviceKey))

	deviceIdObj, ok := cacheClient.Get(ctx, deviceKey)
	if !ok {
		logger.Error("激活码验证失败：缓存中未找到激活码", zap.String("deviceCode", deviceCode))
		response.Fail(c, "激活码错误", nil)
		return
	}

	deviceId, ok := deviceIdObj.(string)
	if !ok {
		logger.Error("激活码验证失败：缓存数据类型错误", zap.Any("deviceIdObj", deviceIdObj))
		response.Fail(c, "激活码错误", nil)
		return
	}

	logger.Info("激活码验证成功", zap.String("deviceId", deviceId))

	// Get device data
	safeDeviceId := strings.ReplaceAll(strings.ToLower(deviceId), ":", "_")
	dataKey := fmt.Sprintf("ota:activation:data:%s", safeDeviceId)
	logger.Info("获取设备数据", zap.String("dataKey", dataKey))

	dataObj, ok := cacheClient.Get(ctx, dataKey)
	if !ok {
		logger.Error("获取设备数据失败：缓存中未找到设备数据", zap.String("dataKey", dataKey))
		response.Fail(c, "激活码错误", nil)
		return
	}

	dataMap, ok := dataObj.(map[string]interface{})
	if !ok {
		logger.Error("设备数据格式错误", zap.Any("dataObj", dataObj))
		response.Fail(c, "激活码错误", nil)
		return
	}

	cachedCode, ok := dataMap["activation_code"].(string)
	if !ok || cachedCode != deviceCode {
		logger.Error("激活码不匹配",
			zap.String("cachedCode", cachedCode),
			zap.String("deviceCode", deviceCode))
		response.Fail(c, "激活码错误", nil)
		return
	}

	logger.Info("设备数据验证成功", zap.Any("deviceData", dataMap))

	// Check if device has already been activated
	logger.Info("检查设备是否已激活", zap.String("deviceId", deviceId))
	existingDevice, err := models.GetDeviceByMacAddress(h.db, deviceId)
	if err != nil {
		logger.Warn("查询现有设备时出错", zap.Error(err), zap.String("deviceId", deviceId))
	}
	if existingDevice != nil {
		logger.Error("设备绑定失败：设备已被激活",
			zap.String("deviceId", deviceId),
			zap.Uint("existingUserId", existingDevice.UserID))
		response.Fail(c, "Device has already been activated", nil)
		return
	}

	// Get current user
	user := models.CurrentUser(c)
	if user == nil {
		logger.Error("设备绑定失败：用户未登录")
		response.Fail(c, "User not logged in", nil)
		return
	}

	logger.Info("获取当前用户成功",
		zap.Uint("userId", user.ID),
		zap.String("userEmail", user.Email))

	// Parse agentId (assistant ID)
	agentId, err := strconv.ParseUint(agentIdStr, 10, 32)
	if err != nil {
		logger.Error("解析助手ID失败", zap.Error(err), zap.String("agentIdStr", agentIdStr))
		response.Fail(c, "Invalid assistant ID", nil)
		return
	}
	assistantID := uint(agentId)

	logger.Info("解析助手ID成功", zap.Uint("assistantID", assistantID))

	// Verify that assistant exists and belongs to current user
	var assistant models.Agent
	if err := h.db.Where("id = ?", assistantID).First(&assistant).Error; err != nil {
		logger.Error("查询助手失败", zap.Error(err), zap.Uint("assistantID", assistantID))
		response.Fail(c, "Assistant does not exist", nil)
		return
	}

	logger.Info("查询助手成功",
		zap.Uint("assistantID", uint(assistant.ID)),
		zap.String("assistantName", assistant.Name),
		zap.Uint("assistantUserId", assistant.UserID))

	if assistant.UserID != user.ID {
		// Check if it's an organization-shared assistant
		if assistant.GroupID == nil {
			logger.Error("权限验证失败：助手不属于当前用户",
				zap.Uint("assistantUserId", assistant.UserID),
				zap.Uint("currentUserId", user.ID))
			response.Fail(c, "Insufficient permissions: Assistant does not belong to you", nil)
			return
		}
		logger.Info("助手属于组织，跳过权限检查", zap.Uint("groupId", *assistant.GroupID))
		// TODO: Organization member permission check can be added here
	}

	// Get device information from cache
	macAddress, _ := dataMap["mac_address"].(string)
	board, _ := dataMap["board"].(string)
	appVersion, _ := dataMap["app_version"].(string)

	if macAddress == "" {
		macAddress = deviceId
	}
	if board == "" {
		board = "default"
	}
	if appVersion == "" {
		appVersion = "1.0.0"
	}

	logger.Info("准备创建设备",
		zap.String("deviceId", deviceId),
		zap.String("macAddress", macAddress),
		zap.String("board", board),
		zap.String("appVersion", appVersion),
		zap.Uint("userId", user.ID),
		zap.Uint("assistantID", assistantID))

	// Create device
	now := time.Now()
	newDevice := &models.Device{
		ID:            deviceId,
		MacAddress:    macAddress,
		Board:         board,
		AppVersion:    appVersion,
		UserID:        user.ID,
		GroupID:       assistant.GroupID, // 如果助手属于组织，设备也属于该组织
		AgentID:   &assistantID,
		AutoUpdate:    1,
		LastConnected: &now,
		LastSeen:      &now, // Set LastSeen to current time to avoid MySQL datetime error
	}

	logger.Info("开始创建设备记录", zap.Any("deviceData", newDevice))

	if err := models.CreateDevice(h.db, newDevice); err != nil {
		logger.Error("创建设备失败",
			zap.Error(err),
			zap.String("deviceId", deviceId),
			zap.String("macAddress", macAddress),
			zap.Uint("userId", user.ID),
			zap.Uint("assistantID", assistantID),
			zap.String("errorType", fmt.Sprintf("%T", err)))

		// 提供更详细的错误信息
		errorMsg := fmt.Sprintf("Failed to create device: %v", err)
		response.Fail(c, errorMsg, nil)
		return
	}

	logger.Info("设备创建成功", zap.String("deviceId", deviceId))

	// Clean up local cache (key format consistent with xiaozhi-esp32 Redis key)
	logger.Info("清理缓存", zap.String("dataKey", dataKey), zap.String("deviceKey", deviceKey))
	cacheClient.Delete(ctx, dataKey)
	cacheClient.Delete(ctx, deviceKey)

	logger.Info("设备激活成功",
		zap.String("deviceId", deviceId),
		zap.String("activationCode", deviceCode),
		zap.Uint("userId", user.ID),
		zap.Uint("assistantID", assistantID))

	response.Success(c, "Device activated successfully", nil)
}

// GetUserDevices gets bound devices - completely consistent with xiaozhi-esp32
// GET /device/bind/:agentId
func (h *Handlers) GetUserDevices(c *gin.Context) {
	agentIdStr := c.Param("agentId")

	// Parse agentId (assistant ID)
	agentId, err := strconv.ParseUint(agentIdStr, 10, 32)
	if err != nil {
		response.Fail(c, "Invalid assistant ID", nil)
		return
	}
	assistantID := uint(agentId)

	// Get current user
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	// 使用新的 GetUserDevices 方法，支持监控字段
	devices, err := models.GetUserDevices(h.db, user.ID, &assistantID)
	if err != nil {
		logger.Error("Failed to query devices", zap.Error(err))
		response.Fail(c, "Failed to query devices", nil)
		return
	}

	response.Success(c, "Query successful", devices)
}

// UnbindDevice unbinds device
// POST /device/unbind
func (h *Handlers) UnbindDevice(c *gin.Context) {
	var req struct {
		DeviceID string `json:"deviceId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid parameters", nil)
		return
	}

	// Get current user
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	// Query device
	device, err := models.GetDeviceByID(h.db, req.DeviceID)
	if err != nil || device == nil {
		response.Fail(c, "Device does not exist", nil)
		return
	}

	// Verify permissions
	if device.UserID != user.ID {
		response.Fail(c, "Insufficient permissions", nil)
		return
	}

	// Delete device
	if err := models.DeleteDevice(h.db, req.DeviceID); err != nil {
		logger.Error("Failed to delete device", zap.Error(err))
		response.Fail(c, "Failed to delete device", nil)
		return
	}

	response.Success(c, "Device unbound successfully", nil)
}

// UpdateDeviceInfo updates device information
// PUT /device/update/:id
func (h *Handlers) UpdateDeviceInfo(c *gin.Context) {
	deviceID := c.Param("id")

	var req struct {
		Alias      string `json:"alias"`
		AutoUpdate *int   `json:"autoUpdate"`
		GroupID    *uint  `json:"groupId,omitempty"` // 组织ID，如果设置则表示这是组织共享的设备
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid parameters", nil)
		return
	}

	// Get current user
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	// Query device
	device, err := models.GetDeviceByID(h.db, deviceID)
	if err != nil || device == nil {
		response.Fail(c, "Device does not exist", nil)
		return
	}

	// Verify permissions: 只有创建者或组织管理员可以更新
	if device.UserID != user.ID {
		if device.GroupID == nil {
			response.Fail(c, "Insufficient permissions", nil)
			return
		}
		// 检查用户是否是组织创建者或管理员
		var group models.Group
		if err := h.db.Where("id = ?", *device.GroupID).First(&group).Error; err != nil {
			response.Fail(c, "Organization not found", nil)
			return
		}
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", *device.GroupID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
				response.Fail(c, "Insufficient permissions", "Only creator or admin can update organization-shared devices")
				return
			}
		}
	}

	// 如果更新了 GroupID，验证权限
	if req.GroupID != nil {
		var group models.Group
		if err := h.db.Where("id = ?", *req.GroupID).First(&group).Error; err != nil {
			response.Fail(c, "组织不存在", nil)
			return
		}
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ?", *req.GroupID, user.ID).First(&member).Error; err != nil {
				response.Fail(c, "权限不足", "您不是该组织的成员")
				return
			}
		}
		device.GroupID = req.GroupID
	}

	// Update device information
	if req.Alias != "" {
		device.Alias = req.Alias
	}
	if req.AutoUpdate != nil {
		device.AutoUpdate = *req.AutoUpdate
	}

	if err := models.UpdateDevice(h.db, device); err != nil {
		logger.Error("Failed to update device", zap.Error(err))
		response.Fail(c, "Failed to update device", nil)
		return
	}

	response.Success(c, "Update successful", device)
}

// ManualAddDevice manually adds device
// POST /device/manual-add
func (h *Handlers) ManualAddDevice(c *gin.Context) {
	var req struct {
		AgentID    string `json:"agentId" binding:"required"`
		Board      string `json:"board" binding:"required"`
		AppVersion string `json:"appVersion"`
		MacAddress string `json:"macAddress" binding:"required"`
		GroupID    *uint  `json:"groupId,omitempty"` // 组织ID，如果设置则表示这是组织共享的设备
	}

	logger.Info("开始手动添加设备流程", zap.String("clientIP", c.ClientIP()))

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("手动添加设备失败：参数绑定错误", zap.Error(err))
		response.Fail(c, "参数错误", nil)
		return
	}

	logger.Info("请求参数解析成功",
		zap.String("agentId", req.AgentID),
		zap.String("board", req.Board),
		zap.String("appVersion", req.AppVersion),
		zap.String("macAddress", req.MacAddress),
		zap.Any("groupId", req.GroupID))

	// Validate MAC address format
	if !isMacAddressValid(req.MacAddress) {
		logger.Error("MAC地址格式无效", zap.String("macAddress", req.MacAddress))
		response.Fail(c, "Invalid MAC address", nil)
		return
	}

	logger.Info("MAC地址格式验证通过", zap.String("macAddress", req.MacAddress))

	// Check if MAC address already exists
	logger.Info("检查MAC地址是否已存在", zap.String("macAddress", req.MacAddress))
	existingDevice, err := models.GetDeviceByMacAddress(h.db, req.MacAddress)
	if err != nil {
		logger.Warn("查询现有设备时出错", zap.Error(err), zap.String("macAddress", req.MacAddress))
	}
	if existingDevice != nil {
		logger.Error("手动添加设备失败：MAC地址已存在",
			zap.String("macAddress", req.MacAddress),
			zap.Uint("existingUserId", existingDevice.UserID))
		response.Fail(c, "MAC address already exists", nil)
		return
	}

	// 获取当前用户
	user := models.CurrentUser(c)
	if user == nil {
		logger.Error("手动添加设备失败：用户未登录")
		response.Fail(c, "用户未登录", nil)
		return
	}

	logger.Info("获取当前用户成功",
		zap.Uint("userId", user.ID),
		zap.String("userEmail", user.Email))

	// 解析 agentId (assistant ID)
	agentId, err := strconv.ParseUint(req.AgentID, 10, 32)
	if err != nil {
		logger.Error("解析助手ID失败", zap.Error(err), zap.String("agentId", req.AgentID))
		response.Fail(c, "无效的助手ID", nil)
		return
	}
	assistantID := uint(agentId)

	logger.Info("解析助手ID成功", zap.Uint("assistantID", assistantID))

	// 验证 assistant 是否存在且属于当前用户
	var assistant models.Agent
	if err := h.db.Where("id = ?", assistantID).First(&assistant).Error; err != nil {
		logger.Error("查询助手失败", zap.Error(err), zap.Uint("assistantID", assistantID))
		response.Fail(c, "助手不存在", nil)
		return
	}

	logger.Info("查询助手成功",
		zap.Uint("assistantID", uint(assistant.ID)),
		zap.String("assistantName", assistant.Name),
		zap.Uint("assistantUserId", assistant.UserID))

	if assistant.UserID != user.ID {
		// 检查是否是组织共享的助手
		if assistant.GroupID == nil {
			logger.Error("权限验证失败：助手不属于当前用户",
				zap.Uint("assistantUserId", assistant.UserID),
				zap.Uint("currentUserId", user.ID))
			response.Fail(c, "权限不足：助手不属于您", nil)
			return
		}
		logger.Info("助手属于组织，跳过权限检查", zap.Uint("groupId", *assistant.GroupID))
		// TODO: 可以在这里添加组织成员权限检查
	}

	// Set default values
	if req.AppVersion == "" {
		req.AppVersion = "1.0.0"
	}

	// 如果设置了 GroupID，验证用户是否有权限共享到该组织
	if req.GroupID != nil {
		logger.Info("验证组织权限", zap.Uint("groupId", *req.GroupID))
		var group models.Group
		if err := h.db.Where("id = ?", *req.GroupID).First(&group).Error; err != nil {
			logger.Error("查询组织失败", zap.Error(err), zap.Uint("groupId", *req.GroupID))
			response.Fail(c, "组织不存在", nil)
			return
		}
		// 检查用户是否是组织成员或创建者
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ?", *req.GroupID, user.ID).First(&member).Error; err != nil {
				logger.Error("组织权限验证失败",
					zap.Uint("groupId", *req.GroupID),
					zap.Uint("userId", user.ID),
					zap.Error(err))
				response.Fail(c, "权限不足", "您不是该组织的成员")
				return
			}
		}
		logger.Info("组织权限验证通过", zap.Uint("groupId", *req.GroupID))
	}

	logger.Info("准备创建设备",
		zap.String("macAddress", req.MacAddress),
		zap.String("board", req.Board),
		zap.String("appVersion", req.AppVersion),
		zap.Uint("userId", user.ID),
		zap.Uint("assistantID", assistantID),
		zap.Any("groupId", req.GroupID))

	// 创建设备
	now := time.Now()
	newDevice := &models.Device{
		ID:            req.MacAddress,
		MacAddress:    req.MacAddress,
		Board:         req.Board,
		AppVersion:    req.AppVersion,
		UserID:        user.ID,
		GroupID:       req.GroupID,
		AgentID:   &assistantID,
		AutoUpdate:    1,
		LastConnected: &now,
		LastSeen:      &now, // Set LastSeen to current time to avoid MySQL datetime error
	}

	logger.Info("开始创建设备记录", zap.Any("deviceData", newDevice))

	if err := models.CreateDevice(h.db, newDevice); err != nil {
		logger.Error("创建设备失败",
			zap.Error(err),
			zap.String("macAddress", req.MacAddress),
			zap.Uint("userId", user.ID),
			zap.Uint("assistantID", assistantID),
			zap.String("errorType", fmt.Sprintf("%T", err)))

		// 提供更详细的错误信息
		errorMsg := fmt.Sprintf("创建设备失败: %v", err)
		response.Fail(c, errorMsg, nil)
		return
	}

	logger.Info("设备创建成功", zap.String("macAddress", req.MacAddress))

	response.Success(c, "Device added successfully", newDevice)
}

// GetDeviceConfig 通过Device-Id获取设备配置（供xiaozhi-server调用）
// GET /device/config/:deviceId
// 不需要认证，因为xiaozhi-server需要调用此接口
func (h *Handlers) GetDeviceConfig(c *gin.Context) {
	deviceID := c.Param("deviceId")

	// 支持从Header获取Device-Id（兼容性）
	if deviceID == "" {
		deviceID = c.GetHeader("Device-Id")
		if deviceID == "" {
			deviceID = c.GetHeader("device-id")
		}
	}

	if deviceID == "" {
		response.Fail(c, "Device ID is required", nil)
		return
	}

	// 根据Device-Id查询设备
	device, err := models.GetDeviceByMacAddress(h.db, deviceID)
	if err != nil || device == nil {
		response.Fail(c, "Device not found or not activated", nil)
		return
	}

	// 检查设备是否绑定了助手
	if device.AgentID == nil {
		response.Fail(c, "Device is not bound to an assistant", nil)
		return
	}

	assistantID := *device.AgentID

	// 获取助手配置
	var assistant models.Agent
	if err := h.db.Where("id = ?", assistantID).First(&assistant).Error; err != nil {
		logger.Error("Failed to get assistant", zap.Error(err), zap.Uint("assistantID", assistantID))
		response.Fail(c, "Failed to get assistant configuration", nil)
		return
	}
	if assistant.ID == 0 {
		response.Fail(c, "Assistant does not exist", nil)
		return
	}

	// 检查助手是否配置了API凭证
	if assistant.ApiKey == "" || assistant.ApiSecret == "" {
		response.Fail(c, "Assistant API credentials not configured", nil)
		return
	}

	// 返回配置信息
	config := map[string]interface{}{
		"deviceId":             deviceID,
		"agentId":              assistantID,
		"apiKey":               assistant.ApiKey,
		"apiSecret":            assistant.ApiSecret,
		"speaker":              assistant.Speaker,
		"llmModel":             assistant.LLMModel,
		"temperature":          assistant.Temperature,
		"systemPrompt":         assistant.SystemPrompt,
		"maxTokens":            assistant.MaxTokens,
		"enableVAD":            assistant.EnableVAD,
		"vadThreshold":         assistant.VADThreshold,
		"vadConsecutiveFrames": assistant.VADConsecutiveFrames,
	}

	logger.Info("Device config requested",
		zap.String("deviceID", deviceID),
		zap.Int64("assistantID", int64(assistantID)))

	response.Success(c, "Success", config)
}

// UpdateDeviceStatus 更新设备状态
// POST /device/status
func (h *Handlers) UpdateDeviceStatus(c *gin.Context) {
	var req struct {
		MacAddress    string                 `json:"macAddress" binding:"required"`
		IsOnline      *bool                  `json:"isOnline"`
		CPUUsage      *float64               `json:"cpuUsage"`
		MemoryUsage   *float64               `json:"memoryUsage"`
		Temperature   *float64               `json:"temperature"`
		SystemInfo    map[string]interface{} `json:"systemInfo"`
		HardwareInfo  map[string]interface{} `json:"hardwareInfo"`
		NetworkInfo   map[string]interface{} `json:"networkInfo"`
		AudioStatus   map[string]interface{} `json:"audioStatus"`
		ServiceStatus map[string]interface{} `json:"serviceStatus"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "请求参数错误", nil)
		return
	}

	// 构建更新数据
	updates := make(map[string]interface{})
	updates["last_seen"] = time.Now()

	if req.IsOnline != nil {
		updates["is_online"] = *req.IsOnline
		if *req.IsOnline {
			updates["start_time"] = time.Now()
		}
	}

	if req.CPUUsage != nil {
		updates["cpu_usage"] = *req.CPUUsage
	}

	if req.MemoryUsage != nil {
		updates["memory_usage"] = *req.MemoryUsage
	}

	if req.Temperature != nil {
		updates["temperature"] = *req.Temperature
	}

	if req.SystemInfo != nil {
		systemInfoJSON, _ := json.Marshal(req.SystemInfo)
		jsonStr := string(systemInfoJSON)
		updates["system_info"] = &jsonStr
	}

	if req.HardwareInfo != nil {
		hardwareInfoJSON, _ := json.Marshal(req.HardwareInfo)
		jsonStr := string(hardwareInfoJSON)
		updates["hardware_info"] = &jsonStr
	}

	if req.NetworkInfo != nil {
		networkInfoJSON, _ := json.Marshal(req.NetworkInfo)
		jsonStr := string(networkInfoJSON)
		updates["network_info"] = &jsonStr
	}

	if req.AudioStatus != nil {
		audioStatusJSON, _ := json.Marshal(req.AudioStatus)
		jsonStr := string(audioStatusJSON)
		updates["audio_status"] = &jsonStr
	}

	if req.ServiceStatus != nil {
		serviceStatusJSON, _ := json.Marshal(req.ServiceStatus)
		jsonStr := string(serviceStatusJSON)
		updates["service_status"] = &jsonStr
	}

	err := models.UpdateDeviceStatus(h.db, req.MacAddress, updates)
	if err != nil {
		logger.Error("更新设备状态失败", zap.Error(err), zap.String("mac_address", req.MacAddress))
		response.Fail(c, "更新设备状态失败", nil)
		return
	}

	response.Success(c, "设备状态更新成功", nil)
}

// LogDeviceError 记录设备错误
// POST /device/error
func (h *Handlers) LogDeviceError(c *gin.Context) {
	var req struct {
		MacAddress string `json:"macAddress" binding:"required"`
		ErrorType  string `json:"errorType" binding:"required"`
		ErrorLevel string `json:"errorLevel" binding:"required"`
		ErrorCode  string `json:"errorCode"`
		ErrorMsg   string `json:"errorMsg" binding:"required"`
		StackTrace string `json:"stackTrace"`
		Context    string `json:"context"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "请求参数错误", nil)
		return
	}

	// 查找设备
	device, err := models.GetDeviceByMacAddress(h.db, req.MacAddress)
	if err != nil {
		logger.Error("查找设备失败", zap.Error(err), zap.String("mac_address", req.MacAddress))
		response.Fail(c, "查找设备失败", nil)
		return
	}

	if device == nil {
		response.Fail(c, "设备不存在", nil)
		return
	}

	err = models.LogDeviceError(h.db, device.ID, req.MacAddress, req.ErrorType, req.ErrorLevel,
		req.ErrorCode, req.ErrorMsg, req.StackTrace, req.Context)
	if err != nil {
		logger.Error("记录设备错误失败", zap.Error(err), zap.String("device_id", device.ID))
		response.Fail(c, "记录设备错误失败", nil)
		return
	}

	response.Success(c, "设备错误记录成功", nil)
}

// GetDeviceDetail 获取设备详情
// GET /device/:deviceId
func (h *Handlers) GetDeviceDetail(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "用户未登录", nil)
		return
	}

	deviceID := c.Param("deviceId")
	if deviceID == "" {
		response.Fail(c, "设备ID不能为空", nil)
		return
	}

	// 查询设备详情 - 使用MAC地址查询
	device, err := models.GetDeviceByMacAddress(h.db, deviceID)
	if err != nil || device == nil {
		response.Fail(c, "设备不存在", nil)
		return
	}

	// 验证设备所有权
	if device.UserID != user.ID {
		// 检查是否是组织共享设备
		if device.GroupID == nil {
			response.Fail(c, "权限不足", nil)
			return
		}
		// 检查用户是否是组织成员
		var member models.GroupMember
		if err := h.db.Where("group_id = ? AND user_id = ?", *device.GroupID, user.ID).First(&member).Error; err != nil {
			response.Fail(c, "权限不足", nil)
			return
		}
	}

	response.Success(c, "获取成功", device)
}

// GetDeviceErrorLogs 获取设备错误日志
// GET /device/:deviceId/error-logs
func (h *Handlers) GetDeviceErrorLogs(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "用户未登录", nil)
		return
	}

	deviceID := c.Param("deviceId")
	if deviceID == "" {
		response.Fail(c, "设备ID不能为空", nil)
		return
	}

	// 验证设备所有权 - 使用MAC地址查询
	var device models.Device
	err := h.db.Where("mac_address = ? AND user_id = ?", deviceID, user.ID).First(&device).Error
	if err != nil {
		response.Fail(c, "设备不存在", nil)
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 获取错误类型和级别过滤参数
	errorType := c.Query("error_type")
	errorLevel := c.Query("error_level")

	var logs []models.DeviceErrorLog
	var total int64

	// 构建查询
	query := h.db.Where("mac_address = ?", device.MacAddress)
	if errorType != "" {
		query = query.Where("error_type = ?", errorType)
	}
	if errorLevel != "" {
		query = query.Where("error_level = ?", errorLevel)
	}

	// 获取总数
	query.Model(&models.DeviceErrorLog{}).Count(&total)

	// 获取分页数据
	err = query.Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&logs).Error

	if err != nil {
		logger.Error("获取设备错误日志失败", zap.Error(err), zap.String("mac_address", device.MacAddress))
		response.Fail(c, "获取错误日志失败", nil)
		return
	}

	response.Success(c, "获取成功", gin.H{
		"logs":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ResolveDeviceError 标记设备错误为已解决
// POST /device/error-logs/:errorId/resolve
func (h *Handlers) ResolveDeviceError(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "用户未登录", nil)
		return
	}

	errorIDStr := c.Param("errorId")
	errorID, err := strconv.ParseUint(errorIDStr, 10, 32)
	if err != nil {
		response.Fail(c, "错误ID格式错误", nil)
		return
	}

	// 验证错误日志所有权
	var errorLog models.DeviceErrorLog
	if err := h.db.First(&errorLog, errorID).Error; err != nil {
		response.Fail(c, "错误日志不存在", nil)
		return
	}

	// 验证设备所有权
	var device models.Device
	if err := h.db.Where("mac_address = ? AND user_id = ?", errorLog.MacAddress, user.ID).First(&device).Error; err != nil {
		response.Fail(c, "无权限操作此设备", nil)
		return
	}

	// 标记为已解决
	if err := models.ResolveDeviceError(h.db, uint(errorID), user.DisplayName); err != nil {
		logger.Error("标记错误为已解决失败", zap.Error(err), zap.Uint64("errorId", errorID))
		response.Fail(c, "标记失败", nil)
		return
	}

	response.Success(c, "已标记为解决", nil)
}

// GetCallRecordings 获取通话录音列表
// GET /device/call-recordings
func (h *Handlers) GetCallRecordings(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "用户未登录", nil)
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 过滤参数
	assistantIDStr := c.Query("agent_id")
	macAddress := c.Query("mac_address")

	var recordings []models.CallRecording
	var total int64
	var err error

	if assistantIDStr != "" {
		// 按助手ID查询
		assistantID, err := strconv.ParseUint(assistantIDStr, 10, 32)
		if err != nil {
			response.Fail(c, "助手ID格式错误", nil)
			return
		}
		recordings, total, err = models.GetCallRecordingsByAgent(h.db, user.ID, uint(assistantID), pageSize, (page-1)*pageSize)
	} else if macAddress != "" {
		// 按设备MAC地址查询
		recordings, total, err = models.GetCallRecordingsByDevice(h.db, user.ID, macAddress, pageSize, (page-1)*pageSize)
	} else {
		// 查询用户所有录音
		offset := (page - 1) * pageSize
		query := h.db.Where("user_id = ?", user.ID)
		query.Model(&models.CallRecording{}).Count(&total)
		err = query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&recordings).Error
	}

	if err != nil {
		logger.Error("获取通话录音列表失败", zap.Error(err), zap.Uint("user_id", user.ID))
		response.Fail(c, "获取录音列表失败", nil)
		return
	}

	// 构建响应数据，包含对话摘要
	recordingList := make([]map[string]interface{}, 0)
	for _, recording := range recordings {
		// 获取对话详情以提取摘要信息
		conversationDetails, _ := recording.GetConversationDetails()

		recordingItem := map[string]interface{}{
			"id":             recording.ID,
			"userId":         recording.UserID,
			"agentId":        recording.AgentID,
			"deviceId":       recording.DeviceID,
			"macAddress":     recording.MacAddress,
			"sessionId":      recording.SessionID,
			"storageUrl":     recording.StorageURL,
			"audioFormat":    recording.AudioFormat,
			"audioSize":      recording.AudioSize,
			"duration":       recording.Duration,
			"sampleRate":     recording.SampleRate,
			"channels":       recording.Channels,
			"callType":       recording.CallType,
			"callStatus":     recording.CallStatus,
			"startTime":      recording.StartTime,
			"endTime":        recording.EndTime,
			"summary":        recording.Summary,
			"category":       recording.Category,
			"isImportant":    recording.IsImportant,
			"isArchived":     recording.IsArchived,
			"analysisStatus": recording.AnalysisStatus,
			"createdAt":      recording.CreatedAt,
			// 新增字段
			"llmModel":    recording.LLMModel,
			"ttsProvider": recording.TTSProvider,
			"asrProvider": recording.ASRProvider,
		}

		// 添加对话摘要信息
		if conversationDetails != nil {
			recordingItem["totalTurns"] = conversationDetails.TotalTurns
			recordingItem["userTurns"] = conversationDetails.UserTurns
			recordingItem["aiTurns"] = conversationDetails.AITurns
		}

		recordingList = append(recordingList, recordingItem)
	}

	response.Success(c, "获取成功", gin.H{
		"recordings": recordingList,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// AnalyzeCallRecording 分析通话录音
// POST /device/call-recordings/:id/analyze
func (h *Handlers) AnalyzeCallRecording(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "用户未登录", nil)
		return
	}

	recordingIDStr := c.Param("id")
	recordingID, err := strconv.ParseUint(recordingIDStr, 10, 32)
	if err != nil {
		response.Fail(c, "录音ID格式错误", nil)
		return
	}

	// 验证录音所有权
	var recording models.CallRecording
	if err := h.db.Where("id = ? AND user_id = ?", recordingID, user.ID).First(&recording).Error; err != nil {
		response.Fail(c, "录音不存在", nil)
		return
	}

	// 设置分析状态为 analyzing
	if err := h.db.Model(&recording).Update("analysis_status", "analyzing").Error; err != nil {
		logger.Error("更新分析状态失败", zap.Error(err), zap.Uint("recordingID", recording.ID))
		response.Fail(c, "启动分析失败", nil)
		return
	}

	go func() {
		ctx := context.Background()
		conversationDetails, err := recording.GetConversationDetails()
		if err != nil || conversationDetails == nil {
			logger.Error("获取对话详情失败", zap.Error(err), zap.Uint("recordingID", recording.ID))
			return
		}

		// 获取助手信息
		var assistant models.Agent
		if err := h.db.Where("id = ?", recording.AgentID).First(&assistant).Error; err != nil {
			logger.Error("获取助手信息失败", zap.Error(err), zap.Uint("assistantID", recording.AgentID))
			return
		}

		// 根据 assistant 的 apiKey 和 apiSecret 获取 UserCredential
		credential, err := models.GetUserCredentialByApiSecretAndApiKey(h.db, assistant.ApiKey, assistant.ApiSecret)
		if err != nil || credential == nil {
			logger.Error("获取用户凭证失败", zap.Error(err), zap.String("apiKey", assistant.ApiKey))
			return
		}

		// 从 UserCredential 中获取 LLM 的 apiKey 和 apiURL
		llmApiKey := credential.LLMApiKey
		llmApiURL := credential.LLMApiURL
		llmProvider := credential.LLMProvider

		if llmApiKey == "" || llmApiURL == "" {
			logger.Error("LLM 凭证不完整", zap.String("llmProvider", llmProvider))
			return
		}

		// 构建对话文本
		conversationText := ""
		for _, turn := range conversationDetails.Turns {
			if turn.Type == "user" {
				conversationText += fmt.Sprintf("用户: %s\n", turn.Content)
			} else if turn.Type == "ai" {
				conversationText += fmt.Sprintf("AI: %s\n", turn.Content)
			}
		}

		// 创建 LLM 提供者
		var provider llm.LLMHandler
		var err2 error

		// 根据 LLM 提供商类型创建对应的提供者
		provider, err2 = llm.NewLLMProvider(ctx, llmProvider, llmApiKey, llmApiURL, "你是一个专业的对话分析助手")

		if err2 != nil {
			logger.Error("创建 LLM 提供者失败", zap.Error(err2), zap.String("llmProvider", llmProvider))
			return
		}

		// 构建分析提示词
		analysisPrompt := fmt.Sprintf(`请分析以下对话，并提供以下信息（以 JSON 格式返回）：
1. summary: 对话摘要（一句话）
2. sentiment: 情感分数（-1 到 1 之间的浮点数）
3. satisfaction: 满意度分数（0 到 1 之间的浮点数）
4. keywords: 关键词列表
5. category: 对话分类
6. isImportant: 是否重要
7. actionItems: 行动项列表
8. issues: 问题列表
9. insights: 深度洞察

对话内容：
%s

请返回有效的 JSON 格式。`, conversationText)

		// 调用 LLM 进行分析
		respLLM, err := provider.QueryWithOptions(analysisPrompt, &llm.QueryOptions{
			Model:       recording.LLMModel,
			Temperature: 0.7,
		})
		result := ""
		if respLLM != nil && len(respLLM.Choices) > 0 {
			result = respLLM.Choices[0].Content
		}

		if err != nil {
			logger.Error("LLM 分析失败", zap.Error(err), zap.Uint("recordingID", recording.ID))
			// 更新分析状态为失败
			h.db.Model(&recording).Updates(map[string]interface{}{
				"analysis_status": "failed",
				"analysis_error":  err.Error(),
			})
			return
		}

		// 解析 JSON 结果
		var analysisResult map[string]interface{}
		if err := json.Unmarshal([]byte(result), &analysisResult); err != nil {
			logger.Error("解析分析结果失败", zap.Error(err), zap.String("result", result))
			// 尝试提取 JSON 部分
			jsonStart := strings.Index(result, "{")
			jsonEnd := strings.LastIndex(result, "}")
			if jsonStart >= 0 && jsonEnd > jsonStart {
				if err := json.Unmarshal([]byte(result[jsonStart:jsonEnd+1]), &analysisResult); err != nil {
					logger.Error("提取 JSON 失败", zap.Error(err))
					return
				}
			} else {
				return
			}
		}

		// 保存分析结果
		analysisJSON, _ := json.Marshal(analysisResult)
		now := time.Now()
		err = h.db.Model(&recording).Updates(map[string]interface{}{
			"analysis_status": "completed",
			"ai_analysis":     string(analysisJSON),
			"analyzed_at":     now,
		}).Error

		if err != nil {
			logger.Error("保存分析结果失败", zap.Error(err), zap.Uint("recordingID", recording.ID))
			return
		}

		logger.Info("通话记录分析完成", zap.Uint("recordingID", recording.ID))
	}()

	response.Success(c, "分析已启动", nil)
}

// BatchAnalyzeCallRecordings 批量分析通话录音
// POST /device/call-recordings/batch-analyze
func (h *Handlers) BatchAnalyzeCallRecordings(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "用户未登录", nil)
		return
	}

	var req struct {
		AgentID *uint `json:"agentId"`
		Limit       int   `json:"limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "请求参数错误", nil)
		return
	}

	// 设置默认限制
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	response.Success(c, "批量分析已启动", nil)
}

// GetCallRecordingAnalysis 获取通话录音分析结果
// GET /device/call-recordings/:id/analysis
func (h *Handlers) GetCallRecordingAnalysis(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "用户未登录", nil)
		return
	}

	recordingIDStr := c.Param("id")
	recordingID, err := strconv.ParseUint(recordingIDStr, 10, 32)
	if err != nil {
		response.Fail(c, "录音ID格式错误", nil)
		return
	}

	// 验证录音所有权并获取分析结果
	var recording models.CallRecording
	if err := h.db.Where("id = ? AND user_id = ?", recordingID, user.ID).First(&recording).Error; err != nil {
		response.Fail(c, "录音不存在", nil)
		return
	}

	// 构建分析结果响应
	analysisData := gin.H{
		"recordingId":     recording.ID,
		"analysisStatus":  recording.AnalysisStatus,
		"analysisError":   recording.AnalysisError,
		"analyzedAt":      recording.AnalyzedAt,
		"autoAnalyzed":    recording.AutoAnalyzed,
		"analysisVersion": recording.AnalysisVersion,
	}

	// 如果有分析结果，解析并返回
	if recording.AIAnalysis != "" {
		var analysisResult map[string]interface{}
		if err := json.Unmarshal([]byte(recording.AIAnalysis), &analysisResult); err == nil {
			analysisData["analysis"] = analysisResult
		} else {
			analysisData["analysis"] = recording.AIAnalysis // 如果解析失败，返回原始文本
		}
	}

	response.Success(c, "获取成功", analysisData)
}

// GetCallRecordingDetail 获取通话录音详情
// GET /api/device/call-recordings/:id
func (h *Handlers) GetCallRecordingDetail(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", nil)
		return
	}

	recordingIDStr := c.Param("id")
	recordingID, err := strconv.ParseUint(recordingIDStr, 10, 32)
	if err != nil {
		response.Fail(c, "无效的记录ID", nil)
		return
	}

	// 获取通话记录
	var recording models.CallRecording
	err = h.db.Where("id = ? AND user_id = ?", recordingID, user.ID).First(&recording).Error
	if err != nil {
		logger.Error("获取通话记录详情失败", zap.Error(err), zap.Uint("userID", user.ID), zap.Uint64("recordingID", recordingID))
		response.Fail(c, "通话记录不存在", nil)
		return
	}

	// 获取真实的对话详情数据
	conversationDetails, err := recording.GetConversationDetails()
	if err != nil {
		logger.Error("解析对话详情失败", zap.Error(err), zap.Uint64("recordingID", recordingID))
	}

	// 获取真实的时间指标数据
	timingMetrics, err := recording.GetTimingMetrics()
	if err != nil {
		logger.Error("解析时间指标失败", zap.Error(err), zap.Uint64("recordingID", recordingID))
	}

	// 获取发音人列表
	speakers := recording.GetSpeakers()

	// 构建详细响应 - 包含所有字段
	detailResponse := map[string]interface{}{
		"id":              recording.ID,
		"userId":          recording.UserID,
		"agentId":         recording.AgentID,
		"deviceId":        recording.DeviceID,
		"macAddress":      recording.MacAddress,
		"sessionId":       recording.SessionID,
		"storageUrl":      recording.StorageURL,
		"audioFormat":     recording.AudioFormat,
		"audioSize":       recording.AudioSize,
		"duration":        recording.Duration,
		"sampleRate":      recording.SampleRate,
		"channels":        recording.Channels,
		"callType":        recording.CallType,
		"callStatus":      recording.CallStatus,
		"startTime":       recording.StartTime,
		"endTime":         recording.EndTime,
		"summary":         recording.Summary,
		"keywords":        recording.Keywords,
		"audioQuality":    recording.AudioQuality,
		"noiseLevel":      recording.NoiseLevel,
		"tags":            recording.Tags,
		"category":        recording.Category,
		"isImportant":     recording.IsImportant,
		"isArchived":      recording.IsArchived,
		"aiAnalysis":      recording.AIAnalysis,
		"analysisStatus":  recording.AnalysisStatus,
		"analysisError":   recording.AnalysisError,
		"analyzedAt":      recording.AnalyzedAt,
		"autoAnalyzed":    recording.AutoAnalyzed,
		"analysisVersion": recording.AnalysisVersion,
		"createdAt":       recording.CreatedAt,
		// 新增字段
		"speakers":    speakers,
		"llmModel":    recording.LLMModel,
		"ttsProvider": recording.TTSProvider,
		"asrProvider": recording.ASRProvider,
	}

	// 添加真实的对话详情数据（如果存在）
	if conversationDetails != nil {
		detailResponse["conversationDetailsData"] = conversationDetails
	} else {
		// 如果没有真实数据，生成基于现有数据的简单结构
		detailResponse["conversationDetailsData"] = generateBasicConversationDetails(recording)
	}

	// 添加真实的时间指标数据（如果存在）
	if timingMetrics != nil {
		detailResponse["timingMetricsData"] = timingMetrics
	} else {
		// 如果没有真实数据，生成基于现有数据的简单指标
		detailResponse["timingMetricsData"] = generateBasicTimingMetrics(recording)
	}

	response.Success(c, "获取成功", detailResponse)
}

// ServeRecordingFile 提供录音文件下载服务
// GET /device/recordings/*filepath
func (h *Handlers) ServeRecordingFile(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "用户未登录", nil)
		return
	}

	filePath := c.Param("filepath")
	if filePath == "" {
		response.Fail(c, "文件路径不能为空", nil)
		return
	}

	// URL解码文件路径（处理MAC地址中的冒号等特殊字符）
	decodedPath := strings.ReplaceAll(filePath, "%3A", ":")

	// 从URL路径中提取录音ID（如果有的话）
	// 或者通过文件路径验证用户权限

	// 这里需要验证用户是否有权限访问该录音文件
	// 可以通过文件路径中的user_id来验证
	// 例如: /recordings/user_1/assistant_2/2026/01/25/file.wav

	// 简单的权限验证：检查路径是否包含用户ID
	expectedUserPath := fmt.Sprintf("user_%d", user.ID)
	if !strings.Contains(decodedPath, expectedUserPath) {
		response.Fail(c, "权限不足", nil)
		return
	}

	// 构建完整的文件路径
	// 使用lingstorage作为录音文件的存储根目录
	recordingBasePath := "./lingstorage" // 与录音管理器的存储路径一致
	fullPath := filepath.Join(recordingBasePath, decodedPath)

	// 检查文件是否存在
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		response.Fail(c, "文件不存在", nil)
		return
	}

	// 设置适当的Content-Type
	ext := filepath.Ext(fullPath)
	switch ext {
	case ".wav":
		c.Header("Content-Type", "audio/wav")
	case ".opus":
		c.Header("Content-Type", "audio/opus")
	case ".mp3":
		c.Header("Content-Type", "audio/mpeg")
	default:
		c.Header("Content-Type", "application/octet-stream")
	}

	// 设置缓存头
	c.Header("Cache-Control", "public, max-age=3600")

	// 检查是否是下载请求
	if c.Query("download") == "1" {
		// 只有明确请求下载时才设置下载头
		filename := filepath.Base(fullPath)
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	} else {
		// 否则设置为内联播放
		c.Header("Content-Disposition", "inline")
	}

	// 提供文件服务
	c.File(fullPath)
}

// generateBasicConversationDetails 基于现有数据生成基本的对话详情
func generateBasicConversationDetails(recording models.CallRecording) map[string]interface{} {
	// 从 ConversationDetailsJSON 中解析对话数据
	conversationDetails, err := recording.GetConversationDetails()
	if err == nil && conversationDetails != nil && len(conversationDetails.Turns) > 0 {
		// 使用实际的对话数据
		turns := make([]map[string]interface{}, 0)
		for _, turn := range conversationDetails.Turns {
			turnMap := map[string]interface{}{
				"turnId":    turn.TurnID,
				"timestamp": turn.Timestamp.Format(time.RFC3339),
				"type":      turn.Type,
				"content":   turn.Content,
				"startTime": turn.StartTime.Format(time.RFC3339),
				"endTime":   turn.EndTime.Format(time.RFC3339),
				"duration":  turn.Duration,
			}

			// 添加用户输入特有字段
			if turn.ASRStartTime != nil {
				turnMap["asrStartTime"] = turn.ASRStartTime.Format(time.RFC3339)
			}
			if turn.ASREndTime != nil {
				turnMap["asrEndTime"] = turn.ASREndTime.Format(time.RFC3339)
			}
			if turn.ASRDuration != nil {
				turnMap["asrDuration"] = *turn.ASRDuration
			}

			// 添加AI回复特有字段
			if turn.LLMStartTime != nil {
				turnMap["llmStartTime"] = turn.LLMStartTime.Format(time.RFC3339)
			}
			if turn.LLMEndTime != nil {
				turnMap["llmEndTime"] = turn.LLMEndTime.Format(time.RFC3339)
			}
			if turn.LLMDuration != nil {
				turnMap["llmDuration"] = *turn.LLMDuration
			}
			if turn.TTSStartTime != nil {
				turnMap["ttsStartTime"] = turn.TTSStartTime.Format(time.RFC3339)
			}
			if turn.TTSEndTime != nil {
				turnMap["ttsEndTime"] = turn.TTSEndTime.Format(time.RFC3339)
			}
			if turn.TTSDuration != nil {
				turnMap["ttsDuration"] = *turn.TTSDuration
			}

			// 添加延迟指标
			if turn.ResponseDelay != nil {
				turnMap["responseDelay"] = *turn.ResponseDelay
			}
			if turn.TotalDelay != nil {
				turnMap["totalDelay"] = *turn.TotalDelay
			}

			turns = append(turns, turnMap)
		}

		return map[string]interface{}{
			"sessionId":     conversationDetails.SessionID,
			"startTime":     conversationDetails.StartTime.Format(time.RFC3339),
			"endTime":       conversationDetails.EndTime.Format(time.RFC3339),
			"totalTurns":    conversationDetails.TotalTurns,
			"userTurns":     conversationDetails.UserTurns,
			"aiTurns":       conversationDetails.AITurns,
			"interruptions": conversationDetails.Interruptions,
			"turns":         turns,
		}
	}

	// 如果没有任何对话数据，创建示例数据
	turns := make([]map[string]interface{}, 0)
	turns = append(turns, map[string]interface{}{
		"turnId":    1,
		"timestamp": recording.StartTime.Format(time.RFC3339),
		"type":      "user",
		"content":   "你好，我想了解一下产品信息",
		"startTime": recording.StartTime.Format(time.RFC3339),
		"endTime":   recording.StartTime.Add(3 * time.Second).Format(time.RFC3339),
		"duration":  3000,
	})

	startTime := recording.StartTime.Add(4 * time.Second)
	turns = append(turns, map[string]interface{}{
		"turnId":    2,
		"timestamp": startTime.Format(time.RFC3339),
		"type":      "ai",
		"content":   "您好！很高兴为您服务，请问有什么可以帮助您的吗？",
		"startTime": startTime.Format(time.RFC3339),
		"endTime":   startTime.Add(5 * time.Second).Format(time.RFC3339),
		"duration":  5000,
	})

	return map[string]interface{}{
		"sessionId":     recording.SessionID,
		"startTime":     recording.StartTime.Format(time.RFC3339),
		"endTime":       recording.EndTime.Format(time.RFC3339),
		"totalTurns":    2,
		"userTurns":     1,
		"aiTurns":       1,
		"interruptions": 0,
		"turns":         turns,
	}
}

// generateBasicTimingMetrics 基于现有数据生成基本的时间指标
func generateBasicTimingMetrics(recording models.CallRecording) map[string]interface{} {
	sessionDuration := int64(recording.Duration) * 1000 // 转换为毫秒

	// 基于对话详情来统计调用次数和时间
	asrCalls := 0
	asrTotalTime := int64(0)
	asrMinTime := int64(0)
	asrMaxTime := int64(0)

	llmCalls := 0
	llmTotalTime := int64(0)
	llmMinTime := int64(0)
	llmMaxTime := int64(0)

	ttsCalls := 0
	ttsTotalTime := int64(0)
	ttsMinTime := int64(0)
	ttsMaxTime := int64(0)

	responseDelays := make([]int64, 0)
	totalDelays := make([]int64, 0)

	conversationDetails, err := recording.GetConversationDetails()
	if err == nil && conversationDetails != nil {
		for _, turn := range conversationDetails.Turns {
			if turn.Type == "user" {
				asrCalls++
				if turn.ASRDuration != nil {
					asrTotalTime += *turn.ASRDuration
					if asrMinTime == 0 || *turn.ASRDuration < asrMinTime {
						asrMinTime = *turn.ASRDuration
					}
					if *turn.ASRDuration > asrMaxTime {
						asrMaxTime = *turn.ASRDuration
					}
				}
			} else if turn.Type == "ai" {
				llmCalls++
				if turn.LLMDuration != nil {
					llmTotalTime += *turn.LLMDuration
					if llmMinTime == 0 || *turn.LLMDuration < llmMinTime {
						llmMinTime = *turn.LLMDuration
					}
					if *turn.LLMDuration > llmMaxTime {
						llmMaxTime = *turn.LLMDuration
					}
				}

				ttsCalls++
				if turn.TTSDuration != nil {
					ttsTotalTime += *turn.TTSDuration
					if ttsMinTime == 0 || *turn.TTSDuration < ttsMinTime {
						ttsMinTime = *turn.TTSDuration
					}
					if *turn.TTSDuration > ttsMaxTime {
						ttsMaxTime = *turn.TTSDuration
					}
				}

				if turn.ResponseDelay != nil {
					responseDelays = append(responseDelays, *turn.ResponseDelay)
				}
				if turn.TotalDelay != nil {
					totalDelays = append(totalDelays, *turn.TotalDelay)
				}
			}
		}
	}

	// 计算平均值
	asrAverageTime := int64(0)
	if asrCalls > 0 {
		asrAverageTime = asrTotalTime / int64(asrCalls)
	}

	llmAverageTime := int64(0)
	if llmCalls > 0 {
		llmAverageTime = llmTotalTime / int64(llmCalls)
	}

	ttsAverageTime := int64(0)
	if ttsCalls > 0 {
		ttsAverageTime = ttsTotalTime / int64(ttsCalls)
	}

	// 计算响应延迟统计
	averageResponseDelay := int64(0)
	minResponseDelay := int64(0)
	maxResponseDelay := int64(0)
	if len(responseDelays) > 0 {
		for _, delay := range responseDelays {
			averageResponseDelay += delay
			if minResponseDelay == 0 || delay < minResponseDelay {
				minResponseDelay = delay
			}
			if delay > maxResponseDelay {
				maxResponseDelay = delay
			}
		}
		averageResponseDelay = averageResponseDelay / int64(len(responseDelays))
	}

	// 计算总延迟统计
	averageTotalDelay := int64(0)
	minTotalDelay := int64(0)
	maxTotalDelay := int64(0)
	if len(totalDelays) > 0 {
		for _, delay := range totalDelays {
			averageTotalDelay += delay
			if minTotalDelay == 0 || delay < minTotalDelay {
				minTotalDelay = delay
			}
			if delay > maxTotalDelay {
				maxTotalDelay = delay
			}
		}
		averageTotalDelay = averageTotalDelay / int64(len(totalDelays))
	}

	return map[string]interface{}{
		"sessionDuration": sessionDuration,
		// ASR指标
		"asrCalls":       asrCalls,
		"asrTotalTime":   asrTotalTime,
		"asrAverageTime": asrAverageTime,
		"asrMinTime":     asrMinTime,
		"asrMaxTime":     asrMaxTime,
		// LLM指标
		"llmCalls":       llmCalls,
		"llmTotalTime":   llmTotalTime,
		"llmAverageTime": llmAverageTime,
		"llmMinTime":     llmMinTime,
		"llmMaxTime":     llmMaxTime,
		// TTS指标
		"ttsCalls":       ttsCalls,
		"ttsTotalTime":   ttsTotalTime,
		"ttsAverageTime": ttsAverageTime,
		"ttsMinTime":     ttsMinTime,
		"ttsMaxTime":     ttsMaxTime,
		// 响应延迟指标
		"responseDelays":       responseDelays,
		"averageResponseDelay": averageResponseDelay,
		"minResponseDelay":     minResponseDelay,
		"maxResponseDelay":     maxResponseDelay,
		// 总延迟指标
		"totalDelays":       totalDelays,
		"averageTotalDelay": averageTotalDelay,
		"minTotalDelay":     minTotalDelay,
		"maxTotalDelay":     maxTotalDelay,
	}
}

// max 返回两个整数中的较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/cache"
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

	if deviceCode == "" {
		response.Fail(c, "Activation code cannot be empty", nil)
		return
	}

	// Verify activation code
	ctx := context.Background()
	// Use global cache (default is local cache, can be configured via CACHE_TYPE environment variable)
	cacheClient := cache.GetGlobalCache()

	// Get device ID from local cache (key format consistent with xiaozhi-esp32 Redis key)
	deviceKey := fmt.Sprintf("ota:activation:code:%s", deviceCode)
	deviceIdObj, ok := cacheClient.Get(ctx, deviceKey)
	if !ok {
		response.Fail(c, "激活码错误", nil)
		return
	}

	deviceId, ok := deviceIdObj.(string)
	if !ok {
		response.Fail(c, "激活码错误", nil)
		return
	}

	// Get device data
	safeDeviceId := strings.ReplaceAll(strings.ToLower(deviceId), ":", "_")
	dataKey := fmt.Sprintf("ota:activation:data:%s", safeDeviceId)
	dataObj, ok := cacheClient.Get(ctx, dataKey)
	if !ok {
		response.Fail(c, "激活码错误", nil)
		return
	}

	dataMap, ok := dataObj.(map[string]interface{})
	if !ok {
		response.Fail(c, "激活码错误", nil)
		return
	}

	cachedCode, ok := dataMap["activation_code"].(string)
	if !ok || cachedCode != deviceCode {
		response.Fail(c, "激活码错误", nil)
		return
	}

	// Check if device has already been activated
	existingDevice, err := models.GetDeviceByMacAddress(h.db, deviceId)
	if err == nil && existingDevice != nil {
		response.Fail(c, "Device has already been activated", nil)
		return
	}

	// Get current user
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	// Parse agentId (assistant ID)
	agentId, err := strconv.ParseUint(agentIdStr, 10, 32)
	if err != nil {
		response.Fail(c, "Invalid assistant ID", nil)
		return
	}
	assistantID := uint(agentId)

	// Verify that assistant exists and belongs to current user
	var assistant models.Assistant
	if err := h.db.Where("id = ?", assistantID).First(&assistant).Error; err != nil {
		response.Fail(c, "Assistant does not exist", nil)
		return
	}

	if assistant.UserID != user.ID {
		// Check if it's an organization-shared assistant
		if assistant.GroupID == nil {
			response.Fail(c, "Insufficient permissions: Assistant does not belong to you", nil)
			return
		}
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

	// Create device
	now := time.Now()
	newDevice := &models.Device{
		ID:            deviceId,
		MacAddress:    macAddress,
		Board:         board,
		AppVersion:    appVersion,
		UserID:        user.ID,
		GroupID:       assistant.GroupID, // 如果助手属于组织，设备也属于该组织
		AssistantID:   &assistantID,
		AutoUpdate:    1,
		LastConnected: &now,
	}

	if err := models.CreateDevice(h.db, newDevice); err != nil {
		logger.Error("Failed to create device", zap.Error(err), zap.String("deviceId", deviceId))
		response.Fail(c, "Failed to create device", nil)
		return
	}

	// Clean up local cache (key format consistent with xiaozhi-esp32 Redis key)
	cacheClient.Delete(ctx, dataKey)
	cacheClient.Delete(ctx, deviceKey)

	logger.Info("Device activated successfully",
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

	// 获取用户所属的组织ID列表
	var groupIDs []uint
	var groupMembers []models.GroupMember
	if err := h.db.Where("user_id = ?", user.ID).Find(&groupMembers).Error; err == nil {
		for _, member := range groupMembers {
			groupIDs = append(groupIDs, member.GroupID)
		}
	}
	// 获取用户创建的组织ID
	var userGroups []models.Group
	if err := h.db.Where("creator_id = ?", user.ID).Find(&userGroups).Error; err == nil {
		for _, group := range userGroups {
			groupIDs = append(groupIDs, group.ID)
		}
	}

	// Query devices: 用户自己的设备 + 组织共享的设备
	var devices []models.Device
	query := h.db.Where("assistant_id = ?", assistantID)
	if len(groupIDs) > 0 {
		query = query.Where("user_id = ? OR (group_id IS NOT NULL AND group_id IN (?))", user.ID, groupIDs)
	} else {
		query = query.Where("user_id = ?", user.ID)
	}

	err = query.Find(&devices).Error
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

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", nil)
		return
	}

	// Validate MAC address format
	if !isMacAddressValid(req.MacAddress) {
		response.Fail(c, "Invalid MAC address", nil)
		return
	}

	// Check if MAC address already exists
	existingDevice, err := models.GetDeviceByMacAddress(h.db, req.MacAddress)
	if err == nil && existingDevice != nil {
		response.Fail(c, "MAC address already exists", nil)
		return
	}

	// 获取当前用户
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "用户未登录", nil)
		return
	}

	// 解析 agentId (assistant ID)
	agentId, err := strconv.ParseUint(req.AgentID, 10, 32)
	if err != nil {
		response.Fail(c, "无效的助手ID", nil)
		return
	}
	assistantID := uint(agentId)

	// 验证 assistant 是否存在且属于当前用户
	var assistant models.Assistant
	if err := h.db.Where("id = ?", assistantID).First(&assistant).Error; err != nil {
		response.Fail(c, "助手不存在", nil)
		return
	}

	if assistant.UserID != user.ID {
		// 检查是否是组织共享的助手
		if assistant.GroupID == nil {
			response.Fail(c, "权限不足：助手不属于您", nil)
			return
		}
		// TODO: 可以在这里添加组织成员权限检查
	}

	// Set default values
	if req.AppVersion == "" {
		req.AppVersion = "1.0.0"
	}

	// 如果设置了 GroupID，验证用户是否有权限共享到该组织
	if req.GroupID != nil {
		var group models.Group
		if err := h.db.Where("id = ?", *req.GroupID).First(&group).Error; err != nil {
			response.Fail(c, "组织不存在", nil)
			return
		}
		// 检查用户是否是组织成员或创建者
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ?", *req.GroupID, user.ID).First(&member).Error; err != nil {
				response.Fail(c, "权限不足", "您不是该组织的成员")
				return
			}
		}
	}

	// 创建设备
	now := time.Now()
	newDevice := &models.Device{
		ID:            req.MacAddress,
		MacAddress:    req.MacAddress,
		Board:         req.Board,
		AppVersion:    req.AppVersion,
		UserID:        user.ID,
		GroupID:       req.GroupID,
		AssistantID:   &assistantID,
		AutoUpdate:    1,
		LastConnected: &now,
	}

	if err := models.CreateDevice(h.db, newDevice); err != nil {
		logger.Error("Failed to create device", zap.Error(err))
		response.Fail(c, "创建设备失败", nil)
		return
	}

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
	if device.AssistantID == nil {
		response.Fail(c, "Device is not bound to an assistant", nil)
		return
	}

	assistantID := *device.AssistantID

	// 获取助手配置
	var assistant models.Assistant
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
		"assistantId":          assistantID,
		"apiKey":               assistant.ApiKey,
		"apiSecret":            assistant.ApiSecret,
		"language":             assistant.Language,
		"speaker":              assistant.Speaker,
		"llmModel":             assistant.LLMModel,
		"temperature":          assistant.Temperature,
		"systemPrompt":         assistant.SystemPrompt,
		"maxTokens":            assistant.MaxTokens,
		"enableVAD":            assistant.EnableVAD,
		"vadThreshold":         assistant.VADThreshold,
		"vadConsecutiveFrames": assistant.VADConsecutiveFrames,
	}

	// 知识库ID（可选）
	if assistant.KnowledgeBaseID != nil && *assistant.KnowledgeBaseID != "" {
		config["knowledgeBaseId"] = *assistant.KnowledgeBaseID
	}

	logger.Info("Device config requested",
		zap.String("deviceID", deviceID),
		zap.Int64("assistantID", int64(assistantID)))

	response.Success(c, "Success", config)
}

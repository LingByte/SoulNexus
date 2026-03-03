package handlers

import (
	"strconv"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CallRecordingHandler 通话记录处理器
type CallRecordingHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewCallRecordingHandler 创建通话记录处理器
func NewCallRecordingHandler(db *gorm.DB) *CallRecordingHandler {
	return &CallRecordingHandler{
		db:     db,
		logger: zap.L().Named("call_recording_handler"),
	}
}

// CallRecordingDetailResponse 详细通话记录响应
type CallRecordingDetailResponse struct {
	*models.CallRecording
	ConversationDetailsData *models.ConversationDetails `json:"conversationDetailsData,omitempty"`
	TimingMetricsData       *models.TimingMetrics       `json:"timingMetricsData,omitempty"`
}

// GetCallRecordings 获取通话记录列表
func (h *CallRecordingHandler) GetCallRecordings(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Fail(c, "未授权", nil)
		return
	}

	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	assistantID, _ := strconv.Atoi(c.Query("assistantId"))
	macAddress := c.Query("macAddress")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var recordings []models.CallRecording
	var total int64
	var err error

	if assistantID > 0 {
		// 按助手查询
		recordings, total, err = models.GetCallRecordingsByAssistant(h.db, userID, uint(assistantID), pageSize, (page-1)*pageSize)
	} else if macAddress != "" {
		// 按设备查询
		recordings, total, err = models.GetCallRecordingsByDevice(h.db, userID, macAddress, pageSize, (page-1)*pageSize)
	} else {
		// 查询所有
		recordings, total, err = models.GetCallRecordingsByUser(h.db, userID, pageSize, (page-1)*pageSize)
	}

	if err != nil {
		h.logger.Error("获取通话记录失败", zap.Error(err), zap.Uint("userID", userID))
		response.Fail(c, "获取通话记录失败", nil)
		return
	}

	response.Success(c, "获取成功", gin.H{
		"recordings": recordings,
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
	})
}

// GetCallRecordingDetail 获取通话记录详情
func (h *CallRecordingHandler) GetCallRecordingDetail(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
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
	err = h.db.Where("id = ? AND user_id = ?", recordingID, userID).First(&recording).Error
	if err != nil {
		h.logger.Error("获取通话记录详情失败", zap.Error(err), zap.Uint("userID", userID), zap.Uint64("recordingID", recordingID))
		response.Fail(c, "通话记录不存在", nil)
		return
	}

	// 构建详细响应
	detailResponse := &CallRecordingDetailResponse{
		CallRecording: &recording,
	}

	// 注意：ConversationDetails 和 TimingMetrics 字段在当前模型中不存在
	// 如果需要这些功能，需要在 CallRecording 模型中添加相应字段
	// 或者从其他来源获取这些数据

	response.Success(c, "获取成功", detailResponse)
}

// DeleteCallRecording 删除通话记录
func (h *CallRecordingHandler) DeleteCallRecording(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Fail(c, "未授权", nil)
		return
	}

	recordingIDStr := c.Param("id")
	recordingID, err := strconv.ParseUint(recordingIDStr, 10, 32)
	if err != nil {
		response.Fail(c, "无效的记录ID", nil)
		return
	}

	// 删除通话记录
	err = h.db.Where("id = ? AND user_id = ?", recordingID, userID).Delete(&models.CallRecording{}).Error
	if err != nil {
		h.logger.Error("删除通话记录失败", zap.Error(err), zap.Uint("userID", userID), zap.Uint64("recordingID", recordingID))
		response.Fail(c, "删除通话记录失败", nil)
		return
	}

	response.Success(c, "删除成功", gin.H{"message": "删除成功"})
}

// GetCallRecordingStats 获取通话记录统计
func (h *CallRecordingHandler) GetCallRecordingStats(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Fail(c, "未授权", nil)
		return
	}

	// 获取查询参数
	assistantID, _ := strconv.Atoi(c.Query("assistantId"))
	macAddress := c.Query("macAddress")

	// Simple stats implementation - just return basic counts for now
	var totalRecordings int64
	query := h.db.Model(&models.CallRecording{}).Where("user_id = ?", userID)

	// Apply filters if provided
	if assistantID > 0 {
		query = query.Where("assistant_id = ?", assistantID)
	}
	if macAddress != "" {
		query = query.Where("mac_address = ?", macAddress)
	}

	err := query.Count(&totalRecordings).Error
	if err != nil {
		h.logger.Error("获取通话记录统计失败", zap.Error(err), zap.Uint("userID", userID))
		response.Fail(c, "获取统计数据失败", nil)
		return
	}

	stats := gin.H{
		"totalRecordings": totalRecordings,
	}

	response.Success(c, "获取成功", stats)
}

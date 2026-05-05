package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

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
	agentID, _ := strconv.Atoi(c.Query("agentId"))
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

	if agentID > 0 {
		var ag models.Agent
		if err = h.db.First(&ag, agentID).Error; err != nil {
			response.Fail(c, "助手不存在", nil)
			return
		}
		if !models.UserIsGroupMember(h.db, userID, ag.GroupID) {
			response.Fail(c, "无权访问", nil)
			return
		}
		recordings, total, err = models.GetCallRecordingsByAgent(h.db, ag.GroupID, uint(agentID), pageSize, (page-1)*pageSize)
	} else if macAddress != "" {
		dev, derr := models.GetDeviceByMacAddress(h.db, macAddress)
		if derr != nil || dev == nil {
			response.Fail(c, "设备不存在", nil)
			return
		}
		if !models.UserIsGroupMember(h.db, userID, dev.GroupID) {
			response.Fail(c, "无权访问", nil)
			return
		}
		recordings, total, err = models.GetCallRecordingsByDevice(h.db, dev.GroupID, macAddress, pageSize, (page-1)*pageSize)
	} else {
		recordings, total, err = models.GetCallRecordingsByMemberUser(h.db, userID, pageSize, (page-1)*pageSize)
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
	recordingID, parseErr := strconv.ParseUint(recordingIDStr, 10, 32)
	if parseErr != nil {
		response.Fail(c, "无效的记录ID", nil)
		return
	}

	groupIDs, err := models.MemberGroupIDs(h.db, userID)
	if err != nil || len(groupIDs) == 0 {
		response.Fail(c, "通话记录不存在", nil)
		return
	}
	var recording models.CallRecording
	err = h.db.Where("id = ? AND group_id IN ?", recordingID, groupIDs).First(&recording).Error
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

	groupIDs, err := models.MemberGroupIDs(h.db, userID)
	if err != nil || len(groupIDs) == 0 {
		response.Fail(c, "删除失败", nil)
		return
	}
	err = h.db.Where("id = ? AND group_id IN ?", recordingID, groupIDs).Delete(&models.CallRecording{}).Error
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
	agentID, _ := strconv.Atoi(c.Query("agentId"))
	macAddress := c.Query("macAddress")

	groupIDs, err := models.MemberGroupIDs(h.db, userID)
	if err != nil || len(groupIDs) == 0 {
		response.Success(c, "获取成功", gin.H{"totalRecordings": int64(0)})
		return
	}

	var totalRecordings int64
	query := h.db.Model(&models.CallRecording{}).Where("group_id IN ?", groupIDs)

	if agentID > 0 {
		query = query.Where("agent_id = ?", agentID)
	}
	if macAddress != "" {
		query = query.Where("mac_address = ?", macAddress)
	}

	err = query.Count(&totalRecordings).Error
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

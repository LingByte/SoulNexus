package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetUsageStatistics gets usage statistics
func (h *Handlers) GetUsageStatistics(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	// 解析查询参数
	var startTime, endTime time.Time
	var credentialID *uint
	var groupID *uint

	startTimeStr := c.Query("startTime")
	endTimeStr := c.Query("endTime")
	credentialIDStr := c.Query("credentialId")
	groupIDStr := c.Query("groupId")

	// 默认时间范围：最近30天
	if startTimeStr == "" {
		startTime = time.Now().AddDate(0, 0, -30)
	} else {
		var err error
		startTime, err = time.Parse("2006-01-02", startTimeStr)
		if err != nil {
			response.Fail(c, "Invalid startTime format. Use YYYY-MM-DD", nil)
			return
		}
	}

	if endTimeStr == "" {
		endTime = time.Now()
	} else {
		var err error
		endTime, err = time.Parse("2006-01-02", endTimeStr)
		if err != nil {
			response.Fail(c, "Invalid endTime format. Use YYYY-MM-DD", nil)
			return
		}
		// 设置为当天的23:59:59
		endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 0, endTime.Location())
	}

	if credentialIDStr != "" {
		id, err := strconv.ParseUint(credentialIDStr, 10, 32)
		if err == nil {
			uid := uint(id)
			credentialID = &uid
		}
	}

	if groupIDStr != "" {
		id, err := strconv.ParseUint(groupIDStr, 10, 32)
		if err == nil {
			uid := uint(id)
			groupID = &uid
			// 验证用户是否有权限访问该组织
			var group models.Group
			if err := h.db.Where("id = ?", uid).First(&group).Error; err != nil {
				response.Fail(c, "organization not found", nil)
				return
			}
			// 检查用户是否是组织成员或创建者
			if group.CreatorID != user.ID {
				var member models.GroupMember
				if err := h.db.Where("group_id = ? AND user_id = ?", uid, user.ID).First(&member).Error; err != nil {
					response.Fail(c, "insufficient permissions", "You are not a member of this organization")
					return
				}
			}
		}
	}

	// Get statistics
	stats, err := models.GetUsageStatistics(h.db, user.ID, startTime, endTime, credentialID, groupID)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, "success", stats)
}

// GetDailyUsageData gets usage data grouped by date
func (h *Handlers) GetDailyUsageData(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	// 解析查询参数
	var startTime, endTime time.Time
	var credentialID *uint

	startTimeStr := c.Query("startTime")
	endTimeStr := c.Query("endTime")
	credentialIDStr := c.Query("credentialId")

	// 默认时间范围：最近30天
	if startTimeStr == "" {
		startTime = time.Now().AddDate(0, 0, -30)
	} else {
		var err error
		startTime, err = time.Parse("2006-01-02", startTimeStr)
		if err != nil {
			response.Fail(c, "Invalid startTime format. Use YYYY-MM-DD", nil)
			return
		}
	}

	if endTimeStr == "" {
		endTime = time.Now()
	} else {
		var err error
		endTime, err = time.Parse("2006-01-02", endTimeStr)
		if err != nil {
			response.Fail(c, "Invalid endTime format. Use YYYY-MM-DD", nil)
			return
		}
		// 设置为当天的23:59:59
		endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 0, endTime.Location())
	}

	if credentialIDStr != "" {
		id, err := strconv.ParseUint(credentialIDStr, 10, 32)
		if err == nil {
			uid := uint(id)
			credentialID = &uid
		}
	}

	var groupID *uint
	if groupIDStr := c.Query("groupId"); groupIDStr != "" {
		if id, err := strconv.ParseUint(groupIDStr, 10, 32); err == nil {
			uid := uint(id)
			groupID = &uid
			// 验证用户是否有权限访问该组织
			var group models.Group
			if err := h.db.Where("id = ?", uid).First(&group).Error; err != nil {
				response.Fail(c, "organization not found", nil)
				return
			}
			// 检查用户是否是组织成员或创建者
			if group.CreatorID != user.ID {
				var member models.GroupMember
				if err := h.db.Where("group_id = ? AND user_id = ?", uid, user.ID).First(&member).Error; err != nil {
					response.Fail(c, "insufficient permissions", "You are not a member of this organization")
					return
				}
			}
		}
	}

	// Get daily usage data
	dailyData, err := models.GetDailyUsageData(h.db, user.ID, startTime, endTime, credentialID, groupID)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, "success", dailyData)
}

// GetUsageRecords gets usage record list
func (h *Handlers) GetUsageRecords(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	// Parse query parameters
	params := make(map[string]interface{})

	// Group ID parameter
	if groupIDStr := c.Query("groupId"); groupIDStr != "" {
		if id, err := strconv.ParseUint(groupIDStr, 10, 32); err == nil {
			uid := uint(id)
			params["groupId"] = uid
			// 验证用户是否有权限访问该组织
			var group models.Group
			if err := h.db.Where("id = ?", uid).First(&group).Error; err != nil {
				response.Fail(c, "organization not found", nil)
				return
			}
			// 检查用户是否是组织成员或创建者
			if group.CreatorID != user.ID {
				var member models.GroupMember
				if err := h.db.Where("group_id = ? AND user_id = ?", uid, user.ID).First(&member).Error; err != nil {
					response.Fail(c, "insufficient permissions", "You are not a member of this organization")
					return
				}
			}
		}
	}

	// Pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	params["page"] = page
	params["size"] = size

	// Credential ID
	if credentialIDStr := c.Query("credentialId"); credentialIDStr != "" {
		if id, err := strconv.ParseUint(credentialIDStr, 10, 32); err == nil {
			params["credentialId"] = uint(id)
		}
	}

	if agentIDStr := c.Query("agentId"); agentIDStr != "" {
		if id, err := strconv.ParseUint(agentIDStr, 10, 32); err == nil {
			params["agentId"] = uint(id)
		}
	}

	// Usage type
	if usageType := c.Query("usageType"); usageType != "" {
		params["usageType"] = models.UsageType(usageType)
	}

	// Time range
	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if t, err := time.Parse("2006-01-02", startTimeStr); err == nil {
			params["startTime"] = t
		}
	}
	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if t, err := time.Parse("2006-01-02", endTimeStr); err == nil {
			// Set to 23:59:59 of the day
			t = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
			params["endTime"] = t
		}
	}

	// Sorting
	if orderBy := c.Query("orderBy"); orderBy != "" {
		params["orderBy"] = orderBy
	}

	// Get records
	records, total, err := models.GetUsageRecords(h.db, user.ID, params)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, "success", gin.H{
		"list":  records,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// ExportUsageRecords exports usage records
func (h *Handlers) ExportUsageRecords(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	// Parse query parameters
	params := make(map[string]interface{})

	// Group ID parameter
	if groupIDStr := c.Query("groupId"); groupIDStr != "" {
		if id, err := strconv.ParseUint(groupIDStr, 10, 32); err == nil {
			uid := uint(id)
			params["groupId"] = uid
			// 验证用户是否有权限访问该组织
			var group models.Group
			if err := h.db.Where("id = ?", uid).First(&group).Error; err != nil {
				response.Fail(c, "organization not found", nil)
				return
			}
			// 检查用户是否是组织成员或创建者
			if group.CreatorID != user.ID {
				var member models.GroupMember
				if err := h.db.Where("group_id = ? AND user_id = ?", uid, user.ID).First(&member).Error; err != nil {
					response.Fail(c, "insufficient permissions", "You are not a member of this organization")
					return
				}
			}
		}
	}

	// Credential ID
	if credentialIDStr := c.Query("credentialId"); credentialIDStr != "" {
		if id, err := strconv.ParseUint(credentialIDStr, 10, 32); err == nil {
			params["credentialId"] = uint(id)
		}
	}

	if agentIDStr := c.Query("agentId"); agentIDStr != "" {
		if id, err := strconv.ParseUint(agentIDStr, 10, 32); err == nil {
			params["agentId"] = uint(id)
		}
	}

	// Usage type
	if usageType := c.Query("usageType"); usageType != "" {
		params["usageType"] = models.UsageType(usageType)
	}

	// Time range
	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if t, err := time.Parse("2006-01-02", startTimeStr); err == nil {
			params["startTime"] = t
		}
	}
	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if t, err := time.Parse("2006-01-02", endTimeStr); err == nil {
			t = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
			params["endTime"] = t
		}
	}

	// Export format
	format := c.DefaultQuery("format", "csv")
	if format != "csv" && format != "excel" {
		format = "csv"
	}

	// Export path
	exportPath := config.GlobalConfig.Features.BackupPath
	if exportPath == "" {
		exportPath = "./exports"
	}

	// Execute export
	var filePath string
	var err error
	if format == "excel" {
		filePath, err = h.exportUsageRecordsToExcel(user.ID, params, exportPath)
	} else {
		filePath, err = h.exportUsageRecords(user.ID, params, format, exportPath)
	}
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		response.Fail(c, "Export file does not exist", nil)
		return
	}

	// Generate download filename (using UTF-8 encoding to avoid Chinese filename issues)
	fileName := filepath.Base(filePath)

	// Set response headers for direct file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", fileName))
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Expires", "0")
	c.Header("Cache-Control", "must-revalidate")
	c.Header("Pragma", "public")

	// 直接发送文件
	c.File(filePath)
}

// GenerateBill generates bill
func (h *Handlers) GenerateBill(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	var req struct {
		CredentialID *uint  `json:"credentialId"`
		GroupID      *uint  `json:"groupId"`
		StartTime    string `json:"startTime" binding:"required"`
		EndTime      string `json:"endTime" binding:"required"`
		Title        string `json:"title"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid request format", err)
		return
	}

	// 验证组织权限（如果提供了组织ID）
	if req.GroupID != nil && *req.GroupID > 0 {
		var group models.Group
		if err := h.db.Where("id = ?", *req.GroupID).First(&group).Error; err != nil {
			response.Fail(c, "organization not found", nil)
			return
		}
		// 检查用户是否是组织成员或创建者
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ?", *req.GroupID, user.ID).First(&member).Error; err != nil {
				response.Fail(c, "insufficient permissions", "You are not a member of this organization")
				return
			}
		}
	}

	// Parse time
	startTime, err := time.Parse("2006-01-02", req.StartTime)
	if err != nil {
		response.Fail(c, "Invalid startTime format. Use YYYY-MM-DD", nil)
		return
	}

	endTime, err := time.Parse("2006-01-02", req.EndTime)
	if err != nil {
		response.Fail(c, "Invalid endTime format. Use YYYY-MM-DD", nil)
		return
	}
	endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 0, endTime.Location())

	// Generate title
	title := req.Title
	if title == "" {
		title = fmt.Sprintf("Bill from %s to %s", startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))
	}

	// Generate bill
	bill, err := models.GenerateBill(h.db, user.ID, req.CredentialID, req.GroupID, startTime, endTime, title)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, "Bill generated successfully", bill)
}

// GetBills gets bill list
func (h *Handlers) GetBills(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	// Parse query parameters
	params := make(map[string]interface{})

	// Group ID parameter
	if groupIDStr := c.Query("groupId"); groupIDStr != "" {
		if id, err := strconv.ParseUint(groupIDStr, 10, 32); err == nil {
			uid := uint(id)
			params["groupId"] = uid
			// 验证用户是否有权限访问该组织
			var group models.Group
			if err := h.db.Where("id = ?", uid).First(&group).Error; err != nil {
				response.Fail(c, "organization not found", nil)
				return
			}
			// 检查用户是否是组织成员或创建者
			if group.CreatorID != user.ID {
				var member models.GroupMember
				if err := h.db.Where("group_id = ? AND user_id = ?", uid, user.ID).First(&member).Error; err != nil {
					response.Fail(c, "insufficient permissions", "You are not a member of this organization")
					return
				}
			}
		}
	}

	// Pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	params["page"] = page
	params["size"] = size

	// Credential ID
	if credentialIDStr := c.Query("credentialId"); credentialIDStr != "" {
		if id, err := strconv.ParseUint(credentialIDStr, 10, 32); err == nil {
			params["credentialId"] = uint(id)
		}
	}

	// Status
	if status := c.Query("status"); status != "" {
		params["status"] = models.BillStatus(status)
	}

	// Time range
	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if t, err := time.Parse("2006-01-02", startTimeStr); err == nil {
			params["startTime"] = t
		}
	}
	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if t, err := time.Parse("2006-01-02", endTimeStr); err == nil {
			params["endTime"] = t
		}
	}

	// Sorting
	if orderBy := c.Query("orderBy"); orderBy != "" {
		params["orderBy"] = orderBy
	}

	// Get bill list
	bills, total, err := models.GetBills(h.db, user.ID, params)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, "success", gin.H{
		"list":  bills,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// GetBill gets a single bill
func (h *Handlers) GetBill(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	billIDStr := c.Param("id")
	billID, err := strconv.ParseUint(billIDStr, 10, 32)
	if err != nil {
		response.Fail(c, "Invalid bill ID", nil)
		return
	}

	bill, err := models.GetBill(h.db, user.ID, uint(billID))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusNotFound, err)
		return
	}

	response.Success(c, "success", bill)
}

// ExportBill exports bill
func (h *Handlers) ExportBill(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	billIDStr := c.Param("id")
	billID, err := strconv.ParseUint(billIDStr, 10, 32)
	if err != nil {
		response.Fail(c, "Invalid bill ID", nil)
		return
	}

	// 获取账单
	bill, err := models.GetBill(h.db, user.ID, uint(billID))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusNotFound, err)
		return
	}

	// Export format
	format := c.DefaultQuery("format", "csv")
	if format != "csv" && format != "excel" {
		format = "csv"
	}

	// Export path
	exportPath := config.GlobalConfig.Features.BackupPath
	if exportPath == "" {
		exportPath = "./exports"
	}

	// Execute export
	var filePath string
	if format == "excel" {
		filePath, err = h.exportBillToExcel(bill, exportPath)
	} else {
		filePath, err = h.exportBill(bill, format, exportPath)
	}
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		response.Fail(c, "导出文件不存在", nil)
		return
	}

	// 更新账单状态
	bill.Status = models.BillStatusExported
	now := time.Now()
	bill.ExportedAt = &now
	bill.ExportFormat = format
	bill.ExportPath = filePath
	models.UpdateBill(h.db, bill)

	// 生成下载文件名（使用UTF-8编码，避免中文文件名问题）
	fileName := filepath.Base(filePath)

	// 设置响应头，直接下载文件
	contentType := "text/csv; charset=utf-8"
	if format == "excel" {
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", fileName))
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Expires", "0")
	c.Header("Cache-Control", "must-revalidate")
	c.Header("Pragma", "public")

	// 直接发送文件
	c.File(filePath)
}

// exportUsageRecordsToExcel 导出使用量记录为Excel（内部方法）
// 注意：当前实现使用CSV格式，Excel可以打开CSV文件
// 如需真正的Excel格式，可以添加github.com/xuri/excelize/v2库
func (h *Handlers) exportUsageRecordsToExcel(userID uint, params map[string]interface{}, exportPath string) (string, error) {
	// 当前实现：使用CSV格式（Excel可以打开）
	// 未来可以添加excelize库实现真正的Excel格式
	return h.exportUsageRecords(userID, params, "csv", exportPath)
}

// exportBillToExcel 导出账单为Excel（内部方法）
// 注意：当前实现使用CSV格式，Excel可以打开CSV文件
// 如需真正的Excel格式，可以添加github.com/xuri/excelize/v2库
func (h *Handlers) exportBillToExcel(bill *models.Bill, exportPath string) (string, error) {
	// 获取账单对应的使用量记录
	params := map[string]interface{}{
		"startTime": bill.StartTime,
		"endTime":   bill.EndTime,
	}
	if bill.CredentialID != nil {
		params["credentialId"] = *bill.CredentialID
	}

	records, _, err := models.GetUsageRecords(h.db, bill.UserID, params)
	if err != nil {
		return "", fmt.Errorf("failed to get usage records: %w", err)
	}

	// 确保导出目录存在
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}

	// 生成文件名（使用.xlsx扩展名，但实际是CSV格式）
	fileName := fmt.Sprintf("bill_%s_%s.xlsx", bill.BillNo, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(exportPath, fileName)

	// 当前实现：使用CSV格式（Excel可以打开）
	// 未来可以添加excelize库实现真正的Excel格式
	return h.exportBillToCSV(bill, records, filePath)
}

// exportUsageRecords 导出使用量记录（内部方法）
func (h *Handlers) exportUsageRecords(userID uint, params map[string]interface{}, format, exportPath string) (string, error) {
	records, _, err := models.GetUsageRecords(h.db, userID, params)
	if err != nil {
		return "", fmt.Errorf("failed to get usage records: %w", err)
	}

	// 确保导出目录存在
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}

	// 生成文件名
	fileName := fmt.Sprintf("usage_records_%s_%d.%s",
		time.Now().Format("20060102_150405"), userID, format)
	filePath := filepath.Join(exportPath, fileName)

	// 导出为CSV
	return h.exportRecordsToCSV(records, filePath)
}

// exportBill 导出账单（内部方法）
func (h *Handlers) exportBill(bill *models.Bill, format, exportPath string) (string, error) {
	// 获取账单对应的使用量记录
	params := map[string]interface{}{
		"startTime": bill.StartTime,
		"endTime":   bill.EndTime,
	}
	if bill.CredentialID != nil {
		params["credentialId"] = *bill.CredentialID
	}

	records, _, err := models.GetUsageRecords(h.db, bill.UserID, params)
	if err != nil {
		return "", fmt.Errorf("failed to get usage records: %w", err)
	}

	// 确保导出目录存在
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}

	// 生成文件名
	fileName := fmt.Sprintf("bill_%s_%s.%s", bill.BillNo, time.Now().Format("20060102_150405"), format)
	filePath := filepath.Join(exportPath, fileName)

	// 导出为CSV
	return h.exportBillToCSV(bill, records, filePath)
}

// exportRecordsToCSV 导出记录为CSV
func (h *Handlers) exportRecordsToCSV(records []models.UsageRecord, filePath string) (string, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	headers := []string{
		"ID", "用户ID", "凭证ID", "助手ID", "会话ID", "使用类型",
		"模型", "Prompt Tokens", "Completion Tokens", "Total Tokens",
		"通话时长(秒)", "通话次数", "音频时长(秒)", "音频大小(字节)",
		"API调用次数", "使用时间", "创建时间",
	}
	if err := writer.Write(headers); err != nil {
		return "", fmt.Errorf("failed to write headers: %w", err)
	}

	// 写入数据
	for _, record := range records {
		row := []string{
			fmt.Sprintf("%d", record.ID),
			fmt.Sprintf("%d", record.UserID),
			fmt.Sprintf("%d", record.CredentialID),
			h.formatUintPtr(record.AgentID),
			record.SessionID,
			string(record.UsageType),
			record.Model,
			fmt.Sprintf("%d", record.PromptTokens),
			fmt.Sprintf("%d", record.CompletionTokens),
			fmt.Sprintf("%d", record.TotalTokens),
			fmt.Sprintf("%d", record.CallDuration),
			fmt.Sprintf("%d", record.CallCount),
			fmt.Sprintf("%d", record.AudioDuration),
			fmt.Sprintf("%d", record.AudioSize),
			fmt.Sprintf("%d", record.APICallCount),
			record.UsageTime.Format("2006-01-02 15:04:05"),
			record.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write row: %w", err)
		}
	}

	return filePath, nil
}

// exportBillToCSV 导出账单为CSV
func (h *Handlers) exportBillToCSV(bill *models.Bill, records []models.UsageRecord, filePath string) (string, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入账单摘要
	summary := [][]string{
		{"账单信息"},
		{"账单编号", bill.BillNo},
		{"账单标题", bill.Title},
		{"状态", string(bill.Status)},
		{"开始时间", bill.StartTime.Format("2006-01-02 15:04:05")},
		{"结束时间", bill.EndTime.Format("2006-01-02 15:04:05")},
		{""},
		{"使用量统计"},
		{"LLM调用次数", fmt.Sprintf("%d", bill.TotalLLMCalls)},
		{"LLM总Token数", fmt.Sprintf("%d", bill.TotalLLMTokens)},
		{"Prompt Tokens", fmt.Sprintf("%d", bill.TotalPromptTokens)},
		{"Completion Tokens", fmt.Sprintf("%d", bill.TotalCompletionTokens)},
		{"总通话时长(秒)", fmt.Sprintf("%d", bill.TotalCallDuration)},
		{"总通话次数", fmt.Sprintf("%d", bill.TotalCallCount)},
		{"ASR总时长(秒)", fmt.Sprintf("%d", bill.TotalASRDuration)},
		{"ASR调用次数", fmt.Sprintf("%d", bill.TotalASRCount)},
		{"TTS总时长(秒)", fmt.Sprintf("%d", bill.TotalTTSDuration)},
		{"TTS调用次数", fmt.Sprintf("%d", bill.TotalTTSCount)},
		{"总API调用次数", fmt.Sprintf("%d", bill.TotalAPICalls)},
		{""},
		{"详细记录"},
	}

	for _, row := range summary {
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write summary: %w", err)
		}
	}

	// 写入表头
	headers := []string{
		"ID", "使用类型", "模型", "Prompt Tokens", "Completion Tokens", "Total Tokens",
		"通话时长(秒)", "通话次数", "音频时长(秒)", "使用时间",
	}
	if err := writer.Write(headers); err != nil {
		return "", fmt.Errorf("failed to write headers: %w", err)
	}

	// 写入详细记录
	for _, record := range records {
		row := []string{
			fmt.Sprintf("%d", record.ID),
			string(record.UsageType),
			record.Model,
			fmt.Sprintf("%d", record.PromptTokens),
			fmt.Sprintf("%d", record.CompletionTokens),
			fmt.Sprintf("%d", record.TotalTokens),
			fmt.Sprintf("%d", record.CallDuration),
			fmt.Sprintf("%d", record.CallCount),
			fmt.Sprintf("%d", record.AudioDuration),
			record.UsageTime.Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write row: %w", err)
		}
	}

	return filePath, nil
}

// formatUintPtr 格式化uint指针
func (h *Handlers) formatUintPtr(ptr *uint) string {
	if ptr == nil {
		return ""
	}
	return fmt.Sprintf("%d", *ptr)
}

// UpdateBill 更新账单
func (h *Handlers) UpdateBill(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	billIDStr := c.Param("id")
	billID, err := strconv.ParseUint(billIDStr, 10, 32)
	if err != nil {
		response.Fail(c, "Invalid bill ID", nil)
		return
	}

	// 获取账单
	bill, err := models.GetBill(h.db, user.ID, uint(billID))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusNotFound, err)
		return
	}

	var req struct {
		Title string `json:"title"`
		Notes string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid request format", err)
		return
	}

	// 更新字段
	if req.Title != "" {
		bill.Title = req.Title
	}
	if req.Notes != "" {
		bill.Notes = req.Notes
	}

	if err := models.UpdateBill(h.db, bill); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, "Bill updated successfully", bill)
}

// DeleteBill 删除账单
func (h *Handlers) DeleteBill(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	billIDStr := c.Param("id")
	billID, err := strconv.ParseUint(billIDStr, 10, 32)
	if err != nil {
		response.Fail(c, "Invalid bill ID", nil)
		return
	}

	// 获取账单
	bill, err := models.GetBill(h.db, user.ID, uint(billID))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusNotFound, err)
		return
	}

	// 删除账单
	if err := h.db.Delete(bill).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, "Bill deleted successfully", nil)
}

// ArchiveBill 归档账单
func (h *Handlers) ArchiveBill(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	billIDStr := c.Param("id")
	billID, err := strconv.ParseUint(billIDStr, 10, 32)
	if err != nil {
		response.Fail(c, "Invalid bill ID", nil)
		return
	}

	// 获取账单
	bill, err := models.GetBill(h.db, user.ID, uint(billID))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusNotFound, err)
		return
	}

	// 更新状态为已归档
	bill.Status = models.BillStatusArchived
	if err := models.UpdateBill(h.db, bill); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, "Bill archived successfully", bill)
}

// UpdateBillNotes 更新账单备注
func (h *Handlers) UpdateBillNotes(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	billIDStr := c.Param("id")
	billID, err := strconv.ParseUint(billIDStr, 10, 32)
	if err != nil {
		response.Fail(c, "Invalid bill ID", nil)
		return
	}

	// 获取账单
	bill, err := models.GetBill(h.db, user.ID, uint(billID))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusNotFound, err)
		return
	}

	var req struct {
		Notes string `json:"notes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid request format", err)
		return
	}

	// 更新备注
	bill.Notes = req.Notes
	if err := models.UpdateBill(h.db, bill); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}

	response.Success(c, "Bill notes updated successfully", bill)
}

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/notification/mail"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// notificationChannelOrgID 当前实现的通知渠道为系统级（OrgID=0）。
// 未来如需多租户，可改为根据用户当前组织上下文返回。
func notificationChannelOrgID(_ *gin.Context) uint { return 0 }

// NotificationChannelUpsertReq 创建/更新通知渠道（email|sms）。
type NotificationChannelUpsertReq struct {
	ChannelType      string `json:"channelType" binding:"required,oneof=email sms"`
	Name             string `json:"name" binding:"required,max=128"`
	SortOrder        int    `json:"sortOrder"`
	Enabled          *bool  `json:"enabled"`
	Remark           string `json:"remark" binding:"max=255"`
	Driver           string `json:"driver"` // for email: smtp|sendcloud
	SMTPHost         string `json:"smtpHost"`
	SMTPPort         int64  `json:"smtpPort"`
	SMTPUsername     string `json:"smtpUsername"`
	SMTPPassword     string `json:"smtpPassword"`
	SMTPFrom         string `json:"smtpFrom"`
	SendcloudAPIUser string `json:"sendcloudApiUser"`
	SendcloudAPIKey  string `json:"sendcloudApiKey"`
	SendcloudFrom    string `json:"sendcloudFrom"`
	FromDisplayName  string `json:"fromDisplayName"`
	SMSProvider      string `json:"smsProvider"`
	SMSConfig        any    `json:"smsConfig"`
}

func (h *Handlers) handleListNotificationChannels(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	t := strings.TrimSpace(c.Query("type"))
	orgID := notificationChannelOrgID(c)
	scope := h.db.Where("org_id = ?", orgID)
	out, err := models.ListNotificationChannels(scope, t, page, pageSize)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", gin.H{
		"list":      out.List,
		"total":     out.Total,
		"page":      out.Page,
		"pageSize":  out.PageSize,
		"size":      out.PageSize,
		"totalPage": out.TotalPage,
	})
}

func (h *Handlers) handleGetNotificationChannel(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.AbortWithStatus(c, http.StatusBadRequest)
		return
	}
	orgID := notificationChannelOrgID(c)
	row, err := models.GetNotificationChannel(h.db.Where("org_id = ?", orgID), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.AbortWithStatus(c, http.StatusNotFound)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	out := gin.H{"channel": row}
	if row.Type == models.NotificationChannelTypeEmail && strings.TrimSpace(row.ConfigJSON) != "" {
		if vf, err := models.DecodeEmailChannelForm(row.ConfigJSON); err == nil {
			out["emailForm"] = vf
		}
	}
	if row.Type == models.NotificationChannelTypeSMS && strings.TrimSpace(row.ConfigJSON) != "" {
		if vf, err := models.DecodeSMSChannelForm(row.ConfigJSON); err == nil {
			out["smsForm"] = vf
		}
	}
	response.Success(c, "success", out)
}

func (h *Handlers) handleCreateNotificationChannel(c *gin.Context) {
	var req NotificationChannelUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	cfgJSON, err := buildChannelConfig(req)
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	user := models.CurrentUser(c)
	orgID := notificationChannelOrgID(c)
	channelType := strings.ToLower(strings.TrimSpace(req.ChannelType))
	row := models.NotificationChannel{
		OrgID:      orgID,
		Type:       channelType,
		Code:       fmt.Sprintf("%s-%d", strings.ToUpper(channelType[:1]), utils.SnowflakeUtil.NextID()),
		Name:       strings.TrimSpace(req.Name),
		SortOrder:  req.SortOrder,
		Enabled:    true,
		Remark:     strings.TrimSpace(req.Remark),
		ConfigJSON: cfgJSON,
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if user != nil {
		row.SetCreateInfo(user.Email)
	}
	if err := h.db.Create(&row).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "创建成功", row)
}

func (h *Handlers) handleUpdateNotificationChannel(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.AbortWithStatus(c, http.StatusBadRequest)
		return
	}
	var req NotificationChannelUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	orgID := notificationChannelOrgID(c)
	var row models.NotificationChannel
	if err := h.db.Where("org_id = ?", orgID).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.AbortWithStatus(c, http.StatusNotFound)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	channelType := strings.ToLower(strings.TrimSpace(req.ChannelType))
	if channelType != strings.ToLower(strings.TrimSpace(row.Type)) {
		response.FailWithCode(c, http.StatusBadRequest, "channelType 不匹配", nil)
		return
	}
	cfgJSON, err := buildChannelConfig(req)
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	switch channelType {
	case models.NotificationChannelTypeEmail:
		if merged, err := models.MergeEmailSecretsOnUpdate(row.ConfigJSON, cfgJSON); err == nil {
			row.ConfigJSON = merged
		} else {
			row.ConfigJSON = cfgJSON
		}
	case models.NotificationChannelTypeSMS:
		if merged, err := models.MergeSMSSecretsOnUpdate(row.ConfigJSON, cfgJSON); err == nil {
			row.ConfigJSON = merged
		} else {
			row.ConfigJSON = cfgJSON
		}
	}
	row.Name = strings.TrimSpace(req.Name)
	row.SortOrder = req.SortOrder
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	row.Remark = strings.TrimSpace(req.Remark)
	if user := models.CurrentUser(c); user != nil {
		row.SetUpdateInfo(user.Email)
	}
	if err := h.db.Save(&row).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "更新成功", row)
}

func (h *Handlers) handleDeleteNotificationChannel(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.AbortWithStatus(c, http.StatusBadRequest)
		return
	}
	orgID := notificationChannelOrgID(c)
	res := h.db.Where("org_id = ?", orgID).Delete(&models.NotificationChannel{}, id)
	if res.Error != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		response.AbortWithStatus(c, http.StatusNotFound)
		return
	}
	response.Success(c, "删除成功", gin.H{"id": id})
}

// buildChannelConfig 把请求字段转换为 config_json。
func buildChannelConfig(req NotificationChannelUpsertReq) (string, error) {
	switch strings.ToLower(strings.TrimSpace(req.ChannelType)) {
	case models.NotificationChannelTypeEmail:
		switch strings.ToLower(strings.TrimSpace(req.Driver)) {
		case mail.ProviderSMTP:
			return models.BuildEmailChannelConfigJSON(
				mail.ProviderSMTP, req.Name,
				req.SMTPHost, req.SMTPPort, req.SMTPUsername, req.SMTPPassword, req.SMTPFrom, req.FromDisplayName,
				"", "", "",
			)
		case mail.ProviderSendCloud:
			return models.BuildEmailChannelConfigJSON(
				mail.ProviderSendCloud, req.Name,
				"", 0, "", "", "", req.FromDisplayName,
				req.SendcloudAPIUser, req.SendcloudAPIKey, req.SendcloudFrom,
			)
		default:
			return "", fmt.Errorf("未知邮件驱动: %q（仅支持 smtp / sendcloud）", req.Driver)
		}
	case models.NotificationChannelTypeSMS:
		return models.BuildSMSChannelConfigJSON(req.SMSProvider, req.SMSConfig)
	default:
		return "", errors.New("未知 channelType")
	}
}

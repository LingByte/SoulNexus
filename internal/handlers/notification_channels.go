package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	dto "github.com/LingByte/SoulNexus/internal/request"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) registerPlatformNotificationAdminRoutes(r *humax.Group) {
	admin := r.Group("")
	admin.Use(middleware.RequirePlatformAdmin())
	notifChannels := admin.Group("admin/notification-channels")
	{
		notifChannels.GET("", h.handleListNotificationChannels)
		notifChannels.POST("", h.handleCreateNotificationChannel)
		notifChannels.GET("/:id", h.handleGetNotificationChannel)
		notifChannels.PUT("/:id", h.handleUpdateNotificationChannel)
		notifChannels.DELETE("/:id", h.handleDeleteNotificationChannel)
	}
}

func (h *Handlers) handleListNotificationChannels(c *gin.Context) {
	page, pageSize := ginutil.QueryPage(c, 200)
	t := strings.TrimSpace(c.Query("type"))
	out, err := models.ListNotificationChannels(h.db, t, page, pageSize)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	ginutil.PageSuccess(c, out.List, out.Total, out.Page, out.PageSize)
}

func (h *Handlers) handleGetNotificationChannel(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetNotificationChannel(h.db, id)
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
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

func (h *Handlers) handleCreateNotificationChannel(c *gin.Context) {
	var req dto.NotificationChannelUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	cfgJSON, err := dto.BuildChannelConfig(req)
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	channelType := strings.ToLower(strings.TrimSpace(req.ChannelType))
	row := models.NotificationChannel{
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
	row.SetCreateInfo(middleware.AuditOperator(c))
	if err := h.db.Create(&row).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeyCreated, row)
}

func (h *Handlers) handleUpdateNotificationChannel(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var req dto.NotificationChannelUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	var row models.NotificationChannel
	if err := h.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
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
	cfgJSON, err := dto.BuildChannelConfigForUpdate(req, row.ConfigJSON)
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
	row.SetUpdateInfo(middleware.AuditOperator(c))
	if err := h.db.Save(&row).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeyUpdated, row)
}

func (h *Handlers) handleDeleteNotificationChannel(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	res := h.db.Delete(&models.NotificationChannel{}, id)
	if res.Error != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		response.AbortWithStatus(c, http.StatusNotFound)
		return
	}
	response.SuccessI18n(c, i18n.KeyDeleted, gin.H{"id": id})
}

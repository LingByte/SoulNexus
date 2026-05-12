// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"errors"
	"net/http"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MailTemplateCreateReq struct {
	Code        string `json:"code" binding:"required,max=64"`
	Name        string `json:"name" binding:"required,max=128"`
	Subject     string `json:"subject" binding:"max=255"`
	HTMLBody    string `json:"htmlBody" binding:"required"`
	Description string `json:"description" binding:"max=512"`
	Variables   string `json:"variables"`
	Locale      string `json:"locale" binding:"max=32"`
	Enabled     *bool  `json:"enabled"`
}

type MailTemplateUpdateReq struct {
	Name        string `json:"name" binding:"required,max=128"`
	Subject     string `json:"subject" binding:"max=255"`
	HTMLBody    string `json:"htmlBody" binding:"required"`
	Description string `json:"description" binding:"max=512"`
	Variables   string `json:"variables"`
	Locale      string `json:"locale" binding:"max=32"`
	Enabled     *bool  `json:"enabled"`
}

func (h *Handlers) handleListMailTemplates(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	offset := (page - 1) * pageSize

	orgID := notificationChannelOrgID(c)
	total, err := models.CountMailTemplatesByOrg(h.db, orgID)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	list, err := models.ListMailTemplatesByOrg(h.db, orgID, offset, pageSize)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	totalPage := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPage++
	}
	response.Success(c, "success", gin.H{
		"list": list, "total": total, "page": page, "pageSize": pageSize, "size": pageSize, "totalPage": totalPage,
	})
}

func (h *Handlers) handleGetMailTemplate(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.AbortWithStatus(c, http.StatusBadRequest)
		return
	}
	orgID := notificationChannelOrgID(c)
	tpl, err := models.GetMailTemplateByOrgAndID(h.db, orgID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.AbortWithStatus(c, http.StatusNotFound)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", tpl)
}

func (h *Handlers) handleCreateMailTemplate(c *gin.Context) {
	var req MailTemplateCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	user := models.CurrentUser(c)
	orgID := notificationChannelOrgID(c)
	tpl := models.MailTemplate{
		OrgID:       orgID,
		Code:        req.Code,
		Name:        req.Name,
		Subject:     req.Subject,
		Description: req.Description,
		Locale:      req.Locale,
		Enabled:     true,
	}
	models.ApplyMailTemplateHTMLDerivedFields(&tpl, req.HTMLBody, req.Variables)
	if req.Enabled != nil {
		tpl.Enabled = *req.Enabled
	}
	if user != nil {
		tpl.SetCreateInfo(user.Email)
	}
	if err := models.CreateMailTemplate(h.db, &tpl); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "创建成功", tpl)
}

func (h *Handlers) handleUpdateMailTemplate(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.AbortWithStatus(c, http.StatusBadRequest)
		return
	}
	var req MailTemplateUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	orgID := notificationChannelOrgID(c)
	tpl, err := models.GetMailTemplateByOrgAndID(h.db, orgID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.AbortWithStatus(c, http.StatusNotFound)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	tpl.Name = req.Name
	tpl.Subject = req.Subject
	tpl.Description = req.Description
	tpl.Locale = req.Locale
	models.ApplyMailTemplateHTMLDerivedFields(tpl, req.HTMLBody, req.Variables)
	if req.Enabled != nil {
		tpl.Enabled = *req.Enabled
	}
	if user := models.CurrentUser(c); user != nil {
		tpl.SetUpdateInfo(user.Email)
	}
	if err := models.SaveMailTemplate(h.db, tpl); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "更新成功", tpl)
}

func (h *Handlers) handleDeleteMailTemplate(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.AbortWithStatus(c, http.StatusBadRequest)
		return
	}
	orgID := notificationChannelOrgID(c)
	n, err := models.DeleteMailTemplateByOrgAndID(h.db, orgID, id)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if n == 0 {
		response.AbortWithStatus(c, http.StatusNotFound)
		return
	}
	response.Success(c, "删除成功", gin.H{"id": id})
}

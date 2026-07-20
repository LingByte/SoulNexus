package handlers

import (
	"errors"
	"net/http"

	"github.com/LingByte/SoulNexus/internal/models"
	dto "github.com/LingByte/SoulNexus/internal/request"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) registerPlatformEmailTemplateAdminRoutes(r *humax.Group) {
	admin := r.Group("")
	admin.Use(middleware.RequirePlatformAdmin())
	mailTemplates := admin.Group("admin/mail-templates")
	{
		mailTemplates.GET("", h.handleListMailTemplates)
		mailTemplates.POST("", h.handleCreateMailTemplate)
		mailTemplates.GET("/:id", h.handleGetMailTemplate)
		mailTemplates.PUT("/:id", h.handleUpdateMailTemplate)
		mailTemplates.DELETE("/:id", h.handleDeleteMailTemplate)
	}
}

// handleListMailTemplates 获取邮件模板分页列表
// GET /api/mail-templates
// Query: page（页码，默认 1），pageSize（每页数量，默认 200）
// 响应：分页包装的模板列表，包含 total、page、pageSize
func (h *Handlers) handleListMailTemplates(c *gin.Context) {
	page, pageSize := ginutil.QueryPage(c, 200)
	list, total, err := models.ListMailTemplatesPage(h.db, page, pageSize)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	ginutil.PageSuccess(c, list, total, page, pageSize)
}

// handleGetMailTemplate 根据 ID 获取单个邮件模板详情
// GET /api/mail-templates/:id
// 响应：MailTemplate 完整对象；模板不存在返回 404
func (h *Handlers) handleGetMailTemplate(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	tpl, err := models.GetMailTemplateByID(h.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.AbortWithStatus(c, http.StatusNotFound)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, tpl)
}

// handleCreateMailTemplate 创建新的邮件模板
// POST /api/mail-templates
// 请求体：MailTemplateCreateReq（code, name, subject, description, locale, html_body, variables, enabled）
// 响应：创建成功的 MailTemplate 对象
func (h *Handlers) handleCreateMailTemplate(c *gin.Context) {
	var req dto.MailTemplateCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	tpl := models.MailTemplate{
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
	tpl.SetCreateInfo(middleware.AuditOperator(c))
	if err := models.CreateMailTemplate(h.db, &tpl); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeyCreated, tpl)
}

// handleUpdateMailTemplate 更新指定邮件模板
// PUT /api/mail-templates/:id
// 请求体：MailTemplateUpdateReq（name, subject, description, locale, html_body, variables, enabled）
// 响应：更新后的 MailTemplate 对象；模板不存在返回 404
func (h *Handlers) handleUpdateMailTemplate(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var req dto.MailTemplateUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	tpl, err := models.GetMailTemplateByID(h.db, id)
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
	tpl.SetUpdateInfo(middleware.AuditOperator(c))
	if err := models.SaveMailTemplate(h.db, tpl); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeyUpdated, tpl)
}

// handleDeleteMailTemplate 删除指定邮件模板
// DELETE /api/mail-templates/:id
// 响应：删除结果（id）；模板不存在返回 404
func (h *Handlers) handleDeleteMailTemplate(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	n, err := models.DeleteMailTemplateByID(h.db, id)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if n == 0 {
		response.AbortWithStatus(c, http.StatusNotFound)
		return
	}
	response.SuccessI18n(c, i18n.KeyDeleted, gin.H{"id": id})
}

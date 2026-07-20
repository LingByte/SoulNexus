package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
)

const maxJSTemplateAvatarBytes = 2 << 20

type jsTemplateWriteReq struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatarUrl"`
	Content   string `json:"content"`
	Usage     string `json:"usage"`
	Status    string `json:"status"`
}

func (h *Handlers) registerJSTemplateRoutes(r *humax.Group) {
	read := r.Group("js-templates")
	read.Use(middleware.RequireTenantPermissionAny("api.assistants.read", "menu.res.assistant"))
	{
		read.GET("", h.listJSTemplates)
		read.GET("/usage", h.listJSTemplateUsage)
		read.GET("/:id", h.getJSTemplate)
	}
	write := r.Group("js-templates")
	write.Use(middleware.RequireTenantPermissionAny("api.assistants.write", "menu.res.assistant"))
	{
		write.POST("", h.createJSTemplate)
		write.PUT("/:id", h.updateJSTemplate)
		write.POST("/:id/avatar", h.uploadJSTemplateAvatar)
		write.DELETE("/:id", h.deleteJSTemplate)
	}
}

func (h *Handlers) listJSTemplates(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	page, size := ginutil.QueryPage(c, 50)
	list, total, err := models.ListJSTemplatesPage(h.db, tid, page, size, c.Query("name"))
	if ginutil.WriteInternalError(c, err) {
		return
	}
	ginutil.PageSuccess(c, list, total, page, size)
}

func (h *Handlers) listJSTemplateUsage(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	page, size := ginutil.QueryPage(c, 20)
	list, total, err := models.ListJSTemplateUsagePage(h.db, tid, c.Query("jsSourceId"), page, size)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	ginutil.PageSuccess(c, list, total, page, size)
}

func (h *Handlers) getJSTemplate(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	tid := middleware.CurrentTenantID(c)
	row, err := models.GetJSTemplateByIDForTenant(h.db, id, tid)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func (h *Handlers) createJSTemplate(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	var req jsTemplateWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	content := strings.TrimSpace(req.Content)
	if name == "" || content == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyNameRequired))
		return
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = models.JSTemplateStatusActive
	}
	row := models.JSTemplate{
		TenantID:  tid,
		Name:      name,
		AvatarURL: strings.TrimSpace(req.AvatarURL),
		Content:   content,
		Usage:     strings.TrimSpace(req.Usage),
		Status:    status,
	}
	row.SetCreateInfo(middleware.AuditOperator(c))
	if ginutil.WriteInternalError(c, h.db.Create(&row).Error) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func (h *Handlers) updateJSTemplate(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	tid := middleware.CurrentTenantID(c)
	row, err := models.GetJSTemplateByIDForTenant(h.db, id, tid)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	var req jsTemplateWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	updates := map[string]any{}
	if name := strings.TrimSpace(req.Name); name != "" {
		updates["name"] = name
	}
	if content := strings.TrimSpace(req.Content); content != "" {
		updates["content"] = content
	}
	updates["usage"] = strings.TrimSpace(req.Usage)
	updates["avatar_url"] = strings.TrimSpace(req.AvatarURL)
	if status := strings.TrimSpace(req.Status); status != "" {
		updates["status"] = status
	}
	meta := row
	meta.SetUpdateInfo(middleware.AuditOperator(c))
	if meta.UpdateBy != "" {
		updates["update_by"] = meta.UpdateBy
	}
	if ginutil.WriteInternalError(c, h.db.Model(&row).Updates(updates).Error) {
		return
	}
	after, _ := models.GetJSTemplateByIDForTenant(h.db, id, tid)
	response.SuccessI18n(c, i18n.KeySuccess, after)
}

func (h *Handlers) uploadJSTemplateAvatar(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	tid := middleware.CurrentTenantID(c)
	row, err := models.GetJSTemplateByIDForTenant(h.db, id, tid)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeySelectImageFile))
		return
	}
	src, err := fh.Open()
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyCannotReadFile))
		return
	}
	defer src.Close()
	body, err := io.ReadAll(io.LimitReader(src, maxJSTemplateAvatarBytes+1))
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	if len(body) > maxJSTemplateAvatarBytes {
		response.Render(c, response.NewI18n(response.CodeValidation, i18n.KeyImageTooLarge))
		return
	}
	ct := http.DetectContentType(body)
	ext := utils.PickImageExtFromContentType(ct)
	if ext == "" {
		response.Render(c, response.NewI18n(response.CodeValidation, i18n.KeyImageFormatInvalid))
		return
	}
	key := path.Join(
		"avatars",
		"t"+strconv.FormatUint(uint64(row.TenantID), 10),
		fmt.Sprintf("js_template_%d_%d%s", row.ID, time.Now().UnixMilli(), ext),
	)
	st := stores.Default()
	if strings.TrimSpace(row.AvatarURL) != "" {
		deleteStoredAvatar(st, row.AvatarURL)
	}
	if err := st.Write(key, bytes.NewReader(body)); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	url := ginutil.UploadURL(c, key)
	if err := h.db.Model(&row).Update("avatar_url", url).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	after, _ := models.GetJSTemplateByIDForTenant(h.db, id, tid)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"avatarUrl": url,
		"template":  after,
	})
}

func (h *Handlers) deleteJSTemplate(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	tid := middleware.CurrentTenantID(c)
	if _, err := models.GetJSTemplateByIDForTenant(h.db, id, tid); ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	if _, err := models.SoftDeleteJSTemplateByIDForTenant(h.db, id, tid, middleware.AuditOperator(c)); ginutil.WriteInternalError(c, err) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id})
}

package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"github.com/LingByte/SoulNexus/pkg/i18n"
)

type tenantWebhookCreateReq struct {
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Secret  string   `json:"secret"`
	Events  []string `json:"events"`
	Enabled *bool    `json:"enabled"`
}

type tenantWebhookUpdateReq struct {
	Name    *string  `json:"name"`
	URL     *string  `json:"url"`
	Secret  *string  `json:"secret"`
	Events  []string `json:"events"`
	Enabled *bool    `json:"enabled"`
}

func parseWebhookEvents(events []string) ([]string, error) {
	if len(events) == 0 {
		return append([]string(nil), constants.AllWebhookEvents...), nil
	}
	out := make([]string, 0, len(events))
	seen := map[string]struct{}{}
	for _, e := range events {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if !notification.ValidWebhookEvent(e) {
			return nil, fmt.Errorf("unsupported event: %s", e)
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("events required")
	}
	return out, nil
}

func (h *Handlers) listTenantWebhooks(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	page, size := ginutil.QueryPage(c, 50)
	list, total, err := models.ListTenantWebhooksPage(h.db, tid, page, size)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	ginutil.PageSuccess(c, list, total, page, size)
}

func (h *Handlers) createTenantWebhook(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	var req tenantWebhookCreateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	url := strings.TrimSpace(req.URL)
	if name == "" || url == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyNameURLRequired))
		return
	}
	if err := utils.ValidateURLForSSRF(url); err != nil {
		response.Render(c, response.WrapErr(response.CodeBadRequest, err))
		return
	}
	events, err := parseWebhookEvents(req.Events)
	if err != nil {
		response.Render(c, response.WrapErr(response.CodeBadRequest, err))
		return
	}
	evJSON, _ := json.Marshal(events)
	row := models.TenantWebhook{
		TenantID: tid,
		Name:     name,
		URL:      url,
		Secret:   strings.TrimSpace(req.Secret),
		Events:   datatypes.JSON(evJSON),
		Enabled:  true,
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if op := middleware.AuditOperator(c); op != "" {
		row.SetCreateInfo(op)
	}
	if err := h.db.Create(&row).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionCreate,
		Resource: "tenant_webhook", ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Created webhook %s", row.Name), Detail: audit.Redact(req),
	}, nil, row)
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func (h *Handlers) updateTenantWebhook(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var row models.TenantWebhook
	if err := h.db.Where("id = ? AND tenant_id = ?", id, tid).First(&row).Error; err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyWebhookNotFound))
		return
	}
	before := row
	var req tenantWebhookUpdateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	u := map[string]any{}
	if req.Name != nil {
		if n := strings.TrimSpace(*req.Name); n != "" {
			u["name"] = n
		}
	}
	if req.URL != nil {
		if u2 := strings.TrimSpace(*req.URL); u2 != "" {
			if err := utils.ValidateURLForSSRF(u2); err != nil {
				response.Render(c, response.WrapErr(response.CodeBadRequest, err))
				return
			}
			u["url"] = u2
		}
	}
	if req.Secret != nil {
		u["secret"] = strings.TrimSpace(*req.Secret)
	}
	if req.Enabled != nil {
		u["enabled"] = *req.Enabled
	}
	if req.Events != nil {
		events, err := parseWebhookEvents(req.Events)
		if err != nil {
			response.Render(c, response.WrapErr(response.CodeBadRequest, err))
			return
		}
		evJSON, _ := json.Marshal(events)
		u["events"] = datatypes.JSON(evJSON)
	}
	if len(u) == 0 {
		response.SuccessI18n(c, i18n.KeySuccess, row)
		return
	}
	if op := middleware.AuditOperator(c); op != "" {
		u["update_by"] = op
	}
	if err := h.db.Model(&row).Updates(u).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	_ = h.db.Where("id = ?", id).First(&row).Error
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionUpdate,
		Resource: "tenant_webhook", ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Updated webhook %s", row.Name), Detail: audit.Redact(req),
	}, before, row)
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func (h *Handlers) deleteTenantWebhook(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var row models.TenantWebhook
	if err := h.db.Where("id = ? AND tenant_id = ?", id, tid).First(&row).Error; err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyWebhookNotFound))
		return
	}
	if err := h.db.Delete(&row).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionDelete,
		Resource: "tenant_webhook", ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Deleted webhook %s", row.Name),
	}, row, nil)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id})
}

func (h *Handlers) listTenantWebhookDeliveries(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	webhookID, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var hook models.TenantWebhook
	if err := h.db.Where("id = ? AND tenant_id = ?", webhookID, tid).First(&hook).Error; err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyWebhookNotFound))
		return
	}
	page, size := ginutil.QueryPage(c, 50)
	q := h.db.Model(&models.TenantWebhookDelivery{}).Where("webhook_id = ? AND tenant_id = ?", webhookID, tid)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	offset := (page - 1) * size
	var list []models.TenantWebhookDelivery
	if err := q.Order("id DESC").Offset(offset).Limit(size).Find(&list).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	ginutil.PageSuccess(c, list, total, page, size)
}

func (h *Handlers) testTenantWebhook(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var hook models.TenantWebhook
	if err := h.db.Where("id = ? AND tenant_id = ?", id, tid).First(&hook).Error; err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyWebhookNotFound))
		return
	}
	notification.DispatchWebhook(h.db, nil, tid, constants.WebhookEventCallStarted,
		"test-call-id", "+10000000000", "+20000000000", "outbound",
		map[string]any{"test": true})
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"sent": true})
}

func (h *Handlers) listWebhookEvents(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"events": constants.AllWebhookEvents})
}

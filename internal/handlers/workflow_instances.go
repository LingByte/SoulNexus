package handlers

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	workflowdef "github.com/LingByte/SoulNexus/internal/workflow"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) workflowDefinitionForUser(c *gin.Context, defID uint) (*models.WorkflowDefinition, bool) {
	user := workflowCurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return nil, false
	}
	var def models.WorkflowDefinition
	if err := h.db.First(&def, defID).Error; err != nil {
		response.Fail(c, "workflow definition not found", err.Error())
		return nil, false
	}
	if !models.UserIsGroupMember(h.db, user.ID, def.GroupID) {
		response.Fail(c, "forbidden", nil)
		return nil, false
	}
	return &def, true
}

// ListWorkflowInstances lists invocation logs for a workflow definition.
func (h *Handlers) ListWorkflowInstances(c *gin.Context) {
	defID, err := strconv.ParseUint(strings.TrimSpace(c.Query("definitionId")), 10, 64)
	if err != nil || defID == 0 {
		response.Fail(c, "definitionId required", nil)
		return
	}
	def, ok := h.workflowDefinitionForUser(c, uint(defID))
	if !ok {
		return
	}
	page, size := ginutil.QueryPage(c, 20)
	filter := models.WorkflowInstanceListFilter{
		DefinitionID:  def.ID,
		GroupID:       def.GroupID,
		TriggerSource: c.Query("source"),
		Keyword:       c.Query("keyword"),
	}
	if from := strings.TrimSpace(c.Query("from")); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.From = &t
		}
	}
	if to := strings.TrimSpace(c.Query("to")); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.To = &t
		}
	}
	list, total, err := models.ListWorkflowInstancesPage(h.db, filter, page, size)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	ginutil.PageSuccess(c, list, total, page, size)
}

// GetWorkflowInstance returns one invocation log with execution details.
func (h *Handlers) GetWorkflowInstance(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var inst models.WorkflowInstance
	if err := h.db.First(&inst, id).Error; err != nil {
		response.Fail(c, "instance not found", err.Error())
		return
	}
	def, ok := h.workflowDefinitionForUser(c, inst.DefinitionID)
	if !ok {
		return
	}
	if inst.GroupID == 0 {
		inst.GroupID = def.GroupID
	}
	response.Success(c, "success", inst)
}

// ExportWorkflowInstances exports invocation logs as CSV.
func (h *Handlers) ExportWorkflowInstances(c *gin.Context) {
	defID, err := strconv.ParseUint(strings.TrimSpace(c.Query("definitionId")), 10, 64)
	if err != nil || defID == 0 {
		response.Fail(c, "definitionId required", nil)
		return
	}
	def, ok := h.workflowDefinitionForUser(c, uint(defID))
	if !ok {
		return
	}
	filter := models.WorkflowInstanceListFilter{
		DefinitionID:  def.ID,
		GroupID:       def.GroupID,
		TriggerSource: c.Query("source"),
		Keyword:       c.Query("keyword"),
	}
	if from := strings.TrimSpace(c.Query("from")); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.From = &t
		}
	}
	if to := strings.TrimSpace(c.Query("to")); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.To = &t
		}
	}
	list, _, err := models.ListWorkflowInstancesPage(h.db, filter, 1, 5000)
	if ginutil.WriteInternalError(c, err) {
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=workflow-%d-logs.csv", def.ID))
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"ID", "来源", "IP", "User-Agent", "标题", "状态", "日志条数", "耗时(ms)", "开始时间", "结束时间", "错误"})
	for _, row := range list {
		started := ""
		if row.StartedAt != nil {
			started = row.StartedAt.Format(time.RFC3339)
		}
		completed := ""
		if row.CompletedAt != nil {
			completed = row.CompletedAt.Format(time.RFC3339)
		}
		ip, ua := clientMetaSummary(row.ClientMeta)
		_ = w.Write([]string{
			strconv.FormatUint(uint64(row.ID), 10),
			row.TriggerSource,
			ip,
			ua,
			row.DefinitionName,
			row.Status,
			strconv.Itoa(row.LogCount),
			strconv.FormatInt(row.DurationMs, 10),
			started,
			completed,
			row.ErrorMessage,
		})
	}
	w.Flush()
}

// StopWorkflowInstance is reserved for future cancellation support.
func (h *Handlers) StopWorkflowInstance(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var inst models.WorkflowInstance
	if err := h.db.First(&inst, id).Error; err != nil {
		response.Fail(c, "instance not found", err.Error())
		return
	}
	if _, ok := h.workflowDefinitionForUser(c, inst.DefinitionID); !ok {
		return
	}
	if inst.Status != "running" {
		response.Success(c, "success", gin.H{"instance_id": id, "status": inst.Status})
		return
	}
	inst.Status = "failed"
	inst.ErrorMessage = "stopped by user"
	now := time.Now()
	inst.CompletedAt = &now
	if err := h.db.Save(&inst).Error; err != nil {
		response.Fail(c, "failed to stop instance", err.Error())
		return
	}
	response.Success(c, "success", gin.H{"instance_id": id, "status": inst.Status})
}

// workflowRequestClientMeta captures HTTP client information for invocation logs.
func workflowRequestClientMeta(c *gin.Context) models.JSONMap {
	if c == nil || c.Request == nil {
		return nil
	}
	meta := models.JSONMap{
		"ip":             c.ClientIP(),
		"userAgent":      strings.TrimSpace(c.GetHeader("User-Agent")),
		"referer":        strings.TrimSpace(c.GetHeader("Referer")),
		"origin":         strings.TrimSpace(c.GetHeader("Origin")),
		"acceptLanguage": strings.TrimSpace(c.GetHeader("Accept-Language")),
		"host":           c.Request.Host,
		"method":         c.Request.Method,
		"path":           c.Request.URL.Path,
		"query":          c.Request.URL.RawQuery,
	}
	if v := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); v != "" {
		meta["forwardedFor"] = v
	}
	if v := strings.TrimSpace(c.GetHeader("X-Real-IP")); v != "" {
		meta["realIp"] = v
	}
	if v := strings.TrimSpace(c.GetHeader("X-Request-Id")); v != "" {
		meta["requestId"] = v
	}
	return meta
}

// workflowInstanceMetaFromContext builds run metadata for manual/editor runs.
func workflowInstanceMetaFromContext(c *gin.Context, def *models.WorkflowDefinition, user *workflowUser, source string, params map[string]interface{}) workflowdef.InstanceRunMeta {
	meta := workflowdef.InstanceRunMeta{
		GroupID:       def.GroupID,
		TriggerSource: source,
		ClientMeta:    workflowRequestClientMeta(c),
	}
	if user != nil {
		meta.UserID = user.ID
		meta.TriggerUser = user.Email
	}
	if len(params) > 0 {
		meta.InputParams = models.JSONMap{}
		for k, v := range params {
			meta.InputParams[k] = v
		}
	}
	return meta
}

func clientMetaSummary(meta models.JSONMap) (ip, ua string) {
	if len(meta) == 0 {
		return "—", "—"
	}
	if v, ok := meta["ip"].(string); ok && strings.TrimSpace(v) != "" {
		ip = strings.TrimSpace(v)
	}
	if v, ok := meta["userAgent"].(string); ok && strings.TrimSpace(v) != "" {
		ua = strings.TrimSpace(v)
	}
	if ip == "" {
		ip = "—"
	}
	if ua == "" {
		ua = "—"
	}
	return ip, ua
}

func workflowParamsWithClientMeta(c *gin.Context, params map[string]interface{}) map[string]interface{} {
	out := params
	if out == nil {
		out = make(map[string]interface{})
	}
	if meta := workflowRequestClientMeta(c); len(meta) > 0 {
		out["_client_meta"] = meta
	}
	return out
}

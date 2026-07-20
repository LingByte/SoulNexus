package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	knworker "github.com/LingByte/SoulNexus/pkg/knowledge/worker"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/task"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	gonanoid "github.com/matoous/go-nanoid"
	"gorm.io/gorm"
)

func (h *Handlers) registerPlatformExecutionTaskRoutes(r *humax.Group) {
	admin := r.Group("")
	admin.Use(middleware.RequirePlatformAdmin())
	g := admin.Group("admin/execution-tasks")
	{
		g.GET("", h.listExecutionTasks)
		g.GET("/stats/summary", h.getExecutionTaskStats)
		g.POST("", h.createExecutionTask)
		g.GET("/:id", h.getExecutionTask)
		g.PUT("/:id", h.updateExecutionTask)
		g.DELETE("/:id", h.deleteExecutionTask)
		g.POST("/:id/cancel", h.cancelExecutionTask)
		g.POST("/:id/retry", h.retryExecutionTask)
	}
}

func (h *Handlers) listExecutionTasks(c *gin.Context) {
	page, pageSize := ginutil.QueryPage(c, 200)
	filter := task.ExecutionTaskListFilter{
		QueueName: c.Query("queueName"),
		Kind:      c.Query("kind"),
		Source:    c.Query("source"),
		Status:    c.Query("status"),
		TaskID:    c.Query("taskId"),
		Search:    c.Query("search"),
	}
	if v := strings.TrimSpace(c.Query("tenantId")); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			filter.TenantID = uint(n)
		}
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
	list, total, err := task.ListExecutionTasksPage(h.db, filter, page, pageSize)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	ginutil.PageSuccess(c, list, total, page, pageSize)
}

func (h *Handlers) getExecutionTaskStats(c *gin.Context) {
	stats, err := task.ExecutionTaskStats(h.db, c.Query("queueName"))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, stats)
}

func (h *Handlers) getExecutionTask(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := task.GetExecutionTaskByID(h.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.AbortWithStatus(c, http.StatusNotFound)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

type executionTaskCreateReq struct {
	QueueName  string `json:"queueName"`
	Title      string `json:"title"`
	Kind       string `json:"kind"`
	Source     string `json:"source"`
	TenantID   uint   `json:"tenantId,string"`
	Priority   int    `json:"priority"`
	ParamsJSON string `json:"paramsJson"`
	MaxRetries int    `json:"maxRetries"`
	Remark     string `json:"remark"`
	EnqueueNow bool   `json:"enqueueNow"`
}

func (h *Handlers) createExecutionTask(c *gin.Context) {
	var req executionTaskCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "invalid request", gin.H{"error": err.Error()})
		return
	}
	queueName := strings.TrimSpace(req.QueueName)
	paramsJSON := strings.TrimSpace(req.ParamsJSON)
	if queueName == "" || paramsJSON == "" {
		response.FailWithCode(c, http.StatusBadRequest, "queueName and paramsJson are required", nil)
		return
	}
	nid, _ := gonanoid.Nanoid()
	row := task.ExecutionTask{
		TaskID:     "task_" + nid,
		QueueName:  queueName,
		Title:      strings.TrimSpace(req.Title),
		Kind:       strings.TrimSpace(req.Kind),
		Source:     strings.TrimSpace(req.Source),
		TenantID:   req.TenantID,
		Priority:   req.Priority,
		ParamsJSON: paramsJSON,
		MaxRetries: req.MaxRetries,
		SubmitTime: time.Now(),
	}
	row.Remark = strings.TrimSpace(req.Remark)
	row.SetCreateInfo(middleware.AuditOperator(c))
	if err := task.SaveExecutionTaskPending(h.db, row); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if req.EnqueueNow {
		if err := h.enqueueExecutionTaskNow(row); err != nil {
			response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
	}
	created, err := task.GetExecutionTaskByTaskID(h.db, row.TaskID)
	if err != nil {
		response.SuccessI18n(c, i18n.KeySuccess, row)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, created)
}

type executionTaskUpdateReq struct {
	Title      *string `json:"title"`
	Priority   *int    `json:"priority"`
	MaxRetries *int    `json:"maxRetries"`
	Remark     *string `json:"remark"`
	TenantID   *uint   `json:"tenantId,string"`
}

func (h *Handlers) updateExecutionTask(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var req executionTaskUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "invalid request", gin.H{"error": err.Error()})
		return
	}
	row, err := task.GetExecutionTaskByID(h.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.AbortWithStatus(c, http.StatusNotFound)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if row.Status != task.TaskStatusPending.String() {
		response.FailWithCode(c, http.StatusBadRequest, "only pending tasks can be edited", nil)
		return
	}
	updates := map[string]any{"update_by": middleware.AuditOperator(c)}
	if req.Title != nil {
		updates["title"] = strings.TrimSpace(*req.Title)
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.MaxRetries != nil {
		updates["max_retries"] = *req.MaxRetries
	}
	if req.Remark != nil {
		updates["remark"] = strings.TrimSpace(*req.Remark)
	}
	if req.TenantID != nil {
		updates["tenant_id"] = *req.TenantID
	}
	if err := task.UpdateExecutionTask(h.db, id, updates); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.AbortWithStatus(c, http.StatusNotFound)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	updated, _ := task.GetExecutionTaskByID(h.db, id)
	response.SuccessI18n(c, i18n.KeySuccess, updated)
}

func (h *Handlers) deleteExecutionTask(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := task.GetExecutionTaskByID(h.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.AbortWithStatus(c, http.StatusNotFound)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if row.Status == task.TaskStatusRunning.String() {
		response.FailWithCode(c, http.StatusBadRequest, "running task cannot be deleted; cancel it first", nil)
		return
	}
	if err := task.SoftDeleteExecutionTask(h.db, id, middleware.AuditOperator(c)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.AbortWithStatus(c, http.StatusNotFound)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, nil)
}

func (h *Handlers) cancelExecutionTask(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := task.GetExecutionTaskByID(h.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.AbortWithStatus(c, http.StatusNotFound)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	switch row.Status {
	case task.TaskStatusSuccess.String(), task.TaskStatusFailed.String(), task.TaskStatusCanceled.String():
		response.FailWithCode(c, http.StatusBadRequest, "task already finished", nil)
		return
	}
	if row.Status == task.TaskStatusPending.String() {
		switch row.QueueName {
		case "knowledge.document":
			if h.kbWorker != nil {
				h.kbWorker.CancelTaskByID(row.TaskID)
			}
		}
	}
	if err := task.MarkExecutionTaskFinished(h.db, row.TaskID, task.TaskStatusCanceled, "canceled by admin"); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	updated, _ := task.GetExecutionTaskByID(h.db, id)
	response.SuccessI18n(c, i18n.KeySuccess, updated)
}

func (h *Handlers) retryExecutionTask(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := task.GetExecutionTaskByID(h.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.AbortWithStatus(c, http.StatusNotFound)
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	switch row.Status {
	case task.TaskStatusFailed.String(), task.TaskStatusCanceled.String():
	default:
		response.FailWithCode(c, http.StatusBadRequest, "only failed or canceled tasks can be retried", nil)
		return
	}
	if row.MaxRetries > 0 && row.RetryCount >= row.MaxRetries {
		response.FailWithCode(c, http.StatusBadRequest, "max retries exceeded", nil)
		return
	}
	if err := h.retryExecutionTaskRow(*row); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	updated, _ := task.GetExecutionTaskByID(h.db, id)
	response.SuccessI18n(c, i18n.KeySuccess, updated)
}

func (h *Handlers) retryExecutionTaskRow(row task.ExecutionTask) error {
	switch row.QueueName {
	case "knowledge.document":
		if h.kbWorker == nil {
			return fmt.Errorf("knowledge worker is not running")
		}
		return h.kbWorker.RetryTaskByID(h.db, row.TaskID)
	default:
		return fmt.Errorf("retry not supported for queue %q", row.QueueName)
	}
}

func (h *Handlers) enqueueExecutionTaskNow(row task.ExecutionTask) error {
	switch row.QueueName {
	case "knowledge.document":
		if h.kbWorker == nil {
			return fmt.Errorf("knowledge worker is not running")
		}
		var job knworker.DocumentJob
		if err := json.Unmarshal([]byte(row.ParamsJSON), &job); err != nil {
			return fmt.Errorf("invalid paramsJson: %w", err)
		}
		if job.Ingest == nil && job.Purge == nil && job.SyncSourceID == 0 {
			return fmt.Errorf("empty knowledge document job")
		}
		h.kbWorker.RequeueJob(row.TaskID, job, row.Priority, row.SubmitTime)
		return nil
	default:
		return fmt.Errorf("enqueue not supported for queue %q", row.QueueName)
	}
}

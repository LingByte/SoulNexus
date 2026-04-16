package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type sipCampaignCreateReq struct {
	Name              string `json:"name"`
	Scenario          string `json:"scenario"`
	MediaProfile      string `json:"media_profile"`
	ScriptID          string `json:"script_id"`
	ScriptVersion     string `json:"script_version"`
	ScriptSpec        string `json:"script_spec"`
	RetrySchedule     string `json:"retry_schedule"`
	MaxAttempts       int    `json:"max_attempts"`
	TaskConcurrency   int    `json:"task_concurrency"`
	GlobalConcurrency int    `json:"global_concurrency"`
	RequestURIFmt     string `json:"request_uri_fmt"`
}

type sipCampaignContactReq struct {
	Phone      string                 `json:"phone"`
	Display    string                 `json:"display"`
	CallerUser string                 `json:"caller_user"`
	CallerName string                 `json:"caller_name"`
	RequestURI string                 `json:"request_uri"`
	Priority   int                    `json:"priority"`
	Variables  map[string]interface{} `json:"variables"`
}

func (h *Handlers) listSIPCampaigns(c *gin.Context) {
	page, size := parsePageSize(c)
	list, total, err := models.ListSIPCampaignsPage(h.db, page, size, c.Query("status"), c.Query("name"))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", gin.H{"list": list, "total": total, "page": page, "size": size})
}

func (h *Handlers) createSIPCampaign(c *gin.Context) {
	var req sipCampaignCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "invalid body", err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		response.Fail(c, "name required", nil)
		return
	}
	var spec datatypes.JSON
	if s := strings.TrimSpace(req.ScriptSpec); s != "" {
		spec = datatypes.JSON([]byte(s))
	}
	row := models.SIPCampaign{
		Name:              name,
		Status:            models.SIPCampaignStatusDraft,
		Scenario:          strings.TrimSpace(req.Scenario),
		MediaProfile:      strings.TrimSpace(req.MediaProfile),
		ScriptID:          strings.TrimSpace(req.ScriptID),
		ScriptVersion:     strings.TrimSpace(req.ScriptVersion),
		ScriptSpec:        spec,
		RetrySchedule:     strings.TrimSpace(req.RetrySchedule),
		MaxAttempts:       req.MaxAttempts,
		TaskConcurrency:   req.TaskConcurrency,
		GlobalConcurrency: req.GlobalConcurrency,
		RequestURIFmt:     strings.TrimSpace(req.RequestURIFmt),
	}
	if row.Scenario == "" {
		row.Scenario = "campaign"
	}
	if row.MediaProfile == "" {
		row.MediaProfile = "script"
	}
	if row.MaxAttempts <= 0 {
		row.MaxAttempts = 3
	}
	if row.TaskConcurrency <= 0 {
		row.TaskConcurrency = 5
	}
	if row.GlobalConcurrency <= 0 {
		row.GlobalConcurrency = 20
	}
	if op := acdOperator(c); op != "" {
		row.SetCreateInfo(op)
	}
	if err := h.db.Create(&row).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	h.appendCampaignEvent(row.ID, 0, 0, "", "", "campaign", "info", "campaign created")
	response.Success(c, "success", row)
}

func (h *Handlers) addSIPCampaignContacts(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	campaign, err := models.GetActiveSIPCampaignByID(h.db, uint(id))
	if err != nil {
		response.Fail(c, "campaign not found", nil)
		return
	}
	var req []sipCampaignContactReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "invalid body", err.Error())
		return
	}
	now := time.Now()
	items := make([]models.SIPCampaignContactBatchItem, 0, len(req))
	for _, it := range req {
		phone := strings.TrimSpace(it.Phone)
		if phone == "" {
			continue
		}
		b, _ := jsonMarshal(it.Variables)
		items = append(items, models.SIPCampaignContactBatchItem{
			Phone:         phone,
			Display:       it.Display,
			CallerUser:    it.CallerUser,
			CallerName:    it.CallerName,
			RequestURI:    it.RequestURI,
			Priority:      it.Priority,
			VariablesJSON: datatypes.JSON(b),
		})
	}
	rows := models.BuildSIPCampaignContactsBatch(uint(id), campaign.MaxAttempts, items, now)
	if len(rows) == 0 {
		response.Success(c, "success", gin.H{"accepted": 0})
		return
	}
	if err := h.db.Create(&rows).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	h.appendCampaignEvent(uint(id), 0, 0, "", "", "contact", "info", fmt.Sprintf("contacts imported: %d", len(rows)))
	response.Success(c, "success", gin.H{"accepted": len(rows)})
}

func (h *Handlers) listSIPCampaignContacts(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	page, size := parsePageSize(c)
	list, total, err := models.ListSIPCampaignContactsPage(h.db, uint(id), page, size)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", gin.H{"list": list, "total": total, "page": page, "size": size})
}

func (h *Handlers) resetSIPCampaignSuppressedContacts(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	now := time.Now()
	n, err := models.ResetSuppressedSIPCampaignContacts(h.db, uint(id), now)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	h.appendCampaignEvent(uint(id), 0, 0, "", "", "contact", "warn", fmt.Sprintf("reset suppressed contacts: %d", n))
	response.Success(c, "success", gin.H{"updated": n})
}

func (h *Handlers) setSIPCampaignStatus(c *gin.Context, status string) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	n, err := models.UpdateActiveSIPCampaignStatus(h.db, uint(id), status, acdOperator(c))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if n == 0 {
		response.Fail(c, "campaign not found", nil)
		return
	}
	if h.campaignSvc != nil && (status == models.SIPCampaignStatusPaused || status == models.SIPCampaignStatusDone) {
		if _, err := h.campaignSvc.CancelCampaignQueuedTasks(context.Background(), uint(id)); err != nil {
			response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
			return
		}
	}
	h.appendCampaignEvent(uint(id), 0, 0, "", "", "campaign", "info", "campaign status changed to "+status)
	response.Success(c, "success", nil)
}

func (h *Handlers) startSIPCampaign(c *gin.Context) {
	h.setSIPCampaignStatus(c, models.SIPCampaignStatusRunning)
}
func (h *Handlers) pauseSIPCampaign(c *gin.Context) {
	h.setSIPCampaignStatus(c, models.SIPCampaignStatusPaused)
}
func (h *Handlers) resumeSIPCampaign(c *gin.Context) {
	h.setSIPCampaignStatus(c, models.SIPCampaignStatusRunning)
}
func (h *Handlers) stopSIPCampaign(c *gin.Context) {
	h.setSIPCampaignStatus(c, models.SIPCampaignStatusDone)
}

func (h *Handlers) deleteSIPCampaign(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	row, err := models.GetActiveSIPCampaignByID(h.db, uint(id))
	if err != nil {
		response.Fail(c, "campaign not found", nil)
		return
	}
	if row.Status == models.SIPCampaignStatusRunning {
		response.Fail(c, "running campaign cannot be deleted", nil)
		return
	}
	n, err := models.SoftDeleteSIPCampaignByID(h.db, uint(id), acdOperator(c))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if n == 0 {
		response.Fail(c, "campaign not found", nil)
		return
	}
	h.appendCampaignEvent(uint(id), 0, 0, "", "", "campaign", "info", "campaign deleted")
	response.Success(c, "success", gin.H{"id": id})
}

func (h *Handlers) getSIPCampaignMetrics(c *gin.Context) {
	invited, _ := models.CountAllSIPCallAttempts(h.db)
	answered, _ := models.CountSIPCampaignContactsWithStatus(h.db, models.SIPCampaignContactAnswered)
	failed, _ := models.CountSIPCampaignContactsWithStatuses(h.db, []string{models.SIPCampaignContactFailed, models.SIPCampaignContactExhausted})
	retrying, _ := models.CountSIPCampaignContactsWithStatus(h.db, models.SIPCampaignContactRetrying)
	suppressed, _ := models.CountSIPCampaignContactsWithStatus(h.db, models.SIPCampaignContactSuppressed)
	response.Success(c, "success", gin.H{
		"invited_total":    invited,
		"answered_total":   answered,
		"failed_total":     failed,
		"retrying_total":   retrying,
		"suppressed_total": suppressed,
	})
}

// getSIPCampaignWorkerMetrics returns in-process dial counters from the embedded campaign worker
// (same data as the former SIP_CAMPAIGN_HTTP server GET .../metrics).
func (h *Handlers) getSIPCampaignWorkerMetrics(c *gin.Context) {
	if h.campaignSvc == nil {
		response.Fail(c, "campaign worker unavailable", nil)
		return
	}
	response.Success(c, "success", h.campaignSvc.SnapshotMetrics())
}

func (h *Handlers) getSIPCampaignLogs(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit <= 0 {
		limit = 100
	}
	if limit > 300 {
		limit = 300
	}
	campaign, err := models.GetActiveSIPCampaignByID(h.db, uint(id))
	if err != nil {
		response.Fail(c, "campaign not found", nil)
		return
	}

	type campaignLogRow struct {
		ID            uint      `json:"id"`
		At            time.Time `json:"at"`
		Type          string    `json:"type"`
		ContactID     uint      `json:"contactId,omitempty"`
		AttemptID     uint      `json:"attemptId,omitempty"`
		AttemptNo     int       `json:"attemptNo,omitempty"`
		Phone         string    `json:"phone,omitempty"`
		CallID        string    `json:"callId,omitempty"`
		CorrelationID string    `json:"correlationId,omitempty"`
		Level         string    `json:"level"`
		Message       string    `json:"message"`
	}
	logs := make([]campaignLogRow, 0, limit*3)

	events, _ := models.ListSIPCampaignEventsDesc(h.db, uint(id), limit)
	for _, e := range events {
		logs = append(logs, campaignLogRow{
			ID:            e.ID,
			At:            e.CreatedAt,
			Type:          strings.TrimSpace(e.Type),
			ContactID:     e.ContactID,
			AttemptID:     e.AttemptID,
			Phone:         "",
			CallID:        e.CallID,
			CorrelationID: e.CorrelationID,
			Level:         nonEmptyOr(strings.TrimSpace(e.Level), "info"),
			Message:       strings.TrimSpace(e.Message),
		})
	}

	attempts, _ := models.ListSIPCallAttemptsDesc(h.db, uint(id), limit)
	for _, a := range attempts {
		var phone string
		if a.ContactID > 0 {
			if p, err := models.GetSIPCampaignContactPhone(h.db, a.ContactID); err == nil {
				phone = p
			}
		}
		at := a.CreatedAt
		if a.DialedAt != nil {
			at = *a.DialedAt
		}
		msg := fmt.Sprintf("attempt#%d state=%s", a.AttemptNo, strings.TrimSpace(a.State))
		if a.SIPStatusCode > 0 {
			msg += fmt.Sprintf(" sip=%d", a.SIPStatusCode)
		}
		if r := strings.TrimSpace(a.FailureReason); r != "" {
			msg += " reason=" + r
		}
		level := "info"
		if strings.EqualFold(strings.TrimSpace(a.State), "failed") {
			level = "error"
		}
		logs = append(logs, campaignLogRow{
			ID:            a.ID,
			At:            at,
			Type:          "attempt",
			ContactID:     a.ContactID,
			AttemptID:     a.ID,
			AttemptNo:     a.AttemptNo,
			Phone:         phone,
			CallID:        a.CallID,
			CorrelationID: a.CorrelationID,
			Level:         level,
			Message:       msg,
		})
	}

	steps, _ := models.ListSIPScriptRunsDesc(h.db, uint(id), limit)
	for _, s := range steps {
		msg := fmt.Sprintf("script step=%s type=%s result=%s", strings.TrimSpace(s.StepID), strings.TrimSpace(s.StepType), strings.TrimSpace(s.Result))
		if out := strings.TrimSpace(s.OutputText); out != "" {
			runes := []rune(out)
			if len(runes) > 80 {
				out = string(runes[:80]) + "..."
			}
			msg += " output=" + out
		}
		logs = append(logs, campaignLogRow{
			ID:            s.ID,
			At:            s.CreatedAt,
			Type:          "script",
			ContactID:     s.ContactID,
			AttemptID:     s.AttemptID,
			Phone:         "",
			CallID:        s.CallID,
			CorrelationID: s.CorrelationID,
			Level:         "info",
			Message:       msg,
		})
	}

	// also include simple campaign status event for operator visibility
	logs = append(logs, campaignLogRow{
		ID:      campaign.ID,
		At:      campaign.UpdatedAt,
		Type:    "campaign",
		Level:   "info",
		Message: "campaign status=" + strings.TrimSpace(campaign.Status),
	})

	// sort desc by event time in-memory
	for i := 0; i < len(logs); i++ {
		for j := i + 1; j < len(logs); j++ {
			if logs[j].At.After(logs[i].At) {
				logs[i], logs[j] = logs[j], logs[i]
			}
		}
	}
	if len(logs) > limit {
		logs = logs[:limit]
	}
	response.Success(c, "success", gin.H{
		"list":  logs,
		"total": len(logs),
	})
}

func (h *Handlers) appendCampaignEvent(campaignID, contactID, attemptID uint, callID, correlationID, typ, level, message string) {
	if h == nil || h.db == nil || campaignID == 0 {
		return
	}
	evt := &models.SIPCampaignEvent{
		CampaignID:    campaignID,
		ContactID:     contactID,
		AttemptID:     attemptID,
		CallID:        strings.TrimSpace(callID),
		CorrelationID: strings.TrimSpace(correlationID),
		Type:          nonEmptyOr(strings.TrimSpace(typ), "campaign"),
		Level:         nonEmptyOr(strings.TrimSpace(level), "info"),
		Message:       nonEmptyOr(strings.TrimSpace(message), "event"),
	}
	_ = models.InsertSIPCampaignEvent(context.Background(), h.db, evt)
}

func nonEmptyOr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func jsonMarshal(v interface{}) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}"), nil
	}
	return b, nil
}

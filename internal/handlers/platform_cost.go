package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"github.com/gin-gonic/gin"
	"github.com/LingByte/SoulNexus/pkg/i18n"
)

type billingPlanWriteReq struct {
	Name               string  `json:"name"`
	Description        string  `json:"description"`
	Currency           string  `json:"currency"`
	CallRatePerMinute  float64 `json:"callRatePerMinute"`
	LLMRatePer1kTokens float64 `json:"llmRatePer1kTokens"`
	Status             string  `json:"status"`
}

func parseCostRangeDays(c *gin.Context) (from, to time.Time) {
	days, _ := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("days", "30")))
	return timeutil.SlidingWindowNow(days, 30, 365)
}

// getPlatformCallCostSummary aggregates platform-wide call costs and trunk
// utilization over a configurable sliding window. Result includes totals plus
// a per-tenant breakdown and the full trunk-rate table used for billing.
//
//	Endpoint: GET /platform/call-cost-summary
//
// Path parameters: none.
//
// Query parameters:
//
//	days (int, default 30; max 365) - size of the sliding window used for the
//	                                    report (window: [now-days, now]).
//
// Request body: none.
//
// Response:
//
//	{
//	  "code": 200, "msg": "success",
//	  "data": {
//	    "from": time.Time, "to": time.Time,
//	    "summary": aggregate totals,
//	    "byTenant": [ per-tenant subtotals ],
//	    "trunkRates": [ ...trunks ]
//	  }
//	}
func (h *Handlers) getPlatformCallCostSummary(c *gin.Context) {
	from, to := parseCostRangeDays(c)
	summary, byTenant, err := models.SummarizePlatformCallCost(h.db, &from, &to)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"from":       from,
		"to":         to,
		"summary":    summary,
		"byTenant":   byTenant,
		"trunkRates": []any{},
	})
}

// listBillingPlans returns a paginated list of all billing plans defined on
// the platform (name, currency, call/LLM rates, status, etc.).
//
//	Endpoint: GET /platform/billing-plans
//
// Path parameters: none.
//
// Query parameters:
//
//	page   (int, default 1)   - page number.
//	size   (int, default 100) - page size.
//	name   (string)           - optional substring filter on the plan name.
//
// Request body: none.
//
// Response (paginated):
//
//	{ "code": 200, "msg": "success", "data": [...models.BillingPlan...],
//	  "total": int, "page": int, "size": int }
func (h *Handlers) listBillingPlans(c *gin.Context) {
	page, size := ginutil.QueryPage(c, 100)
	name := strings.TrimSpace(c.Query("name"))
	list, total, err := models.ListBillingPlansPage(h.db, page, size, name)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	ginutil.PageSuccess(c, list, total, page, size)
}

// getBillingPlan fetches a single billing plan by its id.
//
//	Endpoint: GET /platform/billing-plans/:id
//
// Path parameters:
//
//	id (uint) - billing plan row id.
//
// Query parameters: none.
//
// Request body: none.
//
// Response: { "code": 200, "msg": "success", "data": models.BillingPlan }
func (h *Handlers) getBillingPlan(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetBillingPlanByID(h.db, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func parseBillingPlanStatus(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "active":
		return "active", nil
	case "disabled":
		return "disabled", nil
	default:
		return "", fmt.Errorf("invalid status: %s", s)
	}
}

// createBillingPlan creates a new billing plan.
//
//	Endpoint: POST /platform/billing-plans
//
// Path parameters: none.
//
// Query parameters: none.
//
// Request body (application/json):
//
//	{
//	  "name":               "display name (required)",
//	  "description":        "optional",
//	  "currency":           "CNY (optional)",
//	  "callRatePerMinute":  float,
//	  "llmRatePer1kTokens": float,
//	  "status":             "active | disabled | ..."
//	}
//
// Response: { "code": 200, "msg": "success", "data": models.BillingPlan }
func (h *Handlers) createBillingPlan(c *gin.Context) {
	var req billingPlanWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyNameRequired))
		return
	}
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "CNY"
	}
	status, err := parseBillingPlanStatus(req.Status)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, err.Error(), err))
		return
	}
	row := models.BillingPlan{
		Name:               name,
		Description:        strings.TrimSpace(req.Description),
		Currency:           currency,
		CallRatePerMinute:  req.CallRatePerMinute,
		LLMRatePer1kTokens: req.LLMRatePer1kTokens,
		Status:             status,
	}
	if err := h.db.Create(&row).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		Action: constants.OpActionCreate, Resource: "billing_plan",
		ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Created billing plan %s", row.Name), Detail: audit.Redact(req),
	}, nil, row)
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

// updateBillingPlan updates the mutable fields of a billing plan. Currency
// is kept unchanged if not supplied; otherwise it replaces the current value.
// The update is recorded in the audit log with before/after snapshots.
//
//	Endpoint: PUT /platform/billing-plans/:id
//
// Path parameters:
//
//	id (uint) - billing plan row id.
//
// Query parameters: none.
//
// Request body (application/json): same schema as createBillingPlan
// (name is required).
//
// Response: { "code": 200, "msg": "success", "data": models.BillingPlan }
func (h *Handlers) updateBillingPlan(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	before, err := models.GetBillingPlanByID(h.db, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	var req billingPlanWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyNameRequired))
		return
	}
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = before.Currency
	}
	status, err := parseBillingPlanStatus(req.Status)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, err.Error(), err))
		return
	}
	updates := map[string]any{
		"name":                   name,
		"description":            strings.TrimSpace(req.Description),
		"currency":               currency,
		"call_rate_per_minute":   req.CallRatePerMinute,
		"llm_rate_per_1k_tokens": req.LLMRatePer1kTokens,
		"status":                 status,
	}
	if err := h.db.Model(&models.BillingPlan{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	after, _ := models.GetBillingPlanByID(h.db, id)
	h.recordOpChange(c, OpLogEntry{
		Action: constants.OpActionUpdate, Resource: "billing_plan",
		ResourceID: id, ResourceName: after.Name,
		Summary: fmt.Sprintf("Updated billing plan %s", after.Name), Detail: audit.Redact(req),
	}, before, after)
	response.SuccessI18n(c, i18n.KeySuccess, after)
}

// deleteBillingPlan permanently removes a billing plan and records the operation
// in the audit log with a before snapshot.
//
//	Endpoint: DELETE /platform/billing-plans/:id
//
// Path parameters:
//
//	id (uint) - billing plan row id.
//
// Query parameters: none.
//
// Request body: none.
//
// Response: { "code": 200, "msg": "success", "data": { "id": uint } }
func (h *Handlers) deleteBillingPlan(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	before, err := models.GetBillingPlanByID(h.db, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	if err := h.db.Delete(&models.BillingPlan{}, id).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		Action: constants.OpActionDelete, Resource: "billing_plan",
		ResourceID: id, ResourceName: before.Name,
		Summary: fmt.Sprintf("Deleted billing plan %s", before.Name),
	}, before, nil)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id})
}

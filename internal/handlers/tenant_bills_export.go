package handlers

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// exportTenantBill streams a bill's usage detail as CSV or XLSX.
//
//   - GET /billing/bills/:id/export
//   - Path: id (uint) — the bill id.
//   - Query: format (string, optional) — "csv" (default) or "xlsx" / "excel".
//
// Response: a binary download with Content-Type matching the requested
// format. The default filename uses the bill number.
func (h *Handlers) exportTenantBill(c *gin.Context) {
	bill, ok := loadTenantBillForRequest(c, h)
	if !ok {
		return
	}
	format := strings.ToLower(strings.TrimSpace(c.Query("format")))
	if format == "" {
		format = "csv"
	}
	switch format {
	case "csv":
		h.writeTenantBillCSV(c, bill)
	case "xlsx", "excel":
		h.writeTenantBillExcel(c, bill)
	default:
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyExportFormatInvalid))
	}
}

func loadTenantBillForRequest(c *gin.Context, h *Handlers) (models.TenantBill, bool) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return models.TenantBill{}, false
	}
	row, err := models.GetTenantBill(h.db, id)
	if errors.Is(err, models.ErrTenantBillNotFound) {
		response.Render(c, response.Err(response.CodeNotFound))
		return models.TenantBill{}, false
	}
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "load bill failed", err))
		return models.TenantBill{}, false
	}
	if middleware.AuthPlatformAdminID(c) == 0 && row.TenantID != middleware.CurrentTenantID(c) {
		response.Render(c, response.Err(response.CodeNotFound))
		return models.TenantBill{}, false
	}
	return row, true
}

func (h *Handlers) writeTenantBillCSV(c *gin.Context, bill models.TenantBill) {
	detail, _ := models.ParseTenantBillUsageDetail(bill.UsageDetail)
	buf := &bytes.Buffer{}
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(buf)

	_ = w.Write([]string{"section", "key", "value1", "value2"})
	writeCSVSummary(w, bill)
	for _, d := range detail.Daily {
		_ = w.Write([]string{
			"daily",
			d.Day,
			strconv.FormatInt(d.CallCount, 10),
			strconv.FormatInt(d.BilledMinutes, 10),
		})
	}
	for _, d := range detail.Direction {
		_ = w.Write([]string{
			"direction",
			d.Direction,
			strconv.FormatInt(d.CallCount, 10),
			strconv.FormatInt(d.BilledMinutes, 10),
		})
	}
	w.Flush()

	filename := fmt.Sprintf("%s.csv", strings.TrimSpace(bill.BillNo))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

func writeCSVSummary(w *csv.Writer, bill models.TenantBill) {
	pairs := []struct{ k, v string }{
		{"billNo", bill.BillNo},
		{"periodStart", bill.PeriodStart.Format(time.RFC3339)},
		{"periodEnd", bill.PeriodEnd.Format(time.RFC3339)},
		{"status", bill.Status},
		{"currency", bill.Currency},
		{"totalAmount", fmt.Sprintf("%.4f", bill.TotalAmount)},
		{"callCount", strconv.FormatInt(bill.CallCount, 10)},
		{"connectedCallCount", strconv.FormatInt(bill.ConnectedCallCount, 10)},
		{"billedMinutes", strconv.FormatInt(bill.BilledMinutes, 10)},
		{"inboundCallCount", strconv.FormatInt(bill.InboundCallCount, 10)},
		{"outboundCallCount", strconv.FormatInt(bill.OutboundCallCount, 10)},
		{"aiToHumanCount", strconv.FormatInt(bill.AIToHumanCount, 10)},
		{"analysisCount", strconv.FormatInt(bill.AnalysisCount, 10)},
	}
	for _, p := range pairs {
		_ = w.Write([]string{"summary", p.k, p.v, ""})
	}
}

func (h *Handlers) writeTenantBillExcel(c *gin.Context, bill models.TenantBill) {
	detail, _ := models.ParseTenantBillUsageDetail(bill.UsageDetail)
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	summarySheet := "Summary"
	_ = f.SetSheetName("Sheet1", summarySheet)
	rows := [][]any{
		{"Bill No", bill.BillNo},
		{"Period Start", bill.PeriodStart.Format("2006-01-02")},
		{"Period End", bill.PeriodEnd.Format("2006-01-02")},
		{"Status", bill.Status},
		{"Currency", bill.Currency},
		{"Total Amount", bill.TotalAmount},
		{"Call Count", bill.CallCount},
		{"Connected Calls", bill.ConnectedCallCount},
		{"Billed Minutes", bill.BilledMinutes},
		{"Inbound Calls", bill.InboundCallCount},
		{"Outbound Calls", bill.OutboundCallCount},
		{"AI To Human", bill.AIToHumanCount},
		{"Analysis Count", bill.AnalysisCount},
	}
	for i, row := range rows {
		cellA, _ := excelize.CoordinatesToCellName(1, i+1)
		cellB, _ := excelize.CoordinatesToCellName(2, i+1)
		_ = f.SetCellValue(summarySheet, cellA, row[0])
		_ = f.SetCellValue(summarySheet, cellB, row[1])
	}

	dailySheet := "Daily"
	_, _ = f.NewSheet(dailySheet)
	_ = f.SetSheetRow(dailySheet, "A1", &[]any{"Day", "Call Count", "Billed Minutes"})
	for i, d := range detail.Daily {
		row := []any{d.Day, d.CallCount, d.BilledMinutes}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		_ = f.SetSheetRow(dailySheet, cell, &row)
	}

	dirSheet := "Direction"
	_, _ = f.NewSheet(dirSheet)
	_ = f.SetSheetRow(dirSheet, "A1", &[]any{"Direction", "Call Count", "Billed Minutes"})
	for i, d := range detail.Direction {
		row := []any{d.Direction, d.CallCount, d.BilledMinutes}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		_ = f.SetSheetRow(dirSheet, cell, &row)
	}

	out, err := f.WriteToBuffer()
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "export excel failed", err))
		return
	}
	filename := fmt.Sprintf("%s.xlsx", strings.TrimSpace(bill.BillNo))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", out.Bytes())
}

// getTenantBillingAccount exposes the plan/limits view used by the billing
// dashboard.
//
//   - GET /billing/account
//   - Query: tenantId (string, optional — platform admin only).
//
// Response: a JSON summary { billingMode, prepaidMinutesRemaining, ... }
// derived from the tenant row.
func (h *Handlers) getTenantBillingAccount(c *gin.Context) {
	tenantID := middleware.CurrentTenantID(c)
	if middleware.AuthPlatformAdminID(c) > 0 {
		if tidStr := strings.TrimSpace(c.Query("tenantId")); tidStr != "" {
			if tid, err := strconv.ParseUint(tidStr, 10, 64); err == nil {
				tenantID = uint(tid)
			}
		}
	}
	if tenantID == 0 {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyTenantIDRequired))
		return
	}
	t, err := models.GetActiveTenantByID(h.db, tenantID)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	if middleware.AuthPlatformAdminID(c) == 0 && t.ID != middleware.CurrentTenantID(c) {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	account := models.TenantBillingAccountFrom(t)
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": account})
}

type patchTenantBillingReq struct {
	BillingMode             *string  `json:"billingMode"`
	BillingUnlimited        *bool    `json:"billingUnlimited"`
	PrepaidMinutesRemaining *int64   `json:"prepaidMinutesRemaining"`
	RechargeMinutes         *int64   `json:"rechargeMinutes"`
	BillingRatePerMinute    *float64 `json:"billingRatePerMinute"`
	BillingCurrency         string   `json:"billingCurrency"`
	MaxConcurrentCalls      *int     `json:"maxConcurrentCalls"`
	DailyMinuteLimit        *int64   `json:"dailyMinuteLimit"`
	MonthlyMinuteLimit      *int64   `json:"monthlyMinuteLimit"`
	LicenseExpiresAt        *string  `json:"licenseExpiresAt"`
	QuotaSuspended          *bool    `json:"quotaSuspended"`
	MaxUserCount            *int     `json:"maxUserCount"`
}

// patchTenantBillingPlatform overwrites a tenant's billing fields (plan,
// prepaid minutes, concurrent-calls limit, licenses, etc.). This is a platform-only
// endpoint; the tenant cannot self-serve these settings.
//
//   - PUT /billing/tenants/:id (platform admin)
//   - Path: id (uint) — the tenant id.
//
// Request body: { billingMode, billingUnlimited, prepaidMinutesRemaining,
// rechargeMinutes, billingRatePerMinute, billingCurrency, maxConcurrentCalls,
// dailyMinuteLimit, monthlyMinuteLimit, licenseExpiresAt, quotaSuspended,
// maxUserCount } — all fields are optional; only non-nil fields are applied.
//
// Response: the post-patch tenant billing account view + invalidates the
// cached balance.
func (h *Handlers) patchTenantBillingPlatform(c *gin.Context) {
	id, err := utils.ParseID(c.Param("id"))
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidID))
		return
	}
	if _, err := models.GetActiveTenantByID(h.db, id); err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	var req patchTenantBillingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Render(c, response.Err(response.CodeBadRequest))
		return
	}
	op := middleware.AuditOperator(c)
	patch := models.TenantBillingPatch{
		BillingMode:             req.BillingMode,
		BillingUnlimited:        req.BillingUnlimited,
		PrepaidMinutesRemaining: req.PrepaidMinutesRemaining,
		RechargeMinutes:         req.RechargeMinutes,
		BillingRatePerMinute:    req.BillingRatePerMinute,
		BillingCurrency:         req.BillingCurrency,
		MaxConcurrentCalls:      req.MaxConcurrentCalls,
		DailyMinuteLimit:        req.DailyMinuteLimit,
		MonthlyMinuteLimit:      req.MonthlyMinuteLimit,
		QuotaSuspended:          req.QuotaSuspended,
		MaxUserCount:            req.MaxUserCount,
	}
	if req.LicenseExpiresAt != nil && strings.TrimSpace(*req.LicenseExpiresAt) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.LicenseExpiresAt)); err == nil {
			patch.LicenseExpiresAt = &t
		}
	}
	if err := models.PatchTenantBilling(h.db, id, patch, op); err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "update billing failed", err))
		return
	}
	models.SyncTenantBalance(c.Request.Context(), id)
	t, _ := models.GetActiveTenantByID(h.db, id)
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": models.TenantBillingAccountFrom(t)})
}

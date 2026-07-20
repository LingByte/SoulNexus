package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"github.com/gin-gonic/gin"
)

// registerBillingRoutes mounts the tenant-facing billing endpoints, split
// into read and write branches so the default viewer role cannot finalize
// a bill:
//
//   - GET  /account                        → getTenantBillingAccount    (PermAPIBillingRead)
//   - GET  /usage/summary                  → getBillingUsageSummary     (PermAPIBillingRead)
//   - GET  /usage/events                   → listBillingUsageEvents     (PermAPIBillingRead)
//   - GET  /metrics                        → getBillingBusinessMetrics  (PermAPIBillingRead)
//   - GET  /bills                          → listTenantBills            (PermAPIBillingRead)
//   - GET  /bills/:id                      → getTenantBill              (PermAPIBillingRead)
//   - GET  /bills/:id/export               → exportTenantBill           (PermAPIBillingRead)
//   - POST /bills/:id/finalize             → finalizeTenantBill         (PermAPIBillingWrite)
//   - POST /bills/:id/mark-paid            → markTenantBillPaid         (platform admin only)
func (h *Handlers) registerBillingRoutes(r *humax.Group) {
	read := r.Group("billing")
	read.Use(middleware.RequireTenantPermissionAll(constants.PermAPIBillingRead))
	{
		read.GET("/account", h.getTenantBillingAccount)
		read.GET("/usage/summary", h.getBillingUsageSummary)
		read.GET("/usage/events", h.listBillingUsageEvents)
		read.GET("/metrics", h.getBillingBusinessMetrics)
		read.GET("/bills", h.listTenantBills)
		read.GET("/bills/:id", h.getTenantBill)
		read.GET("/bills/:id/export", h.exportTenantBill)
	}
	write := r.Group("billing")
	write.Use(middleware.RequireTenantPermissionAll(constants.PermAPIBillingWrite))
	{
		write.POST("/bills/:id/finalize", h.finalizeTenantBill)
	}
	platform := r.Group("billing")
	platform.Use(middleware.RequirePlatformAdmin())
	{
		platform.POST("/bills/:id/mark-paid", h.markTenantBillPaid)
	}
}

// listTenantBills paginates bills for the caller's tenant (or all tenants
// for a platform admin).
//
//   - GET /billing/bills
//   - Query: page (int, default 1), size (int, default 20),
//     status (string, optional), period (string, optional),
//     startAt (RFC3339, optional), endAt (RFC3339, optional),
//     tenantId (string, optional — platform admin only).
//
// Response: { list, total, page, size }.
func (h *Handlers) listTenantBills(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	f := models.TenantBillListFilter{
		Status: strings.TrimSpace(c.Query("status")),
		Period: strings.TrimSpace(c.Query("period")),
	}
	if middleware.AuthPlatformAdminID(c) == 0 {
		f.TenantID = middleware.CurrentTenantID(c)
	} else if tidStr := strings.TrimSpace(c.Query("tenantId")); tidStr != "" {
		if tid, err := strconv.ParseUint(tidStr, 10, 64); err == nil {
			f.TenantID = uint(tid)
		}
	}
	f.From = timeutil.ParseOptionalQueryTimePtr(c.Query("startAt"))
	f.To = timeutil.ParseOptionalQueryTimePtr(c.Query("endAt"))
	if f.TenantID > 0 {
		_ = models.EnsureTenantBillsUpToDate(h.db, f.TenantID)
	} else if middleware.AuthPlatformAdminID(c) == 0 {
		_ = models.EnsureTenantBillsUpToDate(h.db, middleware.CurrentTenantID(c))
	}
	rows, total, err := models.ListTenantBills(h.db, page, size, f)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "list bills failed", err))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"list":  rows,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// getTenantBill returns a single bill by id.
//
//   - GET /billing/bills/:id
//   - Path: id (uint) — the bill id; cross-tenant access is rejected for
//     non-platform admins.
//
// Response: the serialized bill.
func (h *Handlers) getTenantBill(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantBill(h.db, id)
	if errors.Is(err, models.ErrTenantBillNotFound) {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "load bill failed", err))
		return
	}
	if middleware.AuthPlatformAdminID(c) == 0 && row.TenantID != middleware.CurrentTenantID(c) {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

// finalizeTenantBill locks a draft bill so no further usage events accrue
// to it.
//
//   - POST /billing/bills/:id/finalize — no body.
//   - Path: id (uint) — the bill id; returns 409 if already finalized.
//
// Response: the finalized bill record.
func (h *Handlers) finalizeTenantBill(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantBill(h.db, id)
	if errors.Is(err, models.ErrTenantBillNotFound) {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "load bill failed", err))
		return
	}
	if middleware.AuthPlatformAdminID(c) == 0 && row.TenantID != middleware.CurrentTenantID(c) {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}

	operator := middleware.AuditOperator(c)
	row, err = models.FinalizeTenantBill(h.db, id, operator)
	if errors.Is(err, models.ErrTenantBillAlreadyLocked) {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "bill already finalized"})
		return
	}
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "finalize bill failed", err))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

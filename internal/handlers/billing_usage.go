package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"github.com/gin-gonic/gin"
	"github.com/LingByte/SoulNexus/pkg/i18n"
)

// getBillingUsageSummary returns the current tenant's billing account summary and
// active quota snapshot (minutes held, monthly quotas, overage flags). Platform
// administrators may query a different tenant by supplying the optional tenantId
// query parameter.
//
// Query parameters:
//   - tenantId (string, uint, optional): Override target tenant ID. Only honoured
//     when the caller authenticates as a platform admin. Ignored otherwise.
//
// Response JSON:
//
//	{
//	  "code":    0,
//	  "msg":     "success",
//	  "data": {
//	    "account": models.TenantBillingAccount,        // plan, balance, billing cycle
//	    "quotas":  []models.TenantQuotaSnapshot        // per-minute-type quota rows
//	  }
//	}
func (h *Handlers) getBillingUsageSummary(c *gin.Context) {
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
	held := models.HeldMinutesForTenant(c.Request.Context(), tenantID)
	quotas, err := models.BuildTenantQuotaSnapshot(h.db, t, held, held)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "load quotas failed", err))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"account": models.TenantBillingAccountFrom(t),
		"quotas":  quotas,
	})
}

func (h *Handlers) listBillingUsageEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	f := models.TenantUsageEventListFilter{}
	if middleware.AuthPlatformAdminID(c) == 0 {
		f.TenantID = middleware.CurrentTenantID(c)
	} else if tidStr := strings.TrimSpace(c.Query("tenantId")); tidStr != "" {
		if tid, err := strconv.ParseUint(tidStr, 10, 64); err == nil {
			f.TenantID = uint(tid)
		}
	}
	f.From = timeutil.ParseOptionalQueryTimePtr(c.Query("startAt"))
	f.To = timeutil.ParseOptionalQueryTimePtr(c.Query("endAt"))
	rows, total, err := models.ListTenantUsageEventsPage(h.db, page, size, f)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "list usage failed", err))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"list": rows, "total": total, "page": page, "size": size})
}

func (h *Handlers) getBillingBusinessMetrics(c *gin.Context) {
	start, end, ok, msgKey := overviewDateRange(c)
	if !ok {
		response.Render(c, response.NewI18n(response.CodeBadRequest, msgKey))
		return
	}
	_ = start
	_ = end
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"calls":   0,
		"minutes": 0,
	})
}

func (h *Handlers) markTenantBillPaid(c *gin.Context) {
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
	row, err = models.MarkTenantBillPaid(h.db, id, operator)
	if errors.Is(err, models.ErrTenantBillAlreadyPaid) {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "bill already paid"})
		return
	}
	if errors.Is(err, models.ErrTenantBillNotFinalized) {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyBillNotFinalized))
		return
	}
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "mark paid failed", err))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

package handlers

import (
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
)

// OpLogEntry is the in-memory representation of an operation-log event
// produced by a handler. It is later written to the database via
// recordOp/recordOpChange and surfaced on the /operation-logs endpoints.
type OpLogEntry struct {
	TenantID     uint
	OperatorKind string
	OperatorID   uint
	Operator     string
	Action       string
	Resource     string
	ResourceID   uint
	ResourceName string
	Summary      string
	Before       any
	After        any
	Detail       any
	Source       string
	Success      bool
	ErrorMsg     string
}

// recordOp writes an OpLogEntry to the operation log for the current HTTP
// request. It resolves the authenticated operator (platform admin,
// credential bearer, tenant user, or system), stamps the request path and
// client IP, and marks the request in the middleware so duplicate log
// entries are not produced by the automatic audit middleware.
func (h *Handlers) recordOp(c *gin.Context, e OpLogEntry) {
	if h.db == nil || c == nil {
		return
	}
	kind, opID, operator := resolveOpLogOperator(c)
	if e.OperatorKind == "" {
		e.OperatorKind = kind
	}
	if e.OperatorID == 0 {
		e.OperatorID = opID
	}
	if strings.TrimSpace(e.Operator) == "" {
		e.Operator = operator
	}
	tenantID := e.TenantID
	if tenantID == 0 {
		tenantID = middleware.CurrentTenantID(c)
	}
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	models.WriteOperationLog(h.db, models.OperationLogInput{
		TenantID:     tenantID,
		OperatorKind: e.OperatorKind,
		OperatorID:   e.OperatorID,
		Operator:     e.Operator,
		Action:       e.Action,
		Resource:     e.Resource,
		ResourceID:   e.ResourceID,
		ResourceName: e.ResourceName,
		Summary:      e.Summary,
		Before:       e.Before,
		After:        e.After,
		Request:      e.Detail,
		Source:       e.Source,
		Success:      e.Success,
		ErrorMsg:     e.ErrorMsg,
		HTTPMethod:   c.Request.Method,
		HTTPPath:     path,
		RequestID:    middleware.ReqIDFromGin(c),
		ClientIP:     c.ClientIP(),
	})
	middleware.MarkOperationLogged(c)
	if e.Success {
		utils.Sig().Emit(constants.SigNotifyOpLog, nil, constants.NotifyOpLogPayload{
			TenantID:     tenantID,
			OperatorKind: e.OperatorKind,
			OperatorID:   e.OperatorID,
			Action:       e.Action,
			Resource:     e.Resource,
			ResourceID:   e.ResourceID,
			ResourceName: e.ResourceName,
			Summary:      e.Summary,
			Success:      e.Success,
		}, h.db)
	}
}

// recordOpChange records a "state change" in the operation log by attaching before
// and after payloads (e.g. the structs produced by a PUT handler). The entry is
// then written through recordOp.
func (h *Handlers) recordOpChange(c *gin.Context, e OpLogEntry, before, after any) {
	e.Before = before
	e.After = after
	h.recordOp(c, e)
}

func resolveOpLogOperator(c *gin.Context) (kind string, id uint, label string) {
	if pid := middleware.AuthPlatformAdminID(c); pid > 0 {
		if s := strings.TrimSpace(middleware.AuthEmail(c)); s != "" {
			return constants.OpOperatorPlatformAdmin, pid, s
		}
		return constants.OpOperatorPlatformAdmin, pid, "platform_admin:" + fmt.Sprintf("%d", pid)
	}
	if cid := middleware.AuthCredentialID(c); cid > 0 {
		return constants.OpOperatorCredential, cid, "credential:" + fmt.Sprintf("%d", cid)
	}
	if uid := middleware.AuthUserID(c); uid > 0 {
		if s := strings.TrimSpace(middleware.AuthEmail(c)); s != "" {
			return constants.OpOperatorTenantUser, uid, s
		}
		return constants.OpOperatorTenantUser, uid, fmt.Sprintf("%d", uid)
	}
	return constants.OpOperatorSystem, 0, middleware.AuditOperator(c)
}

package handlers

import (
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) recordLoginHistory(c *gin.Context, in models.LoginHistoryInput) {
	_ = models.RecordLoginHistory(h.db, in)
}

func (h *Handlers) recordLoginSuccess(c *gin.Context, in models.LoginHistoryInput) {
	if strings.TrimSpace(in.ClientIP) == "" {
		in.ClientIP = c.ClientIP()
	}
	if strings.TrimSpace(in.UserAgent) == "" {
		in.UserAgent = clientUserAgent(c)
	}
	in.Success = true
	in.Operator = loginHistoryOperator(in)
	h.recordLoginHistory(c, in)
}

func (h *Handlers) recordLoginFailure(c *gin.Context, in models.LoginHistoryInput, reason string) {
	if strings.TrimSpace(in.ClientIP) == "" {
		in.ClientIP = c.ClientIP()
	}
	if strings.TrimSpace(in.UserAgent) == "" {
		in.UserAgent = clientUserAgent(c)
	}
	if strings.TrimSpace(in.City) == "" && strings.TrimSpace(in.ClientIP) != "" {
		geo := utils.LoginGeoFromIP(in.ClientIP)
		in.City = geo.City
		in.Location = geo.Location
	}
	in.Success = false
	in.FailureReason = strings.TrimSpace(reason)
	in.Operator = loginHistoryOperator(in)
	h.recordLoginHistory(c, in)
}

func loginHistoryOperator(in models.LoginHistoryInput) string {
	if email := strings.TrimSpace(in.Email); email != "" {
		return email
	}
	if in.PrincipalID > 0 {
		return "principal:" + strings.TrimSpace(in.PrincipalType)
	}
	return "system:login"
}

func (h *Handlers) listMyLoginHistory(c *gin.Context) {
	principalType, principalID, ok := authenticatedDevicePrincipal(c)
	if !ok {
		return
	}
	page, size := ginutil.QueryPage(c, 50)
	list, total, err := models.ListLoginHistoryPage(h.db, principalType, principalID, page, size)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	ginutil.PageSuccess(c, list, total, page, size)
}

func clientUserAgent(c *gin.Context) string {
	return strings.TrimSpace(c.GetHeader("User-Agent"))
}

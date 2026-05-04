package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// requireAccessManage 角色/权限管理（内置 key 仍为 rbac.manage，与库内数据一致）。
func (h *Handlers) requireAccessManage(c *gin.Context) {
	u := models.CurrentUser(c)
	if u == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("authorization required"))
		return
	}
	if models.UserHasPermission(h.db, u.ID, models.PermWildcard) ||
		models.UserHasPermission(h.db, u.ID, models.PermManageRoles) {
		c.Next()
		return
	}
	if u.IsSuperAdmin() || strings.EqualFold(strings.TrimSpace(u.Role), models.RoleSuperAdmin) {
		c.Next()
		return
	}
	response.AbortWithJSONError(c, http.StatusForbidden, errors.New("role and permission management required"))
}

package handlers

import (
	authmodel "github.com/LingByte/SoulNexus/internal/models/auth"
	"errors"
	"net/http"

	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// requireAccessManage 角色/权限管理（内置 key 仍为 rbac.manage，与库内数据一致）。
func (h *Handlers) requireAccessManage(c *gin.Context) {
	u := authmodel.CurrentUser(c)
	if u == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("authorization required"))
		return
	}
	if authmodel.UserHasPermission(h.db, u.ID, authmodel.PermWildcard) ||
		authmodel.UserHasPermission(h.db, u.ID, authmodel.PermManageRoles) {
		c.Next()
		return
	}
	response.AbortWithJSONError(c, http.StatusForbidden, errors.New("role and permission management required"))
}

package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	apiresponse "github.com/LingByte/SoulNexus/internal/response"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// registerTenantOrgRoutes mounts permissions / groups / roles sub-routes
// under the tenant-org prefix. Access is split by constant permission
// codes so a read-only manager cannot mutate roles:
//
//   - GET /permissions              → listOrgPermissions        (PermAPITenantOrgRead)
//   - GET /groups                   → listOrgGroups            (PermAPITenantOrgRead)
//   - GET /roles                    → listOrgRoles             (PermAPITenantOrgRead)
//   - GET /roles/:id                → getOrgRole               (PermAPITenantOrgRead)
//   - POST /groups                  → createOrgGroup           (PermAPITenantOrgWrite)
//   - PUT /groups/:id               → updateOrgGroup           (PermAPITenantOrgWrite)
//   - DELETE /groups/:id            → deleteOrgGroup           (PermAPITenantOrgWrite)
//   - POST /roles                   → createOrgRole            (PermAPITenantOrgWrite)
//   - PUT /roles/:id                → updateOrgRole            (PermAPITenantOrgWrite)
//   - DELETE /roles/:id             → deleteOrgRole            (PermAPITenantOrgWrite)
//   - PUT /roles/:id/permissions    → putOrgRolePermissions    (PermAPITenantOrgWrite)
//   - PUT /users/:userId/roles      → putOrgTenantUserRoles    (PermAPITenantOrgWrite)
//   - PUT /users/:userId/groups     → putOrgTenantUserGroups   (PermAPITenantOrgWrite)
func (h *Handlers) registerTenantOrgRoutes(g *humax.Group) {
	org := g.Group("tenant-org")
	read := org.Group("")
	read.Use(middleware.RequireTenantPermissionAll(constants.PermAPITenantOrgRead))
	{
		read.GET("/permissions", h.listOrgPermissions)
		read.GET("/groups", h.listOrgGroups)
		read.GET("/roles", h.listOrgRoles)
		read.GET("/roles/:id", h.getOrgRole)
	}
	write := org.Group("")
	write.Use(middleware.RequireTenantPermissionAll(constants.PermAPITenantOrgWrite))
	{
		write.POST("/groups", h.createOrgGroup)
		write.PUT("/groups/:id", h.updateOrgGroup)
		write.DELETE("/groups/:id", h.deleteOrgGroup)

		write.POST("/roles", h.createOrgRole)
		write.PUT("/roles/:id", h.updateOrgRole)
		write.DELETE("/roles/:id", h.deleteOrgRole)
		write.PUT("/roles/:id/permissions", h.putOrgRolePermissions)

		write.PUT("/users/:userId/roles", h.putOrgTenantUserRoles)
		write.PUT("/users/:userId/groups", h.putOrgTenantUserGroups)
	}
}

// listOrgPermissions enumerates every permission available in the tenant RBAC
// model. Clients use this list to build role-editing UIs.
//
//   - GET /tenant-org/permissions — no params.
//
// Response: { list: [ { id, code, name, description, kind, parentCode, resource, action } ] }.
func (h *Handlers) listOrgPermissions(c *gin.Context) {
	if _, ok := ginutil.RequireAuthTenant(c); !ok {
		return
	}
	rows, err := models.ListAllPermissions(h.db)
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	pub := make([]gin.H, 0, len(rows))
	for _, p := range rows {
		pub = append(pub, gin.H{
			"id":          utils.FormatID(p.ID),
			"code":        p.Code,
			"name":        p.Name,
			"description": p.Description,
			"kind":        p.Kind,
			"parentCode":  p.ParentCode,
			"resource":    p.Resource,
			"action":      p.Action,
		})
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"list": pub})
}

// listOrgGroups returns all departments defined for the caller's tenant.
//
//   - GET /tenant-org/groups — no params.
//
// Response: { list: [ { id, name, isDefault } ] }.
func (h *Handlers) listOrgGroups(c *gin.Context) {
	tid, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	rows, err := models.ListTenantGroupsForTenant(h.db, tid)
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	pub := make([]gin.H, 0, len(rows))
	for _, g := range rows {
		pub = append(pub, gin.H{"id": utils.FormatID(g.ID), "name": g.Name, "isDefault": g.IsDefault})
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"list": pub})
}

type orgGroupWriteReq struct {
	Name      string `json:"name" binding:"required,min=1,max=128"`
	IsDefault bool   `json:"isDefault"`
}

// createOrgGroup creates a new department in the caller's tenant.
//
//   - POST /tenant-org/groups — body: { name, isDefault }
//
// Setting isDefault=true demotes the current default group to plain groups.
// Response: { id, name, isDefault }.
func (h *Handlers) createOrgGroup(c *gin.Context) {
	tid, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	var req orgGroupWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	op := middleware.AuditOperator(c)
	g := &models.TenantGroup{TenantID: tid, Name: name, IsDefault: req.IsDefault}
	g.SetCreateInfo(op)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if req.IsDefault {
			if err := tx.Model(&models.TenantGroup{}).
				Where("tenant_id = ?", tid).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return models.CreateTenantGroupRecord(tx, g)
	})
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionCreate,
		Resource: constants.OpResourceTenantGroup, ResourceID: g.ID, ResourceName: g.Name,
		Summary: fmt.Sprintf("Created department %s", g.Name), Detail: audit.Redact(req),
	}, nil, g)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": utils.FormatID(g.ID), "name": g.Name, "isDefault": g.IsDefault})
}

// updateOrgGroup changes a department's name and optionally promotes it
// to the default group (demoting the current default).
//
//   - PUT /tenant-org/groups/:id — body: { name, isDefault }
//
// Response: { id, name, isDefault }.
func (h *Handlers) updateOrgGroup(c *gin.Context) {
	tid, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var req orgGroupWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	var row models.TenantGroup
	if err := h.db.Where("id = ? AND tenant_id = ?", id, tid).
		First(&row).Error; err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	name := strings.TrimSpace(req.Name)
	before := row
	op := middleware.AuditOperator(c)
	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		if req.IsDefault {
			if err := tx.Model(&models.TenantGroup{}).
				Where("tenant_id = ?", tid).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		u := map[string]any{
			"name":       name,
			"is_default": req.IsDefault,
		}
		meta := common.BaseModel{}
		meta.SetUpdateInfo(op)
		if meta.UpdateBy != "" {
			u["update_by"] = meta.UpdateBy
		}
		return tx.Model(&models.TenantGroup{}).Where("id = ?", row.ID).Updates(u).Error
	})
	if txErr != nil {
		ginutil.WriteInternalError(c, txErr)
		return
	}
	var after models.TenantGroup
	_ = h.db.Where("id = ? AND tenant_id = ?", row.ID, tid).First(&after).Error
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionUpdate,
		Resource: constants.OpResourceTenantGroup, ResourceID: row.ID, ResourceName: before.Name,
		Summary: fmt.Sprintf("Updated department %s", name), Detail: audit.Redact(req),
	}, before, after)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": utils.FormatID(row.ID), "name": name, "isDefault": req.IsDefault})
}

// deleteOrgGroup soft-deletes a department. Members attached to the group
// are NOT automatically re-assigned.
//
//   - DELETE /tenant-org/groups/:id — no body.
//
// Response: { id } as a snowflake string.
func (h *Handlers) deleteOrgGroup(c *gin.Context) {
	tid, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var row models.TenantGroup
	_ = h.db.Where("id = ? AND tenant_id = ?", id, tid).First(&row).Error
	if err := models.SoftDeleteTenantGroup(h.db, tid, id, middleware.AuditOperator(c)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionDelete,
		Resource: constants.OpResourceTenantGroup, ResourceID: id, ResourceName: row.Name,
		Summary: fmt.Sprintf("Deleted department %s", row.Name),
	}, row, nil)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": utils.FormatID(id)})
}

// listOrgRoles returns the role catalog for the caller's tenant.
//
//   - GET /tenant-org/roles — no params.
//
// Response: { list: [ { id, name, description, isSystem } ] }. System roles
// cannot be renamed or deleted by tenants.
func (h *Handlers) listOrgRoles(c *gin.Context) {
	tid, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	rows, err := models.ListTenantRolesByTenant(h.db, tid)
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	pub := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		pub = append(pub, gin.H{
			"id":          utils.FormatID(r.ID),
			"name":        r.Name,
			"description": r.Description,
			"isSystem":    r.IsSystem,
		})
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"list": pub})
}

// getOrgRole returns a role together with its assigned permission IDs.
//
//   - GET /tenant-org/roles/:id — no body.
//
// Response: { id, name, description, isSystem, permissionIds: [ ... ] }.
func (h *Handlers) getOrgRole(c *gin.Context) {
	tid, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	r, err := models.GetTenantRoleScoped(h.db, tid, id)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	permIDs, err := models.ListPermissionIDsForRole(h.db, r.ID)
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"id":            utils.FormatID(r.ID),
		"name":          r.Name,
		"description":   r.Description,
		"isSystem":      r.IsSystem,
		"permissionIds": utils.FormatIDs(permIDs),
	})
}

type orgRoleCreateReq struct {
	Name        string `json:"name" binding:"required,min=1,max=128"`
	Description string `json:"description"`
}

// createOrgRole defines a new custom role.
//
//   - POST /tenant-org/roles — body: { name, description }
//
// Response: { id, name, description, isSystem: false }.
func (h *Handlers) createOrgRole(c *gin.Context) {
	tid, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	var req orgRoleCreateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	r := &models.TenantRole{
		TenantID:    tid,
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		IsSystem:    false,
	}
	r.SetCreateInfo(middleware.AuditOperator(c))
	if err := models.CreateTenantRole(h.db, r); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionCreate,
		Resource: constants.OpResourceTenantRole, ResourceID: r.ID, ResourceName: r.Name,
		Summary: fmt.Sprintf("Created role %s", r.Name), Detail: audit.Redact(req),
	}, nil, r)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": utils.FormatID(r.ID), "name": r.Name, "description": r.Description, "isSystem": false})
}

type orgRoleUpdateReq struct {
	Name        string `json:"name" binding:"required,min=1,max=128"`
	Description string `json:"description"`
}

// updateOrgRole updates a role's name/description. System roles cannot be
// renamed (only their description can change).
//
//   - PUT /tenant-org/roles/:id — body: { name, description }
//
// Response: { id }.
func (h *Handlers) updateOrgRole(c *gin.Context) {
	tid, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var req orgRoleUpdateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	r, err := models.GetTenantRoleScoped(h.db, tid, id)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	if r.IsSystem && strings.TrimSpace(req.Name) != r.Name {
		response.Render(c, response.NewI18n(response.CodeForbidden, i18n.KeySystemRoleCannotRename))
		return
	}
	before := r
	op := middleware.AuditOperator(c)
	u := map[string]any{
		"name":        strings.TrimSpace(req.Name),
		"description": strings.TrimSpace(req.Description),
	}
	meta := common.BaseModel{}
	meta.SetUpdateInfo(op)
	if meta.UpdateBy != "" {
		u["update_by"] = meta.UpdateBy
	}
	if err := h.db.Model(&models.TenantRole{}).Where("id = ?", r.ID).Updates(u).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	after, _ := models.GetTenantRoleScoped(h.db, tid, r.ID)
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionUpdate,
		Resource: constants.OpResourceTenantRole, ResourceID: r.ID, ResourceName: before.Name,
		Summary: fmt.Sprintf("Updated role %s", after.Name), Detail: audit.Redact(req),
	}, before, after)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": utils.FormatID(r.ID)})
}

// deleteOrgRole removes a custom role. System roles are rejected with 403.
//
//   - DELETE /tenant-org/roles/:id — no body.
//
// Response: { id }.
func (h *Handlers) deleteOrgRole(c *gin.Context) {
	tid, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	r, err := models.GetTenantRoleScoped(h.db, tid, id)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	if r.IsSystem {
		response.Render(c, response.NewI18n(response.CodeForbidden, i18n.KeySystemRoleCannotDelete))
		return
	}
	if err := models.SoftDeleteTenantRole(h.db, tid, r.ID, middleware.AuditOperator(c)); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionDelete,
		Resource: constants.OpResourceTenantRole, ResourceID: r.ID, ResourceName: r.Name,
		Summary: fmt.Sprintf("Deleted role %s", r.Name),
	}, r, nil)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": utils.FormatID(r.ID)})
}

type orgRolePermissionsReq struct {
	PermissionIDs []string `json:"permissionIds"`
}

// putOrgRolePermissions replaces the permission set of a role. Invalid IDs
// surface an i18n error. The admin role (a system role) cannot be edited.
//
//   - PUT /tenant-org/roles/:id/permissions — body: { permissionIds: [...] }
//
// Response: { roleId }.
func (h *Handlers) putOrgRolePermissions(c *gin.Context) {
	tid, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	r, err := models.GetTenantRoleScoped(h.db, tid, id)
	if err != nil {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	if r.IsSystem && r.Name == constants.TenantAdminRoleName {
		response.FailI18n(c, i18n.KeyOrgAdminRoleFixed, nil)
		return
	}
	var req orgRolePermissionsReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	permIDs, err := utils.ParseIDStrings(req.PermissionIDs)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidPermissionID))
		return
	}
	beforeIDs, _ := models.ListPermissionIDsForRole(h.db, id)
	if err := models.ReplaceTenantRolePermissions(h.db, id, permIDs, middleware.AuditOperator(c)); err != nil {
		if errors.Is(err, models.ErrInvalidOrgReference) {
			response.FailI18n(c, i18n.KeyOrgInvalidPermID, nil)
			return
		}
		ginutil.WriteInternalError(c, err)
		return
	}
	if uids, err := models.ListTenantUserIDsByRoleID(h.db, id); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	} else {
		for _, uid := range uids {
			middleware.InvalidatePermissionCache(uid)
		}
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionUpdate,
		Resource: constants.OpResourceTenantRole, ResourceID: id, ResourceName: r.Name,
		Summary: fmt.Sprintf("Updated permissions for role %s", r.Name), Detail: audit.Redact(req),
	}, gin.H{"permissionIds": utils.FormatIDs(beforeIDs)}, gin.H{"permissionIds": req.PermissionIDs})
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"roleId": utils.FormatID(id)})
}

type orgUserRolesReq struct {
	RoleIDs []string `json:"roleIds"`
}

// putOrgTenantUserRoles replaces the role set assigned to a member.
//
//   - PUT /tenant-org/users/:userId/roles — body: { roleIds: [...] }
//
// Response is the updated member public view; permission caches for the
// user are invalidated server-side.
func (h *Handlers) putOrgTenantUserRoles(c *gin.Context) {
	tid, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	uid, ok := ginutil.ParamID(c, "userId")
	if !ok {
		return
	}
	u, err := models.GetActiveTenantUserByID(h.db, uid)
	if err != nil || u.TenantID != tid {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	var req orgUserRolesReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	roleIDs, err := utils.ParseIDStrings(req.RoleIDs)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidRoleID))
		return
	}
	beforeRoles, _ := models.ListTenantRolesForUser(h.db, u.ID)
	if err := models.ReplaceTenantUserRoles(h.db, tid, u.ID, roleIDs, middleware.AuditOperator(c)); err != nil {
		if errors.Is(err, models.ErrInvalidOrgReference) {
			response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidRoleID))
			return
		}
		ginutil.WriteInternalError(c, err)
		return
	}
	middleware.InvalidatePermissionCache(u.ID)
	afterRoles, _ := models.ListTenantRolesForUser(h.db, u.ID)
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionUpdate,
		Resource: constants.OpResourceTenantUser, ResourceID: u.ID, ResourceName: u.Email,
		Summary: fmt.Sprintf("Updated roles for member %s", u.Email), Detail: audit.Redact(req),
	}, beforeRoles, afterRoles)
	next, err := models.GetActiveTenantUserByID(h.db, u.ID)
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewTenantUserResponse(h.db, next))
}

type orgUserGroupsReq struct {
	GroupIDs []string `json:"groupIds"`
}

// putOrgTenantUserGroups replaces the departments a member belongs to.
//
//   - PUT /tenant-org/users/:userId/groups — body: { groupIds: [...] }
//
// Response is the updated member public view.
func (h *Handlers) putOrgTenantUserGroups(c *gin.Context) {
	tid, ok := ginutil.RequireAuthTenant(c)
	if !ok {
		return
	}
	uid, ok := ginutil.ParamID(c, "userId")
	if !ok {
		return
	}
	u, err := models.GetActiveTenantUserByID(h.db, uid)
	if err != nil || u.TenantID != tid {
		response.Render(c, response.Err(response.CodeNotFound))
		return
	}
	var req orgUserGroupsReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	groupIDs, err := utils.ParseIDStrings(req.GroupIDs)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidDepartmentID))
		return
	}
	beforeGroups, _ := models.ListTenantGroupsForUser(h.db, u.ID)
	if err := models.ReplaceTenantUserGroups(h.db, tid, u.ID, groupIDs, middleware.AuditOperator(c)); err != nil {
		if errors.Is(err, models.ErrInvalidOrgReference) {
			response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidDepartmentID))
			return
		}
		ginutil.WriteInternalError(c, err)
		return
	}
	afterGroups, _ := models.ListTenantGroupsForUser(h.db, u.ID)
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionUpdate,
		Resource: constants.OpResourceTenantUser, ResourceID: u.ID, ResourceName: u.Email,
		Summary: fmt.Sprintf("Updated departments for member %s", u.Email), Detail: audit.Redact(req),
	}, beforeGroups, afterGroups)
	next, err := models.GetActiveTenantUserByID(h.db, u.ID)
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, apiresponse.NewTenantUserResponse(h.db, next))
}

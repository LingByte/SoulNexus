package handlers

import (
	"strconv"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"github.com/LingByte/SoulNexus/pkg/i18n"
)

// listAssistantVersions returns the immutable published snapshots for an
// assistant ordered by version id desc.
//
// Path parameters:
//   - id (uint): assistant id
//
// Response: { code: 0, msg: "success", data: [AssistantVersion] }
func (h *Handlers) listAssistantVersions(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	ast, err := h.getAssistantRow(c, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	list, err := models.ListAssistantVersions(h.db, ast.ID)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, list)
}

// publishAssistant freezes the current draft config into a new immutable
// AssistantVersion row. This is a safe operation: it never modifies an
// existing version.
//
// Path parameters:
//   - id (uint): assistant id
//
// Response: { code: 0, msg: "success", data: { version: AssistantVersion } }
func (h *Handlers) publishAssistant(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	ast, err := h.getAssistantRow(c, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	ver, err := models.PublishAssistantVersion(h.db, ast, middleware.AuditOperator(c))
	if ginutil.WriteInternalError(c, err) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"version": ver})
}

type assistantRollbackReq struct {
	VersionID string `json:"versionId" binding:"required"`
}

// rollbackAssistant reverts an assistant's draft configuration to match a
// previously published version. Useful to undo a broken draft.
//
// Path parameters:
//   - id (uint): assistant id
//
// Request body: { versionId: string } — id of the version to restore
//
// Response: { code: 0, msg: "success", data: AssistantVersion }
func (h *Handlers) rollbackAssistant(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	var req assistantRollbackReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	verID, err := utils.ParseID(req.VersionID)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidVersionID))
		return
	}
	ast, err := h.getAssistantRow(c, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	ver, err := models.GetAssistantVersion(h.db, ast.ID, verID)
	if ginutil.WriteGORMError(c, err, "version not found") {
		return
	}
	after, err := models.RollbackAssistantToVersion(h.db, ast, ver, middleware.AuditOperator(c))
	if ginutil.WriteInternalError(c, err) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, after)
}

// diffAssistantVersions returns the top-level key differences between two
// snapshots. If both `from` and `to` are omitted, the draft config is diff'd
// against the currently published (resolved) config.
//
// Path parameters:
//   - id (uint): assistant id
//
// Query parameters:
//   - from (string, uint): start version id
//   - to (string, uint): end version id
//
// Response: { code: 0, msg: "success", data: { changedKeys: [...], from: _, to: _ } }
func (h *Handlers) diffAssistantVersions(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	ast, err := h.getAssistantRow(c, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	fromQ, toQ := c.Query("from"), c.Query("to")
	if fromQ == "" && toQ == "" {
		spec, _ := models.ResolveAssistantSpec(h.db, ast)
		pubSpec, perr := models.AssistantResolvedSnapshot(spec)
		draftSpec, derr := models.AssistantConfigSnapshot(ast)
		if perr != nil || derr != nil {
			response.Render(c, response.NewI18n(response.CodeInternal, i18n.KeyDiffFailed))
			return
		}
		changed, diffErr := models.DiffAssistantConfigs(draftSpec, pubSpec)
		if diffErr != nil {
			response.Render(c, response.Wrap(response.CodeInternal, "diff failed", diffErr))
			return
		}
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{
			"changedKeys":  changed,
			"from":         "draft",
			"to":           "published",
			"fromSnapshot": draftSpec,
			"toSnapshot":   pubSpec,
		})
		return
	}
	fromID, err1 := strconv.ParseUint(fromQ, 10, 64)
	toID, err2 := strconv.ParseUint(toQ, 10, 64)
	if err1 != nil || err2 != nil || fromID == 0 || toID == 0 {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyFromToVersionIdsRequired))
		return
	}
	fromVer, err := models.GetAssistantVersion(h.db, ast.ID, uint(fromID))
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyVersionNotFound))
		return
	}
	toVer, err := models.GetAssistantVersion(h.db, ast.ID, uint(toID))
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyVersionNotFound))
		return
	}
	fromSnap, _ := models.AssistantVersionSnapshot(fromVer)
	toSnap, _ := models.AssistantVersionSnapshot(toVer)
	changed, err := models.DiffAssistantConfigs(fromSnap, toSnap)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "diff failed", err))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"changedKeys":  changed,
		"from":         fromVer,
		"to":           toVer,
		"fromSnapshot": fromSnap,
		"toSnapshot":   toSnap,
	})
}

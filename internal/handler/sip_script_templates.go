package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

type sipScriptTemplateWriteReq struct {
	Name        string `json:"name"`
	ScriptID    string `json:"scriptId"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled"`
	ScriptSpec  string `json:"scriptSpec"`
}

func (h *Handlers) listSIPScriptTemplates(c *gin.Context) {
	page, size := parsePageSize(c)
	list, total, err := models.ListSIPScriptTemplatesPage(h.db, page, size, c.Query("scriptId"), c.Query("name"))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", gin.H{"list": list, "total": total, "page": page, "size": size})
}

func (h *Handlers) getSIPScriptTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	row, err := models.GetActiveSIPScriptTemplateByID(h.db, uint(id))
	if err != nil {
		response.Fail(c, "not found", nil)
		return
	}
	response.Success(c, "success", row)
}

func (h *Handlers) createSIPScriptTemplate(c *gin.Context) {
	var req sipScriptTemplateWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "invalid body", err.Error())
		return
	}
	spec, err := models.ParseScriptTemplateSpec(req.ScriptSpec)
	if err != nil {
		response.Fail(c, "invalid scriptSpec JSON", err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		response.Fail(c, "name required", nil)
		return
	}
	scriptID := strings.TrimSpace(req.ScriptID)
	if scriptID == "" {
		scriptID = models.RandomScriptTemplateID()
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := models.NewSIPScriptTemplateForCreate(
		name,
		scriptID,
		strings.TrimSpace(req.Version),
		strings.TrimSpace(req.Description),
		enabled,
		spec,
	)
	if op := acdOperator(c); op != "" {
		row.SetCreateInfo(op)
	}
	if err := h.db.Create(&row).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, "success", row)
}

func (h *Handlers) updateSIPScriptTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	var req sipScriptTemplateWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "invalid body", err.Error())
		return
	}
	row, err := models.GetActiveSIPScriptTemplateByID(h.db, uint(id))
	if err != nil {
		response.Fail(c, "not found", nil)
		return
	}
	updateBy := acdOperator(c)
	updates, err := models.BuildSIPScriptTemplateUpdates(
		row,
		req.Name,
		req.ScriptID,
		req.Version,
		req.Description,
		req.Enabled,
		req.ScriptSpec,
		updateBy,
	)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}
	if err := h.db.Model(&row).Updates(updates).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	row, _ = models.ReloadSIPScriptTemplateByID(h.db, uint(id))
	response.Success(c, "success", row)
}

func (h *Handlers) deleteSIPScriptTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	n, err := models.SoftDeleteSIPScriptTemplateByID(h.db, uint(id), acdOperator(c))
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if n == 0 {
		response.Fail(c, "not found", nil)
		return
	}
	response.Success(c, "success", gin.H{"id": id})
}

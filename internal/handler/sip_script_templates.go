package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type sipScriptTemplateWriteReq struct {
	Name        string `json:"name"`
	ScriptID    string `json:"scriptId"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled"`
	ScriptSpec  string `json:"scriptSpec"`
}

func genScriptID() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("script_%d", timeNowUnix())
	}
	return "script_" + strings.ToLower(hex.EncodeToString(buf))
}

func timeNowUnix() int64 {
	return time.Now().Unix()
}

func validateScriptSpec(raw string) (datatypes.JSON, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("scriptSpec is empty")
	}
	var probe map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return nil, err
	}
	return datatypes.JSON([]byte(raw)), nil
}

func (h *Handlers) listSIPScriptTemplates(c *gin.Context) {
	page, size := parsePageSize(c)
	q := h.db.Model(&models.SIPScriptTemplate{}).Where("is_deleted = ?", models.SoftDeleteStatusActive)
	if s := strings.TrimSpace(c.Query("scriptId")); s != "" {
		q = q.Where("script_id = ?", s)
	}
	if name := strings.TrimSpace(c.Query("name")); name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	var list []models.SIPScriptTemplate
	offset := (page - 1) * size
	if err := q.Order("id DESC").Offset(offset).Limit(size).Find(&list).Error; err != nil {
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
	var row models.SIPScriptTemplate
	if err := h.db.Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).First(&row).Error; err != nil {
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
	spec, err := validateScriptSpec(req.ScriptSpec)
	if err != nil {
		response.Fail(c, "invalid scriptSpec JSON", err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	scriptID := strings.TrimSpace(req.ScriptID)
	version := strings.TrimSpace(req.Version)
	if name == "" {
		response.Fail(c, "name required", nil)
		return
	}
	if scriptID == "" {
		scriptID = genScriptID()
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := models.SIPScriptTemplate{
		Name:        name,
		ScriptID:    scriptID,
		Version:     version,
		Description: strings.TrimSpace(req.Description),
		Enabled:     enabled,
		ScriptSpec:  spec,
	}
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
	var row models.SIPScriptTemplate
	if err := h.db.Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).First(&row).Error; err != nil {
		response.Fail(c, "not found", nil)
		return
	}
	updates := map[string]interface{}{
		"name":        strings.TrimSpace(req.Name),
		"script_id":   strings.TrimSpace(req.ScriptID),
		"version":     strings.TrimSpace(req.Version),
		"description": strings.TrimSpace(req.Description),
	}
	if strings.TrimSpace(req.Name) == "" {
		response.Fail(c, "name required", nil)
		return
	}
	if strings.TrimSpace(req.ScriptID) == "" {
		updates["script_id"] = row.ScriptID
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if strings.TrimSpace(req.ScriptSpec) != "" {
		spec, err := validateScriptSpec(req.ScriptSpec)
		if err != nil {
			response.Fail(c, "invalid scriptSpec JSON", err.Error())
			return
		}
		updates["script_spec"] = spec
	}
	if op := acdOperator(c); op != "" {
		updates["update_by"] = op
	}
	if err := h.db.Model(&row).Updates(updates).Error; err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	_ = h.db.Where("id = ?", id).First(&row).Error
	response.Success(c, "success", row)
}

func (h *Handlers) deleteSIPScriptTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}
	updates := map[string]interface{}{"is_deleted": models.SoftDeleteStatusDeleted}
	if op := acdOperator(c); op != "" {
		updates["update_by"] = op
	}
	res := h.db.Model(&models.SIPScriptTemplate{}).Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).Updates(updates)
	if res.Error != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		response.Fail(c, "not found", nil)
		return
	}
	response.Success(c, "success", gin.H{"id": id})
}


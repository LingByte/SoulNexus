// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/chat"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
)

type dialogSkillWriteReq struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Kind          string `json:"kind"`
	Body          string `json:"body"`
	ScriptContent string `json:"scriptContent"`
	EntryFile     string `json:"entryFile"`
	Enabled       *bool  `json:"enabled"`
}

func dialogSkillResp(row models.TenantDialogSkill) gin.H {
	return gin.H{
		"id":            row.ID,
		"tenantId":      row.TenantID,
		"code":          row.Code,
		"name":          row.Name,
		"description":   row.Description,
		"kind":          row.Kind,
		"body":          row.Body,
		"scriptContent": row.ScriptContent,
		"entryFile":     row.EntryFile,
		"hasAssets":     row.HasAssets,
		"enabled":       row.Enabled,
		"toolName":      "run_skill_" + models.SkillToolSuffix(row.Code),
		"createdAt":     row.CreatedAt,
		"updatedAt":     row.UpdatedAt,
	}
}

func writeDialogSkillValidationError(c *gin.Context, err error) {
	response.Render(c, response.New(response.CodeBadRequest, err.Error()))
}

func (h *Handlers) listDialogSkills(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	enabledOnly := c.Query("enabled") == "1" || c.Query("enabled") == "true"
	if c.Query("page") != "" || c.Query("size") != "" {
		page, size := ginutil.QueryPage(c, 50)
		list, total, err := models.ListTenantDialogSkillsPage(h.db, tid, page, size)
		if ginutil.WriteInternalError(c, err) {
			return
		}
		out := make([]gin.H, 0, len(list))
		for _, row := range list {
			out = append(out, dialogSkillResp(row))
		}
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{"list": out, "total": total, "page": page, "size": size})
		return
	}
	rows, err := models.ListTenantDialogSkills(h.db, tid, enabledOnly)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, dialogSkillResp(row))
	}
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

func (h *Handlers) getDialogSkill(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantDialogSkill(h.db, tid, id)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, dialogSkillResp(row))
}

func (h *Handlers) createDialogSkill(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	var req dialogSkillWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := &models.TenantDialogSkill{
		TenantID:      tid,
		Code:          req.Code,
		Name:          req.Name,
		Description:   req.Description,
		Kind:          req.Kind,
		Body:          req.Body,
		ScriptContent: req.ScriptContent,
		EntryFile:     req.EntryFile,
		Enabled:       enabled,
	}
	if err := models.CreateTenantDialogSkill(h.db, row); err != nil {
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
			writeDialogSkillValidationError(c, err)
			return
		}
		ginutil.WriteInternalError(c, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, dialogSkillResp(*row))
}

func (h *Handlers) updateDialogSkill(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	existing, err := models.GetTenantDialogSkill(h.db, tid, id)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	var req dialogSkillWriteReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	// Re-validate merged row for kind/script rules.
	merged := existing
	if strings.TrimSpace(req.Code) != "" {
		merged.Code = req.Code
	}
	if strings.TrimSpace(req.Name) != "" {
		merged.Name = req.Name
	}
	merged.Description = strings.TrimSpace(req.Description)
	if strings.TrimSpace(req.Kind) != "" {
		merged.Kind = req.Kind
	}
	if req.Body != "" || models.NormalizeDialogSkillKind(merged.Kind) == models.DialogSkillKindPrompt {
		merged.Body = req.Body
	}
	if req.ScriptContent != "" || req.Kind != "" {
		merged.ScriptContent = req.ScriptContent
	}
	if strings.TrimSpace(req.EntryFile) != "" {
		merged.EntryFile = req.EntryFile
	}
	if req.Enabled != nil {
		merged.Enabled = *req.Enabled
	}
	if err := models.ValidateTenantDialogSkill(&merged); err != nil {
		writeDialogSkillValidationError(c, err)
		return
	}
	updates := map[string]any{
		"code":           merged.Code,
		"name":           merged.Name,
		"description":    merged.Description,
		"kind":           merged.Kind,
		"body":           merged.Body,
		"script_content": merged.ScriptContent,
		"entry_file":     merged.EntryFile,
		"enabled":        merged.Enabled,
	}
	if ginutil.WriteInternalError(c, models.UpdateTenantDialogSkill(h.db, tid, id, updates)) {
		return
	}
	row, _ := models.GetTenantDialogSkill(h.db, tid, id)
	response.SuccessI18n(c, i18n.KeySuccess, dialogSkillResp(row))
}

func (h *Handlers) deleteDialogSkill(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantDialogSkill(h.db, tid, id)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	_ = os.RemoveAll(chat.SkillAssetsDir(tid, row.Code))
	if ginutil.WriteInternalError(c, models.DeleteTenantDialogSkill(h.db, tid, id)) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"ok": true})
}

func (h *Handlers) uploadDialogSkillAssets(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetTenantDialogSkill(h.db, tid, id)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	kind := models.NormalizeDialogSkillKind(row.Kind)
	if kind == models.DialogSkillKindPrompt {
		writeDialogSkillValidationError(c, fmt.Errorf("prompt skills do not accept script assets; set kind to python or node"))
		return
	}
	fh, ferr := c.FormFile("file")
	if ferr != nil || fh == nil {
		response.Render(c, response.New(response.CodeBadRequest, "multipart file required (zip)"))
		return
	}
	if fh.Size > 20<<20 {
		response.Render(c, response.New(response.CodeBadRequest, "zip too large (max 20MB)"))
		return
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".zip" {
		response.Render(c, response.New(response.CodeBadRequest, "only .zip uploads are supported"))
		return
	}
	tmp, err := os.CreateTemp("", "dialog-skill-*.zip")
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)
	if err := c.SaveUploadedFile(fh, tmpPath); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	if err := chat.ExtractSkillZip(tid, row.Code, tmpPath); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	_ = models.UpdateTenantDialogSkill(h.db, tid, id, map[string]any{"has_assets": true})
	row, _ = models.GetTenantDialogSkill(h.db, tid, id)
	response.SuccessI18n(c, i18n.KeySuccess, dialogSkillResp(row))
}

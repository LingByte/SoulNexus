package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
)

type assistantMembersReq struct {
	UserIDs []string `json:"userIds"`
}

// listAssistantMembers returns collaborator user ids for an assistant.
func (h *Handlers) listAssistantMembers(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	ast, err := h.getAssistantRow(c, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	rows, err := models.ListAssistantMembers(h.db, ast.ID)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	type memberRow struct {
		UserID string `json:"userId"`
		Role   string `json:"role"`
		Email  string `json:"email,omitempty"`
		Name   string `json:"name,omitempty"`
	}
	out := make([]memberRow, 0, len(rows))
	for _, row := range rows {
		item := memberRow{UserID: strconv.FormatUint(uint64(row.UserID), 10), Role: row.Role}
		if u, err := models.GetActiveTenantUserByID(h.db, row.UserID); err == nil {
			item.Email = u.Email
			item.Name = u.DisplayName
		}
		out = append(out, item)
	}
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

// updateAssistantMembers replaces collaborator list for an assistant.
func (h *Handlers) updateAssistantMembers(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	ast, err := h.getAssistantRow(c, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	var req assistantMembersReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	userIDs := parseAssistantMemberUserIDs(req.UserIDs)
	if err := models.ReplaceAssistantMembers(h.db, ast.ID, userIDs, middleware.AuditOperator(c)); err != nil {
		writeAssistantMemberError(c, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"count": len(userIDs)})
}

// addAssistantMembers appends collaborators without replacing the full list.
func (h *Handlers) addAssistantMembers(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	ast, err := h.getAssistantRow(c, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	var req assistantMembersReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	userIDs := parseAssistantMemberUserIDs(req.UserIDs)
	if err := models.AddAssistantMembers(h.db, ast.ID, userIDs, middleware.AuditOperator(c)); err != nil {
		writeAssistantMemberError(c, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"count": len(userIDs)})
}

// removeAssistantMember removes one collaborator.
func (h *Handlers) removeAssistantMember(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	userID, ok := ginutil.ParamID(c, "userId")
	if !ok {
		return
	}
	ast, err := h.getAssistantRow(c, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	if err := models.RemoveAssistantMember(h.db, ast.ID, userID); err != nil {
		writeAssistantMemberError(c, err)
		return
	}
	response.SuccessI18n(c, i18n.KeyDeleted, nil)
}

func parseAssistantMemberUserIDs(raw []string) []uint {
	userIDs := make([]uint, 0, len(raw))
	for _, s := range raw {
		if uid := utils.ParseOptionalID(s); uid > 0 {
			userIDs = append(userIDs, uid)
		}
	}
	return userIDs
}

func writeAssistantMemberError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "duplicate entry") || strings.Contains(msg, "unique constraint") {
		response.Render(c, response.NewI18n(response.CodeConflict, i18n.KeyAssistantMemberDuplicate))
		return
	}
	response.Render(c, response.Wrap(response.CodeInternal, i18n.TGin(c, i18n.KeyInternalError), err))
}

const maxAssistantAvatarBytes = 2 << 20

// uploadAssistantAvatar stores an avatar image for an assistant.
func (h *Handlers) uploadAssistantAvatar(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	ast, err := h.getAssistantRow(c, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeySelectImageFile))
		return
	}
	src, err := fh.Open()
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyCannotReadFile))
		return
	}
	defer src.Close()
	body, err := io.ReadAll(io.LimitReader(src, maxAssistantAvatarBytes+1))
	if err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	if len(body) > maxAssistantAvatarBytes {
		response.Render(c, response.NewI18n(response.CodeValidation, i18n.KeyImageTooLarge))
		return
	}
	ct := http.DetectContentType(body)
	ext := utils.PickImageExtFromContentType(ct)
	if ext == "" {
		response.Render(c, response.NewI18n(response.CodeValidation, i18n.KeyImageFormatInvalid))
		return
	}
	key := path.Join(
		"avatars",
		"t"+strconv.FormatUint(uint64(ast.TenantID), 10),
		fmt.Sprintf("assistant_%d_%d%s", ast.ID, time.Now().UnixMilli(), ext),
	)
	st := stores.Default()
	if strings.TrimSpace(ast.AvatarURL) != "" {
		deleteStoredAvatar(st, ast.AvatarURL)
	}
	if err := st.Write(key, bytes.NewReader(body)); err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	url := ginutil.UploadURL(c, key)
	if err := h.db.Model(&ast).Update("avatar_url", url).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	next, _ := models.ReloadAssistantByID(h.db, ast.ID)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"avatarUrl": url,
		"assistant": next,
	})
}

// patchAssistantSettings updates lightweight app metadata (name, description).
func (h *Handlers) patchAssistantSettings(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	before, err := h.getAssistantRow(c, id)
	if ginutil.WriteGORMError(c, err, "not found") {
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if !ginutil.BindJSON(c, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = before.Name
	}
	updates := map[string]any{
		"name":        name,
		"description": strings.TrimSpace(req.Description),
	}
	if err := h.db.Model(&before).Updates(updates).Error; err != nil {
		ginutil.WriteInternalError(c, err)
		return
	}
	after, _ := models.ReloadAssistantByID(h.db, id)
	response.SuccessI18n(c, i18n.KeySuccess, after)
}

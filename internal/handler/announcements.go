package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

func parseAnnouncementTime(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	s := strings.TrimSpace(*raw)
	if s == "" {
		return nil, nil
	}
	if strings.Contains(s, "T") {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (h *Handlers) handleListAnnouncements(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	now := time.Now()
	query := h.db.Model(&models.Announcement{}).
		Where("is_deleted = ?", models.SoftDeleteStatusActive).
		Where("status = ?", models.AnnouncementStatusPublished).
		Where("(publish_at IS NULL OR publish_at <= ?)", now).
		Where("(expire_at IS NULL OR expire_at > ?)", now)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list announcements failed", err)
		return
	}
	var items []models.Announcement
	if err := query.Order("pinned DESC, publish_at DESC, created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Fail(c, "list announcements failed", err)
		return
	}
	response.Success(c, "announcements fetched", gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleGetAnnouncement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid announcement id"))
		return
	}
	now := time.Now()
	var item models.Announcement
	if err := h.db.Where("id = ? AND is_deleted = ? AND status = ?", id, models.SoftDeleteStatusActive, models.AnnouncementStatusPublished).
		Where("(publish_at IS NULL OR publish_at <= ?)", now).
		Where("(expire_at IS NULL OR expire_at > ?)", now).
		First(&item).Error; err != nil {
		response.Fail(c, "announcement not found", err)
		return
	}
	response.Success(c, "announcement fetched", item)
}

func (h *Handlers) handleAdminListAnnouncements(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	status := strings.TrimSpace(c.Query("status"))
	search := strings.TrimSpace(c.Query("search"))
	query := h.db.Model(&models.Announcement{}).Where("is_deleted = ?", models.SoftDeleteStatusActive)
	if status != "" {
		query = query.Where("status = ?", models.NormalizeAnnouncementStatus(status))
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("title LIKE ? OR summary LIKE ? OR content LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list announcements failed", err)
		return
	}
	var items []models.Announcement
	if err := query.Order("pinned DESC, updated_at DESC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Fail(c, "list announcements failed", err)
		return
	}
	response.Success(c, "announcements fetched", gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminGetAnnouncement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid announcement id"))
		return
	}
	var item models.Announcement
	if err := h.db.Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).First(&item).Error; err != nil {
		response.Fail(c, "announcement not found", err)
		return
	}
	response.Success(c, "announcement fetched", item)
}

func (h *Handlers) handleAdminCreateAnnouncement(c *gin.Context) {
	var req struct {
		Title     string  `json:"title"`
		Summary   string  `json:"summary"`
		Content   string  `json:"content"`
		Status    string  `json:"status"`
		Pinned    *bool   `json:"pinned"`
		PublishAt *string `json:"publishAt"`
		ExpireAt  *string `json:"expireAt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if title == "" || content == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("title and content are required"))
		return
	}
	publishAt, err := parseAnnouncementTime(req.PublishAt)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid publishAt format"))
		return
	}
	expireAt, err := parseAnnouncementTime(req.ExpireAt)
	if err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid expireAt format"))
		return
	}
	admin := models.CurrentUser(c)
	operator := "system"
	if admin != nil && strings.TrimSpace(admin.Email) != "" {
		operator = strings.TrimSpace(admin.Email)
	}
	item := models.Announcement{
		Title:     title,
		Summary:   strings.TrimSpace(req.Summary),
		Content:   content,
		Status:    models.NormalizeAnnouncementStatus(strings.TrimSpace(req.Status)),
		Pinned:    req.Pinned != nil && *req.Pinned,
		PublishAt: publishAt,
		ExpireAt:  expireAt,
		BaseModel: models.BaseModel{
			CreateBy: operator,
			UpdateBy: operator,
		},
	}
	if err := h.db.Create(&item).Error; err != nil {
		response.Fail(c, "create announcement failed", err)
		return
	}
	response.Success(c, "announcement created", item)
}

func (h *Handlers) handleAdminUpdateAnnouncement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid announcement id"))
		return
	}
	var req struct {
		Title     *string `json:"title"`
		Summary   *string `json:"summary"`
		Content   *string `json:"content"`
		Status    *string `json:"status"`
		Pinned    *bool   `json:"pinned"`
		PublishAt *string `json:"publishAt"`
		ExpireAt  *string `json:"expireAt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	updates := map[string]any{}
	if req.Title != nil {
		updates["title"] = strings.TrimSpace(*req.Title)
	}
	if req.Summary != nil {
		updates["summary"] = strings.TrimSpace(*req.Summary)
	}
	if req.Content != nil {
		updates["content"] = strings.TrimSpace(*req.Content)
	}
	if req.Status != nil {
		updates["status"] = models.NormalizeAnnouncementStatus(strings.TrimSpace(*req.Status))
	}
	if req.Pinned != nil {
		updates["pinned"] = *req.Pinned
	}
	if req.PublishAt != nil {
		publishAt, parseErr := parseAnnouncementTime(req.PublishAt)
		if parseErr != nil {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid publishAt format"))
			return
		}
		updates["publish_at"] = publishAt
	}
	if req.ExpireAt != nil {
		expireAt, parseErr := parseAnnouncementTime(req.ExpireAt)
		if parseErr != nil {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid expireAt format"))
			return
		}
		updates["expire_at"] = expireAt
	}
	admin := models.CurrentUser(c)
	if admin != nil && strings.TrimSpace(admin.Email) != "" {
		updates["update_by"] = strings.TrimSpace(admin.Email)
	}
	if len(updates) == 0 {
		response.Success(c, "nothing changed", nil)
		return
	}
	result := h.db.Model(&models.Announcement{}).
		Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).
		Updates(updates)
	if result.Error != nil {
		response.Fail(c, "update announcement failed", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c, "announcement not found", errors.New("announcement not found"))
		return
	}
	var item models.Announcement
	_ = h.db.First(&item, id).Error
	response.Success(c, "announcement updated", item)
}

func (h *Handlers) handleAdminPublishAnnouncement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid announcement id"))
		return
	}
	now := time.Now()
	updates := map[string]any{
		"status":     models.AnnouncementStatusPublished,
		"publish_at": &now,
	}
	admin := models.CurrentUser(c)
	if admin != nil && strings.TrimSpace(admin.Email) != "" {
		updates["update_by"] = strings.TrimSpace(admin.Email)
	}
	result := h.db.Model(&models.Announcement{}).
		Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).
		Updates(updates)
	if result.Error != nil {
		response.Fail(c, "publish announcement failed", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c, "announcement not found", errors.New("announcement not found"))
		return
	}
	response.Success(c, "announcement published", nil)
}

func (h *Handlers) handleAdminOfflineAnnouncement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid announcement id"))
		return
	}
	updates := map[string]any{
		"status": models.AnnouncementStatusOffline,
	}
	admin := models.CurrentUser(c)
	if admin != nil && strings.TrimSpace(admin.Email) != "" {
		updates["update_by"] = strings.TrimSpace(admin.Email)
	}
	result := h.db.Model(&models.Announcement{}).
		Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).
		Updates(updates)
	if result.Error != nil {
		response.Fail(c, "offline announcement failed", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c, "announcement not found", errors.New("announcement not found"))
		return
	}
	response.Success(c, "announcement offlined", nil)
}

func (h *Handlers) handleAdminDeleteAnnouncement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid announcement id"))
		return
	}
	updates := map[string]any{"is_deleted": models.SoftDeleteStatusDeleted}
	admin := models.CurrentUser(c)
	if admin != nil && strings.TrimSpace(admin.Email) != "" {
		updates["update_by"] = strings.TrimSpace(admin.Email)
	}
	result := h.db.Model(&models.Announcement{}).
		Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).
		Updates(updates)
	if result.Error != nil {
		response.Fail(c, "delete announcement failed", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c, "announcement not found", errors.New("announcement not found"))
		return
	}
	response.Success(c, "announcement deleted", nil)
}

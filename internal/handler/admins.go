package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type adminUserUpsertReq struct {
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	DisplayName string  `json:"displayName"`
	FirstName   string  `json:"firstName"`
	LastName    string  `json:"lastName"`
	Role        string  `json:"role"`
	Status      *string `json:"status"`
	Phone       string  `json:"phone"`
	Locale      string  `json:"locale"`
	Timezone    string  `json:"timezone"`
	City        string  `json:"city"`
	Region      string  `json:"region"`
	Gender      string  `json:"gender"`
	Avatar      string  `json:"avatar"`

	EmailNotifications *bool `json:"emailNotifications"`
	PushNotifications  *bool `json:"pushNotifications"`
}

type adminConfigUpsertReq struct {
	Key      string `json:"key"`
	Desc     string `json:"desc"`
	Value    string `json:"value"`
	Format   string `json:"format"`
	Autoload *bool  `json:"autoload"`
	Public   *bool  `json:"public"`
}

type oauthClientUpsertReq struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Name         string `json:"name"`
	RedirectURI  string `json:"redirectUri"`
	Status       *int8  `json:"status"`
}

type adminGroupUpdateReq struct {
	Name       string                  `json:"name"`
	Type       string                  `json:"type"`
	Extra      string                  `json:"extra"`
	IsArchived *bool                   `json:"isArchived"`
	Permission *models.GroupPermission `json:"permission"`
}

type adminCredentialStatusReq struct {
	Status         string  `json:"status"`
	BannedReason   string  `json:"bannedReason"`
	ExpiresAt      *string `json:"expiresAt"`
	TokenQuota     *int64  `json:"tokenQuota"`
	RequestQuota   *int64  `json:"requestQuota"`
	UseNativeQuota *bool   `json:"useNativeQuota"`
	UnlimitedQuota *bool   `json:"unlimitedQuota"`
}

type adminAgentUpdateReq struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	SystemPrompt     string   `json:"systemPrompt"`
	Temperature      *float32 `json:"temperature"`
	MaxTokens        *int     `json:"maxTokens"`
	Speaker          string   `json:"speaker"`
	TtsProvider      string   `json:"ttsProvider"`
	LLMModel         string   `json:"llmModel"`
	EnableJSONOutput *bool    `json:"enableJSONOutput"`
}

func normalizeOAuthRedirectURI(raw string) string {
	parts := strings.Split(raw, ";")
	normalized := make([]string, 0, len(parts))
	for _, item := range parts {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return strings.Join(normalized, ";")
}

func (h *Handlers) requireAdmin(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("authorization required"))
		return
	}
	if models.UserHasAdminAccess(h.db, user.ID) {
		c.Next()
		return
	}
	response.AbortWithJSONError(c, http.StatusForbidden, errors.New("admin permission required"))
}

func (h *Handlers) parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := 20
	if raw := strings.TrimSpace(c.Query("pageSize")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			pageSize = parsed
		}
	}
	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			pageSize = parsed
		}
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func roleSlugExists(db *gorm.DB, slug string) bool {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return false
	}
	var cnt int64
	if err := db.Model(&models.Role{}).Where("slug = ? AND is_deleted = ?", slug, models.SoftDeleteStatusActive).Count(&cnt).Error; err != nil {
		return false
	}
	return cnt > 0
}

func (h *Handlers) handleAdminListUsers(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	roleFilter := strings.TrimSpace(strings.ToLower(c.Query("role")))
	statusQuery := strings.TrimSpace(c.Query("status"))
	enabledQuery := strings.TrimSpace(c.Query("enabled"))
	hasPhoneQuery := strings.TrimSpace(c.Query("hasPhone"))

	query := h.db.Model(&models.User{}).
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.id").
		Where("users.is_deleted = ?", models.SoftDeleteStatusActive)
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("users.email LIKE ? OR user_profiles.display_name LIKE ? OR user_profiles.first_name LIKE ? OR user_profiles.last_name LIKE ?",
			like, like, like, like)
	}
	if roleFilter != "" && roleSlugExists(h.db, roleFilter) {
		query = query.Joins("INNER JOIN user_roles ur ON ur.user_id = users.id").
			Joins("INNER JOIN roles r ON r.id = ur.role_id AND r.is_deleted = ?", models.SoftDeleteStatusActive).
			Where("r.slug = ?", roleFilter)
	}
	if statusQuery != "" {
		if st := models.NormalizeUserStatus(statusQuery); st != "" {
			query = query.Where("users.status = ?", st)
		}
	} else if enabledQuery != "" {
		if enabled, err := strconv.ParseBool(enabledQuery); err == nil {
			if enabled {
				query = query.Where("users.status = ?", models.UserStatusActive)
			} else {
				query = query.Where("users.status <> ?", models.UserStatusActive)
			}
		}
	}
	if hasPhoneQuery != "" {
		if hasPhone, err := strconv.ParseBool(hasPhoneQuery); err == nil {
			if hasPhone {
				query = query.Where("users.phone IS NOT NULL AND users.phone <> ''")
			} else {
				query = query.Where("users.phone IS NULL OR users.phone = ''")
			}
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list users failed", err)
		return
	}

	var users []models.User
	if err := query.Order("users.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		response.Fail(c, "list users failed", err)
		return
	}

	response.Success(c, "users fetched", gin.H{
		"users":    users,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminGetUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid user id"))
		return
	}
	var user models.User
	if err = h.db.Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).First(&user).Error; err != nil {
		response.Fail(c, "user not found", err)
		return
	}
	response.Success(c, "user fetched", user)
}

func (h *Handlers) handleAdminCreateUser(c *gin.Context) {
	var req adminUserUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("email is required"))
		return
	}
	roleSlug := strings.TrimSpace(strings.ToLower(req.Role))
	if !roleSlugExists(h.db, roleSlug) {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid role"))
		return
	}
	password := strings.TrimSpace(req.Password)
	if password == "" {
		password = utils.RandString(16)
	}
	if models.IsExistsByEmail(h.db, req.Email) {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("email already exists"))
		return
	}

	status := models.UserStatusActive
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		if st := models.NormalizeUserStatus(*req.Status); st != "" {
			status = st
		} else {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid status"))
			return
		}
	}

	user, err := models.CreateUserByEmailWithMeta(h.db, req.DisplayName, req.DisplayName, req.Email, password, models.UserSourceAdmin, status)
	if err != nil {
		response.Fail(c, "create user failed", err)
		return
	}

	coreVals := map[string]any{}
	if strings.TrimSpace(req.Phone) != "" {
		coreVals["phone"] = strings.TrimSpace(req.Phone)
	}
	operator := "system"
	if current := models.CurrentUser(c); current != nil {
		operator = current.Email
		if operator == "" {
			operator = "system"
		}
	}
	coreVals["create_by"] = operator
	coreVals["update_by"] = operator
	if strings.TrimSpace(req.Locale) != "" {
		coreVals["preferred_locale"] = strings.TrimSpace(req.Locale)
	}
	if strings.TrimSpace(req.Timezone) != "" {
		coreVals["preferred_timezone"] = strings.TrimSpace(req.Timezone)
	}
	if err = h.db.Model(user).Updates(coreVals).Error; err != nil {
		response.Fail(c, "create user failed", err)
		return
	}

	profVals := map[string]any{
		"display_name": req.DisplayName,
		"first_name":   req.FirstName,
		"last_name":    req.LastName,
		"city":         req.City,
		"region":       req.Region,
		"gender":       req.Gender,
		"avatar":       req.Avatar,
	}
	if req.EmailNotifications != nil {
		profVals["email_notifications"] = *req.EmailNotifications
	}
	if req.PushNotifications != nil {
		profVals["push_notifications"] = *req.PushNotifications
	}
	if err = models.UpdateUserProfileFields(h.db, user.ID, profVals); err != nil {
		response.Fail(c, "create user failed", err)
		return
	}
	_ = h.db.First(user, user.ID).Error
	if err = models.AssignUserSingleRoleBySlug(h.db, user.ID, roleSlug); err != nil {
		response.Fail(c, "assign role failed", err)
		return
	}
	response.Success(c, "user created", user)
}

func (h *Handlers) handleAdminUpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid user id"))
		return
	}

	var req adminUserUpsertReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	var user models.User
	if err = h.db.Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).First(&user).Error; err != nil {
		response.Fail(c, "user not found", err)
		return
	}

	coreVals := map[string]any{}
	profVals := map[string]any{}
	roleChanged := false
	var newRoleSlug string
	if req.Email != "" {
		coreVals["email"] = strings.TrimSpace(strings.ToLower(req.Email))
	}
	if req.DisplayName != "" {
		profVals["display_name"] = req.DisplayName
	}
	if req.FirstName != "" {
		profVals["first_name"] = req.FirstName
	}
	if req.LastName != "" {
		profVals["last_name"] = req.LastName
	}
	if req.Phone != "" {
		coreVals["phone"] = strings.TrimSpace(req.Phone)
	}
	if req.Locale != "" {
		coreVals["preferred_locale"] = strings.TrimSpace(req.Locale)
	}
	if req.Timezone != "" {
		coreVals["preferred_timezone"] = strings.TrimSpace(req.Timezone)
	}
	if req.City != "" {
		profVals["city"] = req.City
	}
	if req.Region != "" {
		profVals["region"] = req.Region
	}
	if req.Gender != "" {
		profVals["gender"] = req.Gender
	}
	if req.Avatar != "" {
		profVals["avatar"] = req.Avatar
	}
	if req.Role != "" {
		rs := strings.TrimSpace(strings.ToLower(req.Role))
		if !roleSlugExists(h.db, rs) {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid role"))
			return
		}
		roleChanged = true
		newRoleSlug = rs
	}
	if req.Password != "" {
		coreVals["password"] = models.HashPassword(req.Password)
	}
	if req.Status != nil {
		raw := strings.TrimSpace(*req.Status)
		if raw == "" {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("status cannot be empty"))
			return
		}
		if st := models.NormalizeUserStatus(raw); st == "" {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid status"))
			return
		} else {
			coreVals["status"] = st
		}
	}
	if req.EmailNotifications != nil {
		profVals["email_notifications"] = *req.EmailNotifications
	}
	if req.PushNotifications != nil {
		profVals["push_notifications"] = *req.PushNotifications
	}
	if len(coreVals) == 0 && len(profVals) == 0 {
		response.Success(c, "nothing changed", &user)
		return
	}
	operator := "system"
	if current := models.CurrentUser(c); current != nil {
		operator = current.Email
		if operator == "" {
			operator = "system"
		}
	}

	if len(coreVals) > 0 {
		coreVals["update_by"] = operator
		if err = h.db.Model(&user).Updates(coreVals).Error; err != nil {
			response.Fail(c, "update user failed", err)
			return
		}
	}
	if len(profVals) > 0 {
		if err = models.UpdateUserProfileFields(h.db, user.ID, profVals); err != nil {
			response.Fail(c, "update user profile failed", err)
			return
		}
	}
	_ = h.db.First(&user, user.ID).Error
	if roleChanged {
		if err = models.AssignUserSingleRoleBySlug(h.db, user.ID, newRoleSlug); err != nil {
			response.Fail(c, "assign role failed", err)
			return
		}
	}
	response.Success(c, "user updated", &user)
}

func (h *Handlers) handleAdminDeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid user id"))
		return
	}
	operator := "system"
	if current := models.CurrentUser(c); current != nil {
		operator = current.Email
		if operator == "" {
			operator = "system"
		}
	}
	if err = h.db.Model(&models.User{}).Where("id = ? AND is_deleted = ?", id, models.SoftDeleteStatusActive).
		Updates(map[string]any{"is_deleted": models.SoftDeleteStatusDeleted, "status": models.UserStatusBanned, "update_by": operator}).Error; err != nil {
		response.Fail(c, "delete user failed", err)
		return
	}
	response.Success(c, "user deleted", nil)
}

func (h *Handlers) handleAdminListConfigs(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	autoloadQuery := strings.TrimSpace(c.Query("autoload"))
	publicQuery := strings.TrimSpace(c.Query("public"))

	query := h.db.Model(&utils.Config{})
	if search != "" {
		like := "%" + strings.ToUpper(search) + "%"
		query = query.Where("`key` LIKE ? OR `desc` LIKE ?", like, like)
	}
	if autoloadQuery != "" {
		if autoload, err := strconv.ParseBool(autoloadQuery); err == nil {
			query = query.Where("autoload = ?", autoload)
		}
	}
	if publicQuery != "" {
		if isPublic, err := strconv.ParseBool(publicQuery); err == nil {
			query = query.Where("public = ?", isPublic)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list configs failed", err)
		return
	}

	var configs []utils.Config
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&configs).Error; err != nil {
		response.Fail(c, "list configs failed", err)
		return
	}

	response.Success(c, "configs fetched", gin.H{
		"configs": configs,
		"total":   total,
		"page":    page,
		"size":    pageSize,
	})
}

func (h *Handlers) handleAdminGetConfig(c *gin.Context) {
	key := strings.TrimSpace(strings.ToUpper(c.Param("key")))
	if key == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("config key is required"))
		return
	}
	var cfg utils.Config
	if err := h.db.Where("`key` = ?", key).First(&cfg).Error; err != nil {
		response.Fail(c, "config not found", err)
		return
	}
	response.Success(c, "config fetched", gin.H{"config": cfg})
}

func (h *Handlers) handleAdminCreateConfig(c *gin.Context) {
	var req adminConfigUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	req.Key = strings.TrimSpace(strings.ToUpper(req.Key))
	if req.Key == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("config key is required"))
		return
	}
	if req.Value == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("config value is required"))
		return
	}
	format := strings.TrimSpace(strings.ToLower(req.Format))
	if format == "" {
		format = "text"
	}
	autoload := req.Autoload != nil && *req.Autoload
	public := req.Public != nil && *req.Public

	cfg := utils.Config{
		Key:      req.Key,
		Desc:     req.Desc,
		Value:    req.Value,
		Format:   format,
		Autoload: autoload,
		Public:   public,
	}
	if err := h.db.Create(&cfg).Error; err != nil {
		response.Fail(c, "create config failed", err)
		return
	}
	utils.SetValue(h.db, cfg.Key, cfg.Value, cfg.Format, cfg.Autoload, cfg.Public)
	response.Success(c, "config created", gin.H{"config": cfg})
}

func (h *Handlers) handleAdminUpdateConfig(c *gin.Context) {
	key := strings.TrimSpace(strings.ToUpper(c.Param("key")))
	if key == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("config key is required"))
		return
	}
	var req adminConfigUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	var cfg utils.Config
	if err := h.db.Where("`key` = ?", key).First(&cfg).Error; err != nil {
		response.Fail(c, "config not found", err)
		return
	}

	updateVals := map[string]any{}
	if req.Desc != "" {
		updateVals["desc"] = req.Desc
		cfg.Desc = req.Desc
	}
	if req.Value != "" {
		updateVals["value"] = req.Value
		cfg.Value = req.Value
	}
	if req.Format != "" {
		updateVals["format"] = strings.ToLower(strings.TrimSpace(req.Format))
		cfg.Format = strings.ToLower(strings.TrimSpace(req.Format))
	}
	if req.Autoload != nil {
		updateVals["autoload"] = *req.Autoload
		cfg.Autoload = *req.Autoload
	}
	if req.Public != nil {
		updateVals["public"] = *req.Public
		cfg.Public = *req.Public
	}
	if len(updateVals) == 0 {
		response.Success(c, "nothing changed", gin.H{"config": cfg})
		return
	}
	if err := h.db.Model(&cfg).Updates(updateVals).Error; err != nil {
		response.Fail(c, "update config failed", err)
		return
	}
	utils.SetValue(h.db, cfg.Key, cfg.Value, cfg.Format, cfg.Autoload, cfg.Public)
	response.Success(c, "config updated", gin.H{"config": cfg})
}

func (h *Handlers) handleAdminDeleteConfig(c *gin.Context) {
	key := strings.TrimSpace(strings.ToUpper(c.Param("key")))
	if key == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("config key is required"))
		return
	}
	if err := h.db.Where("`key` = ?", key).Delete(&utils.Config{}).Error; err != nil {
		response.Fail(c, "delete config failed", err)
		return
	}
	response.Success(c, "config deleted", nil)
}

func (h *Handlers) handleAdminListOAuthClients(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	query := h.db.Model(&models.OAuthClient{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list oauth clients failed", err)
		return
	}
	var clients []models.OAuthClient
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&clients).Error; err != nil {
		response.Fail(c, "list oauth clients failed", err)
		return
	}
	response.Success(c, "oauth clients fetched", gin.H{
		"clients":   clients,
		"total":     total,
		"page":      page,
		"pageSize":  pageSize,
		"page_size": pageSize,
	})
}

func (h *Handlers) handleAdminGetOAuthClient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid oauth client id"))
		return
	}
	var client models.OAuthClient
	if err = h.db.First(&client, id).Error; err != nil {
		response.Fail(c, "oauth client not found", err)
		return
	}
	response.Success(c, "oauth client fetched", client)
}

func (h *Handlers) handleAdminCreateOAuthClient(c *gin.Context) {
	var req oauthClientUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	req.ClientID = strings.TrimSpace(req.ClientID)
	req.Name = strings.TrimSpace(req.Name)
	req.RedirectURI = normalizeOAuthRedirectURI(req.RedirectURI)
	if req.ClientID == "" || req.Name == "" || req.RedirectURI == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("clientId, name, redirectUri are required"))
		return
	}
	secret := strings.TrimSpace(req.ClientSecret)
	if secret == "" {
		secret = utils.RandString(32)
	}
	status := int8(models.OAuthClientStatusEnabled)
	if req.Status != nil {
		status = *req.Status
	}
	client := models.OAuthClient{
		ClientID:     req.ClientID,
		ClientSecret: secret,
		Name:         req.Name,
		RedirectURI:  req.RedirectURI,
		Status:       status,
	}
	if err := h.db.Create(&client).Error; err != nil {
		response.Fail(c, "create oauth client failed", err)
		return
	}
	response.Success(c, "oauth client created", client)
}

func (h *Handlers) handleAdminListOperationLogs(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	var userID uint64
	var err error
	if raw := strings.TrimSpace(c.Query("user_id")); raw != "" {
		userID, err = strconv.ParseUint(raw, 10, 64)
		if err != nil {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid user_id"))
			return
		}
	}
	action := strings.TrimSpace(c.Query("action"))
	target := strings.TrimSpace(c.Query("target"))

	query := h.db.Model(&middleware.OperationLog{})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if target != "" {
		query = query.Where("target LIKE ?", "%"+target+"%")
	}

	var total int64
	if err = query.Count(&total).Error; err != nil {
		response.Fail(c, "list operation logs failed", err)
		return
	}
	var logs []middleware.OperationLog
	if err = query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
		response.Fail(c, "list operation logs failed", err)
		return
	}
	response.Success(c, "operation logs fetched", gin.H{
		"logs":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *Handlers) handleAdminGetOperationLog(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid operation log id"))
		return
	}
	var logEntry middleware.OperationLog
	if err = h.db.First(&logEntry, id).Error; err != nil {
		response.Fail(c, "operation log not found", err)
		return
	}
	response.Success(c, "operation log fetched", gin.H{"log": logEntry})
}

func (h *Handlers) handleAdminListLoginHistory(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	var userID uint64
	var err error
	if raw := strings.TrimSpace(c.Query("user_id")); raw != "" {
		userID, err = strconv.ParseUint(raw, 10, 64)
		if err != nil {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid user_id"))
			return
		}
	}
	search := strings.TrimSpace(c.Query("search"))
	successRaw := strings.TrimSpace(c.Query("success"))
	suspiciousRaw := strings.TrimSpace(c.Query("is_suspicious"))

	query := h.db.Model(&models.LoginHistory{})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("email LIKE ? OR ip_address LIKE ? OR location LIKE ?", like, like, like)
	}
	if successRaw != "" {
		if success, parseErr := strconv.ParseBool(successRaw); parseErr == nil {
			query = query.Where("success = ?", success)
		}
	}
	if suspiciousRaw != "" {
		if suspicious, parseErr := strconv.ParseBool(suspiciousRaw); parseErr == nil {
			query = query.Where("is_suspicious = ?", suspicious)
		}
	}

	var total int64
	if err = query.Count(&total).Error; err != nil {
		response.Fail(c, "list login history failed", err)
		return
	}
	var histories []models.LoginHistory
	if err = query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&histories).Error; err != nil {
		response.Fail(c, "list login history failed", err)
		return
	}

	response.Success(c, "login history fetched", gin.H{
		"histories":  histories,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"pageSize":   pageSize,
		"totalPages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func (h *Handlers) handleAdminGetLoginHistory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid login history id"))
		return
	}
	var history models.LoginHistory
	if err = h.db.First(&history, id).Error; err != nil {
		response.Fail(c, "login history not found", err)
		return
	}
	response.Success(c, "login history fetched", gin.H{"history": history})
}

func (h *Handlers) handleAdminListAccountLocks(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	var userID uint64
	var err error
	if raw := strings.TrimSpace(c.Query("user_id")); raw != "" {
		userID, err = strconv.ParseUint(raw, 10, 64)
		if err != nil {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid user_id"))
			return
		}
	}
	email := strings.TrimSpace(c.Query("email"))
	isActiveRaw := strings.TrimSpace(c.Query("is_active"))

	query := h.db.Model(&models.AccountLock{})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if email != "" {
		query = query.Where("email LIKE ?", "%"+email+"%")
	}
	if isActiveRaw != "" {
		if active, parseErr := strconv.ParseBool(isActiveRaw); parseErr == nil {
			query = query.Where("is_active = ?", active)
		}
	}

	var total int64
	if err = query.Count(&total).Error; err != nil {
		response.Fail(c, "list account locks failed", err)
		return
	}
	var locks []models.AccountLock
	if err = query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&locks).Error; err != nil {
		response.Fail(c, "list account locks failed", err)
		return
	}
	response.Success(c, "account locks fetched", gin.H{
		"locks":      locks,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"totalPages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func (h *Handlers) handleAdminUnlockAccount(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid account lock id"))
		return
	}
	var lock models.AccountLock
	if err = h.db.First(&lock, id).Error; err != nil {
		response.Fail(c, "account lock not found", err)
		return
	}
	if err = h.db.Model(&lock).Update("is_active", false).Error; err != nil {
		response.Fail(c, "unlock account failed", err)
		return
	}
	response.Success(c, "account unlocked", nil)
}

func (h *Handlers) handleAdminListGroups(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	groupType := strings.TrimSpace(c.Query("type"))
	isArchivedRaw := strings.TrimSpace(c.Query("isArchived"))

	query := h.db.Model(&models.Group{}).Preload("Creator")
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR `type` LIKE ?", like, like)
	}
	if groupType != "" {
		query = query.Where("`type` = ?", groupType)
	}
	if isArchivedRaw != "" {
		if isArchived, err := strconv.ParseBool(isArchivedRaw); err == nil {
			query = query.Where("is_archived = ?", isArchived)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list groups failed", err)
		return
	}

	var groups []models.Group
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&groups).Error; err != nil {
		response.Fail(c, "list groups failed", err)
		return
	}

	type groupWithCount struct {
		models.Group
		MemberCount int64 `json:"memberCount"`
	}
	result := make([]groupWithCount, 0, len(groups))
	for _, g := range groups {
		var memberCount int64
		_ = h.db.Model(&models.GroupMember{}).Where("group_id = ?", g.ID).Count(&memberCount).Error
		result = append(result, groupWithCount{
			Group:       g,
			MemberCount: memberCount,
		})
	}

	response.Success(c, "groups fetched", gin.H{
		"groups":    result,
		"total":     total,
		"page":      page,
		"pageSize":  pageSize,
		"page_size": pageSize,
	})
}

func (h *Handlers) handleAdminGetGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid group id"))
		return
	}
	var group models.Group
	if err = h.db.Preload("Creator").First(&group, id).Error; err != nil {
		response.Fail(c, "group not found", err)
		return
	}
	response.Success(c, "group fetched", group)
}

func (h *Handlers) handleAdminUpdateGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid group id"))
		return
	}
	var req adminGroupUpdateReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	var group models.Group
	if err = h.db.First(&group, id).Error; err != nil {
		response.Fail(c, "group not found", err)
		return
	}

	updateVals := map[string]any{}
	if req.Name != "" {
		updateVals["name"] = req.Name
	}
	if req.Type != "" {
		updateVals["type"] = req.Type
	}
	if req.Extra != "" {
		updateVals["extra"] = req.Extra
	}
	if req.IsArchived != nil {
		updateVals["is_archived"] = *req.IsArchived
	}
	if req.Permission != nil {
		updateVals["permission"] = *req.Permission
	}
	if len(updateVals) == 0 {
		response.Success(c, "nothing changed", group)
		return
	}

	if err = h.db.Model(&group).Updates(updateVals).Error; err != nil {
		response.Fail(c, "update group failed", err)
		return
	}
	_ = h.db.Preload("Creator").First(&group, group.ID).Error
	response.Success(c, "group updated", group)
}

func (h *Handlers) handleAdminDeleteGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid group id"))
		return
	}
	if err = h.db.Delete(&models.Group{}, id).Error; err != nil {
		response.Fail(c, "delete group failed", err)
		return
	}
	response.Success(c, "group deleted", nil)
}

func (h *Handlers) handleAdminListCredentials(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	status := strings.TrimSpace(c.Query("status"))
	userIDRaw := strings.TrimSpace(c.Query("user_id"))

	query := h.db.Model(&models.UserCredential{})
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR api_key LIKE ? OR llm_provider LIKE ?", like, like, like)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if userIDRaw != "" {
		if uid, err := strconv.ParseUint(userIDRaw, 10, 64); err == nil {
			query = query.Where("user_id = ?", uid)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list credentials failed", err)
		return
	}
	var creds []models.UserCredential
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&creds).Error; err != nil {
		response.Fail(c, "list credentials failed", err)
		return
	}
	response.Success(c, "credentials fetched", gin.H{
		"credentials": creds,
		"total":       total,
		"page":        page,
		"pageSize":    pageSize,
		"page_size":   pageSize,
	})
}

func (h *Handlers) handleAdminGetCredential(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid credential id"))
		return
	}
	var cred models.UserCredential
	if err = h.db.First(&cred, id).Error; err != nil {
		response.Fail(c, "credential not found", err)
		return
	}
	response.Success(c, "credential fetched", cred)
}

func (h *Handlers) handleAdminUpdateCredentialStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid credential id"))
		return
	}
	var req adminCredentialStatusReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	var cred models.UserCredential
	if err = h.db.First(&cred, id).Error; err != nil {
		response.Fail(c, "credential not found", err)
		return
	}

	status := models.CredentialStatus(strings.TrimSpace(req.Status))
	updateVals := map[string]any{
		"status": status,
	}
	switch status {
	case models.CredentialStatusActive:
		updateVals["banned_at"] = nil
		updateVals["banned_reason"] = ""
		updateVals["banned_by"] = nil
	case models.CredentialStatusBanned:
		now := time.Now()
		updateVals["banned_at"] = &now
		updateVals["banned_reason"] = req.BannedReason
	case models.CredentialStatusSuspended:
		// no extra fields
	default:
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid credential status"))
		return
	}
	if req.ExpiresAt != nil {
		raw := strings.TrimSpace(*req.ExpiresAt)
		if raw == "" {
			updateVals["expires_at"] = nil
		} else {
			var parsed time.Time
			var parseErr error
			if strings.Contains(raw, "T") {
				parsed, parseErr = time.Parse(time.RFC3339, raw)
			} else {
				parsed, parseErr = time.ParseInLocation("2006-01-02 15:04:05", raw, time.Local)
			}
			if parseErr != nil {
				response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid expiresAt format, expected 'YYYY-MM-DD HH:MM:SS' or RFC3339"))
				return
			}
			updateVals["expires_at"] = &parsed
		}
	}
	if req.TokenQuota != nil {
		if *req.TokenQuota < 0 {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("tokenQuota must be >= 0"))
			return
		}
		updateVals["token_quota"] = *req.TokenQuota
	}
	if req.RequestQuota != nil {
		if *req.RequestQuota < 0 {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("requestQuota must be >= 0"))
			return
		}
		updateVals["request_quota"] = *req.RequestQuota
	}
	if req.UseNativeQuota != nil {
		updateVals["use_native_quota"] = *req.UseNativeQuota
	}
	if req.UnlimitedQuota != nil {
		updateVals["unlimited_quota"] = *req.UnlimitedQuota
	}
	if err = h.db.Model(&cred).Updates(updateVals).Error; err != nil {
		response.Fail(c, "update credential status failed", err)
		return
	}
	_ = h.db.First(&cred, cred.ID).Error
	response.Success(c, "credential status updated", cred)
}

func (h *Handlers) handleAdminDeleteCredential(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid credential id"))
		return
	}
	if err = h.db.Delete(&models.UserCredential{}, id).Error; err != nil {
		response.Fail(c, "delete credential failed", err)
		return
	}
	response.Success(c, "credential deleted", nil)
}

func (h *Handlers) handleAdminListJSTemplates(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	status := strings.TrimSpace(c.Query("status"))
	templateType := strings.TrimSpace(c.Query("type"))
	userIDRaw := strings.TrimSpace(c.Query("user_id"))

	query := h.db.Model(&models.JSTemplate{})
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR js_source_id LIKE ?", like, like)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if templateType != "" {
		query = query.Where("type = ?", templateType)
	}
	if userIDRaw != "" {
		if uid, err := strconv.ParseUint(userIDRaw, 10, 64); err == nil {
			query = query.Where("user_id = ?", uid)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list js templates failed", err)
		return
	}
	var templates []models.JSTemplate
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&templates).Error; err != nil {
		response.Fail(c, "list js templates failed", err)
		return
	}
	response.Success(c, "js templates fetched", gin.H{
		"templates": templates,
		"total":     total,
		"page":      page,
		"pageSize":  pageSize,
		"page_size": pageSize,
	})
}

func (h *Handlers) handleAdminGetJSTemplate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid js template id"))
		return
	}
	var template models.JSTemplate
	if err := h.db.Where("id = ?", id).First(&template).Error; err != nil {
		response.Fail(c, "js template not found", err)
		return
	}
	response.Success(c, "js template fetched", template)
}

func (h *Handlers) handleAdminUpdateJSTemplate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid js template id"))
		return
	}
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	delete(req, "id")
	delete(req, "user_id")
	delete(req, "created_at")
	delete(req, "createdAt")
	if len(req) == 0 {
		response.Success(c, "nothing changed", nil)
		return
	}

	if err := h.db.Model(&models.JSTemplate{}).Where("id = ?", id).Updates(req).Error; err != nil {
		response.Fail(c, "update js template failed", err)
		return
	}
	var template models.JSTemplate
	_ = h.db.Where("id = ?", id).First(&template).Error
	response.Success(c, "js template updated", template)
}

func (h *Handlers) handleAdminDeleteJSTemplate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid js template id"))
		return
	}
	if err := h.db.Where("id = ?", id).Delete(&models.JSTemplate{}).Error; err != nil {
		response.Fail(c, "delete js template failed", err)
		return
	}
	response.Success(c, "js template deleted", nil)
}

func (h *Handlers) handleAdminListBills(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	status := strings.TrimSpace(c.Query("status"))
	search := strings.TrimSpace(c.Query("search"))
	userIDRaw := strings.TrimSpace(c.Query("user_id"))
	groupIDRaw := strings.TrimSpace(c.Query("group_id"))

	query := h.db.Model(&models.Bill{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("bill_no LIKE ? OR title LIKE ?", like, like)
	}
	if userIDRaw != "" {
		if uid, err := strconv.ParseUint(userIDRaw, 10, 64); err == nil {
			query = query.Where("user_id = ?", uid)
		}
	}
	if groupIDRaw != "" {
		if gid, err := strconv.ParseUint(groupIDRaw, 10, 64); err == nil {
			query = query.Where("group_id = ?", gid)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list bills failed", err)
		return
	}
	var bills []models.Bill
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&bills).Error; err != nil {
		response.Fail(c, "list bills failed", err)
		return
	}
	response.Success(c, "bills fetched", gin.H{
		"bills":     bills,
		"total":     total,
		"page":      page,
		"pageSize":  pageSize,
		"page_size": pageSize,
	})
}

func (h *Handlers) handleAdminGetBill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid bill id"))
		return
	}
	var bill models.Bill
	if err = h.db.First(&bill, id).Error; err != nil {
		response.Fail(c, "bill not found", err)
		return
	}
	response.Success(c, "bill fetched", bill)
}

func (h *Handlers) handleAdminUpdateBill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid bill id"))
		return
	}
	var req struct {
		Title  string `json:"title"`
		Notes  string `json:"notes"`
		Status string `json:"status"`
	}
	if err = c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	updateVals := map[string]any{}
	if req.Title != "" {
		updateVals["title"] = req.Title
	}
	if req.Notes != "" {
		updateVals["notes"] = req.Notes
	}
	if req.Status != "" {
		updateVals["status"] = req.Status
	}
	if len(updateVals) == 0 {
		response.Success(c, "nothing changed", nil)
		return
	}
	if err = h.db.Model(&models.Bill{}).Where("id = ?", id).Updates(updateVals).Error; err != nil {
		response.Fail(c, "update bill failed", err)
		return
	}
	var bill models.Bill
	_ = h.db.First(&bill, id).Error
	response.Success(c, "bill updated", bill)
}

func (h *Handlers) handleAdminDeleteBill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid bill id"))
		return
	}
	if err = h.db.Delete(&models.Bill{}, id).Error; err != nil {
		response.Fail(c, "delete bill failed", err)
		return
	}
	response.Success(c, "bill deleted", nil)
}

func (h *Handlers) handleAdminUpdateOAuthClient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid oauth client id"))
		return
	}
	var req oauthClientUpsertReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	var client models.OAuthClient
	if err = h.db.First(&client, id).Error; err != nil {
		response.Fail(c, "oauth client not found", err)
		return
	}
	updateVals := map[string]any{}
	if req.ClientID != "" {
		updateVals["client_id"] = strings.TrimSpace(req.ClientID)
	}
	if req.ClientSecret != "" {
		updateVals["client_secret"] = strings.TrimSpace(req.ClientSecret)
	}
	if req.Name != "" {
		updateVals["name"] = strings.TrimSpace(req.Name)
	}
	if req.RedirectURI != "" {
		normalized := normalizeOAuthRedirectURI(req.RedirectURI)
		if normalized == "" {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("redirectUri cannot be empty"))
			return
		}
		updateVals["redirect_uri"] = normalized
	}
	if req.Status != nil {
		updateVals["status"] = *req.Status
	}
	if len(updateVals) == 0 {
		response.Success(c, "nothing changed", client)
		return
	}
	if err = h.db.Model(&client).Updates(updateVals).Error; err != nil {
		response.Fail(c, "update oauth client failed", err)
		return
	}
	_ = h.db.First(&client, client.ID).Error
	response.Success(c, "oauth client updated", client)
}

func (h *Handlers) handleAdminDeleteOAuthClient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid oauth client id"))
		return
	}
	if err = h.db.Delete(&models.OAuthClient{}, id).Error; err != nil {
		response.Fail(c, "delete oauth client failed", err)
		return
	}
	response.Success(c, "oauth client deleted", nil)
}

func (h *Handlers) handleAdminListAgents(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	search := strings.TrimSpace(c.Query("search"))

	query := h.db.Model(&models.Agent{})
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR description LIKE ? OR persona_tag LIKE ? OR llm_model LIKE ?", like, like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list assistants failed", err)
		return
	}

	var assistants []models.Agent
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&assistants).Error; err != nil {
		response.Fail(c, "list assistants failed", err)
		return
	}

	response.Success(c, "agents fetched", gin.H{
		"agents":   assistants,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminGetAgent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid assistant id"))
		return
	}
	var assistant models.Agent
	if err = h.db.First(&assistant, id).Error; err != nil {
		response.Fail(c, "assistant not found", err)
		return
	}
	response.Success(c, "assistant fetched", assistant)
}

func (h *Handlers) handleAdminUpdateAgent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid assistant id"))
		return
	}
	var req adminAgentUpdateReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	var assistant models.Agent
	if err = h.db.First(&assistant, id).Error; err != nil {
		response.Fail(c, "assistant not found", err)
		return
	}
	updateVals := map[string]any{}
	if req.Name != "" {
		updateVals["name"] = strings.TrimSpace(req.Name)
	}
	if req.Description != "" {
		updateVals["description"] = strings.TrimSpace(req.Description)
	}
	if req.SystemPrompt != "" {
		updateVals["system_prompt"] = req.SystemPrompt
	}
	if req.Temperature != nil {
		updateVals["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		updateVals["max_tokens"] = *req.MaxTokens
	}
	if req.Speaker != "" {
		updateVals["speaker"] = strings.TrimSpace(req.Speaker)
	}
	if req.TtsProvider != "" {
		updateVals["tts_provider"] = strings.TrimSpace(req.TtsProvider)
	}
	if req.LLMModel != "" {
		updateVals["llm_model"] = strings.TrimSpace(req.LLMModel)
	}
	if req.EnableJSONOutput != nil {
		updateVals["enable_json_output"] = *req.EnableJSONOutput
	}

	if len(updateVals) == 0 {
		response.Success(c, "nothing changed", assistant)
		return
	}
	if err = h.db.Model(&assistant).Updates(updateVals).Error; err != nil {
		response.Fail(c, "update assistant failed", err)
		return
	}
	_ = h.db.First(&assistant, id).Error
	response.Success(c, "assistant updated", assistant)
}

func (h *Handlers) handleAdminDeleteAgent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid assistant id"))
		return
	}
	if err = h.db.Delete(&models.Agent{}, id).Error; err != nil {
		response.Fail(c, "delete assistant failed", err)
		return
	}
	response.Success(c, "assistant deleted", nil)
}

func (h *Handlers) handleAdminListChatSessions(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	search := strings.TrimSpace(c.Query("search"))

	query := h.db.Model(&models.ChatSession{})
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("id LIKE ? OR user_id LIKE ? OR title LIKE ? OR provider LIKE ? OR model LIKE ?", like, like, like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list chat sessions failed", err)
		return
	}
	var items []models.ChatSession
	if err := query.Order("updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Fail(c, "list chat sessions failed", err)
		return
	}
	response.Success(c, "chat sessions fetched", gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminListChatMessages(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	sessionID := strings.TrimSpace(c.Query("session_id"))

	query := h.db.Model(&models.ChatMessage{})
	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("id LIKE ? OR session_id LIKE ? OR role LIKE ? OR content LIKE ? OR request_id LIKE ?", like, like, like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list chat messages failed", err)
		return
	}
	var items []models.ChatMessage
	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Fail(c, "list chat messages failed", err)
		return
	}
	response.Success(c, "chat messages fetched", gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminListLLMUsage(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	sessionID := strings.TrimSpace(c.Query("session_id"))
	success := strings.TrimSpace(c.Query("success"))

	query := h.db.Model(&models.LLMUsage{})
	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if success != "" {
		if parsed, err := strconv.ParseBool(success); err == nil {
			query = query.Where("success = ?", parsed)
		}
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("request_id LIKE ? OR session_id LIKE ? OR provider LIKE ? OR model LIKE ? OR ip_address LIKE ? OR user_agent LIKE ? OR response_content LIKE ?", like, like, like, like, like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list llm usage failed", err)
		return
	}
	var items []models.LLMUsage
	if err := query.Order("requested_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Fail(c, "list llm usage failed", err)
		return
	}
	response.Success(c, "llm usage fetched", gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminListGenericTable(c *gin.Context, tableName string, searchableCols ...string) {
	page, pageSize := h.parsePagination(c)
	search := strings.TrimSpace(c.Query("search"))

	query := h.db.Table(tableName)
	if search != "" && len(searchableCols) > 0 {
		like := "%" + search + "%"
		parts := make([]string, 0, len(searchableCols))
		args := make([]any, 0, len(searchableCols))
		for _, col := range searchableCols {
			parts = append(parts, col+" LIKE ?")
			args = append(args, like)
		}
		query = query.Where(strings.Join(parts, " OR "), args...)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list records failed", err)
		return
	}

	var items []map[string]any
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Fail(c, "list records failed", err)
		return
	}

	response.Success(c, "records fetched", gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminGetGenericTableItem(c *gin.Context, tableName string) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var item map[string]any
	if err = h.db.Table(tableName).Where("id = ?", id).Take(&item).Error; err != nil {
		response.Fail(c, "record not found", err)
		return
	}
	response.Success(c, "record fetched", item)
}

func (h *Handlers) handleAdminDeleteGenericTableItem(c *gin.Context, tableName string) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if err = h.db.Table(tableName).Where("id = ?", id).Delete(nil).Error; err != nil {
		response.Fail(c, "delete record failed", err)
		return
	}
	response.Success(c, "record deleted", nil)
}

func (h *Handlers) handleAdminListVoiceTrainingTasks(c *gin.Context) {
	h.handleAdminListGenericTable(c, "voice_training_tasks", "task_name", "task_id", "asset_id")
}

func (h *Handlers) handleAdminGetVoiceTrainingTask(c *gin.Context) {
	h.handleAdminGetGenericTableItem(c, "voice_training_tasks")
}

func (h *Handlers) handleAdminDeleteVoiceTrainingTask(c *gin.Context) {
	h.handleAdminDeleteGenericTableItem(c, "voice_training_tasks")
}

func (h *Handlers) handleAdminListWorkflowDefinitions(c *gin.Context) {
	h.handleAdminListGenericTable(c, "workflow_definitions", "name", "slug", "description", "status")
}

func (h *Handlers) handleAdminGetWorkflowDefinition(c *gin.Context) {
	h.handleAdminGetGenericTableItem(c, "workflow_definitions")
}

func (h *Handlers) handleAdminDeleteWorkflowDefinition(c *gin.Context) {
	h.handleAdminDeleteGenericTableItem(c, "workflow_definitions")
}

func (h *Handlers) handleAdminListNodePlugins(c *gin.Context) {
	h.handleAdminListGenericTable(c, "node_plugins", "name", "slug", "display_name", "category", "status")
}

func (h *Handlers) handleAdminGetNodePlugin(c *gin.Context) {
	h.handleAdminGetGenericTableItem(c, "node_plugins")
}

func (h *Handlers) handleAdminDeleteNodePlugin(c *gin.Context) {
	h.handleAdminDeleteGenericTableItem(c, "node_plugins")
}

func (h *Handlers) handleAdminListWorkflowPlugins(c *gin.Context) {
	h.handleAdminListGenericTable(c, "workflow_plugins", "name", "slug", "display_name", "category", "status")
}

func (h *Handlers) handleAdminGetWorkflowPlugin(c *gin.Context) {
	h.handleAdminGetGenericTableItem(c, "workflow_plugins")
}

func (h *Handlers) handleAdminDeleteWorkflowPlugin(c *gin.Context) {
	h.handleAdminDeleteGenericTableItem(c, "workflow_plugins")
}

func (h *Handlers) handleAdminListInternalNotifications(c *gin.Context) {
	h.handleAdminListGenericTable(c, "internal_notifications", "title", "content")
}

func (h *Handlers) handleAdminGetInternalNotification(c *gin.Context) {
	h.handleAdminGetGenericTableItem(c, "internal_notifications")
}

func (h *Handlers) handleAdminDeleteInternalNotification(c *gin.Context) {
	h.handleAdminDeleteGenericTableItem(c, "internal_notifications")
}

func (h *Handlers) handleAdminListDevices(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	search := strings.TrimSpace(c.Query("search"))

	query := h.db.Table("devices")
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("id LIKE ? OR mac_address LIKE ? OR device_name LIKE ? OR alias LIKE ?", like, like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list devices failed", err)
		return
	}

	var items []map[string]any
	if err := query.Order("updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Fail(c, "list devices failed", err)
		return
	}

	response.Success(c, "devices fetched", gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminGetDevice(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid device id"))
		return
	}
	var item map[string]any
	if err := h.db.Table("devices").Where("id = ?", id).Take(&item).Error; err != nil {
		response.Fail(c, "device not found", err)
		return
	}
	response.Success(c, "device fetched", item)
}

func (h *Handlers) handleAdminDeleteDevice(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid device id"))
		return
	}
	if err := h.db.Table("devices").Where("id = ?", id).Delete(nil).Error; err != nil {
		response.Fail(c, "delete device failed", err)
		return
	}
	response.Success(c, "device deleted", nil)
}

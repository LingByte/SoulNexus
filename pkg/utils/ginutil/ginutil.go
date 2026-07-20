// Package ginutil provides shared Gin handler helpers (param parsing, binding, errors).
package ginutil

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ParamID parses c.Param(name) as uint. On failure writes {"msg":"invalid id"} and returns ok=false.
func ParamID(c *gin.Context, name string) (id uint, ok bool) {
	v, err := utils.ParseID(c.Param(name))
	if err != nil {
		response.Render(c, response.New(response.CodeBadRequest, i18n.TGin(c, i18n.KeyInvalidParams)))
		return 0, false
	}
	return v, true
}

// QueryPage reads page/size query params and clamps them (default max size 100).
func QueryPage(c *gin.Context, maxSize int) (page, size int) {
	p, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	s, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	return utils.NormalizePage(p, s, maxSize)
}

// BindJSON binds JSON body into dest. On failure writes invalid body response and returns false.
func BindJSON(c *gin.Context, dest any) bool {
	if err := c.ShouldBindJSON(dest); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, i18n.TGin(c, i18n.KeyInvalidBody), err))
		return false
	}
	return true
}

// RequireAuthTenant returns JWT tenant id or writes unauthorized and returns ok=false.
func RequireAuthTenant(c *gin.Context) (tenantID uint, ok bool) {
	tid := middleware.AuthTenantID(c)
	if tid == 0 {
		response.Render(c, response.New(response.CodeUnauthorized, i18n.TGin(c, i18n.KeyUnauthorized)))
		return 0, false
	}
	return tid, true
}

// RequireAuthUser returns JWT user id or writes unauthorized and returns ok=false.
func RequireAuthUser(c *gin.Context) (userID uint, ok bool) {
	uid := middleware.AuthUserID(c)
	if uid == 0 {
		response.Render(c, response.New(response.CodeUnauthorized, i18n.TGin(c, i18n.KeyUnauthorized)))
		return 0, false
	}
	return uid, true
}

// WriteGORMError maps err to client response. Returns true when err != nil (caller should return).
func WriteGORMError(c *gin.Context, err error, notFoundMsg string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if notFoundMsg == "" {
			notFoundMsg = i18n.TGin(c, i18n.KeyNotFound)
		}
		response.Render(c, response.New(response.CodeNotFound, notFoundMsg))
		return true
	}
	response.Render(c, response.Wrap(response.CodeInternal, "internal error", err))
	return true
}

// WriteInternalError aborts with HTTP 500 and returns true (caller should return).
func WriteInternalError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
	return true
}

// WriteAppError renders an AppError with proper status and i18n message.
func WriteAppError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	response.Render(c, err)
}

// PageSuccess writes a standard paginated list payload.
func PageSuccess(c *gin.Context, list any, total int64, page, size int) {
	response.SuccessI18n(c, i18n.KeySuccess, utils.PagePayload(list, total, page, size))
}

// UploadURL 生成文件访问链接
func UploadURL(c *gin.Context, key string) string {
	rawKey := strings.TrimSpace(key)
	cleanKey := strings.TrimPrefix(rawKey, "/")
	st := stores.Default()
	ossURL := strings.TrimSpace(st.PublicURL(cleanKey))
	lowerURL := strings.ToLower(ossURL)
	if strings.HasPrefix(lowerURL, "http://") || strings.HasPrefix(lowerURL, "https://") {
		return ossURL
	}
	// 兼容反向代理X-Forwarded-Proto
	proto := strings.TrimSpace(c.Request.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		proto = "http"
		if c.Request.TLS != nil {
			proto = "https"
		}
	}
	escapedKey := url.QueryEscape(cleanKey)
	path := "/uploads/" + escapedKey
	if host := strings.TrimSpace(c.Request.Host); host != "" {
		return proto + "://" + host + path
	}
	return path
}

package middleware

import (
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/utils/cache"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// permCache caches user permission codes for 30s to avoid DB hit per request.
var permCache = cache.NewExpiredLRUCache[uint, []string](2048, 30*time.Second)

// InvalidatePermissionCache removes a user's cached permission codes.
// Call this after role/permission assignment changes.
func InvalidatePermissionCache(userID uint) {
	permCache.Remove(userID)
}

func dbFromContext(c *gin.Context) (*gorm.DB, bool) {
	raw, ok := c.Get(constants.DbField)
	if !ok {
		return nil, false
	}
	db, ok := raw.(*gorm.DB)
	return db, ok
}

// AuthCredentialID is non-zero when the request was authenticated via API key.
func AuthCredentialID(c *gin.Context) uint {
	if v, ok := c.Get(CtxAuthCredentialID); ok {
		if n, ok := v.(uint); ok {
			return n
		}
	}
	return 0
}

func abortForbiddenPermission(c *gin.Context, key string, args ...any) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": i18n.TGin(c, key, args...), "data": nil})
}

// RequireTenantPermissionAll requires an authenticated tenant user JWT to have every listed permission code.
// Platform-admin JWT bypasses tenant RBAC; handlers must scope data (e.g. cross-tenant list branches).
// API key requests must satisfy credential.permission_codes (wildcards via "*").
func RequireTenantPermissionAll(codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthPlatformAdminID(c) != 0 {
			c.Next()
			return
		}
		if cid := AuthCredentialID(c); cid != 0 {
			db, ok := dbFromContext(c)
			if !ok || db == nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": i18n.TGin(c, i18n.KeyDatabaseUnavailable), "data": nil})
				return
			}
			if AuthTenantID(c) == 0 {
				abortForbiddenPermission(c, i18n.KeyPermNeedTenantContext)
				return
			}
			if len(codes) == 0 {
				c.Next()
				return
			}
			okPerm, err := models.CredentialMatchesPermissionCodes(db, cid, codes, true)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					abortForbiddenPermission(c, i18n.KeyPermInvalidCredential)
					return
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": i18n.TGin(c, i18n.KeyInternalError), "data": nil})
				return
			}
			if !okPerm {
				abortForbiddenPermission(c, i18n.KeyPermInsufficientCredential)
				return
			}
			c.Next()
			return
		}
		db, ok := dbFromContext(c)
		if !ok || db == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "database unavailable", "data": nil})
			return
		}
		uid := AuthUserID(c)
		if uid == 0 || AuthTenantID(c) == 0 {
			abortForbiddenPermission(c, i18n.KeyPermNeedTenantUser)
			return
		}
		if len(codes) == 0 {
			c.Next()
			return
		}
		userCodes := cachedPermissionCodes(db, uid)
		for _, req := range codes {
			if !slices.Contains(userCodes, req) {
				abortForbiddenPermission(c, i18n.KeyPermInsufficientCode, req)
				return
			}
		}
		c.Next()
	}
}

// RequireTenantPermissionAny requires the tenant user to have at least one of the listed codes.
func RequireTenantPermissionAny(codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthPlatformAdminID(c) != 0 {
			c.Next()
			return
		}
		if cid := AuthCredentialID(c); cid != 0 {
			db, ok := dbFromContext(c)
			if !ok || db == nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": i18n.TGin(c, i18n.KeyDatabaseUnavailable), "data": nil})
				return
			}
			if AuthTenantID(c) == 0 {
				abortForbiddenPermission(c, i18n.KeyPermNeedTenantContext)
				return
			}
			if len(codes) == 0 {
				c.Next()
				return
			}
			okPerm, err := models.CredentialMatchesPermissionCodes(db, cid, codes, false)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					abortForbiddenPermission(c, i18n.KeyPermInvalidCredential)
					return
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": i18n.TGin(c, i18n.KeyInternalError), "data": nil})
				return
			}
			if !okPerm {
				abortForbiddenPermission(c, i18n.KeyPermInsufficientCredential)
				return
			}
			c.Next()
			return
		}
		db, ok := dbFromContext(c)
		if !ok || db == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "database unavailable", "data": nil})
			return
		}
		uid := AuthUserID(c)
		if uid == 0 || AuthTenantID(c) == 0 {
			abortForbiddenPermission(c, i18n.KeyPermNeedTenantUser)
			return
		}
		if len(codes) == 0 {
			c.Next()
			return
		}
		userCodes := cachedPermissionCodes(db, uid)
		for _, req := range codes {
			if slices.Contains(userCodes, req) {
				c.Next()
				return
			}
		}
		abortForbiddenPermission(c, i18n.KeyPermInsufficient)
	}
}

// cachedPermissionCodes returns permission codes from cache or DB.
func cachedPermissionCodes(db *gorm.DB, uid uint) []string {
	if codes, ok := permCache.Get(uid); ok {
		return codes
	}
	codes, _ := models.ListEffectivePermissionCodesForTenantUser(db, uid)
	permCache.Add(uid, codes)
	return codes
}

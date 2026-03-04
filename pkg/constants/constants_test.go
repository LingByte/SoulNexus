package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	// Test cache related constants
	assert.Equal(t, "CONFIG_CACHE_SIZE", ENV_CONFIG_CACHE_SIZE, "ENV_CONFIG_CACHE_SIZE should be 'CONFIG_CACHE_SIZE'")
	assert.Equal(t, "CONFIG_CACHE_EXPIRED", ENV_CONFIG_CACHE_EXPIRED, "ENV_CONFIG_CACHE_EXPIRED should be 'CONFIG_CACHE_EXPIRED'")

	// Test session related constants
	assert.Equal(t, "SESSION_FIELD", ENV_SESSION_FIELD, "ENV_SESSION_FIELD should be 'SESSION_FIELD'")
	assert.Equal(t, "SESSION_SECRET", ENV_SESSION_SECRET, "ENV_SESSION_SECRET should be 'SESSION_SECRET'")
	assert.Equal(t, "SESSION_EXPIRE_DAYS", ENV_SESSION_EXPIRE_DAYS, "ENV_SESSION_EXPIRE_DAYS should be 'SESSION_EXPIRE_DAYS'")

	// Test database related constants
	assert.Equal(t, "DB_DRIVER", ENV_DB_DRIVER, "ENV_DB_DRIVER should be 'DB_DRIVER'")
	assert.Equal(t, "DSN", ENV_DSN, "ENV_DSN should be 'DSN'")
	assert.Equal(t, "_lingecho_db", DbField, "DbField should be '_lingecho_db'")
	assert.Equal(t, "_lingecho_uid", UserField, "UserField should be '_lingecho_uid'")
	assert.Equal(t, "_lingecho_gid", GroupField, "GroupField should be '_lingecho_gid'")
	assert.Equal(t, "_lingecho_tz", TzField, "TzField should be '_lingecho_tz'")
	assert.Equal(t, "_lingecho_assets", AssetsField, "AssetsField should be '_lingecho_assets'")
	assert.Equal(t, "_lingecho_templates", TemplatesField, "TemplatesField should be '_lingecho_templates'")

	// Test key related constants
	assert.Equal(t, "VERIFY_EMAIL_EXPIRED", KEY_VERIFY_EMAIL_EXPIRED, "KEY_VERIFY_EMAIL_EXPIRED should be 'VERIFY_EMAIL_EXPIRED'")
	assert.Equal(t, "AUTH_TOKEN_EXPIRED", KEY_AUTH_TOKEN_EXPIRED, "KEY_AUTH_TOKEN_EXPIRED should be 'AUTH_TOKEN_EXPIRED'")
	assert.Equal(t, "SITE_NAME", KEY_SITE_NAME, "KEY_SITE_NAME should be 'SITE_NAME'")
	assert.Equal(t, "SITE_ADMIN", KEY_SITE_ADMIN, "KEY_SITE_ADMIN should be 'SITE_ADMIN'")
	assert.Equal(t, "SITE_URL", KEY_SITE_URL, "KEY_SITE_URL should be 'SITE_URL'")
	assert.Equal(t, "SITE_KEYWORDS", KEY_SITE_KEYWORDS, "KEY_SITE_KEYWORDS should be 'SITE_KEYWORDS'")
	assert.Equal(t, "SITE_DESCRIPTION", KEY_SITE_DESCRIPTION, "KEY_SITE_DESCRIPTION should be 'SITE_DESCRIPTION'")
	assert.Equal(t, "SITE_GA", KEY_SITE_GA, "KEY_SITE_GA should be 'SITE_GA'")

	// Test site URL related constants
	assert.Equal(t, "SITE_LOGO_URL", KEY_SITE_LOGO_URL, "KEY_SITE_LOGO_URL should be 'SITE_LOGO_URL'")
	assert.Equal(t, "SITE_FAVICON_URL", KEY_SITE_FAVICON_URL, "KEY_SITE_FAVICON_URL should be 'SITE_FAVICON_URL'")
	assert.Equal(t, "SITE_TERMS_URL", KEY_SITE_TERMS_URL, "KEY_SITE_TERMS_URL should be 'SITE_TERMS_URL'")
	assert.Equal(t, "SITE_PRIVACY_URL", KEY_SITE_PRIVACY_URL, "KEY_SITE_PRIVACY_URL should be 'SITE_PRIVACY_URL'")
	assert.Equal(t, "SITE_SIGNIN_URL", KEY_SITE_SIGNIN_URL, "KEY_SITE_SIGNIN_URL should be 'SITE_SIGNIN_URL'")
	assert.Equal(t, "SITE_SIGNUP_URL", KEY_SITE_SIGNUP_URL, "KEY_SITE_SIGNUP_URL should be 'SITE_SIGNUP_URL'")
	assert.Equal(t, "SITE_LOGOUT_URL", KEY_SITE_LOGOUT_URL, "KEY_SITE_LOGOUT_URL should be 'SITE_LOGOUT_URL'")
	assert.Equal(t, "SITE_RESET_PASSWORD_URL", KEY_SITE_RESET_PASSWORD_URL, "KEY_SITE_RESET_PASSWORD_URL should be 'SITE_RESET_PASSWORD_URL'")
	assert.Equal(t, "SITE_SIGNIN_API", KEY_SITE_SIGNIN_API, "KEY_SITE_SIGNIN_API should be 'SITE_SIGNIN_API'")
	assert.Equal(t, "SITE_SIGNUP_API", KEY_SITE_SIGNUP_API, "KEY_SITE_SIGNUP_API should be 'SITE_SIGNUP_API'")
	assert.Equal(t, "SITE_RESET_PASSWORD_DONE_API", KEY_SITE_RESET_PASSWORD_DONE_API, "KEY_SITE_RESET_PASSWORD_DONE_API should be 'SITE_RESET_PASSWORD_DONE_API'")
	assert.Equal(t, "SITE_LOGIN_NEXT", KEY_SITE_LOGIN_NEXT, "KEY_SITE_LOGIN_NEXT should be 'SITE_LOGIN_NEXT'")
	assert.Equal(t, "SITE_USER_ID_TYPE", KEY_SITE_USER_ID_TYPE, "KEY_SITE_USER_ID_TYPE should be 'SITE_USER_ID_TYPE'")
	assert.Equal(t, "USER_ACTIVATED", KEY_USER_ACTIVATED, "KEY_USER_ACTIVATED should be 'USER_ACTIVATED'")

	// Test static related constants
	assert.Equal(t, "STATIC_PREFIX", ENV_STATIC_PREFIX, "ENV_STATIC_PREFIX should be 'STATIC_PREFIX'")
	assert.Equal(t, "STATIC_ROOT", ENV_STATIC_ROOT, "ENV_STATIC_ROOT should be 'STATIC_ROOT'")
}

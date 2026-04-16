package constants

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

const (
	USER_TABLE_NAME                 = "users"
	USER_CREDENTIAL_TABLE_NAME      = "user_credentials"
	USER_DEVICE_TABLE_NAME          = "user_devices"
	LOGIN_HISTORY_TABLE_NAME        = "login_histories"
	ACCOUNT_LOCK_TABLE_NAME         = "account_locks"
	CALL_RECORDING_TABLE_NAME       = "call_recordings"
	DEVICE_ERROR_LOG_TABLE_NAME     = "device_error_logs"
	SIP_USER_TABLE_NAME             = "sip_users"
	SIP_CALL_TABLE_NAME             = "sip_calls"
	SIP_CAMPAIGN_TABLE_NAME         = "sip_campaigns"
	SIP_CAMPAIGN_CONTACT_TABLE_NAME = "sip_campaign_contacts"
	SIP_CALL_ATTEMPT_TABLE_NAME     = "sip_call_attempts"
	SIP_SCRIPT_RUN_TABLE_NAME       = "sip_script_runs"
	SIP_CAMPAIGN_EVENT_TABLE_NAME   = "sip_campaign_events"
	SIP_SCRIPT_TEMPLATE_TABLE_NAME  = "sip_script_templates"
	ACD_POOL_TARGET_TABLE_NAME      = "acd_pool_targets" // ACD: unified SIP + Web routing pool (targets + weights)
)

// Default Value: 1024
const ENV_CONFIG_CACHE_SIZE = "CONFIG_CACHE_SIZE"

// Default Value: 10s
const ENV_CONFIG_CACHE_EXPIRED = "CONFIG_CACHE_EXPIRED"

// Gin session field name
const ENV_SESSION_FIELD = "SESSION_FIELD"

// Session
const ENV_SESSION_SECRET = "SESSION_SECRET"
const ENV_SESSION_EXPIRE_DAYS = "SESSION_EXPIRE_DAYS"

// DB
const ENV_DB_DRIVER = "DB_DRIVER"
const ENV_DSN = "DSN"
const DbField = "_lingecho_db"
const UserField = "_lingecho_uid"
const GroupField = "_lingecho_gid"
const TzField = "_lingecho_tz"
const AssetsField = "_lingecho_assets"
const TemplatesField = "_lingecho_templates"

const KEY_VERIFY_EMAIL_EXPIRED = "VERIFY_EMAIL_EXPIRED"
const KEY_AUTH_TOKEN_EXPIRED = "AUTH_TOKEN_EXPIRED"
const KEY_SITE_NAME = "SITE_NAME"
const KEY_SITE_ADMIN = "SITE_ADMIN"
const KEY_SITE_URL = "SITE_URL"
const KEY_SITE_KEYWORDS = "SITE_KEYWORDS"
const KEY_SITE_DESCRIPTION = "SITE_DESCRIPTION"
const KEY_SITE_GA = "SITE_GA"

const KEY_SITE_LOGO_URL = "SITE_LOGO_URL"
const KEY_SITE_FAVICON_URL = "SITE_FAVICON_URL"
const KEY_SITE_TERMS_URL = "SITE_TERMS_URL"
const KEY_SITE_PRIVACY_URL = "SITE_PRIVACY_URL"
const KEY_SITE_SIGNIN_URL = "SITE_SIGNIN_URL"
const KEY_SITE_SIGNUP_URL = "SITE_SIGNUP_URL"
const KEY_SITE_LOGOUT_URL = "SITE_LOGOUT_URL"
const KEY_SITE_RESET_PASSWORD_URL = "SITE_RESET_PASSWORD_URL"
const KEY_SITE_SIGNIN_API = "SITE_SIGNIN_API"
const KEY_SITE_SIGNUP_API = "SITE_SIGNUP_API"
const KEY_SITE_RESET_PASSWORD_DONE_API = "SITE_RESET_PASSWORD_DONE_API"
const KEY_SITE_LOGIN_NEXT = "SITE_LOGIN_NEXT"
const KEY_SITE_USER_ID_TYPE = "SITE_USER_ID_TYPE"
const KEY_USER_ACTIVATED = "USER_ACTIVATED"
const KEY_STORAGE_KIND = "STORAGE_KIND"

// Search configuration keys
const KEY_SEARCH_ENABLED = "SEARCH_ENABLED"
const KEY_SEARCH_PATH = "SEARCH_PATH"
const KEY_SEARCH_BATCH_SIZE = "SEARCH_BATCH_SIZE"
const KEY_SEARCH_INDEX_SCHEDULE = "SEARCH_INDEX_SCHEDULE"

// Voice clone configuration keys
const KEY_VOICE_CLONE_XUNFEI_CONFIG = "VOICE_CLONE_XUNFEI_CONFIG"
const KEY_VOICE_CLONE_VOLCENGINE_CONFIG = "VOICE_CLONE_VOLCENGINE_CONFIG"

// Voiceprint recognition configuration keys
const KEY_VOICEPRINT_ENABLED = "VOICEPRINT_ENABLED"
const KEY_VOICEPRINT_CONFIG = "VOICEPRINT_CONFIG"

// OTA and device configuration keys
const KEY_SERVER_WEBSOCKET = "server.websocket"
const KEY_SERVER_MQTT_GATEWAY = "server.mqtt_gateway"
const KEY_SERVER_MQTT_SIGNATURE_KEY = "server.mqtt_signature_key"
const KEY_SERVER_FRONTED_URL = "server.fronted_url"

const ENV_STATIC_PREFIX = "STATIC_PREFIX"
const ENV_STATIC_ROOT = "STATIC_ROOT"

const AUTHORIZATION_PREFIX = "Bearer "
const CREDENTIAL_API_KEY = "X-API-KEY"
const CREDENTIAL_API_SECRET = "X-API-SECRET"

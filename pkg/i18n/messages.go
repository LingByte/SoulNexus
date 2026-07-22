package i18n

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// Message keys — use with TGin(c, KeyXxx) or response.FailI18n.
const (
	KeySuccess = "common.success"
	KeyCreated = "common.created"
	KeyUpdated = "common.updated"
	KeyDeleted = "common.deleted"
	KeySent    = "common.sent"

	KeyInvalidBody         = "common.invalid_body"
	KeyNotFound            = "common.not_found"
	KeyUnauthorized        = "common.unauthorized"
	KeyForbidden           = "common.forbidden"
	KeyInternalError       = "common.internal_error"
	KeyNoFieldsToUpdate    = "common.no_fields_to_update"
	KeyInvalidParams       = "common.invalid_params"
	KeyDatabaseUnavailable = "common.database_unavailable"
	KeyConflict            = "common.conflict"
	KeyRateLimited         = "common.rate_limited"
	KeyTenantMismatch      = "common.tenant_mismatch"
	KeyQuotaExceeded       = "common.quota_exceeded"
	KeyUpstreamTimeout     = "common.upstream_timeout"
	KeyServiceUnavailable  = "common.service_unavailable"

	KeyAuthInvalidCredentials        = "auth.invalid_credentials"
	KeyAuthAccountLocked             = "auth.account_locked"
	KeyAuthNeedsTotp                 = "auth.needs_totp"
	KeyAuthNeedsDeviceVerify         = "auth.needs_device_verify"
	KeyAuthNeedsRemoteVerify         = "auth.needs_remote_verify"
	KeyAuthSessionRevoked            = "auth.session_revoked"
	KeyAuthInvalidTotp               = "auth.invalid_totp"
	KeyAuthJWTNotReady               = "auth.jwt_not_ready"
	KeyAuthMissingToken              = "auth.missing_token"
	KeyAuthInvalidToken              = "auth.invalid_token"
	KeyAuthLogoutSuccess             = "auth.logout_success"
	KeyAuthEmailCodeSent             = "auth.email_code_sent"
	KeyAuthEmailCodeCooldown         = "auth.email_code_cooldown"
	KeyAuthEmailCodeSendFailed       = "auth.email_code_send_failed"
	KeyAuthInvalidEmailCode          = "auth.invalid_email_code"
	KeyAuthInvalidVoiceprint         = "auth.invalid_voiceprint"
	KeyAuthEmailNotRegistered        = "auth.email_not_registered"
	KeyAuthSMSCodeSent               = "auth.sms_code_sent"
	KeyAuthSMSUnavailable            = "auth.sms_unavailable"
	KeyAuthSMSSendFailed             = "auth.sms_send_failed"
	KeyAuthPhoneNotRegistered        = "auth.phone_not_registered"
	KeyAuthInvalidSMSCode            = "auth.invalid_sms_code"
	KeyAuthEmailAlreadyRegistered    = "auth.email_already_registered"
	KeyAuthEmailSameAsCurrent        = "auth.email_same_as_current"
	KeyAuthEmailChanged              = "auth.email_changed"
	KeyAccountDeletionRequested      = "account.deletion_requested"
	KeyAccountDeletionCancelled      = "account.deletion_cancelled"
	KeyAccountDeletionPending        = "account.deletion_pending"
	KeyAccountDeletionAlreadyPending = "account.deletion_already_pending"
	KeyAccountDeletionNotPending     = "account.deletion_not_pending"

	KeyTenantRegisterDisabled = "tenant.register_disabled"
	KeyTenantEmailExists      = "tenant.email_exists"
	KeyTenantInvalidEmail     = "tenant.invalid_email"
	KeyTenantNotFound         = "tenant.not_found"
	KeyTenantSuspended        = "tenant.suspended"
	KeyTenantUserUnavailable  = "tenant.user_unavailable"
	KeyTenantSignTokenFailed  = "tenant.sign_token_failed"

	KeyPermInsufficient              = "perm.insufficient"
	KeyPermInsufficientCode          = "perm.insufficient_code"
	KeyPermInsufficientCredential    = "perm.insufficient_credential"
	KeyPermNeedTenantUser            = "perm.need_tenant_user"
	KeyPermNeedTenantContext         = "perm.need_tenant_context"
	KeyPermPlatformNoTenantRBAC      = "perm.platform_no_tenant_rbac"
	KeyPermInvalidCredential         = "perm.invalid_credential"
	KeyAKSKRouteNotAllowed           = "aksk.route_not_allowed"
	KeyAKSKCredentialRouteNotAllowed = "aksk.credential_route_not_allowed"

	KeyCredPermRequired     = "credential.permission_required"
	KeyCredAIConfigRequired = "credential.ai_config_required"
	KeyCredPlatformKindForbidden = "credential.platform_kind_forbidden"
	KeyCredPlatformNoCapacity      = "credential.platform_no_capacity"
	KeyCredRequiredForDebug        = "credential.required_for_debug"
	KeyCredPermEmptyArray   = "credential.permission_empty"
	KeyCredAllowIPRequired  = "credential.allow_ip_required"
	KeyCredAllowIPEmpty     = "credential.allow_ip_empty"
	KeyCredInvalidPerm      = "credential.invalid_permission"
	KeyCredNameEmpty        = "credential.name_empty"
	KeyCredNoticeSecretOnce = "credential.secret_once"

	KeyUploadsRecordingAuth = "uploads.recording_auth_required"

	KeyOrgInvalidPermID  = "org.invalid_permission_id"
	KeyOrgInvalidRoleID  = "org.invalid_role_id"
	KeyOrgAdminRoleFixed = "org.admin_role_fixed"

	KeyPasswordWrong   = "account.password_wrong"
	KeyPasswordChanged = "account.password_changed"
	KeyTotpAlreadyOn   = "account.totp_already_on"
	KeyTotpNotOn       = "account.totp_not_on"
	KeyTotpInvalidCode = "account.totp_invalid"
	KeyTotpEnabled     = "account.totp_enabled"
	KeyTotpDisabled    = "account.totp_disabled"
	KeyTotpSetupFirst  = "account.totp_setup_first"

	KeyValidationUsernameShort   = "validation.username_short"
	KeyValidationUsernameFormat  = "validation.username_format"
	KeyValidationPasswordShort   = "validation.password_short"
	KeyValidationCaptchaRequired = "validation.captcha_required"
	KeyValidationCaptchaInvalid  = "validation.captcha_invalid"

	KeyWebhookReceived              = "webhook.received"
	KeyWebhookProcessed             = "webhook.processed"
	KeyNotificationAllMarked        = "notification.all_marked"
	KeyNameRequired                 = "validation.name_required"
	KeyInvalidID                    = "validation.invalid_id"
	KeyCampaignNotFound             = "entity.campaign_not_found"
	KeyUserNotFound                 = "entity.user_not_found"
	KeyWebhookNotFound              = "entity.webhook_not_found"
	KeyTemplateNotFound             = "entity.template_not_found"
	KeyChunkNotFound                = "entity.chunk_not_found"
	KeyContactNotFound              = "entity.contact_not_found"
	KeyVersionNotFound              = "entity.version_not_found"
	KeyRecordingNotFound            = "entity.recording_not_found"
	KeyUnknownSession               = "entity.unknown_session"
	KeyUsernameExists               = "duplicate.username_exists"
	KeyPhoneExists                  = "duplicate.phone_exists"
	KeyEmailRequired                = "validation.email_required"
	KeyQueryRequired                = "validation.query_required"
	KeyTenantIDRequired             = "validation.tenant_id_required"
	KeySessionIDRequired            = "validation.session_id_required"
	KeyAssistantIDRequired          = "validation.assistant_id_required"
	KeyAssistantMemberDuplicate     = "assistant.member_duplicate"
	KeyConfigKeyRequired            = "validation.config_key_required"
	KeyContentRequired              = "validation.content_required"
	KeyDocumentEmpty                = "validation.document_empty"
	KeyPasswordRequired             = "validation.password_required"
	KeyUsernameRequired             = "validation.username_required"
	KeyPhoneRequired                = "validation.phone_required"
	KeyDescriptionRequired          = "validation.description_required"
	KeyTargetValueRequired          = "validation.target_value_required"
	KeyInvalidTenantID              = "validation.invalid_tenant_id"
	KeyInvalidVersionID             = "validation.invalid_version_id"
	KeyInvalidAssistantID           = "validation.invalid_assistant_id"
	KeyInvalidDepartmentID          = "validation.invalid_department_id"
	KeyInvalidRoleID                = "validation.invalid_role_id"
	KeyInvalidPermissionID          = "validation.invalid_permission_id"
	KeyInvalidScriptTemplateID      = "validation.invalid_script_template_id"
	KeyInvalidContactEmail          = "validation.invalid_contact_email"
	KeyInvalidExpiresAt             = "validation.invalid_expires_at"
	KeyInvalidFormat                = "validation.invalid_format"
	KeyInvalidStatus                = "validation.invalid_status"
	KeyInvalidChunkIndex            = "validation.invalid_chunk_index"
	KeyWorkStateInvalid             = "validation.work_state_invalid"
	KeyModeInvalid                  = "validation.mode_invalid"
	KeyVoiceModeInvalid             = "validation.voice_mode_invalid"
	KeyCategoryInvalid              = "validation.category_invalid"
	KeyAssistantIDRequiredCat       = "validation.assistant_id_required_cat"
	KeyRoutePolicyRequiresRoute     = "validation.route_policy_requires_route"
	KeyExportFormatInvalid          = "validation.export_format_invalid"
	KeyImageTooLarge                = "validation.image_too_large"
	KeyImageFormatInvalid           = "validation.image_format_invalid"
	KeySelectImageFile              = "validation.select_image_file"
	KeyCannotReadFile               = "validation.cannot_read_file"
	KeyNewPasswordSameAsOld         = "validation.new_password_same_as_old"
	KeyCurrentPwdRequired           = "validation.current_pwd_required"
	KeyTotpDisableFirstToRebind     = "auth.totp_disable_first_to_rebind"
	KeyTenantRequired               = "auth.tenant_required"
	KeyKnowledgeUnavailable         = "service.knowledge_unavailable"
	KeyKnowledgeWorkerUnavail       = "service.knowledge_worker_unavailable"
	KeyVoiceSessionUnavailable      = "service.voice_session_unavailable"
	KeyDiffFailed                   = "service.diff_failed"
	KeySessionCreateFailed          = "service.session_create_failed"
	KeyCannotDeleteSelf             = "perm.cannot_delete_self"
	KeyCannotDisableSelf            = "perm.cannot_disable_self"
	KeyCannotDeleteLastAdmin        = "perm.cannot_delete_last_admin"
	KeyCannotDisableLastAdmin       = "perm.cannot_disable_last_admin"
	KeySystemRoleCannotDelete       = "perm.system_role_cannot_delete"
	KeySystemRoleCannotRename       = "perm.system_role_cannot_rename"
	KeyPlatformNoAvatarUpload       = "perm.platform_no_avatar_upload"
	KeyWavFileTooLarge              = "validation.wav_file_too_large"
	KeyOnlyQdrantSupported          = "validation.only_qdrant_supported"
	KeyVectorProviderInvalid        = "validation.vector_provider_invalid"
	KeyEndpointURLRequired          = "validation.endpoint_url_required"
	KeyNameEndpointURLRequired      = "validation.name_endpoint_url_required"
	KeyNameURLRequired              = "validation.name_url_required"
	KeyAllowedRouteIDsRequired      = "validation.allowed_route_ids_required"
	KeyBillNotFinalized             = "validation.bill_not_finalized"
	KeyEmailExistsConflict          = "duplicate.email_exists"
	KeyConfigKeyExists              = "duplicate.config_key_exists"
	KeyFromToVersionIdsRequired     = "validation.from_to_version_ids_required"
	KeyWavFileRequired              = "validation.wav_file_required"
	KeyTenantUserLimitReached       = "tenant.user_limit_reached"
	KeyUnsupportedTransport         = "validation.unsupported_transport"
	KeyProviderRequired             = "validation.provider_required"
	KeyVoiceIDRequired              = "validation.voice_id_required"
	KeyVoicePreviewConfigMissing    = "validation.voice_preview_config_missing"
	KeyVoicePreviewEmpty            = "validation.voice_preview_empty"
	KeyVoicePreviewRealtimeUnsupported = "validation.voice_preview_realtime_unsupported"
	KeyStatusActiveOrDisabled       = "validation.status_active_or_disabled"
	KeyScriptSpecOrTemplateRequired = "validation.script_spec_or_template_required"
	KeyRatingInvalid                = "validation.rating_invalid"
	KeyRouteNotOpenPlatform         = "aksk.route_not_open_platform"
	KeyRouteIDNotOpenPlatform       = "aksk.route_id_not_open_platform"
	KeyDateEndAfterToday            = "validation.date_end_after_today"
	KeyDateRangeExceed90            = "validation.date_range_exceed_90"
)

func catalog(locale string) map[string]string {
	switch ResolveLocale(locale) {
	case LocaleEnUS:
		return enUS
	case LocaleZhTW:
		return zhTW
	case LocaleJaJP:
		return jaJP
	default:
		return zhCN
	}
}

// T returns a localized string for the given locale and key.
func T(locale, key string, args ...any) string {
	msg, ok := catalog(locale)[key]
	if !ok || msg == "" {
		if fb, ok := zhCN[key]; ok {
			msg = fb
		} else {
			return key
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

// TGin resolves locale from gin context.
func TGin(c *gin.Context, key string, args ...any) string {
	return T(LocaleFromGin(c), key, args...)
}

package i18n

// Message keys for i18n translations
// Use these constants instead of hardcoded strings

// Common error messages
const (
	MsgInvalidParams     = "common.invalid_params"
	MsgDatabaseError     = "common.database_error"
	MsgRetryLater        = "common.retry_later"
	MsgGenerateFailed    = "common.generate_failed"
	MsgNotFound          = "common.not_found"
	MsgUnauthorized      = "common.unauthorized"
	MsgForbidden         = "common.forbidden"
	MsgInvalidId         = "common.invalid_id"
	MsgIdEmpty           = "common.id_empty"
	MsgFeatureDisabled   = "common.feature_disabled"
	MsgOperationSuccess  = "common.operation_success"
	MsgOperationFailed   = "common.operation_failed"
	MsgUpdateSuccess     = "common.update_success"
	MsgUpdateFailed      = "common.update_failed"
	MsgCreateSuccess     = "common.create_success"
	MsgCreateFailed      = "common.create_failed"
	MsgDeleteSuccess     = "common.delete_success"
	MsgDeleteFailed      = "common.delete_failed"
	MsgAlreadyExists     = "common.already_exists"
	MsgNameCannotBeEmpty = "common.name_cannot_be_empty"
	MsgBatchTooMany      = "common.batch_too_many"
)

// Auth middleware messages
const (
	MsgAuthNotLoggedIn           = "auth.not_logged_in"
	MsgAuthAccessTokenInvalid    = "auth.access_token_invalid"
	MsgAuthUserInfoInvalid       = "auth.user_info_invalid"
	MsgAuthUserIdNotProvided     = "auth.user_id_not_provided"
	MsgAuthUserIdFormatError     = "auth.user_id_format_error"
	MsgAuthUserIdMismatch        = "auth.user_id_mismatch"
	MsgAuthUserBanned            = "auth.user_banned"
	MsgAuthInsufficientPrivilege = "auth.insufficient_privilege"
)

// Token related messages
const (
	MsgTokenNameTooLong          = "token.name_too_long"
	MsgTokenQuotaNegative        = "token.quota_negative"
	MsgTokenQuotaExceedMax       = "token.quota_exceed_max"
	MsgTokenGenerateFailed       = "token.generate_failed"
	MsgTokenGetInfoFailed        = "token.get_info_failed"
	MsgTokenExpiredCannotEnable  = "token.expired_cannot_enable"
	MsgTokenExhaustedCannotEable = "token.exhausted_cannot_enable"
	MsgTokenInvalid              = "token.invalid"
	MsgTokenNotProvided          = "token.not_provided"
	MsgTokenExpired              = "token.expired"
	MsgTokenExhausted            = "token.exhausted"
	MsgTokenStatusUnavailable    = "token.status_unavailable"
	MsgTokenDbError              = "token.db_error"
)

// Redemption related messages
const (
	MsgRedemptionNameLength        = "redemption.name_length"
	MsgRedemptionCountPositive     = "redemption.count_positive"
	MsgRedemptionCountMax          = "redemption.count_max"
	MsgRedemptionCreateFailed      = "redemption.create_failed"
	MsgRedemptionInvalid           = "redemption.invalid"
	MsgRedemptionUsed              = "redemption.used"
	MsgRedemptionExpired           = "redemption.expired"
	MsgRedemptionFailed            = "redemption.failed"
	MsgRedemptionNotProvided       = "redemption.not_provided"
	MsgRedemptionExpireTimeInvalid = "redemption.expire_time_invalid"
)

// User related messages
const (
	MsgUserPasswordLoginDisabled     = "user.password_login_disabled"
	MsgUserRegisterDisabled          = "user.register_disabled"
	MsgUserPasswordRegisterDisabled  = "user.password_register_disabled"
	MsgUserUsernameOrPasswordEmpty   = "user.username_or_password_empty"
	MsgUserUsernameOrPasswordError   = "user.username_or_password_error"
	MsgUserEmailOrPasswordEmpty      = "user.email_or_password_empty"
	MsgUserExists                    = "user.exists"
	MsgUserNotExists                 = "user.not_exists"
	MsgUserDisabled                  = "user.disabled"
	MsgUserSessionSaveFailed         = "user.session_save_failed"
	MsgUserRequire2FA                = "user.require_2fa"
	MsgUserEmailVerificationRequired = "user.email_verification_required"
	MsgUserVerificationCodeError     = "user.verification_code_error"
	MsgUserInputInvalid              = "user.input_invalid"
	MsgUserNoPermissionSameLevel     = "user.no_permission_same_level"
	MsgUserNoPermissionHigherLevel   = "user.no_permission_higher_level"
	MsgUserCannotCreateHigherLevel   = "user.cannot_create_higher_level"
	MsgUserCannotDeleteRootUser      = "user.cannot_delete_root_user"
	MsgUserCannotDisableRootUser     = "user.cannot_disable_root_user"
	MsgUserCannotDemoteRootUser      = "user.cannot_demote_root_user"
	MsgUserAlreadyAdmin              = "user.already_admin"
	MsgUserAlreadyCommon             = "user.already_common"
	MsgUserAdminCannotPromote        = "user.admin_cannot_promote"
	MsgUserOriginalPasswordError     = "user.original_password_error"
	MsgUserInviteQuotaInsufficient   = "user.invite_quota_insufficient"
	MsgUserTransferQuotaMinimum      = "user.transfer_quota_minimum"
	MsgUserTransferSuccess           = "user.transfer_success"
	MsgUserTransferFailed            = "user.transfer_failed"
	MsgUserTopUpProcessing           = "user.topup_processing"
	MsgUserRegisterFailed            = "user.register_failed"
	MsgUserDefaultTokenFailed        = "user.default_token_failed"
	MsgUserAffCodeEmpty              = "user.aff_code_empty"
	MsgUserEmailEmpty                = "user.email_empty"
	MsgUserGitHubIdEmpty             = "user.github_id_empty"
	MsgUserDiscordIdEmpty            = "user.discord_id_empty"
	MsgUserOidcIdEmpty               = "user.oidc_id_empty"
	MsgUserWeChatIdEmpty             = "user.wechat_id_empty"
	MsgUserTelegramIdEmpty           = "user.telegram_id_empty"
	MsgUserTelegramNotBound          = "user.telegram_not_bound"
	MsgUserLinuxDOIdEmpty            = "user.linux_do_id_empty"
	MsgUserQuotaChangeZero           = "user.quota_change_zero"
)

// Quota related messages
const (
	MsgQuotaNegative        = "quota.negative"
	MsgQuotaExceedMax       = "quota.exceed_max"
	MsgQuotaInsufficient    = "quota.insufficient"
	MsgQuotaWarningInvalid  = "quota.warning_invalid"
	MsgQuotaThresholdGtZero = "quota.threshold_gt_zero"
)

// Subscription related messages
const (
	MsgSubscriptionNotEnabled       = "subscription.not_enabled"
	MsgSubscriptionTitleEmpty       = "subscription.title_empty"
	MsgSubscriptionPriceNegative    = "subscription.price_negative"
	MsgSubscriptionPriceMax         = "subscription.price_max"
	MsgSubscriptionPurchaseLimitNeg = "subscription.purchase_limit_negative"
	MsgSubscriptionQuotaNegative    = "subscription.quota_negative"
	MsgSubscriptionGroupNotExists   = "subscription.group_not_exists"
	MsgSubscriptionResetCycleGtZero = "subscription.reset_cycle_gt_zero"
	MsgSubscriptionPurchaseMax      = "subscription.purchase_max"
	MsgSubscriptionInvalidId        = "subscription.invalid_id"
	MsgSubscriptionInvalidUserId    = "subscription.invalid_user_id"
)

// Payment related messages
const (
	MsgPaymentNotConfigured    = "payment.not_configured"
	MsgPaymentMethodNotExists  = "payment.method_not_exists"
	MsgPaymentCallbackError    = "payment.callback_error"
	MsgPaymentCreateFailed     = "payment.create_failed"
	MsgPaymentStartFailed      = "payment.start_failed"
	MsgPaymentAmountTooLow     = "payment.amount_too_low"
	MsgPaymentStripeNotConfig  = "payment.stripe_not_configured"
	MsgPaymentWebhookNotConfig = "payment.webhook_not_configured"
	MsgPaymentPriceIdNotConfig = "payment.price_id_not_configured"
	MsgPaymentCreemNotConfig   = "payment.creem_not_configured"
)

// Topup related messages
const (
	MsgTopupNotProvided    = "topup.not_provided"
	MsgTopupOrderNotExists = "topup.order_not_exists"
	MsgTopupOrderStatus    = "topup.order_status"
	MsgTopupFailed         = "topup.failed"
	MsgTopupInvalidQuota   = "topup.invalid_quota"
)

// Channel related messages
const (
	MsgChannelNotExists          = "channel.not_exists"
	MsgChannelIdFormatError      = "channel.id_format_error"
	MsgChannelNoAvailableKey     = "channel.no_available_key"
	MsgChannelGetListFailed      = "channel.get_list_failed"
	MsgChannelGetTagsFailed      = "channel.get_tags_failed"
	MsgChannelGetKeyFailed       = "channel.get_key_failed"
	MsgChannelGetOllamaFailed    = "channel.get_ollama_failed"
	MsgChannelQueryFailed        = "channel.query_failed"
	MsgChannelNoValidUpstream    = "channel.no_valid_upstream"
	MsgChannelUpstreamSaturated  = "channel.upstream_saturated"
	MsgChannelGetAvailableFailed = "channel.get_available_failed"
)

// Model related messages
const (
	MsgModelNameEmpty     = "model.name_empty"
	MsgModelNameExists    = "model.name_exists"
	MsgModelIdMissing     = "model.id_missing"
	MsgModelGetListFailed = "model.get_list_failed"
	MsgModelGetFailed     = "model.get_failed"
	MsgModelResetSuccess  = "model.reset_success"
)

// Vendor related messages
const (
	MsgVendorNameEmpty  = "vendor.name_empty"
	MsgVendorNameExists = "vendor.name_exists"
	MsgVendorIdMissing  = "vendor.id_missing"
)

// Group related messages
const (
	MsgGroupNameTypeEmpty = "group.name_type_empty"
	MsgGroupNameExists    = "group.name_exists"
	MsgGroupIdMissing     = "group.id_missing"
)

// Checkin related messages
const (
	MsgCheckinDisabled     = "checkin.disabled"
	MsgCheckinAlreadyToday = "checkin.already_today"
	MsgCheckinFailed       = "checkin.failed"
	MsgCheckinQuotaFailed  = "checkin.quota_failed"
)

// Passkey related messages
const (
	MsgPasskeyCreateFailed  = "passkey.create_failed"
	MsgPasskeyLoginAbnormal = "passkey.login_abnormal"
	MsgPasskeyUpdateFailed  = "passkey.update_failed"
	MsgPasskeyInvalidUserId = "passkey.invalid_user_id"
	MsgPasskeyVerifyFailed  = "passkey.verify_failed"
)

// 2FA related messages
const (
	MsgTwoFANotEnabled    = "twofa.not_enabled"
	MsgTwoFAUserIdEmpty   = "twofa.user_id_empty"
	MsgTwoFAAlreadyExists = "twofa.already_exists"
	MsgTwoFARecordIdEmpty = "twofa.record_id_empty"
	MsgTwoFACodeInvalid   = "twofa.code_invalid"
)

// Rate limit related messages
const (
	MsgRateLimitReached      = "rate_limit.reached"
	MsgRateLimitTotalReached = "rate_limit.total_reached"
)

// Setting related messages
const (
	MsgSettingInvalidType      = "setting.invalid_type"
	MsgSettingWebhookEmpty     = "setting.webhook_empty"
	MsgSettingWebhookInvalid   = "setting.webhook_invalid"
	MsgSettingEmailInvalid     = "setting.email_invalid"
	MsgSettingBarkUrlEmpty     = "setting.bark_url_empty"
	MsgSettingBarkUrlInvalid   = "setting.bark_url_invalid"
	MsgSettingGotifyUrlEmpty   = "setting.gotify_url_empty"
	MsgSettingGotifyTokenEmpty = "setting.gotify_token_empty"
	MsgSettingGotifyUrlInvalid = "setting.gotify_url_invalid"
	MsgSettingUrlMustHttp      = "setting.url_must_http"
	MsgSettingSaved            = "setting.saved"
)

// Deployment related messages (io.net)
const (
	MsgDeploymentNotEnabled     = "deployment.not_enabled"
	MsgDeploymentIdRequired     = "deployment.id_required"
	MsgDeploymentContainerIdReq = "deployment.container_id_required"
	MsgDeploymentNameEmpty      = "deployment.name_empty"
	MsgDeploymentNameTaken      = "deployment.name_taken"
	MsgDeploymentHardwareIdReq  = "deployment.hardware_id_required"
	MsgDeploymentHardwareInvId  = "deployment.hardware_invalid_id"
	MsgDeploymentApiKeyRequired = "deployment.api_key_required"
	MsgDeploymentInvalidPayload = "deployment.invalid_payload"
	MsgDeploymentNotFound       = "deployment.not_found"
)

// Performance related messages
const (
	MsgPerfDiskCacheCleared = "performance.disk_cache_cleared"
	MsgPerfStatsReset       = "performance.stats_reset"
	MsgPerfGcExecuted       = "performance.gc_executed"
)

// Ability related messages
const (
	MsgAbilityDbCorrupted   = "ability.db_corrupted"
	MsgAbilityRepairRunning = "ability.repair_running"
)

// OAuth related messages
const (
	MsgOAuthInvalidCode     = "oauth.invalid_code"
	MsgOAuthGetUserErr      = "oauth.get_user_error"
	MsgOAuthAccountUsed     = "oauth.account_used"
	MsgOAuthUnknownProvider = "oauth.unknown_provider"
	MsgOAuthStateInvalid    = "oauth.state_invalid"
	MsgOAuthNotEnabled      = "oauth.not_enabled"
	MsgOAuthUserDeleted     = "oauth.user_deleted"
	MsgOAuthUserBanned      = "oauth.user_banned"
	MsgOAuthBindSuccess     = "oauth.bind_success"
	MsgOAuthAlreadyBound    = "oauth.already_bound"
	MsgOAuthConnectFailed   = "oauth.connect_failed"
	MsgOAuthTokenFailed     = "oauth.token_failed"
	MsgOAuthUserInfoEmpty   = "oauth.user_info_empty"
	MsgOAuthTrustLevelLow   = "oauth.trust_level_low"
)

// Model layer error messages (for translation in controller)
const (
	MsgRedeemFailed          = "redeem.failed"
	MsgCreateDefaultTokenErr = "user.create_default_token_error"
	MsgUuidDuplicate         = "common.uuid_duplicate"
	MsgInvalidInput          = "common.invalid_input"
)

// Distributor related messages
const (
	MsgDistributorInvalidRequest      = "distributor.invalid_request"
	MsgDistributorInvalidChannelId    = "distributor.invalid_channel_id"
	MsgDistributorChannelDisabled     = "distributor.channel_disabled"
	MsgDistributorTokenNoModelAccess  = "distributor.token_no_model_access"
	MsgDistributorTokenModelForbidden = "distributor.token_model_forbidden"
	MsgDistributorModelNameRequired   = "distributor.model_name_required"
	MsgDistributorGroupAccessDenied   = "distributor.group_access_denied"
	MsgDistributorGetChannelFailed    = "distributor.get_channel_failed"
	MsgDistributorNoAvailableChannel  = "distributor.no_available_channel"
	MsgDistributorInvalidMidjourney   = "distributor.invalid_midjourney_request"
	MsgDistributorInvalidParseModel   = "distributor.invalid_request_parse_model"
)

// Response validation messages
const (
	MsgResponseUsernameTooShort    = "response.username_too_short"
	MsgResponseUsernameInvalidFmt = "response.username_invalid_format"
	MsgResponseEmailExists        = "response.email_already_exists"
	MsgResponsePasswordTooShort   = "response.password_too_short"
	MsgResponseCaptchaRequired    = "response.captcha_required"
	MsgResponseCaptchaInvalid     = "response.captcha_invalid"
)

// Handler messages
const (
	MsgHandlerTwofaLoadFailed         = "handler.twofa.load_failed"
	MsgHandlerTwofaStatus             = "handler.twofa.status"
	MsgHandlerTwofaNotEnabled         = "handler.twofa.not_enabled"
	MsgHandlerTwofaBackupGenFailed    = "handler.twofa.backup_generate_failed"
	MsgHandlerTwofaBackupGenerated    = "handler.twofa.backup_generated"
	MsgHandlerTwofaBackupAccepted     = "handler.twofa.backup_accepted"
	MsgHandlerTwofaDisableFailed      = "handler.twofa.disable_failed"
	MsgHandlerTwofaDisabled           = "handler.twofa.disabled"
	MsgHandlerWechatLoginCodeGenerated = "handler.wechat.login_code_generated"
	MsgHandlerWechatConfigCheck       = "handler.wechat.config_check"
	MsgHandlerWechatSessionExpired    = "handler.wechat.session_expired"
	MsgHandlerWechatSessionStatus     = "handler.wechat.session_status"
	MsgHandlerWechatCheckLogin        = "handler.wechat.check_login"
	MsgHandlerWechatPending           = "handler.wechat.pending"
)

// Auth handler messages
const (
	MsgHandlerAuthWechatLoginCode  = "handler.auth.wechat_login_code"
	MsgHandlerAuthWechatConfigCheck = "handler.auth.wechat_config_check"
	MsgHandlerAuthWechatSessionExpired = "handler.auth.wechat_session_expired"
	MsgHandlerAuthWechatSessionStatus = "handler.auth.wechat_session_status"
	MsgHandlerAuthWechatCheckLogin = "handler.auth.wechat_check_login"
	MsgHandlerAuthWechatBindCode  = "handler.auth.wechat_bind_code"
	MsgHandlerAuthWechatBindStatus = "handler.auth.wechat_bind_status"
	MsgHandlerAuthOIDCTokenIssued = "handler.auth.oidc_token_issued"
	MsgHandlerAuthTokenRefreshed  = "handler.auth.token_refreshed"
	MsgHandlerAuthLogoutSuccess   = "handler.auth.logout_success"
	MsgHandlerAuthLoginSuccess    = "handler.auth.login_success"
	MsgHandlerAuthUserNotExists   = "handler.auth.user_not_exists"
)

// Auth handler additional messages
const (
	MsgHandlerAuthWechatBindCodeGenerated = "handler.auth.wechat_bind_code_generated"
	MsgHandlerAuthLoginFailed             = "handler.auth.login_failed"
	MsgHandlerAuthTooManyAttempts         = "handler.auth.too_many_attempts"
	MsgHandlerAuthAccountLocked           = "handler.auth.account_locked"
	MsgHandlerAuthEmailRequired           = "handler.auth.email_required"
	MsgHandlerAuthPasswordEmpty           = "handler.auth.password_empty"
	MsgHandlerAuthUserNotFoundEmail       = "handler.auth.user_not_found_email"
	MsgHandlerAuthPasswordError           = "handler.auth.password_error"
	MsgHandlerAuthEmailVerificationRequired = "handler.auth.email_verification_required"
	MsgHandlerAuthCaptchaError            = "handler.auth.captcha_error"
	MsgHandlerAuthInvalidAuthToken        = "handler.auth.invalid_auth_token"
	MsgHandlerAuthUserNotFound            = "handler.auth.user_not_found"
	MsgHandlerAuthUserNoAuth              = "handler.auth.user_no_auth"
	MsgHandlerAuthTwoFARequired           = "handler.auth.twofa_required"
	MsgHandlerAuthTwoFACodeInvalid        = "handler.auth.twofa_code_invalid"
	MsgHandlerAuthDeviceVerificationRequired = "handler.auth.device_verification_required"
	MsgHandlerAuthLoginSuccessful         = "handler.auth.login_successful"
	MsgHandlerAuthSignupSuccess           = "handler.auth.signup_success"
	MsgHandlerAuthInvalidRequest          = "handler.auth.invalid_request"
	MsgHandlerAuthUpdateUserFailed        = "handler.auth.update_user_failed"
	MsgHandlerAuthUpdateProfileFailed     = "handler.auth.update_profile_failed"
	MsgHandlerAuthGetUserFailed           = "handler.auth.get_user_failed"
	MsgHandlerAuthUpdateUserSuccess       = "handler.auth.update_user_success"
)

// Server handler messages
const (
	MsgServerLLMUsageFetchFailed    = "handler.server.llm_usage_fetch_failed"
	MsgServerLLMUsageFetched        = "handler.server.llm_usage_fetched"
	MsgServerLLMUsageSummaryFailed  = "handler.server.llm_usage_summary_failed"
	MsgServerLLMUsageSummary        = "handler.server.llm_usage_summary"
	MsgServerChannelListFailed      = "handler.server.channel_list_failed"
	MsgServerChannelFetched         = "handler.server.channel_fetched"
	MsgServerChannelNotFound        = "handler.server.channel_not_found"
	MsgServerChannelCreateFailed    = "handler.server.channel_create_failed"
	MsgServerChannelCreated         = "handler.server.channel_created"
	MsgServerChannelUpdateFailed    = "handler.server.channel_update_failed"
	MsgServerChannelUpdated         = "handler.server.channel_updated"
	MsgServerChannelDeleteFailed    = "handler.server.channel_delete_failed"
	MsgServerChannelDeleted         = "handler.server.channel_deleted"
	MsgServerAbilitySyncFailed      = "handler.server.ability_sync_failed"
	MsgServerAbilitySynced          = "handler.server.ability_synced"
	MsgServerAbilityListFailed      = "handler.server.ability_list_failed"
	MsgServerAbilityFetched         = "handler.server.ability_fetched"
	MsgServerModelMetaListFailed    = "handler.server.model_meta_list_failed"
	MsgServerModelMetaFetched       = "handler.server.model_meta_fetched"
	MsgServerModelMetaCreateFailed  = "handler.server.model_meta_create_failed"
	MsgServerModelMetaCreated       = "handler.server.model_meta_created"
	MsgServerModelMetaUpdateFailed  = "handler.server.model_meta_update_failed"
	MsgServerModelMetaUpdated       = "handler.server.model_meta_updated"
	MsgServerModelMetaDeleteFailed  = "handler.server.model_meta_delete_failed"
	MsgServerModelMetaDeleted       = "handler.server.model_meta_deleted"
)

// Group handler messages
const (
	MsgGroupUnauthorized       = "handler.group.unauthorized"
	MsgGroupInvalidParams      = "handler.group.invalid_params"
	MsgGroupPersonalAutoCreated = "handler.group.personal_auto_created"
	MsgGroupCreateFailed       = "handler.group.create_failed"
	MsgGroupCreateMemberFailed = "handler.group.create_member_failed"
	MsgGroupCreateSuccess      = "handler.group.create_success"
	MsgGroupQueryListFailed    = "handler.group.query_list_failed"
	MsgGroupQuerySuccess       = "handler.group.query_success"
	MsgGroupInvalidId          = "handler.group.invalid_id"
	MsgGroupNotExists          = "handler.group.not_exists"
	MsgGroupQueryFailed        = "handler.group.query_failed"
	MsgGroupPermissionDenied   = "handler.group.permission_denied"
	MsgGroupQueryMembersFailed = "handler.group.query_members_failed"
)

// Chat handler messages (Swipe / Branch / WorldInfo / Persona)
const (
	// Swipe / Branching
	MsgChatAlternativesFetched   = "handler.chat.alternatives_fetched"
	MsgChatAlternativesFailed    = "handler.chat.alternatives_failed"
	MsgChatAlternativeActivated  = "handler.chat.alternative_activated"
	MsgChatActivateFailed        = "handler.chat.activate_failed"
	MsgChatRegenerateFailed      = "handler.chat.regenerate_failed"
	MsgChatRegenerated           = "handler.chat.regenerated"
	MsgChatBranchCreated         = "handler.chat.branch_created"
	MsgChatBranchFailed          = "handler.chat.branch_failed"

	// World Info
	MsgWorldInfoListFetched     = "handler.chat.world_info_list_fetched"
	MsgWorldInfoListFailed      = "handler.chat.world_info_list_failed"
	MsgWorldInfoCreated         = "handler.chat.world_info_created"
	MsgWorldInfoCreateFailed    = "handler.chat.world_info_create_failed"
	MsgWorldInfoUpdated         = "handler.chat.world_info_updated"
	MsgWorldInfoUpdateFailed    = "handler.chat.world_info_update_failed"
	MsgWorldInfoDeleted         = "handler.chat.world_info_deleted"
	MsgWorldInfoDeleteFailed    = "handler.chat.world_info_delete_failed"
	MsgWorldInfoActivateFailed  = "handler.chat.world_info_activate_failed"
	MsgWorldInfoActivated       = "handler.chat.world_info_activated"
	MsgWorldInfoInjectFailed    = "handler.chat.world_info_inject_failed"
	MsgWorldInfoInjected        = "handler.chat.world_info_injected"

	// Persona
	MsgPersonaListFetched       = "handler.chat.persona_list_fetched"
	MsgPersonaListFailed        = "handler.chat.persona_list_failed"
	MsgPersonaCreated           = "handler.chat.persona_created"
	MsgPersonaCreateFailed      = "handler.chat.persona_create_failed"
	MsgPersonaUpdated           = "handler.chat.persona_updated"
	MsgPersonaUpdateFailed      = "handler.chat.persona_update_failed"
	MsgPersonaDeleted           = "handler.chat.persona_deleted"
	MsgPersonaDeleteFailed      = "handler.chat.persona_delete_failed"
	MsgPersonaDefaultSet        = "handler.chat.persona_default_set"
	MsgPersonaDefaultSetFailed  = "handler.chat.persona_default_set_failed"
	MsgPersonaInjectFailed      = "handler.chat.persona_inject_failed"
	MsgPersonaInjected          = "handler.chat.persona_injected"
)

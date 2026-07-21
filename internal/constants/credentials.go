package constants

// Credential lifecycle.
const (
	CredentialStatusActive       = "active"
	CredentialStatusDisabled     = "disabled"
	CredentialPermissionWildcard = "*"

	// CredentialKindPlatformBundle uses tenant AI config from platform admin (no per-key provider JSON).
	CredentialKindPlatformBundle = "platform_bundle"
	// CredentialKindUserBundle stores llm/asr/tts/realtime JSON on the key (tenant self-service).
	CredentialKindUserBundle = "user_bundle"

	// CredentialKindAIBundle is a legacy alias for user_bundle.
	CredentialKindAIBundle = "user_bundle"
	// CredentialKindAPIAccess deprecated; treated as user_bundle for storage only.
	CredentialKindAPIAccess = "api_access"
)

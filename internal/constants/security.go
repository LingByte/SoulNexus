package constants

// Security-related environment variable names (fail-closed unless explicitly true).

const (
	ENVUploadsRecordingsPublic     = "UPLOADS_RECORDINGS_PUBLIC"
	ENVVoiceDialogAllowEmptyToken  = "VOICE_DIALOG_ALLOW_EMPTY_TOKEN"
	ENVCredentialAllowEmptyAllowIP = "CREDENTIAL_ALLOW_EMPTY_ALLOW_IP" // dev-only: AK/SK without IP allowlist
)

package constants

// Security-related environment variable names (fail-closed unless explicitly true).

const (
	ENVUploadsRecordingsPublic      = "UPLOADS_RECORDINGS_PUBLIC"
	ENVVoiceDialogAllowEmptyToken   = "VOICE_DIALOG_ALLOW_EMPTY_TOKEN"
	ENVCredentialAllowEmptyAllowIP  = "CREDENTIAL_ALLOW_EMPTY_ALLOW_IP" // dev-only: AK/SK without IP allowlist
	ENVNLUEnabled                   = "NLU_ENABLED"                     // master switch for NLU lab (ONNX intent)
	ENVNLUMode                      = "NLU_MODE"                        // embedding (default) or classifier
	ENVNLUModel                     = "NLU_MODEL"                       // path to .onnx (embedding or classifier)
	ENVNLUTokenizer                 = "NLU_TOKENIZER"                   // path to tokenizer.json
	ENVNLUORTLib                    = "NLU_ORT_LIB"                     // onnxruntime shared library; or ONNXRUNTIME_SHARED_LIBRARY_PATH
	ENVNLUIntentsConfig             = "NLU_INTENTS_CONFIG"              // optional intents JSON; empty = embedded default
	ENVNLUSeq                       = "NLU_SEQ"                         // sequence length (default 128)
	ENVNLUCoreML                    = "NLU_COREML"                      // darwin: use CoreML EP when true
)

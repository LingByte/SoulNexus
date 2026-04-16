package constants

// SIP hybrid script step types.
const (
	SIPScriptStepSay       = "say"
	SIPScriptStepListen    = "listen"
	SIPScriptStepLLMReply  = "llm_reply"
	SIPScriptStepCondition = "condition"
	SIPScriptStepEnd       = "end"
)

// SIP hybrid script runtime event results.
const (
	SIPScriptRunStarted     = "started"
	SIPScriptRunMatched     = "matched" // listen step: ASR text received (after OnListen succeeds)
	SIPScriptRunFailed      = "failed"
	SIPScriptRunTimeout     = "timeout"
	SIPScriptRunEnded       = "ended"
	SIPScriptRunRouteFailed = "route_failed" // listen: LLM branch routing failed or LLM not configured
)

// Env vars for campaign script listen latency (read via utils.GetEnv).
const (
	// SIP_SCRIPT_LISTEN_AFTER_TTS_TAIL: extra ms after say before listen timeout fully applies.
	// "0", "false", "off", "no" disables the tail (only script listen_timeout_ms + silence_timeout_ms apply).
	EnvSIPScriptListenAfterTTSTail = "SIP_SCRIPT_LISTEN_AFTER_TTS_TAIL"
	// SIP_SCRIPT_LISTEN_TAIL_MS_MAX caps the computed tail (default 2000; was 6000).
	EnvSIPScriptListenTailMSMax = "SIP_SCRIPT_LISTEN_TAIL_MS_MAX"
	// SIP_SCRIPT_LISTEN_TAIL_MS_MIN floor for non-zero tail (default 400).
	EnvSIPScriptListenTailMSMin = "SIP_SCRIPT_LISTEN_TAIL_MS_MIN"
	// SIP_SCRIPT_LISTEN_POLL_MS: DB poll interval while waiting for next user turn (default 120).
	EnvSIPScriptListenPollMS = "SIP_SCRIPT_LISTEN_POLL_MS"

	// CHECK_LLM_*: OpenAI-compatible API used to pick listen-step branches (DashScope, OpenAI, etc.).
	EnvCHECKLLMProvider       = "CHECK_LLM_PROVIDER"
	EnvCHECKLLMBaseURL        = "CHECK_LLM_BASEURL"
	EnvCHECKLLMAPIKey         = "CHECK_LLM_APIKEY"
	EnvCHECKLLMModel          = "CHECK_LLM_MODEL"
	EnvCHECKLLMRouteTimeoutMS = "CHECK_LLM_ROUTE_TIMEOUT_MS"
	EnvCHECKLLMRouteDisabled  = "CHECK_LLM_ROUTE_DISABLED" // "1" or "true" disables LLM listen routing (then listen+transitions is invalid at runtime)
	// CHECK_LLM_ROUTE_LEGACY_JSON=1: model returns {"next_id":"..."}; default uses compact {"i":N} index (faster, fewer tokens).
	EnvCHECKLLMRouteLegacyJSON = "CHECK_LLM_ROUTE_LEGACY_JSON"
	// CHECK_LLM_ROUTE_MAX_TOKENS: max completion tokens for route call (default 32, range 8–128).
	EnvCHECKLLMRouteMaxTokens = "CHECK_LLM_ROUTE_MAX_TOKENS"
	// SIP_SCRIPT_LLM_FAIL_PROMPT: spoken when listen LLM routing is unavailable or fails (default below).
	EnvSIPScriptLLMFailPrompt = "SIP_SCRIPT_LLM_FAIL_PROMPT"
)

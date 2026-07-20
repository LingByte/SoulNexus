package constants

// Operation log action verbs (stored in operation_logs.action).
const (
	OpActionCreate     = "create"
	OpActionUpdate     = "update"
	OpActionDelete     = "delete"
	OpActionRestore    = "restore"
	OpActionEnable     = "enable"
	OpActionDisable    = "disable"
	OpActionReorder    = "reorder"
	OpActionStart      = "start"
	OpActionPause      = "pause"
	OpActionResume     = "resume"
	OpActionStop       = "stop"
	OpActionAPICall    = "api_call"
	OpActionPublish    = "publish"
	OpActionRegenerate = "regenerate"
)

// Operation log resource types (stored in operation_logs.resource).
const (
	OpResourceCredential        = "credential"
	OpResourceTenantUser        = "tenant_user"
	OpResourceTenant            = "tenant"
	OpResourceTenantRole        = "tenant_role"
	OpResourceTenantGroup       = "tenant_group"
	OpResourcePlatformAdmin     = "platform_admin"
	OpResourceSystemConfig      = "system_config"
	OpResourceVoiceClone        = "voice_clone"
	OpResourceVoiceTrainingTask = "voice_training_task"
	OpResourceVoiceSynthesis    = "voice_synthesis"
	OpResourceAPI               = "api"
	OpResourceAssistant         = "assistant"
	OpResourceVoiceprint        = "voiceprint"
	OpResourceWorkflow          = "workflow"
	OpResourceMCPMarket         = "mcp_market"
)

// OperatorKind identifies who performed the operation.
const (
	OpOperatorTenantUser    = "tenant_user"
	OpOperatorPlatformAdmin = "platform_admin"
	OpOperatorCredential    = "credential"
	OpOperatorSystem        = "system"
)

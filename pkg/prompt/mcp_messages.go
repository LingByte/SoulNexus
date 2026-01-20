package prompt

// Method constant definitions
// Using consistent naming with schema.json
const (
	// Base protocol
	MethodInitialize               = "initialize"
	MethodNotificationsInitialized = "notifications/initialized"

	// Tool related
	MethodToolsList = "tools/list"
	MethodToolsCall = "tools/call"

	// Prompt related
	MethodPromptsList        = "prompts/list"
	MethodPromptsGet         = "prompts/get"
	MethodCompletionComplete = "completion/complete"

	// Resource related
	MethodResourcesList          = "resources/list"
	MethodResourcesRead          = "resources/read"
	MethodResourcesTemplatesList = "resources/templates/list"
	MethodResourcesSubscribe     = "resources/subscribe"
	MethodResourcesUnsubscribe   = "resources/unsubscribe"

	// Utilities
	MethodLoggingSetLevel = "logging/setLevel"
	MethodPing            = "ping"
)

// Implementation describes the name and version of an MCP implementation
// Corresponds to the "Implementation" definition in schema.json
type Implementation struct {
	// Name of the implementation
	Name string `json:"name"`
	// Version of the implementation
	Version string `json:"version"`
}

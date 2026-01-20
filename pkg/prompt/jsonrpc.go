package prompt

const (
	// JSONRPCVersion specifies the JSON-RPC version
	JSONRPCVersion = "2.0"

	// Standard JSON-RPC error codes
	ErrCodeParse          = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603

	// MCP custom error code range: -32000 to -32099
)

// RequestId is the base request id struct for all MCP requests.
type RequestId interface{}

// JSONRPCMessage represents a JSON-RPC message.
type JSONRPCMessage interface{}

// JSONRPCRequest represents a JSON-RPC request
// Conforms to the JSONRPCRequest definition in schema.json
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      RequestId   `json:"id,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Request
}

// JSONRPCError represents a JSON-RPC error response
// Conforms to the JSONRPCError definition in schema.json
type JSONRPCError struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      RequestId `json:"id,omitempty"`
	Error   struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error"`
}

// JSONRPCNotification represents a JSON-RPC notification
// Conforms to the JSONRPCNotification definition in schema.json
type JSONRPCNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Notification
}

// newJSONRPCErrorResponse creates a new JSON-RPC error response
func newJSONRPCErrorResponse(id interface{}, code int, message string, data interface{}) *JSONRPCError {
	errResp := &JSONRPCError{
		JSONRPC: JSONRPCVersion,
		ID:      id,
	}
	errResp.Error.Code = code
	errResp.Error.Message = message
	errResp.Error.Data = data
	return errResp
}

// newJSONRPCNotification creates a new JSON-RPC notification
func newJSONRPCNotification(notification Notification) *JSONRPCNotification {
	return &JSONRPCNotification{
		JSONRPC:      JSONRPCVersion,
		Notification: notification,
	}
}

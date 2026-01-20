package prompt

import (
	"testing"
)

func TestNewJSONRPCErrorResponse(t *testing.T) {
	id := "test-id"
	code := ErrCodeInvalidParams
	message := "Invalid parameters"
	data := map[string]string{"field": "value"}

	errResp := newJSONRPCErrorResponse(id, code, message, data)

	if errResp.JSONRPC != JSONRPCVersion {
		t.Errorf("Expected JSONRPC version %s, got %s", JSONRPCVersion, errResp.JSONRPC)
	}

	if errResp.ID != id {
		t.Errorf("Expected ID %v, got %v", id, errResp.ID)
	}

	if errResp.Error.Code != code {
		t.Errorf("Expected error code %d, got %d", code, errResp.Error.Code)
	}

	if errResp.Error.Message != message {
		t.Errorf("Expected error message %s, got %s", message, errResp.Error.Message)
	}

	if errResp.Error.Data == nil {
		t.Errorf("Expected error data, got nil")
	}
	// Verify data is set (can't directly compare maps)
	dataMap, ok := errResp.Error.Data.(map[string]string)
	if !ok {
		t.Errorf("Expected error data to be map[string]string")
	} else if dataMap["field"] != "value" {
		t.Errorf("Expected error data field 'value', got %v", dataMap["field"])
	}
}

func TestNewJSONRPCErrorResponse_NilID(t *testing.T) {
	errResp := newJSONRPCErrorResponse(nil, ErrCodeInternal, "Internal error", nil)

	if errResp.ID != nil {
		t.Errorf("Expected nil ID, got %v", errResp.ID)
	}
}

func TestNewJSONRPCErrorResponse_NilData(t *testing.T) {
	errResp := newJSONRPCErrorResponse("id", ErrCodeInternal, "Error", nil)

	if errResp.Error.Data != nil {
		t.Errorf("Expected nil data, got %v", errResp.Error.Data)
	}
}

func TestNewJSONRPCNotification(t *testing.T) {
	notification := Notification{
		Method: "test/method",
		Params: NotificationParams{
			AdditionalFields: map[string]interface{}{
				"key": "value",
			},
		},
	}

	jsonrpcNotif := newJSONRPCNotification(notification)

	if jsonrpcNotif.JSONRPC != JSONRPCVersion {
		t.Errorf("Expected JSONRPC version %s, got %s", JSONRPCVersion, jsonrpcNotif.JSONRPC)
	}

	if jsonrpcNotif.Method != notification.Method {
		t.Errorf("Expected method %s, got %s", notification.Method, jsonrpcNotif.Method)
	}

	if jsonrpcNotif.Params.AdditionalFields["key"] != "value" {
		t.Errorf("Expected params key 'value', got %v", jsonrpcNotif.Params.AdditionalFields["key"])
	}
}

func TestJSONRPCConstants(t *testing.T) {
	if JSONRPCVersion != "2.0" {
		t.Errorf("Expected JSONRPC version 2.0, got %s", JSONRPCVersion)
	}

	if ErrCodeParse != -32700 {
		t.Errorf("Expected parse error code -32700, got %d", ErrCodeParse)
	}

	if ErrCodeInvalidRequest != -32600 {
		t.Errorf("Expected invalid request error code -32600, got %d", ErrCodeInvalidRequest)
	}

	if ErrCodeMethodNotFound != -32601 {
		t.Errorf("Expected method not found error code -32601, got %d", ErrCodeMethodNotFound)
	}

	if ErrCodeInvalidParams != -32602 {
		t.Errorf("Expected invalid params error code -32602, got %d", ErrCodeInvalidParams)
	}

	if ErrCodeInternal != -32603 {
		t.Errorf("Expected internal error code -32603, got %d", ErrCodeInternal)
	}
}

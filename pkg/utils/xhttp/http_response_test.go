package xhttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccess_NoData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Success() status code = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["code"] != float64(0) {
		t.Fatalf("Success() code = %v, want 0", response["code"])
	}

	if response["msg"] != "success" {
		t.Fatalf("Success() msg = %q, want %q", response["msg"], "success")
	}

	if _, exists := response["data"]; exists {
		t.Fatalf("Success() should not have data field when no data provided")
	}
}

func TestSuccess_WithData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	testData := map[string]string{
		"key": "value",
	}

	Success(c, testData)

	if w.Code != http.StatusOK {
		t.Fatalf("Success() status code = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["code"] != float64(0) {
		t.Fatalf("Success() code = %v, want 0", response["code"])
	}

	if response["msg"] != "success" {
		t.Fatalf("Success() msg = %q, want %q", response["msg"], "success")
	}

	if response["data"] == nil {
		t.Fatalf("Success() should have data field when data provided")
	}
}

func TestSuccess_WithNilData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("Success() status code = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// When nil is explicitly passed, it should be treated as no data
	if _, exists := response["data"]; exists {
		t.Fatalf("Success() should not have data field when nil data provided")
	}
}

func TestSuccess_WithStringData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, "test string")

	if w.Code != http.StatusOK {
		t.Fatalf("Success() status code = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["data"] != "test string" {
		t.Fatalf("Success() data = %v, want %q", response["data"], "test string")
	}
}

func TestError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	errorCode := int32(401)
	errorMsg := "Unauthorized"

	Error(c, errorCode, errorMsg)

	if w.Code != http.StatusOK {
		t.Fatalf("Error() status code = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["code"] != float64(errorCode) {
		t.Fatalf("Error() code = %v, want %d", response["code"], errorCode)
	}

	if response["msg"] != errorMsg {
		t.Fatalf("Error() msg = %q, want %q", response["msg"], errorMsg)
	}

	if _, exists := response["data"]; exists {
		t.Fatalf("Error() should not have data field")
	}
}

func TestError_WithDifferentCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		code int32
		msg  string
	}{
		{400, "Bad Request"},
		{404, "Not Found"},
		{500, "Internal Server Error"},
		{601, "Custom Error"},
	}

	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			Error(c, tc.code, tc.msg)

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if response["code"] != float64(tc.code) {
				t.Fatalf("Error() code = %v, want %d", response["code"], tc.code)
			}

			if response["msg"] != tc.msg {
				t.Fatalf("Error() msg = %q, want %q", response["msg"], tc.msg)
			}
		})
	}
}

func TestResp_SetResult(t *testing.T) {
	resp := &Resp{}

	code := int32(200)
	msg := "Custom message"

	resp.SetResult(code, msg)

	if resp.Code != code {
		t.Fatalf("SetResult() code = %d, want %d", resp.Code, code)
	}

	if resp.Msg != msg {
		t.Fatalf("SetResult() msg = %q, want %q", resp.Msg, msg)
	}
}

func TestResp_Structure(t *testing.T) {
	// Test that Resp embeds Result correctly
	resp := &Resp{
		Result: Result{
			Code: 0,
			Msg:  "success",
		},
		Data: "test data",
	}

	if resp.Code != 0 {
		t.Fatalf("Resp.Code = %d, want 0", resp.Code)
	}

	if resp.Msg != "success" {
		t.Fatalf("Resp.Msg = %q, want %q", resp.Msg, "success")
	}

	if resp.Data != "test data" {
		t.Fatalf("Resp.Data = %v, want %q", resp.Data, "test data")
	}
}

package mediagen

import (
	"encoding/json"
	"fmt"
	"strings"
)

type volcengineErrorBody struct {
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func upstreamHTTPError(prefix string, status int, body []byte) *UpstreamError {
	msg := fmt.Sprintf("%s: HTTP %d", prefix, status)
	detail := parseVolcengineErrorDetail(body)
	if detail != "" {
		msg += " — " + detail
	}
	if status == 404 {
		msg += "（请在火山方舟控制台开通 Seedance 模型，并将 SEEDANCE_MODEL_ID 设为控制台「推理接入点」ID 或模型 ID）"
	}
	return &UpstreamError{Message: msg}
}

func parseVolcengineErrorDetail(body []byte) string {
	raw := strings.TrimSpace(string(body))
	if raw == "" {
		return ""
	}
	var parsed volcengineErrorBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		if len(raw) > 240 {
			raw = raw[:240] + "..."
		}
		return raw
	}
	if parsed.Error != nil {
		code := strings.TrimSpace(parsed.Error.Code)
		message := strings.TrimSpace(parsed.Error.Message)
		switch {
		case code != "" && message != "":
			return code + ": " + message
		case message != "":
			return message
		case code != "":
			return code
		}
	}
	if message := strings.TrimSpace(parsed.Message); message != "" {
		if code := strings.TrimSpace(parsed.Code); code != "" {
			return code + ": " + message
		}
		return message
	}
	if len(raw) > 240 {
		raw = raw[:240] + "..."
	}
	return raw
}

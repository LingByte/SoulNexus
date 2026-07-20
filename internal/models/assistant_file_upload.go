package models

import (
	"encoding/json"
	"strings"

	"gorm.io/datatypes"
)

// AgentFileUploadConfig controls debug/file chat attachments for an assistant.
type AgentFileUploadConfig struct {
	Enabled         bool `json:"enabled"`
	DocumentEnabled bool `json:"documentEnabled"`
	ImageEnabled    bool `json:"imageEnabled"`
	MaxFiles        int  `json:"maxFiles"`
	PDFEnhanced     bool `json:"pdfEnhanced"`
}

// ParseAgentFileUpload reads file upload settings from agent_config JSON.
func ParseAgentFileUpload(raw datatypes.JSON) AgentFileUploadConfig {
	out := AgentFileUploadConfig{MaxFiles: 8, DocumentEnabled: true}
	if len(raw) == 0 {
		return out
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return out
	}
	fu, ok := root["fileUpload"].(map[string]any)
	if !ok {
		return out
	}
	if v, ok := fu["enabled"].(bool); ok {
		out.Enabled = v
	}
	if v, ok := fu["documentEnabled"].(bool); ok {
		out.DocumentEnabled = v
	}
	if v, ok := fu["imageEnabled"].(bool); ok {
		out.ImageEnabled = v
	}
	if v, ok := fu["pdfEnhanced"].(bool); ok {
		out.PDFEnhanced = v
	}
	if n, ok := fu["maxFiles"].(float64); ok && n > 0 {
		out.MaxFiles = int(n)
	}
	if out.MaxFiles <= 0 {
		out.MaxFiles = 8
	}
	return out
}

// AgentFileUploadAllowsMIME checks whether the assistant accepts this content type.
func (c AgentFileUploadConfig) AllowsMIME(mime string) bool {
	if !c.Enabled {
		return false
	}
	mime = strings.ToLower(strings.TrimSpace(mime))
	if strings.HasPrefix(mime, "image/") {
		return c.ImageEnabled
	}
	return c.DocumentEnabled
}

package request

import "encoding/json"

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

type CredentialCreateReq struct {
	Name            string   `json:"name"`
	Kind            string   `json:"kind"`
	AllowIP         string   `json:"allowIp"`
	PermissionCodes []string `json:"permissionCodes"`
	AllowedRouteIDs []string `json:"allowedRouteIds"`
	ExpiresAt       *string  `json:"expiresAt"`
	VoiceMode       string   `json:"voiceMode"`
	AsrConfig       json.RawMessage `json:"asrConfig"`
	TtsConfig       json.RawMessage `json:"ttsConfig"`
	LlmConfig       json.RawMessage `json:"llmConfig"`
	RealtimeConfig  json.RawMessage `json:"realtimeConfig"`
}

type CredentialUpdateReq struct {
	Name            *string  `json:"name"`
	Kind            *string  `json:"kind"`
	AllowIP         *string  `json:"allowIp"`
	PermissionCodes []string `json:"permissionCodes"`
	AllowedRouteIDs []string `json:"allowedRouteIds"`
	ExpiresAt       *string  `json:"expiresAt"`
	VoiceMode       *string  `json:"voiceMode"`
	AsrConfig       json.RawMessage `json:"asrConfig"`
	TtsConfig       json.RawMessage `json:"ttsConfig"`
	LlmConfig       json.RawMessage `json:"llmConfig"`
	RealtimeConfig  json.RawMessage `json:"realtimeConfig"`
}

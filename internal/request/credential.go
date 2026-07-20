package request

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

type CredentialCreateReq struct {
	Name            string   `json:"name"`
	AllowIP         string   `json:"allowIp"`
	PermissionCodes []string `json:"permissionCodes"`
	AllowedRouteIDs []string `json:"allowedRouteIds"`
	ExpiresAt       *string  `json:"expiresAt"`
}

type CredentialUpdateReq struct {
	Name            *string  `json:"name"`
	AllowIP         *string  `json:"allowIp"`
	PermissionCodes []string `json:"permissionCodes"`
	AllowedRouteIDs []string `json:"allowedRouteIds"`
	ExpiresAt       *string  `json:"expiresAt"`
}

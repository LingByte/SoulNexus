// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package humax

import (
	dto "github.com/LingByte/SoulNexus/internal/request"
)

// RegisterKnownBodies registers shared DTOs from internal/request onto common routes.
func RegisterKnownBodies() {
	RegisterJSONBody("POST", "/api/login", dto.TenantLoginReq{})
	RegisterJSONBody("POST", "/api/change-email", dto.ChangeEmailReq{})
	RegisterJSONBody("POST", "/api/account/delete", dto.AccountDeletionReq{})

	RegisterJSONBody("POST", "/api/credentials", dto.CredentialCreateReq{})
	RegisterJSONBody("PUT", "/api/credentials/{id}", dto.CredentialUpdateReq{})
	RegisterJSONBody("PATCH", "/api/credentials/{id}", dto.CredentialUpdateReq{})

	RegisterJSONBodyBoth("/api/admin/notification-channels", dto.NotificationChannelUpsertReq{}, "POST")
	RegisterJSONBodyBoth("/api/admin/notification-channels/{id}", dto.NotificationChannelUpsertReq{}, "PUT", "PATCH")

	RegisterJSONBody("POST", "/api/configs", dto.SystemConfigCreateReq{})
	RegisterJSONBody("PUT", "/api/configs/{key}", dto.SystemConfigUpdateReq{})
	RegisterJSONBody("POST", "/api/admin/configs", dto.SystemConfigCreateReq{})
	RegisterJSONBody("PUT", "/api/admin/configs/{key}", dto.SystemConfigUpdateReq{})

	// Handler-local + dto bindings from AST (go run ./cmd/tools/genbodies).
	// Call handlers.RegisterOpenAPIBodies() from apidocs after import.
}

// RegisterAllBodies registers request-package DTOs. Handler-local bodies are
// registered via handlers.RegisterOpenAPIBodies() (generated).
func RegisterAllBodies() {
	RegisterKnownBodies()
}

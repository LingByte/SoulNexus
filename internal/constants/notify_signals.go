package constants

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Notify signals — handlers emit; internal/listeners deliver inbox/email asynchronously.
const (
	SigNotifyOpLog             = "notify:op_log"
	SigNotifyTenantProvisioned = "notify:tenant_provisioned"
)

// NotifyOpLogPayload mirrors operation-log fields needed for inbox delivery.
type NotifyOpLogPayload struct {
	TenantID     uint
	OperatorKind string
	OperatorID   uint
	Action       string
	Resource     string
	ResourceID   uint
	ResourceName string
	Summary      string
	Success      bool
}

// NotifyTenantProvisionedPayload is emitted when a tenant is created (self-register or platform).
type NotifyTenantProvisionedPayload struct {
	TenantID         uint
	TenantName       string
	AdminUserID      uint
	AdminEmail       string
	AdminDisplayName string
	Source           string
	ClientIP         string
}

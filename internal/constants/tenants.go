package constants

import "time"

// JWT role claims for tenant and platform principals.
const (
	JWTRoleTenantAdmin   = "tenant_admin"
	JWTRoleTenantMember  = "tenant_member"
	JWTRolePlatformSuper = "platform_super"
)

// TenantAccessTokenTTL is the default tenant/platform login access token lifetime.
const TenantAccessTokenTTL = 24 * time.Hour

// AccountDeletionCoolingPeriod is the grace window before self-requested account deletion is finalized.
const AccountDeletionCoolingPeriod = 7 * 24 * time.Hour

// Tenant lifecycle.
const (
	TenantStatusActive    = "active"
	TenantStatusSuspended = "suspended"
)

// Tenant user account lifecycle and provisioning source.
const (
	TenantUserStatusActive          = "active"
	TenantUserStatusDisabled        = "disabled"
	TenantUserStatusPending         = "pending"
	TenantUserStatusPendingDeletion = "pending_deletion"

	TenantUserSourceRegister = "register"
	TenantUserSourceManual   = "manual"
	TenantUserSourceGitHub   = "github"
)

// TenantAdminRoleName is the system full-access role created on tenant registration.
const TenantAdminRoleName = "管理员"

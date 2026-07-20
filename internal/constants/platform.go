package constants

// Platform admin account lifecycle (platform_admins.status).
// ActivePlatformAdmins only returns rows with status=active; login rejects disabled rows.
// Manage via HTTP: PUT /api/.../platform-admins/:id/status (active | disabled).

const (
	PlatformAdminStatusActive          = "active"
	PlatformAdminStatusDisabled        = "disabled"
	PlatformAdminStatusPendingDeletion = "pending_deletion"
)

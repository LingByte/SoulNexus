package constants

import (
	"testing"
	"time"

	constants2 "github.com/LingByte/SoulNexus/pkg/constants"
)

func TestTableNames(t *testing.T) {
	tables := map[string]string{
		"TENANT_TABLE_NAME":                 constants2.TENANT_TABLE_NAME,
		"TENANT_GROUP_TABLE_NAME":           constants2.TENANT_GROUP_TABLE_NAME,
		"TENANT_USER_TABLE_NAME":            constants2.TENANT_USER_TABLE_NAME,
		"TENANT_USER_GROUP_TABLE_NAME":      constants2.TENANT_USER_GROUP_TABLE_NAME,
		"PERMISSION_TABLE_NAME":             constants2.PERMISSION_TABLE_NAME,
		"TENANT_ROLE_TABLE_NAME":            constants2.TENANT_ROLE_TABLE_NAME,
		"TENANT_ROLE_PERMISSION_TABLE_NAME": constants2.TENANT_ROLE_PERMISSION_TABLE_NAME,
		"TENANT_USER_ROLE_TABLE_NAME":       constants2.TENANT_USER_ROLE_TABLE_NAME,
		"CREDENTIAL_TABLE_NAME":             constants2.CREDENTIAL_TABLE_NAME,
		"PLATFORM_ADMIN_TABLE_NAME":         constants2.PLATFORM_ADMIN_TABLE_NAME,
		"OPERATION_LOG_TABLE_NAME":          constants2.OPERATION_LOG_TABLE_NAME,
		"TENANT_BILL_TABLE_NAME":            constants2.TENANT_BILL_TABLE_NAME,
		"TENANT_USAGE_EVENT_TABLE_NAME":     constants2.TENANT_USAGE_EVENT_TABLE_NAME,
		"ASSISTANT_TABLE_NAME":              constants2.ASSISTANT_TABLE_NAME,
		"AI_INVOCATION_LOGS_TABLE_NAME":     constants2.AI_INVOCATION_LOGS_TABLE_NAME,
	}
	for name, value := range tables {
		if value == "" {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestTableNameValues(t *testing.T) {
	if constants2.TENANT_TABLE_NAME != "tenants" {
		t.Errorf("TENANT_TABLE_NAME = %q, want 'tenants'", constants2.TENANT_TABLE_NAME)
	}
	if constants2.TENANT_USER_TABLE_NAME != "tenant_users" {
		t.Errorf("TENANT_USER_TABLE_NAME = %q, want 'tenant_users'", constants2.TENANT_USER_TABLE_NAME)
	}
	if constants2.ASSISTANT_TABLE_NAME != "assistants" {
		t.Errorf("ASSISTANT_TABLE_NAME = %q, want 'assistants'", constants2.ASSISTANT_TABLE_NAME)
	}
}

func TestPermissionKinds(t *testing.T) {
	kinds := []string{
		PermissionKindModule,
		PermissionKindMenu,
		PermissionKindButton,
		PermissionKindAPI,
		PermissionKindData,
	}
	for _, kind := range kinds {
		if kind == "" {
			t.Error("permission kind is empty")
		}
	}
}

func TestPermissionCodes(t *testing.T) {
	codes := []string{
		PermAPIAssistantsRead,
		PermAPIAssistantsWrite,
		PermAPIWorkflowRead,
		PermAPIWorkflowWrite,
		PermAPITenantOrgRead,
		PermAPITenantOrgWrite,
		PermAPITenantUsersRead,
		PermAPITenantUsersWrite,
		PermAPICredentialsRead,
		PermAPICredentialsWrite,
		PermAPIVoiceRead,
		PermAPIVoiceWrite,
		PermAPIOperationLogsRead,
		PermAPIAIInvocationsRead,
		PermAPIReportsRead,
		PermAPIKBRead,
		PermAPIKBWrite,
		PermAPIBillingRead,
		PermAPIBillingWrite,
		PermMenuWorkspaceOverview,
		PermMenuResAssistant,
		PermMenuKBRead,
	}
	for _, code := range codes {
		if code == "" {
			t.Error("permission code is empty")
		}
	}
	if PermAPIAssistantsRead != "api.assistants.read" {
		t.Fatalf("PermAPIAssistantsRead=%q", PermAPIAssistantsRead)
	}
}

func TestAppConstants(t *testing.T) {
	if KEY_SITE_NAME == "" || KEY_SITE_URL == "" {
		t.Fatal("site keys empty")
	}
	if KEY_API_AKSK_ROUTE_POLICY == "" {
		t.Fatal("aksk policy key empty")
	}
}

func TestTenantAccessTokenTTL(t *testing.T) {
	if TenantAccessTokenTTL <= 0 {
		t.Fatal("TenantAccessTokenTTL must be positive")
	}
	_ = time.Duration(TenantAccessTokenTTL)
}

func TestOpActionConstants(t *testing.T) {
	actions := []string{
		OpActionCreate, OpActionUpdate, OpActionDelete, OpActionRestore,
		OpActionEnable, OpActionDisable, OpActionPublish, OpActionRegenerate,
	}
	for _, a := range actions {
		if a == "" {
			t.Error("op action empty")
		}
	}
}

func TestOpResourceConstants(t *testing.T) {
	resources := []string{
		OpResourceCredential,
		OpResourceTenantUser,
		OpResourceTenant,
		OpResourceTenantRole,
		OpResourceTenantGroup,
		OpResourcePlatformAdmin,
		OpResourceSystemConfig,
		OpResourceVoiceClone,
		OpResourceAPI,
		OpResourceAssistant,
		OpResourceVoiceprint,
		OpResourceWorkflow,
		OpResourceMCPMarket,
	}
	for _, r := range resources {
		if r == "" {
			t.Error("op resource empty")
		}
	}
}

func TestOpOperatorConstants(t *testing.T) {
	ops := []string{
		OpOperatorTenantUser,
		OpOperatorPlatformAdmin,
		OpOperatorCredential,
		OpOperatorSystem,
	}
	for _, o := range ops {
		if o == "" {
			t.Error("op operator empty")
		}
	}
}

func TestPersistQueryResult(t *testing.T) {
	r := PersistQueryResult{Value: 1, Value2: "x"}
	if r.Value != 1 || r.Value2 != "x" || r.Err != nil {
		t.Fatalf("%+v", r)
	}
}

func TestPersistSignalConstants(t *testing.T) {
	if SigSignalingLogInsert == "" {
		t.Fatal("SigSignalingLogInsert empty")
	}
}

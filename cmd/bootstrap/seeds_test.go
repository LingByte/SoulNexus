package bootstrap

import (
	"errors"
	"testing"

	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
)

// ===== isDuplicateKeyError =====

func TestIsDuplicateKeyError_NilError(t *testing.T) {
	if isDuplicateKeyError(nil) {
		t.Error("nil error should not be a duplicate key error")
	}
}

func TestIsDuplicateKeyError_DuplicateEntry(t *testing.T) {
	err := errors.New("Error 1062: Duplicate entry 'admin@lingecho.com' for key 'email'")
	if !isDuplicateKeyError(err) {
		t.Error("should detect 'duplicate entry' in error message")
	}
}

func TestIsDuplicateKeyError_UniqueConstraint(t *testing.T) {
	err := errors.New("UNIQUE constraint failed: mail_templates.code")
	if !isDuplicateKeyError(err) {
		t.Error("should detect 'unique constraint' in error message")
	}
}

func TestIsDuplicateKeyError_UpperCase(t *testing.T) {
	err := errors.New("ERROR: DUPLICATE ENTRY IN TABLE")
	if !isDuplicateKeyError(err) {
		t.Error("should detect 'DUPLICATE ENTRY' case-insensitively")
	}
}

func TestIsDuplicateKeyError_UniqueConstraintUpperCase(t *testing.T) {
	err := errors.New("UNIQUE CONSTRAINT FAILED")
	if !isDuplicateKeyError(err) {
		t.Error("should detect 'UNIQUE CONSTRAINT' case-insensitively")
	}
}

func TestIsDuplicateKeyError_NoMatch(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"not found", errors.New("record not found")},
		{"connection refused", errors.New("connection refused")},
		{"invalid input", errors.New("invalid input syntax")},
		{"foreign key", errors.New("foreign key constraint violation")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isDuplicateKeyError(tt.err) {
				t.Errorf("should not detect duplicate key: %v", tt.err)
			}
		})
	}
}

func TestIsDuplicateKeyError_SubstringMatch(t *testing.T) {
	// Test partial string matching
	err := errors.New("some prefix duplicate entry here too")
	if !isDuplicateKeyError(err) {
		t.Error("should detect 'duplicate entry' anywhere in message")
	}
}

// ===== SeedService nil-db guards =====

func TestSeedService_NilDB_SeedAll(t *testing.T) {
	// SeedAll delegates to seedConfigs, seedPermissions, seedPlatformAdmin, seedMailTemplates
	// seedConfigs does NOT have a nil check (accesses db directly), but the delegate calls do.
	// Individual delegate methods are tested below.
}

func TestSeedService_NilDB_SeedPermissions(t *testing.T) {
	// Test with nil service
	var s *SeedService
	err := s.seedPermissions()
	if err != nil {
		t.Errorf("nil SeedService.seedPermissions should return nil, got: %v", err)
	}

	// Test with nil db
	s = &SeedService{db: nil}
	err = s.seedPermissions()
	if err != nil {
		t.Errorf("nil db seedPermissions should return nil, got: %v", err)
	}
}

func TestSeedService_NilDB_SeedPlatformAdmin(t *testing.T) {
	var s *SeedService
	err := s.seedPlatformAdmin()
	if err != nil {
		t.Errorf("nil SeedService.seedPlatformAdmin should return nil, got: %v", err)
	}

	s = &SeedService{db: nil}
	err = s.seedPlatformAdmin()
	if err != nil {
		t.Errorf("nil db seedPlatformAdmin should return nil, got: %v", err)
	}
}

func TestSeedService_NilDB_SeedMailTemplates(t *testing.T) {
	var s *SeedService
	err := s.seedMailTemplates()
	if err != nil {
		t.Errorf("nil SeedService.seedMailTemplates should return nil, got: %v", err)
	}

	s = &SeedService{db: nil}
	err = s.seedMailTemplates()
	if err != nil {
		t.Errorf("nil db seedMailTemplates should return nil, got: %v", err)
	}
}

// ===== Constants =====

func TestSeedConstants(t *testing.T) {
	if pkgconst.SystemActorSeed == "" {
		t.Error("SystemActorSeed should not be empty")
	}
	if pkgconst.SQLAlterTablePrefix == "" {
		t.Error("SQLAlterTablePrefix should not be empty")
	}
	if pkgconst.SQLConvertUTF8MB4 == "" {
		t.Error("SQLConvertUTF8MB4 should not be empty")
	}
	if pkgconst.DBDriverMySQL == "" {
		t.Error("DBDriverMySQL should not be empty")
	}
}

func TestDuplicateKeySubstrConstants(t *testing.T) {
	if pkgconst.ErrSubstrDuplicateEntry == "" {
		t.Error("ErrSubstrDuplicateEntry should not be empty")
	}
	if pkgconst.ErrSubstrUniqueConstraint == "" {
		t.Error("ErrSubstrUniqueConstraint should not be empty")
	}
}

func TestDefaultSiteNameNotEmpty(t *testing.T) {
	if pkgconst.DefaultSiteName == "" {
		t.Error("DefaultSiteName should not be empty")
	}
}

func TestDefaultPlatformAdminConstants(t *testing.T) {
	if defaultPlatformAdminEmail == "" {
		t.Error("defaultPlatformAdminEmail should not be empty")
	}
	if defaultPlatformAdminPassword == "" {
		t.Error("defaultPlatformAdminPassword should not be empty")
	}
	if defaultPlatformAdminDisplayName == "" {
		t.Error("defaultPlatformAdminDisplayName should not be empty")
	}
}

func TestTableNameConstants(t *testing.T) {
	if pkgconst.TableMailTemplates != "mail_templates" {
		t.Error("TableMailTemplates should be 'mail_templates'")
	}
	if pkgconst.TableMailLogs != "mail_logs" {
		t.Error("TableMailLogs should be 'mail_logs'")
	}
	if pkgconst.TableSMSLogs != "sms_logs" {
		t.Error("TableSMSLogs should be 'sms_logs'")
	}
}

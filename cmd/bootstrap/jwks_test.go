package bootstrap

import (
	"testing"

	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
)

// ===== minKeysToProvision =====

func TestMinKeysToProvision_MinimumEnforced(t *testing.T) {
	// This function reads from config, which may not be initialized.
	// We test that the logic is correct: if KeepOldKeys < 3, return 3.
	// We can't easily test this without config, so we test the constant.
	if pkgconst.DefaultMinJWKSKeys < 2 {
		t.Error("DefaultMinJWKSKeys should be at least 2 to allow key rotation overlap")
	}
	if pkgconst.DefaultJWKSMinimumKeepOldKeys < 1 {
		t.Error("DefaultJWKSMinimumKeepOldKeys should be at least 1")
	}
}

// ===== StopKeyRotationChecker =====

func TestStopKeyRotationChecker_NilChannel(t *testing.T) {
	// Save original and set to nil
	oldChan := stopRotationChan
	stopRotationChan = nil
	defer func() { stopRotationChan = oldChan }()

	// Should not panic
	StopKeyRotationChecker()
}

func TestStopKeyRotationChecker_ActiveChannel(t *testing.T) {
	// Set up an active channel
	oldChan := stopRotationChan
	ch := make(chan struct{})
	stopRotationChan = ch

	// Close the channel from a goroutine to prevent deadlock
	go StopKeyRotationChecker()

	// Restore original
	defer func() { stopRotationChan = oldChan }()
}

// ===== RotateKeys constants =====

func TestKeyRotationDefaults(t *testing.T) {
	if pkgconst.DefaultKeyRotationCheckInterval <= 0 {
		t.Error("DefaultKeyRotationCheckInterval should be positive")
	}
	if pkgconst.HoursPerDay != 24 {
		t.Error("HoursPerDay should be 24")
	}
}

func TestKeyDirPerm(t *testing.T) {
	if pkgconst.KeyDirPerm != 0700 {
		t.Error("KeyDirPerm should be 0700 (owner-only)")
	}
}

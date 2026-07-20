package models

import (
	"testing"
	"time"
)

func TestDeviceWithinTrustWindow(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)
	if !DeviceWithinTrustWindow(UserDevice{TrustedUntil: &future}, now) {
		t.Fatal("future trust")
	}
	past := now.Add(-time.Hour)
	if DeviceWithinTrustWindow(UserDevice{TrustedUntil: &past}, now) {
		t.Fatal("past trust")
	}
}

func TestUserDevicePublicRow_trustExpiry(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	row := UserDevice{
		DeviceKey:    "d1",
		IsTrusted:    true,
		TrustedUntil: &past,
	}
	pub := UserDevicePublicRow(row, "d1")
	if pub.IsTrusted {
		t.Fatal("expired trust should clear flag")
	}
	if !pub.IsCurrent {
		t.Fatal("current device")
	}
}

func TestIsUserDeviceSessionActive_permissive(t *testing.T) {
	ok, err := IsUserDeviceSessionActive(nil, 0, "")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestListUserDevices_nilDB(t *testing.T) {
	if _, err := ListUserDevices(nil, UserDevicePrincipalTenantUser, 1); err == nil {
		t.Fatal("expected error")
	}
}

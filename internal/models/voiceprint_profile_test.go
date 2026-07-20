package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestVoiceprintCandidateIDs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&VoiceprintProfile{}); err != nil {
		t.Fatal(err)
	}
	rows := []VoiceprintProfile{
		{TenantID: 1, Scene: VoiceprintSceneBusiness, Name: "A", Provider: "xunfei", FeatureID: "f1", Status: VoiceprintStatusActive},
		{TenantID: 1, Scene: VoiceprintSceneBusiness, Name: "B", Provider: "xunfei", FeatureID: "f2", Status: VoiceprintStatusFailed},
		{TenantID: 1, Scene: VoiceprintSceneBusiness, Name: "C", Provider: "xunfei", FeatureID: "f3", Status: VoiceprintStatusActive},
	}
	for i := range rows {
		if err := CreateVoiceprintProfile(db, &rows[i]); err != nil {
			t.Fatal(err)
		}
	}
	ids, err := VoiceprintCandidateFeatureIDs(db, 1, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("got %#v", ids)
	}
	want := map[string]bool{"f1": true, "f3": true}
	for _, id := range ids {
		if !want[id] {
			t.Fatalf("unexpected id %q in %#v", id, ids)
		}
	}
}

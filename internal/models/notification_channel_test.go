package models

import (
	"strings"
	"testing"
)

func TestBuildEmailChannelConfigJSON(t *testing.T) {
	_, err := BuildEmailChannelConfigJSON("smtp", "n", "", 0, "", "", "", "", "", "", "")
	if err == nil {
		t.Fatal("missing host")
	}
	raw, err := BuildEmailChannelConfigJSON("smtp", "n", "smtp.example.com", 587, "u", "p", "from@example.com", "From", "", "", "")
	if err != nil || !strings.Contains(raw, "smtp.example.com") {
		t.Fatalf("raw=%q err=%v", raw, err)
	}
}

func TestMergeEmailSecretsOnUpdate_preserves(t *testing.T) {
	old := `{"provider":"smtp","password":"secret"}`
	newJSON := `{"provider":"smtp","password":""}`
	got, err := MergeEmailSecretsOnUpdate(old, newJSON)
	if err != nil || !strings.Contains(got, "secret") {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestListNotificationChannels_nilDB(t *testing.T) {
	if _, err := ListNotificationChannels(nil, "", 1, 10); err == nil {
		t.Fatal("expected error")
	}
}

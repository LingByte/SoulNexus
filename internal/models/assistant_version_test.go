package models

import (
	"testing"

	"gorm.io/datatypes"
)

func TestParseOptionalJSONColumn(t *testing.T) {
	if _, err := ParseOptionalJSONColumn(""); err == nil {
		t.Fatal("empty should error")
	}
	if _, err := ParseOptionalJSONColumn("not-json"); err == nil {
		t.Fatal("invalid json")
	}
	j, err := ParseOptionalJSONColumn(`{"a":1}`)
	if err != nil || len(j) == 0 {
		t.Fatalf("j=%v err=%v", j, err)
	}
}

func TestParseOptionalJSONColumnNullable(t *testing.T) {
	j, err := ParseOptionalJSONColumnNullable("")
	if err != nil || j != nil {
		t.Fatalf("j=%v err=%v", j, err)
	}
}

func TestDiffAssistantConfigs(t *testing.T) {
	a := datatypes.JSON(`{"x":1}`)
	b := datatypes.JSON(`{"x":2}`)
	keys, err := DiffAssistantConfigs(a, b)
	if err != nil || len(keys) != 1 || keys[0] != "x" {
		t.Fatalf("keys=%v err=%v", keys, err)
	}
}

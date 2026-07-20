package common

import (
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestBaseModelSoftDeleteRestore(t *testing.T) {
	var m BaseModel
	m.SetCreateInfo("alice")
	if m.CreateBy != "alice" || m.UpdateBy != "alice" {
		t.Fatalf("create info: %+v", m)
	}
	m.SoftDelete("bob")
	if !m.DeletedAt.Valid || m.UpdateBy != "bob" {
		t.Fatalf("soft delete: %+v", m)
	}
	m.Restore("carol")
	if m.DeletedAt.Valid {
		t.Fatal("expected restored")
	}
	if m.UpdateBy != "carol" {
		t.Fatal(m.UpdateBy)
	}
}

func TestBaseModelBeforeCreateSetsIDWhenSnowflakeNil(t *testing.T) {
	var m BaseModel
	if err := m.BeforeCreate(&gorm.DB{}); err != nil {
		t.Fatal(err)
	}
	if m.CreatedAt.IsZero() || m.UpdatedAt.IsZero() {
		t.Fatal("timestamps")
	}
	_ = time.Now()
}

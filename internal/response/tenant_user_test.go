package response

import (
	"testing"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestNewTenantUserResponse(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	_ = db.AutoMigrate(&models.TenantUser{}, &models.TenantGroup{}, &models.TenantUserGroup{}, &models.TenantRole{}, &models.TenantUserRole{})
	u := models.TenantUser{Email: "a@b.c", Username: "alice", Status: "active", DisplayName: "Alice"}
	got := NewTenantUserResponse(db, u)
	if got.Email != "a@b.c" || got.Username != "alice" {
		t.Fatalf("%+v", got)
	}
}

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/models"
)

// Backfill admin permissions for existing system tenant「管理员」roles.
//
// Usage:
//
//	go run ./cmd/backfill
//	go run ./cmd/backfill -mode production
func main() {
	mode := flag.String("mode", "", "running environment (development, test, production)")
	flag.Parse()

	if *mode != "" {
		_ = os.Setenv("MODE", *mode)
	}
	if err := config.Load(); err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	db, err := bootstrap.ConnectDatabase(os.Stdout)
	if err != nil {
		log.Fatalf("database connect failed: %v", err)
	}

	if err := models.SyncPermissionCatalog(db); err != nil {
		log.Fatalf("sync permission catalog failed: %v", err)
	}
	fmt.Println("permission catalog synced")

	added, err := models.BackfillSystemTenantAdminPermissionsStats(db, "cmd/backfill")
	if err != nil {
		log.Fatalf("backfill tenant admin permissions failed: %v", err)
	}
	fmt.Printf("tenant admin permissions backfilled (%d new bindings)\n", added)
}

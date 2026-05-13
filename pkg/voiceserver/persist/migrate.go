// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package persist

import (
	"errors"

	"gorm.io/gorm"
)

// Models returns the slice of GORM-managed entities owned by this package, in
// dependency order. Used by `cmd/bootstrap` to AutoMigrate them alongside any
// other application schemas.
//
// Adding a new entity? Append it here AND register table creation in any
// hand-written migration scripts under `cmd/bootstrap/migrations`.
func Models() []any {
	return []any{
		&SIPCall{},
		&SIPUser{},
		&CallEvent{},
		&CallMediaStats{},
		&CallRecording{},
	}
}

// Migrate runs GORM AutoMigrate on every entity in Models(). Idempotent; safe
// to invoke on every boot.
func Migrate(db *gorm.DB) error {
	if db == nil {
		return errors.New("persist: nil db")
	}
	for _, m := range Models() {
		if err := db.AutoMigrate(m); err != nil {
			return err
		}
	}
	return nil
}

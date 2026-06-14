// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package bootstrap

import (
	"fmt"

	"github.com/LingByte/SoulNexus/i18n"
)

// InitI18n loads embedded locale files into the global i18n bundle.
func InitI18n() error {
	if err := i18n.Init(); err != nil {
		return fmt.Errorf("i18n init failed: %w", err)
	}
	return nil
}

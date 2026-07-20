// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package quota

// Policy documents overage / suspend surface area.
// Concrete enforcement remains in internal/models tenant quota helpers.
type Policy struct {
	HardStopOnExhausted bool
	GraceMinutes        int
}

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rnnoise

import "errors"

// ErrUnavailable is returned by stub builds and by New() when the library
// cannot be used. Real CGO builds never return this from New() on success;
// it stays defined so callers can write transport-agnostic code:
//
//	d, err := rnnoise.New()
//	if errors.Is(err, rnnoise.ErrUnavailable) { ... fallback ... }
var ErrUnavailable = errors.New("rnnoise: unavailable (build with -tags rnnoise, CGO_ENABLED=1, and install librnnoise)")

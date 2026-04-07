// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rnnoise

import "errors"

// ErrUnavailable is returned by stub builds and by New() when the library cannot be used.
// Real CGO builds never return this from New() on success; it remains defined so callers
// can use errors.Is(err, rnnoise.ErrUnavailable) regardless of build tags.
var ErrUnavailable = errors.New("rnnoise: unavailable (build with -tags rnnoise, CGO_ENABLED=1, and install librnnoise)")

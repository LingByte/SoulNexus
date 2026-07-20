// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package encoder

import "errors"

// ErrLECodecUnavailable is returned when built without -tags lingcodec.
var ErrLECodecUnavailable = errors.New("encoder: lecodec unavailable (build with -tags lingcodec and run lecodec/build.sh)")

var (
	errLENil    = errors.New("encoder: nil lecodec handle")
	errLECreate = errors.New("encoder: lecodec create failed")
	errLEEncode = errors.New("encoder: lecodec encode failed")
	errLEDecode = errors.New("encoder: lecodec decode failed")
	errLEBuf    = errors.New("encoder: lecodec output buffer too small")
)

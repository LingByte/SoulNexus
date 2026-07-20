package errors

import "errors"

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

var ErrLastActivePlatformAdmin = errors.New("cannot disable or remove the last active platform admin")

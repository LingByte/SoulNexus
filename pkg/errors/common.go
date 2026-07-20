package errors

import "errors"

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

var (
	ErrDBNil              = errors.New("db is nil")
	ErrDSNRequired        = errors.New("database DSN is required")
	ErrServerAddrRequired = errors.New("server address is required")
)

var (
	ErrAKSKRouteIDsRequired   = errors.New("at least one API route id is required")
	ErrAKSKSystemRoutesClosed = errors.New("platform has not opened any API Key routes")
	ErrAKSKRouteIDNotOpen     = errors.New("route id is not open at platform level")
)

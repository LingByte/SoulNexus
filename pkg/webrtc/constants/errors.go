package constants

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import "errors"

var (
	ErrSessionIsShutdown  = errors.New("session is shutdown")
	ErrSessionIsRunning   = errors.New("session is running")
	ErrNotInputTransport  = errors.New("not input transport")
	ErrNotOutputTransport = errors.New("not output transport")
	ErrTransportIsClosed  = errors.New("transport is closed")
	ErrInvalidFrameType   = errors.New("invalid frame type")
	ErrCodecNotSupported  = errors.New("codec not supported")
)

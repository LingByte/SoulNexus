// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import "errors"

// Sentinel errors exported so callers can branch on `errors.Is`.
var (
	ErrRoomFull          = errors.New("sfu: room is full")
	ErrRoomNotFound      = errors.New("sfu: room not found")
	ErrTooManyRooms      = errors.New("sfu: too many rooms")
	ErrParticipantGone   = errors.New("sfu: participant disconnected")
	ErrForbidden         = errors.New("sfu: permission denied")
	ErrProtocol          = errors.New("sfu: protocol violation")
	ErrDuplicateIdentity = errors.New("sfu: identity already in room")
	ErrOriginRejected    = errors.New("sfu: websocket origin not allowed")
	ErrManagerClosed     = errors.New("sfu: manager is closed")
)

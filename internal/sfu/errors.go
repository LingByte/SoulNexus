// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import "errors"

var (
	ErrMaxRooms    = errors.New("sfu: maximum room count reached for this process")
	ErrRoomFull    = errors.New("sfu: maximum peers per room reached")
	ErrP2PRoomFull = errors.New("sfu: p2p room already has two distinct peers; use SFU for a third participant")
)

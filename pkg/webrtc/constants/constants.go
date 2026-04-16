package constants

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"time"
)

const (
	DefaultICETimeout = 25 * time.Second
	// ICEGatherSignalingMaxWait caps how long CreateOffer/CreateAnswer block waiting for
	// ICE gathering. Full ICETimeout is not needed for signaling; trickle adds candidates later.
	ICEGatherSignalingMaxWait = 2 * time.Second
	DefaultStreamID   = "ling-echo"
	DefaultCodec      = "pcmu"
	WebRTCOffer       = "offer"
	WebRTCAnswer      = "answer"
	WebRTCCandidate   = "candidate"
)

const (
	CodecPCMU = "pcmu"
	CodecPCMA = "pcma"
	CodecG722 = "g722"
	CodecOPUS = "opus"
	CodecG711 = "g711"
)

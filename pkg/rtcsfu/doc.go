// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package rtcsfu provides control-plane primitives for a hybrid multi-SFU WebRTC
// deployment: room-to-node routing, sticky assignment, subscription shaping (Last-N),
// and simulcast layer selection with hysteresis.
//
// It does not implement RTP forwarding; integrate with your SFU process separately.
// For audio interoperability with this repo's voice stack, prefer Opus at 48 kHz
// mono/stereo negotiated consistently with pkg/media paths.
package rtcsfu

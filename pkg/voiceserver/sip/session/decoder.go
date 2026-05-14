package session

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import "github.com/LingByte/SoulNexus/pkg/media"

// passthroughDTMFDecode wraps an audio decoder so DTMFPackets (RFC 2833
// telephone-events) are forwarded unchanged while audio payloads go
// through dec. Without this wrapper the codec decoder would reject or
// corrupt the DTMF telephone-event payload.
func passthroughDTMFDecode(dec media.EncoderFunc) media.EncoderFunc {
	return func(p media.MediaPacket) ([]media.MediaPacket, error) {
		if _, ok := p.(*media.DTMFPacket); ok {
			return []media.MediaPacket{p}, nil
		}
		return dec(p)
	}
}

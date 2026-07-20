// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package welcomeaudio

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// MaxTransferRingBytes caps transfer ringback WAV uploads and runtime fetches.
// 2 MiB ≈ 12 s at 48 kHz / 16-bit / mono — enough for a short loop, small
// enough to keep first-play decode + disk cache fast.
const MaxTransferRingBytes = 2 << 20

// MaxTransferRingDurationSec is the longest allowed transfer ringback clip.
// Playback loops until bridge; very long files delay first audible ringback
// and waste memory on decode.
const MaxTransferRingDurationSec = 30

// ErrTransferRingTooLarge is returned when a transfer ring WAV exceeds MaxTransferRingBytes.
var ErrTransferRingTooLarge = errors.New("transfer ringing wav exceeds max size")

// ErrTransferRingTooLong is returned when a transfer ring WAV exceeds MaxTransferRingDurationSec.
var ErrTransferRingTooLong = errors.New("transfer ringing wav exceeds max duration")

// ValidateTransferRingBytes checks RIFF/WAVE magic, size, and duration on an in-memory blob.
func ValidateTransferRingBytes(b []byte) error {
	if err := ValidateBytes(b); err != nil {
		return err
	}
	if len(b) > MaxTransferRingBytes {
		return fmt.Errorf("%w: %d bytes (max %d)", ErrTransferRingTooLarge, len(b), MaxTransferRingBytes)
	}
	dur := wavDurationSec(b)
	if dur <= 0 {
		return fmt.Errorf("%w: could not determine duration", ErrNotAudio)
	}
	if dur > MaxTransferRingDurationSec {
		return fmt.Errorf("%w: %d sec (max %d)", ErrTransferRingTooLong, dur, MaxTransferRingDurationSec)
	}
	return nil
}

// ValidateTransferRingURL probes reachability, downloads up to MaxTransferRingBytes,
// and validates WAV magic + duration. Used when admins paste or save a ringback URL.
func ValidateTransferRingURL(ctx context.Context, rawURL string) error {
	if err := ValidateURL(ctx, rawURL); err != nil {
		return err
	}
	body, err := downloadWAVBody(ctx, rawURL, MaxTransferRingBytes)
	if err != nil {
		return err
	}
	return ValidateTransferRingBytes(body)
}

func wavDurationSec(wav []byte) int {
	if len(wav) < 44 {
		return 0
	}
	if string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		return 0
	}
	var byteRate uint32
	var dataSize uint32
	i := 12
	for i+8 <= len(wav) {
		chunkID := string(wav[i : i+4])
		chunkSize := int(binary.LittleEndian.Uint32(wav[i+4 : i+8]))
		payloadStart := i + 8
		if payloadStart+chunkSize > len(wav) {
			return 0
		}
		switch chunkID {
		case "fmt ":
			if chunkSize >= 16 {
				byteRate = binary.LittleEndian.Uint32(wav[payloadStart+8 : payloadStart+12])
			}
		case "data":
			dataSize = uint32(chunkSize)
		}
		i = payloadStart + chunkSize
		if chunkSize%2 != 0 {
			i++
		}
	}
	if byteRate == 0 || dataSize == 0 {
		return 0
	}
	sec := int((uint64(dataSize) + uint64(byteRate) - 1) / uint64(byteRate))
	if sec < 1 && dataSize > 0 {
		return 1
	}
	return sec
}

// FetchTransferRingPCM is like FetchPCM but enforces transfer-ring size/duration limits.
func FetchTransferRingPCM(ctx context.Context, rawURL string, sampleRate int, decodeWAV func(raw []byte, sampleRate int) ([]byte, error)) ([]byte, error) {
	return fetchWAVPCMAt(ctx, rawURL, sampleRate, transferRingFetchPolicy, decodeWAV)
}

var transferRingFetchPolicy = wavFetchPolicy{
	maxBytes:    MaxTransferRingBytes,
	validate:    ValidateTransferRingBytes,
	cachePrefix: "ring:",
}

func downloadWAVBody(ctx context.Context, rawURL string, maxBytes int) ([]byte, error) {
	u, err := parseHTTPURL(rawURL)
	if err != nil {
		return nil, err
	}
	cctx, cancel := context.WithTimeout(ctx, validateTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build request: %v", ErrUnreachable, err)
	}
	resp, err := httpClientForValidate().Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("%w: GET %d", ErrUnreachable, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrUnreachable, err)
	}
	if len(body) > maxBytes {
		return nil, fmt.Errorf("%w: body exceeds %d bytes", ErrTransferRingTooLarge, maxBytes)
	}
	return body, nil
}

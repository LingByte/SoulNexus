// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package welcomeaudio

import (
	"context"
	"encoding/binary"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func pcmWAVWithDuration(sec, sampleRate int) []byte {
	if sec <= 0 {
		sec = 1
	}
	if sampleRate <= 0 {
		sampleRate = 8000
	}
	dataBytes := sampleRate * 2 * sec
	buf := make([]byte, 44+dataBytes)
	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], uint32(36+dataBytes))
	copy(buf[8:12], "WAVE")
	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], 16)
	binary.LittleEndian.PutUint16(buf[20:22], 1)
	binary.LittleEndian.PutUint16(buf[22:24], 1)
	binary.LittleEndian.PutUint32(buf[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(buf[28:32], uint32(sampleRate*2))
	binary.LittleEndian.PutUint16(buf[32:34], 2)
	binary.LittleEndian.PutUint16(buf[34:36], 16)
	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataBytes))
	return buf
}

func TestValidateTransferRingBytes(t *testing.T) {
	ok := pcmWAVWithDuration(5, 8000)
	if err := ValidateTransferRingBytes(ok); err != nil {
		t.Fatalf("5s ring: %v", err)
	}
	long := pcmWAVWithDuration(MaxTransferRingDurationSec+1, 8000)
	if err := ValidateTransferRingBytes(long); !errors.Is(err, ErrTransferRingTooLong) {
		t.Fatalf("too long: got %v want ErrTransferRingTooLong", err)
	}
	huge := make([]byte, MaxTransferRingBytes+1)
	copy(huge, minimalWAVHeader())
	if err := ValidateTransferRingBytes(huge); !errors.Is(err, ErrTransferRingTooLarge) {
		t.Fatalf("too large: got %v want ErrTransferRingTooLarge", err)
	}
}

func TestValidateTransferRingURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write(pcmWAVWithDuration(3, 8000))
	}))
	defer srv.Close()
	if err := ValidateTransferRingURL(context.Background(), srv.URL+"/ring.wav"); err != nil {
		t.Fatalf("valid url: %v", err)
	}
}

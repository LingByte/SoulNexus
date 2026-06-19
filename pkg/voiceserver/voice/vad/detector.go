// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package vad

import (
	lingllmvad "github.com/LingByte/lingllm/vad"
	"go.uber.org/zap"
)

// Detector wraps lingllm's energy-based RMS barge-in detector with
// voice-server defaults (higher threshold + more consecutive frames for
// uncancelled speaker echo). Endpointing still relies on cloud ASR VAD.
type Detector struct {
	*lingllmvad.RMSDetector
}

// NewDetector builds a barge-in tuned detector for 20 ms @ 16 kHz PCM.
func NewDetector() *Detector {
	inner := lingllmvad.NewRMSDetector()
	inner.SetThreshold(4500.0)
	inner.SetConsecutiveFrames(5)
	return &Detector{RMSDetector: inner}
}

// SetLogger is kept for API compatibility; lingllm uses logrus internally.
func (d *Detector) SetLogger(_ *zap.Logger) {}

// SetConsecutiveFrames clamps to at least 1 before delegating.
func (d *Detector) SetConsecutiveFrames(frames int) {
	if frames < 1 {
		frames = 1
	}
	d.RMSDetector.SetConsecutiveFrames(frames)
}

// Enabled reports whether detection is active.
func (d *Detector) Enabled() bool {
	return true
}

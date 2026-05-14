package media

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT
//
// audio_util.go provides small audio helpers shared by synthesizer / recognizer.
// They were moved here from pkg/utils so that pkg/utils can stay free of
// audio-specific concerns.

import (
	"strings"
	"time"
)

// frame-period bounds shared with the encoder/registry split-frames logic.
const (
	minFramePeriod     = 10 * time.Millisecond
	maxFramePeriod     = 300 * time.Millisecond
	defaultFramePeriod = 20 * time.Millisecond
)

// NormalizeFramePeriod parses a frame-period string (e.g. "20ms", "60ms") and
// clamps it to the supported range [10ms, 300ms]. Empty / unparseable / zero /
// out-of-range values are coerced to a safe default of 20ms so downstream
// RTP/codec packetizers always receive a usable cadence.
//
// The string form is what synthesizer/recognizer Options carry over the wire
// (JSON / form encoding). Pure-Duration callers can simply pass d.String().
func NormalizeFramePeriod(s string) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultFramePeriod
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return defaultFramePeriod
	}
	if d < minFramePeriod {
		return defaultFramePeriod
	}
	if d > maxFramePeriod {
		return maxFramePeriod
	}
	return d
}

// ComputeSampleByteCount returns the number of bytes produced by one
// millisecond of linear-PCM audio at the given configuration. It is used by
// both the recognizer (to compute byte budgets per second) and the
// synthesizer (to size frame slices).
//
// Formula: bytes_per_ms = sampleRate * (bitDepth/8) * channels / 1000.
//
// Inputs <= 0 are treated as zero so the function never panics; callers can
// guard against zero results to detect misconfiguration.
func ComputeSampleByteCount(sampleRate, bitDepth, channels int) int {
	if sampleRate <= 0 || bitDepth <= 0 || channels <= 0 {
		return 0
	}
	return sampleRate * (bitDepth / 8) * channels / 1000
}

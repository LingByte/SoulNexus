// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package vad is an energy-based (RMS) voice activity detector tuned for
// barge-in: detecting that a human has started to talk while the AI's
// TTS is still playing, so the gateway can interrupt synthesis and let
// the user take the floor.
//
// The detector is transport-agnostic — it consumes PCM16 LE mono frames
// and emits a boolean "user is speaking now". All three transports in
// VoiceServer (SIP / xiaozhi / WebRTC) funnel decoded PCM through the
// shared ASR pipeline; Detector plugs in at that same junction so a
// single implementation handles barge-in for every transport.
//
// Defaults are calibrated for 20 ms @ 16 kHz frames (320 bytes) from a
// typical VoIP mic: threshold 1500 RMS, adaptive noise floor tracked
// over 20 quiet frames, one over-threshold frame fires. Tune via the
// Set* methods when running on headset, conference-room, or WebRTC
// browser inputs — browser auto-gain tends to need a lower threshold.
package vad

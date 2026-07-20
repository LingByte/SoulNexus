// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package realtime hosts the native dialog engine for the realtime
// (multimodal) voice mode. Where pkg/dialog/cascaded composes
// ASR/LLM/TTS as separate Stages, this package wraps a single
// full-duplex provider Agent (Qwen-Omni / GPT-4o realtime / …) into
// one agentStage that:
//
//   - consumes caller PCM (KindPCM frames) and pushes them to the
//     Agent's input channel;
//   - listens to Agent events (user transcripts, assistant text,
//     assistant audio, server-VAD barge-in) and translates them into
//     downstream pipeline.Frames so the same hotword / persist /
//     transfer / intent stages used by the cascaded engine can run
//     unchanged on the realtime path.
//
// The package is provider-agnostic: it does NOT depend on
// pkg/realtime directly. Instead it accepts a small local Agent
// interface (see types.go) that the media adapter implements via
// pkg/realtime.NewAgentFromCredential. Keeping pkg/realtime out of
// the import graph avoids leaking transport-layer concerns (WS
// handshake, vendor decoders) into the engine package and lets unit
// tests inject a synchronous in-memory fake without spinning up a
// WebSocket.
//
// PR-10a wires the engine + agentStage scaffolding. Voice OnACK uses
// StreamingPort → session.AttachEngine(ModeRealtime) via the session attach
// path (dialog engine mode selection).
package realtime

/*
 * Copyright 2026 LingByte Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package voice provides transport-agnostic voice capabilities for AI calls.
//
// It is deliberately independent of WebSocket, WebRTC, voice, or any other
// media transport. Transports feed PCM (or encoded audio decoded upstream)
// into a voice session and receive synthesized audio back through callbacks.
//
// Layering:
//
//	protocol/voice/dialog    session orchestration + Event/Command contract
//	protocol/voice/gateway   dialog-plane WebSocket client
//	protocol/voice/webrtc    browser WebRTC (HTTP SDP + SRTP/Opus)
//	protocol/voice/xiaozhi   xiaozhi-esp32 / browser WebSocket (pipeline + realtime)
//	protocol/voice/transport per-call SessionFactory wiring
//	protocol/voice/asr       uplink pipeline components
//	protocol/voice/tts       downlink synthesis + cache
//
// Uplink order matters: VAD runs on raw microphone PCM before echo suppression so
// barge-in still works while the recognizer feed is silenced. An optional Denoiser
// (RNNoise, WebRTC AEC3, hardware AEC) may run after decode and before VAD.
// PlaybackGate tracks streaming, queued TTS, and a post-playback tail for room-echo
// suppression when true AEC is unavailable.
//
// Conversation logic (LLM, tools, business rules) lives outside this package.
// The dialog subpackage defines the event/command contract between the voice
// plane and an external Dialog application.
//
// LLM→TTS streaming path:
//
//	Dialog app streams CmdTTSStream chunks as the LLM emits tokens.
//	TextSegmenter: first segment may break at comma (≥6 runes) for low latency;
//	later segments break only on sentence punctuation (。！？.!?) or stream end.
//	Speaker runs synth+play workers with prefetch=3 to overlap TTS with playback.
//	WebRTC enables PaceRealtime so frames are not burst-written to RTP.
//
// Typical integration:
//
//	sess, _ := dialog.NewSession(ctx, dialog.Config{
//	    CallID: "call-1",
//	    Engine: recognizerEngine,
//	    TTSService: tts.FromSynthesisEngine(synth),
//	    OnAudioOut: transport.SendDownlink,
//	    OnEvent:    dialogApp.HandleEvent,
//	})
//	sess.Start(ctx)
//	transport.OnUplink(func(pcm []byte) { sess.ProcessAudio(ctx, pcm) })
//	dialogApp.OnCommand(sess.HandleCommand)
package voice

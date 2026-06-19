// Package vad provides barge-in detection for voice-server transports.
//
// The implementation delegates to github.com/LingByte/lingllm/vad (RMS energy
// detector). Remote HTTP/WebSocket VAD services are also available from the
// same lingllm module when needed.
package vad

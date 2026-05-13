// Package voice provides generic, provider-agnostic ASR and TTS channels
// that plug into any audio source/sink (SIP MediaLeg, WebSocket, WebRTC, file).
//
// Layering
//
//   pkg/voice/asr   PCM16 in  →  partial/final transcript out
//   pkg/voice/tts   text in   →  PCM16 frames out (pace-realtime optional)
//
// Design goals
//
//   1. Zero coupling to any concrete ASR/TTS vendor — the vendor is injected
//      via the recognizer.TranscribeService and synthesizer.SynthesisService
//      (or the narrower voice/tts.Service) interfaces.
//   2. Pure PCM16 mono bridge: callers decide input/output sample rates, the
//      pipelines resample internally via pkg/media.ResamplePCM.
//   3. Minimal-allocation hot path: the ASR feeder copies only when
//      resampling; the TTS pipeline slices a single growing buffer per turn.
//   4. Reusable across every business shape:
//        - SIP UAS voice bot  (cmd/voiceserver, LingEchoX, SoulNexus)
//        - SIP UAC dialer     (cmd/voiceserver -outbound)
//        - WS voice gateway   (LingEchoX voicedialog, SoulNexus voice handler)
//        - WebRTC room bot    (future)
//
// See pkg/voice/asr and pkg/voice/tts for the public APIs.
package voice

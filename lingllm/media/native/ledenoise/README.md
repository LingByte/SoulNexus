# ledenoise

Pure-Rust **RNNoise** uplink denoise (from voice-engine `NoiseReducer` via `nnnoiseless`),
with Kaiser polyphase resample to/from 48 kHz. C ABI for Go (`-tags ledenoise`).

## Build

```bash
./build.sh
# from lingllm/
make build-ledenoise
CGO_ENABLED=1 go test -tags ledenoise ./media/denoise/ -run Ledenoise -v
```

## SNR (`ld_snr_*`)

Same crate also exports a **time-domain noise-floor SNR** estimator (WebRTC-style
minimum statistics, no FFT / no `webrtc-audio-processing` dependency). Go links it
via `media/denoise.LedenoiseSNR` when `-tags ledenoise` is set; otherwise
`pkg/dialog/audio.SNRMonitor` falls back to an equivalent Go path. Estimation
stays **in-process** (CGO) — there is no Unix-socket media sidecar.

## Note

voice-engine has **no AEC**. Denoise here is RNNoise only. Assistant
`noiseSuppressionType=ledenoise` (aliases: `native`, `nnnoiseless`).

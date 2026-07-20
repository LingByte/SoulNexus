# levad

Pure-Rust **TinySilero** and **TinyTen** VAD (ported from voice-engine `media/vad/`),
exposed as a C ABI for Go CGO (`-tags levad`).

## Build

```bash
./build.sh
# from lingllm/
make build-levad
CGO_ENABLED=1 go test -tags levad ./vad/local/ -run Levad -v
```

## C API

- `lv_vad_open(LV_KIND_SILERO|LV_KIND_TEN, sample_rate, threshold)`
- `lv_vad_process(vad, pcm16, n, &speech)` → `LV_OK` / `LV_NEED_MORE`
- `lv_vad_close(vad)`

Expects **16 kHz** mono PCM16LE (TinyTen requires 16 kHz).

## Weights (`weights/*.bin`)

| File | Purpose |
|------|---------|
| `silero_weights.bin` | TinySilero network weights (~1.2 MB), `include_bytes!` in `tiny_silero.rs` |
| `tiny_tenvad.bin` | TinyTen network weights (~300 KB), `include_bytes!` in `tiny_ten.rs` |

These are **compile-time inputs** for `liblevad`, not runtime downloads. Keep them
in git (do **not** gitignore). Origin: same binary weight dumps as voice-engine’s
TinySilero / TinyTen models.

# lecodec

Rust VoIP codecs used by `media/encoder` (`-tags lingcodec`).

```bash
./build.sh
# → target/release/liblecodec.{a,dylib|so}
```

| Piece | Name |
|-------|------|
| Crate / dylib | `lecodec` / `liblecodec` |
| C header | `include/lecodec.h` |
| Symbols | `le_encoder_open`, `le_decoder_decode`, … |
| Modules | `codec::*`, `capi`, `resample` |
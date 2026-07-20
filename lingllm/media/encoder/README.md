# Package encoder

Audio codec registry for SoulNexus (`CreateEncode` / `CreateDecode`).

## Backends

| Build | Implementation |
|-------|----------------|
| **`-tags lingcodec`** | Rust crate in `lecodec/` (`liblecodec`, C ABI `le_*`) |
| default | Legacy pure-Go / hraban (`legacy_*.go`) |

```bash
./lecodec/build.sh
CGO_ENABLED=1 go test -tags lingcodec ./media/encoder/ -run LECodec -v
```

## Layout

```
encoder/
  registry.go              # name → factory
  pcm.go / pool.go
  lecodec.go               # CGO bind (+ stub / api / pipeline / errors / test)
  legacy_*.go              # !lingcodec
  lecodec/                 # Rust crate "lecodec"
    Cargo.toml
    include/lecodec.h
    src/
      lib.rs / capi.rs / resample.rs
      codec/{pcmu,pcma,g722,g729,opus,telephone_event}.rs
```

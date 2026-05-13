# SFU (Selective Forwarding Unit)

Multi-party realtime audio/video for VoiceServer. Forwards media between
browser peers without transcoding; one PeerConnection per participant on
both the publish and subscribe path. Independent of the SIP / xiaozhi /
1v1-WebRTC plane — the SFU does **not** bridge to the dialog backend.

## Wire-up

```go
mgr, err := sfu.NewManager(&sfu.Config{
    AuthSecret:             "your-shared-hmac-secret",
    MaxParticipantsPerRoom: 16,
    EnableRecording:        true,
    RecordBucket:           "sfu-recordings",
    WebhookURL:             "https://example.com/sfu-events",
}, logger)
http.HandleFunc("/sfu/v1/ws", mgr.ServeWS)
```

In `cmd/voiceserver` this is automatic via the `-sfu` family of flags.
Quick demo:

```sh
voiceserver -http :7080 -sfu -sfu-allow-anon
# then open http://localhost:7080/sfu/v1/demo in two tabs
```

## Access tokens

Tokens are mini-JWTs (`base64url(payload).base64url(hmac256)`). Mint on
the business backend with the same secret the SFU runs with:

```go
tok, _ := sfu.NewAccessToken(secret, sfu.AccessTokenClaims{
    Room: "team-standup",
    Identity: "alice@example.com",
    Name: "Alice",
    ExpiresAt: time.Now().Add(time.Hour).Unix(),
    Permissions: &sfu.Permissions{CanPublish: true, CanSubscribe: true},
})
```

The token is sent as the `token` field in the first `join` message.

## Signaling protocol

WebSocket JSON envelope: `{type, data, requestId?}`. See `protocol.go`
for the full message catalogue. Highlights:

| direction | type                | purpose                                |
|-----------|---------------------|----------------------------------------|
| c → s     | `join`              | first message, carries access token    |
| s → c     | `joined`            | ack with peer list + ICE servers       |
| both      | `offer` / `answer`  | SDP exchange (initial + renegotiation) |
| both      | `iceCandidate`      | trickle ICE                            |
| s → c     | `participantJoined` / `participantLeft` | room membership   |
| s → c     | `trackPublished` / `trackUnpublished` | track lifecycle     |
| c → s     | `setMute`           | server-side track suppression          |
| s → c     | `iceRestart`        | server forced an ICE restart           |
| both      | `ping` / `pong`     | (WS-level pings are also used)         |

## Media

* Codecs: Opus (audio, PT 111) and VP8 (video, PT 96).
* Simulcast: video publishers may send up to 3 layers (`f`/`h`/`q`); the
  SFU picks the highest-quality layer that has produced a keyframe and
  switches atomically when bandwidth changes.
* Per-subscriber RTCP feedback: PLI / FIR are forwarded back to the
  original publisher to drive keyframe regeneration.
* ICE restart on `failed` is automatic.

## Recording

`EnableRecording=true` causes every published audio track to be decoded
to PCM16 mono 48 kHz, wrapped in a WAV container, and uploaded to the
default `pkg/stores` backend at participant teardown. Video is **not**
recorded — composed video would require a real container/encoder; if you
need it, run a downstream Composer that subscribes like a normal
participant.

The webhook (`recording.finished`) carries the public URL returned by
the store. With the local store, this URL is served by voiceserver
itself at `/media/...`.

## Webhooks

When `WebhookURL` is configured, the SFU POSTs JSON to it on lifecycle
events. Each request is signed with `X-SFU-Signature: hex(HMAC_SHA256(AuthSecret, body))`.

Event types: `room.started`, `room.ended`, `participant.joined`,
`participant.left`, `track.published`, `recording.finished`. Delivery is
best-effort, in-order, and never blocks the SFU hot path (a 128-event
buffer absorbs bursts; overflow is logged).

## File map

| file              | responsibility                                         |
|-------------------|--------------------------------------------------------|
| `config.go`       | `Config` + `Normalise()` defaults                      |
| `auth.go`         | HMAC-SHA256 access tokens + permissions                |
| `protocol.go`     | WS JSON envelope + payload structs                     |
| `engine.go`       | pion `MediaEngine` (Opus + VP8 + simulcast extensions) |
| `simulcast.go`    | layer multiplexer with active-layer atomics            |
| `participant.go`  | one peer's PC + tracks + (un)subscribe + renegotiate   |
| `room.go`         | room registry + `Manager`                              |
| `ws.go`           | gorilla/websocket wrapper with serialised writes       |
| `handler.go`      | HTTP upgrade + dispatch loop + heartbeat               |
| `recording.go`    | per-track Opus → WAV → stores.Default                  |
| `webhook.go`      | async signed event emitter                             |

## Limits / non-goals

* Video recording (see above).
* Server-side composition / mixing — all forwarding is per-track.
* Per-subscriber bandwidth adaptation beyond simulcast layer choice
  (we don't run TWCC-driven layer selection per subscriber yet).
* Data channels (no `MsgPublishData` route despite the permission flag
  existing on the token).

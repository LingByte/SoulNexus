# TURN server for `cmd/voice` WebRTC

**What**: a drop-in coturn stack that lets WebRTC calls survive symmetric
NAT / strict firewalls. Without it, ~30-50% of real-world browsers fail
to connect (corporate users, mobile carriers, university WiFi).

**When you need it**: the moment you take voiceserver past `localhost` and
want browsers from outside your LAN to work. STUN alone only fixes cone
NAT.

## Topology

```
Browser ────┐                       ┌──── voiceserver (public or NAT'd)
            │                       │
            └──► coturn (public IP) ◄┘
             relays SRTP when a direct
             path cannot be established
```

coturn runs standalone (separate docker host, separate compose). It does
**NOT** need to live on the voiceserver host — in fact you probably want
it closer to the user (edge region).

## Quick start

```bash
cd deploy/voice-coturn/coturn
cp .env.example .env
vim .env   # set EXTERNAL_IP, TURN_PASSWORD, TURN_REALM
docker compose up -d
docker compose logs -f coturn | grep 'Relay'
```

Open these ports on the host firewall / cloud SG:

| Port | Proto | Purpose |
|---|---|---|
| 3478 | UDP, TCP | STUN + TURN (unencrypted) |
| 5349 | UDP, TCP | TURN/TLS (optional but recommended) |
| 49160-49200 | UDP | relay range (capacity = 40 concurrent calls × ~80 Kbps) |

## Wire into voiceserver

Credentials ride inline in `WEBRTC_ICE_SERVERS` as `?username=U&credential=C`
query params — voiceserver's `ParseICEServers` reads them and hands them
to pion, which forwards them to the browser at `/webrtc/v1/offer` time.

```bash
export WEBRTC_ICE_SERVERS="stun:turn.example.com:3478,turn:turn.example.com:3478?username=voiceserver&credential=your-password-from-.env"
```

Multiple servers separated by commas; mix of stun/turn/turns all fine.
Restart voiceserver after changing the env var (it's read once at
startup).

Verify end-to-end:

1. Go to https://webrtc.github.io/samples/src/content/peerconnection/trickle-ice/
2. Clear the default server list
3. Add `turn:turn.example.com:3478` with your username/credential
4. Click **Add Server** then **Gather candidates**
5. You should see at least one `relay` candidate — that's the TURN path

## Scaling notes

- **Bandwidth**: each call through TURN eats `2 × bitrate` on coturn's uplink (inbound from A, outbound to B). Opus mono ≈ 32 Kbps, so plan ~64 Kbps per active call. 40 concurrent calls × 64 Kbps ≈ 2.5 Mbps.
- **Widen the port range** for more concurrency: edit `--min-port` / `--max-port` in `docker-compose.yml` and reopen the range on your firewall.
- **Rotate credentials** monthly. For multi-tenant/public deployments use `--use-auth-secret` + a TOTP-style ephemeral password generator instead of the long-term static credential shown here.
- **Geo-distribute**: coturn in ap-east-1 for APAC users, another in us-east-1 for NA users. voiceserver can list both in `WEBRTC_ICE_SERVERS` — ICE will pick the closer pair automatically.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Browser: `ICE failed` immediately | TURN server unreachable | Check UDP/3478 is open, `docker compose logs coturn` |
| Browser: `ICE failed` after ~30s | Relay allocated but relay ports closed | Open UDP 49160-49200 on firewall |
| `401 Unauthorized` in coturn logs | Credential mismatch | `TURN_PASSWORD` in .env must match the `credential=` query value in `WEBRTC_ICE_SERVERS` |
| Relay candidate uses wrong IP | `EXTERNAL_IP` missed | Set `EXTERNAL_IP` to public IPv4, restart compose |
| Works on LAN, fails from 4G | You only have STUN, not TURN | This is exactly why you deployed coturn — check voiceserver read the env vars |

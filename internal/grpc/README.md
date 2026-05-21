# Internal gRPC (`internal/grpc`)

Outbound gRPC clients and per-service servers for split binaries (`cmd/server`, `cmd/auth`, …).

## Layout

```
internal/grpc/
  errors.go          # shared persistence → gRPC status mapping
  clients/           # Bundle aggregate; DialAuth()
  auth/
    proto/auth/v1/   # .proto definitions
    pb/auth/v1/        # generated Go (make proto-auth)
    convert.go         # DB models ↔ proto
    client/            # used by cmd/server
    server/            # used by cmd/auth
```

Add a new service under `internal/grpc/<name>/` (same `proto/`, `pb/`, `client/`, `server/` layout), then add a field on `clients.Bundle`.

## Auth service

| Process | Env | Default |
|---------|-----|---------|
| `cmd/auth` gRPC listen | `AUTH_GRPC_ADDR` | `:7075` |
| `cmd/auth` HTTP | `USER_SERVICE_HTTP_ADDR` | `:7074` |
| `cmd/server` dial target | `AUTH_SERVICE_GRPC_ADDR` | `127.0.0.1:7075` |

Regenerate code after proto edits:

```bash
make proto-auth
```

## Usage in handlers

- JWT/session identity: `auth.CurrentUser(c)` (no gRPC).
- DB-backed user rows, API keys/secrets: `h.rpc.Auth.*`.

```go
user, err := h.rpc.Auth.GetUser(ctx, id)
cred, err := h.rpc.Auth.ResolveCredential(ctx, apiKey, apiSecret)
```

# API Service

Implemented baseline in Go at `api/cmd/server/main.go`.

Current status:
- In-memory store (for execution bootstrap)
- Auth: login + refresh
- Streams: create, patch, state change, runtime
- Playback: start, heartbeat billing, stop, kick
- Monitoring: health + metrics + error summary
- Internal playback validation endpoint for NGINX auth_request

Run locally:

```bash
cd api
go run ./cmd/server
```

Production next step:
- replace in-memory store with Postgres + Redis
- add proper JWT signing/validation
- add RBAC middleware and rate limiting


## Package layout

- `cmd/server`: entrypoint
- `internal/httpapi`: HTTP transport + route handlers
- `internal/service`: business rules (sessions, points, tokens)
- `internal/store`: repository implementation (currently in-memory)
- `internal/model`: domain models
- `internal/auth`: token helpers


Additional endpoints now available:
- `GET /monitoring/errors`
- `POST /playback/renew`


Security now implemented:
- HMAC-signed JWT-like access tokens
- RBAC on admin routes (streams mutate, monitoring, kick)
- Use `Authorization: Bearer <access_token>` for admin endpoints

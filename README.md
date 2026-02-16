# MPV-Based Streaming Platform (Execution Started)

This repository has been reset from the old single-file relay and now follows the requested platform plan structure.

## Current implemented baseline

- Control-plane API scaffold (Go, stdlib HTTP) under `api/` with modular architecture (`internal/auth`, `internal/model`, `internal/store`, `internal/service`, `internal/httpapi`).
- Player CLI scaffold (Go) under `player/`.
- Dev stack: Postgres, Redis, MinIO, API, NGINX in `docker-compose.dev.yml`.
- NGINX token-gated route template (`/play/{session_id}/*`) and direct test HLS route.
- SQL schema migration file for core tables.
- Living progress tracker: `docs/EXECUTION_CHECKLIST.md`.

## Repo structure

- `infra/` infra configs
- `api/` control-plane service
- `admin/` admin dashboard scaffold
- `player/` mpv launcher scaffold
- `pipeline/` worker scaffold
- `scripts/` helper scripts
- `db/migrations/` SQL schema
- `docs/` progress tracker

## Quick start (dev)

```bash
./scripts/dev-up.sh
```

Check services:

```bash
curl -sS http://127.0.0.1:8080/healthz
```

## API quick flow

1) login:

```bash
curl -sS -X POST http://127.0.0.1:8080/auth/login \
  -H 'content-type: application/json' \
  -d '{"email":"demo@local","password":"demo"}'
```

2) set stream live:

```bash
curl -sS -X POST http://127.0.0.1:8080/streams/stream-1/state \
  -H 'content-type: application/json' \
  -d '{"state":"live"}'
```

3) start playback session:

```bash
curl -sS -X POST http://127.0.0.1:8080/playback/start \
  -H 'content-type: application/json' \
  -d '{"stream_id":"stream-1","token":"token:u_demo:user"}'
```

## Player CLI

Build:

```bash
cd player && go build ./cmd/player
```

Use:

```bash
./player login
./player play stream-1
./player logout
```

## Implementation tracking

Use `docs/EXECUTION_CHECKLIST.md` as the source of truth and mark items as work is completed.


### New hardening added

- Playback token renew endpoint: `POST /playback/renew`
- Basic in-memory rate limiting on login and playback start endpoints

- Monitoring error summary endpoint: `GET /monitoring/errors`
- Monitoring points spend metric included in `GET /monitoring/metrics` as `points_spent_per_minute`

- Stream restart endpoint added: `POST /streams/{id}/restart`
- Metrics now include `active_sessions`, `points_spent_per_minute`, `login_failures`, `playback_errors`

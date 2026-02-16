# MPV Streaming Platform Execution Checklist

This file is mandatory progress tracking for the platform plan.

Legend:
- [x] Done
- [~] Partial
- [ ] Not done

Last updated: 2026-02-15

---

## Repository structure
- [x] `/infra/dev`
- [x] `/infra/prod`
- [x] `/api`
- [x] `/admin`
- [x] `/player`
- [x] `/pipeline`
- [x] `/scripts`

## Core technology stack
- [x] API chosen: Go (modular clean architecture baseline)
- [x] DB: Postgres service in compose
- [x] Cache: Redis service in compose
- [x] Object storage: MinIO service in compose
- [x] Gateway: NGINX template
- [~] Transcode pipeline: scaffold only
- [~] Admin UI: scaffold only
- [x] Launcher baseline: Go CLI scaffold


## Architecture quality
- [x] API split into packages (auth/model/store/service/http)
- [x] Transport layer separated from business logic
- [x] Business logic separated from storage implementation
- [~] In-memory repository abstraction (DB repository pending)
- [ ] End-to-end integration tests

## Data model
- [x] SQL schema file created (`db/migrations/0001_init.sql`)
- [x] users
- [x] wallets
- [x] wallet_ledger
- [x] streams
- [x] stream_runtime
- [x] playback_sessions
- [ ] Migration runner integrated in service startup

## Video pipeline
Admin configurable fields:
- [~] ingest mode (schema + stream fields)
- [ ] ABR profiles
- [x] segment duration (schema field)
- [x] window minutes (schema field)
- [x] points rate (schema field)

Worker responsibilities:
- [ ] start ffmpeg worker from desired state
- [ ] package HLS to MinIO
- [ ] runtime heartbeat every 10s
- [ ] update `last_manifest_at`

Controller service:
- [ ] desired_state watcher
- [ ] worker lifecycle manager
- [ ] runtime status sync

## NGINX gateway
Routes:
- [x] `/hls/*` route template
- [x] `/play/{session_id}/*` route template

Requirements:
- [~] token/session validation hook (`auth_request`) wired
- [x] segment caching
- [x] manifest no-store behavior
- [x] stream live + session active enforced in playback token validation (DB-backed pending)

Security rules:
- [ ] direct storage blocked in production
- [~] short-lived play token implemented in API mock
- [ ] manifest token enforcement validated with integration tests

## API functionality
Auth:
- [x] login
- [x] refresh
- [ ] JWT middleware (real)
- [ ] RBAC middleware (real)

Stream admin:
- [x] create stream
- [x] edit stream config (patch)
- [x] change state
- [x] restart stream endpoint (controller hook pending)
- [x] runtime status endpoint

Playback:
- [x] start session
- [x] renew session token endpoint
- [x] stop session
- [x] kick session

Points system:
- [x] deduction per heartbeat endpoint
- [x] atomic deduction with mutex
- [x] ledger insert
- [x] block session at zero points

Monitoring endpoints:
- [x] health
- [x] metrics baseline
- [x] error summary endpoint
- [x] points consumption per minute metric

## MPV launcher
- [x] `player login`
- [x] `player play <stream_id>`
- [x] `player logout`
- [x] heartbeat every 10s
- [x] kill mpv when blocked

## Admin dashboard
- [ ] Stream list page
- [ ] Stream detail page
- [ ] Sessions page
- [ ] Wallet page

## Monitoring
API metrics:
- [x] active_sessions
- [x] points_spent_per_minute
- [x] login_failures
- [x] playback_errors

Pipeline metrics:
- [ ] ingest bitrate
- [ ] segment rate
- [ ] manifest age

Gateway metrics:
- [ ] 2xx/4xx/5xx
- [ ] cache hit ratio
- [ ] bandwidth egress

## Concurrency controls
- [x] max sessions per user (at stream limit in start playback)
- [x] login/playback rate limit (basic in-memory)
- [x] IP + user-agent stored per session
- [x] admin kick session
- [~] short token TTL (play token 90 seconds)

## Dev completion checklist
- [x] docker-compose full baseline stack up file
- [ ] stream control from admin UI
- [ ] worker launch when stream live
- [ ] HLS generated to MinIO
- [~] gateway playback token validation API hook
- [x] user login flow baseline
- [x] playback start baseline
- [x] mpv heartbeat billing stop baseline
- [ ] admin live viewer dashboard
- [ ] pause stream stops worker
- [x] kick user session baseline endpoint
- [~] ledger baseline persisted in memory (DB integration pending)

## Production readiness checklist
- [ ] API stateless + DB-backed
- [ ] Redis HA
- [ ] Postgres backups
- [ ] MinIO distributed setup
- [ ] Gateway clustering + LB
- [ ] CDN
- [ ] centralized logs
- [ ] metrics dashboard
- [ ] alerting
- [ ] secrets management

---

## Work done this iteration
- [x] Removed previous simple Python relay/web UI code.
- [x] Refactored API into modular packages for scalability and maintainability.
- [x] Added basic rate limiting + playback token renew endpoint.
- [x] Added monitoring error summary + points/min metric endpoints.
- [x] Added Go player CLI baseline with mpv + heartbeat enforcement.
- [x] Added SQL migration for full data model tables.
- [x] Updated compose to include API service.
- [x] Updated README to new platform execution state.


### Executed next 5 steps:
- [x] Restart stream endpoint
- [x] active_sessions metric finalized
- [x] points_spent_per_minute metric finalized
- [x] login_failures metric
- [x] playback_errors metric

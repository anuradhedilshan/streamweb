CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('admin', 'user')),
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wallets (
  user_id TEXT PRIMARY KEY REFERENCES users(id),
  balance_points BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wallet_ledger (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id),
  delta_points BIGINT NOT NULL,
  reason TEXT NOT NULL,
  stream_id TEXT,
  session_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS streams (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('draft','live','paused','disabled')),
  ingest_mode TEXT,
  ingest_url TEXT,
  abr_profiles_json JSONB,
  segment_duration_sec INT NOT NULL,
  playlist_window_minutes INT NOT NULL,
  storage_prefix TEXT,
  points_rate INT NOT NULL,
  max_concurrent_sessions INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS stream_runtime (
  stream_id TEXT PRIMARY KEY REFERENCES streams(id),
  desired_state TEXT,
  actual_state TEXT,
  worker_id TEXT,
  last_heartbeat_at TIMESTAMPTZ,
  last_manifest_at TIMESTAMPTZ,
  current_viewers INT NOT NULL DEFAULT 0,
  last_error TEXT
);

CREATE TABLE IF NOT EXISTS playback_sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id),
  stream_id TEXT NOT NULL REFERENCES streams(id),
  state TEXT NOT NULL CHECK (state IN ('active','blocked','stopped')),
  started_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL,
  ip TEXT,
  user_agent TEXT
);

CREATE INDEX IF NOT EXISTS idx_playback_sessions_user_state ON playback_sessions(user_id, state);
CREATE INDEX IF NOT EXISTS idx_wallet_ledger_user_created ON wallet_ledger(user_id, created_at DESC);

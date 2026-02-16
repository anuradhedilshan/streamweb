package model

import "time"

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"-"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type Wallet struct {
	UserID  string `json:"user_id"`
	Balance int64  `json:"balance_points"`
}

type Stream struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	Status                string `json:"status"`
	IngestMode            string `json:"ingest_mode"`
	IngestURL             string `json:"ingest_url"`
	SegmentDurationSec    int    `json:"segment_duration_sec"`
	PlaylistWindowMinutes int    `json:"playlist_window_minutes"`
	PointsRate            int    `json:"points_rate"`
	MaxConcurrentSessions int    `json:"max_concurrent_sessions"`
}

type Session struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	StreamID   string    `json:"stream_id"`
	State      string    `json:"state"`
	StartedAt  time.Time `json:"started_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
}

type LedgerEntry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Delta     int64     `json:"delta_points"`
	Reason    string    `json:"reason"`
	StreamID  string    `json:"stream_id"`
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
}

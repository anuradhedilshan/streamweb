package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"streamweb/api/internal/auth"
	"streamweb/api/internal/model"
	"streamweb/api/internal/store"
)

type Service struct {
	repo store.Repository
}

func New(repo store.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Login(email, password string) (map[string]any, error) {
	u, ok := s.repo.FindUserByEmail(email)
	if !ok || u.Password != password {
		s.repo.IncrementError("login_failures")
		return nil, fmt.Errorf("invalid credentials")
	}
	tok := auth.TokenForUser(u.ID, u.Role)
	return map[string]any{"access_token": tok, "refresh_token": tok, "user": u}, nil
}

func (s *Service) Refresh(refreshToken string) (string, error) {
	uid, role, err := auth.ParseUserToken(refreshToken)
	if err != nil {
		return "", err
	}
	return auth.TokenForUser(uid, role), nil
}

func (s *Service) CreateStream(st model.Stream) model.Stream {
	if st.ID == "" {
		st.ID = fmt.Sprintf("stream-%d", time.Now().Unix())
	}
	if st.Status == "" {
		st.Status = "draft"
	}
	return s.repo.CreateStream(st)
}

func (s *Service) PatchStream(id string, body map[string]any) (model.Stream, bool) {
	return s.repo.UpdateStream(id, func(st *model.Stream) {
		if v, ok := body["name"].(string); ok {
			st.Name = v
		}
		if v, ok := body["ingest_url"].(string); ok {
			st.IngestURL = v
		}
		if v, ok := body["status"].(string); ok {
			st.Status = v
		}
		if v, ok := body["points_rate"].(float64); ok {
			st.PointsRate = int(v)
		}
	})
}

func (s *Service) SetStreamState(id, state string) bool {
	_, ok := s.repo.UpdateStream(id, func(st *model.Stream) { st.Status = state })
	return ok
}

func (s *Service) StreamRuntime(id string) (map[string]any, bool) {
	st, ok := s.repo.GetStream(id)
	if !ok {
		return nil, false
	}
	return map[string]any{"stream": st, "current_viewers": s.repo.ActiveViewerCount(id), "last_manifest_at": time.Now().UTC()}, true
}

func (s *Service) StartPlayback(streamID, token, ip, userAgent string) (map[string]string, int, error) {
	uid, _, err := auth.ParseUserToken(token)
	if err != nil {
		s.repo.IncrementError("playback_errors")
		return nil, 401, err
	}
	st, ok := s.repo.GetStream(streamID)
	if !ok || st.Status != "live" {
		s.repo.IncrementError("playback_errors")
		return nil, 400, fmt.Errorf("stream not live")
	}
	wallet, ok := s.repo.GetWallet(uid)
	if !ok || wallet.Balance <= 0 {
		s.repo.IncrementError("playback_errors")
		return nil, 402, fmt.Errorf("insufficient points")
	}
	if s.repo.ActiveUserSessionCount(uid) >= st.MaxConcurrentSessions {
		s.repo.IncrementError("playback_errors")
		return nil, 429, fmt.Errorf("too many concurrent sessions")
	}
	ss := s.repo.CreateSession(uid, streamID, ip, userAgent)
	playToken := fmt.Sprintf("play:%s:%d", ss.ID, time.Now().Add(90*time.Second).Unix())
	playURL := fmt.Sprintf("http://localhost:8088/play/%s/master.m3u8?token=%s", ss.ID, playToken)
	return map[string]string{"session_id": ss.ID, "play_token": playToken, "play_url": playURL}, 200, nil
}

func (s *Service) RenewPlayback(sessionID string) (map[string]string, int, error) {
	ss, ok := s.repo.GetSession(sessionID)
	if !ok {
		s.repo.IncrementError("playback_errors")
		return nil, 404, fmt.Errorf("session not found")
	}
	if ss.State != "active" {
		s.repo.IncrementError("playback_errors")
		return nil, 403, fmt.Errorf("session not active")
	}
	s.repo.TouchSession(ss.ID)
	playToken := fmt.Sprintf("play:%s:%d", ss.ID, time.Now().Add(90*time.Second).Unix())
	playURL := fmt.Sprintf("http://localhost:8088/play/%s/master.m3u8?token=%s", ss.ID, playToken)
	return map[string]string{"session_id": ss.ID, "play_token": playToken, "play_url": playURL}, 200, nil
}

func (s *Service) Heartbeat(sessionID string) (map[string]any, int) {
	ss, ok := s.repo.GetSession(sessionID)
	if !ok {
		s.repo.IncrementError("playback_errors")
		return map[string]any{"error": "session not found"}, 404
	}
	st, ok := s.repo.GetStream(ss.StreamID)
	if !ok {
		s.repo.IncrementError("playback_errors")
		return map[string]any{"error": "stream not found"}, 404
	}
	remaining, err := s.repo.DeductPoints(ss.UserID, ss.StreamID, ss.ID, int64(st.PointsRate))
	if err != nil || remaining <= 0 {
		s.repo.IncrementError("playback_errors")
		s.repo.UpdateSessionState(ss.ID, "blocked")
		return map[string]any{"state": "blocked", "balance_points": remaining}, 402
	}
	s.repo.TouchSession(ss.ID)
	return map[string]any{"state": "active", "balance_points": remaining}, 200
}

func (s *Service) StopSession(sessionID string) { s.repo.UpdateSessionState(sessionID, "stopped") }
func (s *Service) KickSession(sessionID string) { s.repo.UpdateSessionState(sessionID, "blocked") }
func (s *Service) Metrics() map[string]any {
	m := map[string]any{}
	for k, v := range s.repo.Metrics() {
		m[k] = v
	}
	m["points_spent_per_minute"] = s.repo.PointsSpentLastMinute()
	errs := s.repo.ErrorSummary()
	m["login_failures"] = errs["login_failures"]
	m["playback_errors"] = errs["playback_errors"]
	return m
}

func (s *Service) ErrorSummary() map[string]int {
	return s.repo.ErrorSummary()
}

func (s *Service) ValidatePlaybackToken(token, sessionID string) (int, string) {
	parts := strings.Split(token, ":")
	if len(parts) != 3 || parts[0] != "play" || parts[1] != sessionID {
		s.repo.IncrementError("playback_errors")
		return 401, "invalid"
	}
	exp, _ := strconv.ParseInt(parts[2], 10, 64)
	if time.Now().Unix() > exp {
		s.repo.IncrementError("playback_errors")
		return 401, "expired"
	}
	ss, ok := s.repo.GetSession(sessionID)
	if !ok || ss.State != "active" {
		s.repo.IncrementError("playback_errors")
		return 403, "blocked"
	}
	stream, ok := s.repo.GetStream(ss.StreamID)
	if !ok || stream.Status != "live" {
		s.repo.IncrementError("playback_errors")
		return 403, "stream_not_live"
	}
	return 200, "ok"
}

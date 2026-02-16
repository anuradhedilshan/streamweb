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
	store *store.MemoryStore
}

func New(st *store.MemoryStore) *Service { return &Service{store: st} }

func (s *Service) Login(email, password string) (map[string]any, error) {
	u, ok := s.store.FindUserByEmail(email)
	if !ok || u.Password != password {
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
	return s.store.CreateStream(st)
}

func (s *Service) PatchStream(id string, body map[string]any) (model.Stream, bool) {
	return s.store.UpdateStream(id, func(st *model.Stream) {
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
	_, ok := s.store.UpdateStream(id, func(st *model.Stream) { st.Status = state })
	return ok
}

func (s *Service) StreamRuntime(id string) (map[string]any, bool) {
	st, ok := s.store.GetStream(id)
	if !ok {
		return nil, false
	}
	return map[string]any{"stream": st, "current_viewers": s.store.ActiveViewerCount(id), "last_manifest_at": time.Now().UTC()}, true
}

func (s *Service) StartPlayback(streamID, token, ip, userAgent string) (map[string]string, int, error) {
	uid, _, err := auth.ParseUserToken(token)
	if err != nil {
		return nil, 401, err
	}
	st, ok := s.store.GetStream(streamID)
	if !ok || st.Status != "live" {
		return nil, 400, fmt.Errorf("stream not live")
	}
	wallet, ok := s.store.GetWallet(uid)
	if !ok || wallet.Balance <= 0 {
		return nil, 402, fmt.Errorf("insufficient points")
	}
	if s.store.ActiveUserSessionCount(uid) >= st.MaxConcurrentSessions {
		return nil, 429, fmt.Errorf("too many concurrent sessions")
	}
	ss := s.store.CreateSession(uid, streamID, ip, userAgent)
	playToken := fmt.Sprintf("play:%s:%d", ss.ID, time.Now().Add(90*time.Second).Unix())
	playURL := fmt.Sprintf("http://localhost:8088/play/%s/master.m3u8?token=%s", ss.ID, playToken)
	return map[string]string{"session_id": ss.ID, "play_token": playToken, "play_url": playURL}, 200, nil
}

func (s *Service) Heartbeat(sessionID string) (map[string]any, int) {
	ss, ok := s.store.GetSession(sessionID)
	if !ok {
		return map[string]any{"error": "session not found"}, 404
	}
	st, ok := s.store.GetStream(ss.StreamID)
	if !ok {
		return map[string]any{"error": "stream not found"}, 404
	}
	remaining, err := s.store.DeductPoints(ss.UserID, ss.StreamID, ss.ID, int64(st.PointsRate))
	if err != nil || remaining <= 0 {
		s.store.UpdateSessionState(ss.ID, "blocked")
		return map[string]any{"state": "blocked", "balance_points": remaining}, 402
	}
	s.store.TouchSession(ss.ID)
	return map[string]any{"state": "active", "balance_points": remaining}, 200
}

func (s *Service) StopSession(sessionID string) { s.store.UpdateSessionState(sessionID, "stopped") }
func (s *Service) KickSession(sessionID string) { s.store.UpdateSessionState(sessionID, "blocked") }
func (s *Service) Metrics() map[string]int      { return s.store.Metrics() }

func (s *Service) ValidatePlaybackToken(token, sessionID string) (int, string) {
	parts := strings.Split(token, ":")
	if len(parts) != 3 || parts[0] != "play" || parts[1] != sessionID {
		return 401, "invalid"
	}
	exp, _ := strconv.ParseInt(parts[2], 10, 64)
	if time.Now().Unix() > exp {
		return 401, "expired"
	}
	ss, ok := s.store.GetSession(sessionID)
	if !ok || ss.State != "active" {
		return 403, "blocked"
	}
	return 200, "ok"
}

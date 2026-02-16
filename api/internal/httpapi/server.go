package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"streamweb/api/internal/auth"
	"streamweb/api/internal/model"
	"streamweb/api/internal/service"
)

type Server struct {
	svc    *service.Service
	rateMu sync.Mutex
	rate   map[string][]time.Time
}

func NewServer(svc *service.Service) *Server {
	return &Server{svc: svc, rate: map[string][]time.Time{}}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func parseBody(r *http.Request, dst any) error { return json.NewDecoder(r.Body).Decode(dst) }

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/auth/login", s.login)
	mux.HandleFunc("/auth/refresh", s.refresh)
	mux.HandleFunc("/streams", s.createStream)
	mux.HandleFunc("/streams/", s.streamRoutes)
	mux.HandleFunc("/playback/start", s.playbackStart)
	mux.HandleFunc("/playback/renew", s.playbackRenew)
	mux.HandleFunc("/playback/heartbeat", s.playbackHeartbeat)
	mux.HandleFunc("/playback/stop", s.playbackStop)
	mux.HandleFunc("/playback/kick", s.playbackKick)
	mux.HandleFunc("/monitoring/health", s.monitorHealth)
	mux.HandleFunc("/monitoring/metrics", s.monitorMetrics)
	mux.HandleFunc("/monitoring/errors", s.monitorErrors)
	mux.HandleFunc("/internal/validate-playback", s.validatePlayback)
}

func (s *Server) allowRate(r *http.Request, bucket string, limit int, window time.Duration) bool {
	key := bucket + ":" + r.RemoteAddr
	now := time.Now()
	cut := now.Add(-window)
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	arr := s.rate[key]
	kept := arr[:0]
	for _, t := range arr {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= limit {
		s.rate[key] = kept
		return false
	}
	kept = append(kept, now)
	s.rate[key] = kept
	return true
}

func bearerToken(r *http.Request) string {
	authz := r.Header.Get("Authorization")
	return strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
}

func (s *Server) requireAuth(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	tok := bearerToken(r)
	if tok == "" {
		writeJSON(w, 401, map[string]string{"error": "missing bearer token"})
		return "", "", false
	}
	uid, role, err := auth.ParseUserToken(tok)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "invalid token"})
		return "", "", false
	}
	return uid, role, true
}

func (s *Server) requireRole(w http.ResponseWriter, r *http.Request, wanted string) bool {
	_, role, ok := s.requireAuth(w, r)
	if !ok {
		return false
	}
	if role != wanted {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return false
	}
	return true
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "ok", "time": time.Now().UTC()})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.allowRate(r, "login", 20, time.Minute) {
		writeJSON(w, 429, map[string]string{"error": "rate limit"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method"})
		return
	}
	var body struct{ Email, Password string }
	if err := parseBody(r, &body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid body"})
		return
	}
	resp, err := s.svc.Login(body.Email, body.Password)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, resp)
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method"})
		return
	}
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := parseBody(r, &body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid body"})
		return
	}
	tok, err := s.svc.Refresh(body.RefreshToken)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "invalid token"})
		return
	}
	writeJSON(w, 200, map[string]string{"access_token": tok})
}

func (s *Server) createStream(w http.ResponseWriter, r *http.Request) {
	if !s.requireRole(w, r, "admin") {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method"})
		return
	}
	var st model.Stream
	if err := parseBody(r, &st); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid body"})
		return
	}
	writeJSON(w, 201, s.svc.CreateStream(st))
}

func (s *Server) streamRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/streams/")
	if strings.HasSuffix(path, "/state") {
		if !s.requireRole(w, r, "admin") {
			return
		}
		id := strings.TrimSuffix(path, "/state")
		if r.Method != http.MethodPost {
			writeJSON(w, 405, map[string]string{"error": "method"})
			return
		}
		var body struct {
			State string `json:"state"`
		}
		_ = parseBody(r, &body)
		if !s.svc.SetStreamState(id, body.State) {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, 200, map[string]string{"stream_id": id, "state": body.State})
		return
	}
	if strings.HasSuffix(path, "/runtime") {
		if _, _, ok := s.requireAuth(w, r); !ok {
			return
		}
		id := strings.TrimSuffix(path, "/runtime")
		resp, ok := s.svc.StreamRuntime(id)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, 200, resp)
		return
	}
	if r.Method == http.MethodPatch {
		if !s.requireRole(w, r, "admin") {
			return
		}
		var body map[string]any
		if err := parseBody(r, &body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid body"})
			return
		}
		st, ok := s.svc.PatchStream(path, body)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, 200, st)
		return
	}
	writeJSON(w, 404, map[string]string{"error": "not found"})
}

func (s *Server) playbackStart(w http.ResponseWriter, r *http.Request) {
	if !s.allowRate(r, "playback_start", 30, time.Minute) {
		writeJSON(w, 429, map[string]string{"error": "rate limit"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method"})
		return
	}
	var body struct {
		StreamID string `json:"stream_id"`
		Token    string `json:"token"`
	}
	_ = parseBody(r, &body)
	resp, code, err := s.svc.StartPlayback(body.StreamID, body.Token, r.RemoteAddr, r.UserAgent())
	if err != nil {
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, resp)
}

func (s *Server) playbackRenew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method"})
		return
	}
	var body struct {
		SessionID string `json:"session_id"`
	}
	_ = parseBody(r, &body)
	resp, code, err := s.svc.RenewPlayback(body.SessionID)
	if err != nil {
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, resp)
}

func (s *Server) playbackHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method"})
		return
	}
	var body struct {
		SessionID string `json:"session_id"`
	}
	_ = parseBody(r, &body)
	resp, code := s.svc.Heartbeat(body.SessionID)
	writeJSON(w, code, resp)
}

func (s *Server) playbackStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method"})
		return
	}
	var body struct {
		SessionID string `json:"session_id"`
	}
	_ = parseBody(r, &body)
	s.svc.StopSession(body.SessionID)
	writeJSON(w, 200, map[string]string{"status": "stopped"})
}

func (s *Server) playbackKick(w http.ResponseWriter, r *http.Request) {
	if !s.requireRole(w, r, "admin") {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method"})
		return
	}
	var body struct {
		SessionID string `json:"session_id"`
	}
	_ = parseBody(r, &body)
	s.svc.KickSession(body.SessionID)
	writeJSON(w, 200, map[string]string{"status": "kicked"})
}

func (s *Server) monitorHealth(w http.ResponseWriter, r *http.Request) {
	if !s.requireRole(w, r, "admin") {
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) monitorMetrics(w http.ResponseWriter, r *http.Request) {
	if !s.requireRole(w, r, "admin") {
		return
	}
	writeJSON(w, 200, s.svc.Metrics())
}

func (s *Server) monitorErrors(w http.ResponseWriter, r *http.Request) {
	if !s.requireRole(w, r, "admin") {
		return
	}
	writeJSON(w, 200, s.svc.ErrorSummary())
}

func (s *Server) validatePlayback(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	sid := r.Header.Get("X-Session-Id")
	status, message := s.svc.ValidatePlaybackToken(token, sid)
	if status != 200 {
		http.Error(w, message, status)
		return
	}
	w.WriteHeader(200)
}

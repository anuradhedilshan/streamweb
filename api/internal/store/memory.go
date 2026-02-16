package store

import (
	"fmt"
	"sync"
	"time"

	"streamweb/api/internal/model"
)

type MemoryStore struct {
	mu       sync.Mutex
	users    map[string]model.User
	wallets  map[string]model.Wallet
	streams  map[string]model.Stream
	sessions map[string]model.Session
	ledger   []model.LedgerEntry
}

func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{
		users:    map[string]model.User{},
		wallets:  map[string]model.Wallet{},
		streams:  map[string]model.Stream{},
		sessions: map[string]model.Session{},
		ledger:   []model.LedgerEntry{},
	}
	admin := model.User{ID: "u_admin", Email: "admin@local", Password: "admin", Role: "admin", Status: "active"}
	demo := model.User{ID: "u_demo", Email: "demo@local", Password: "demo", Role: "user", Status: "active"}
	s.users[admin.Email] = admin
	s.users[demo.Email] = demo
	s.wallets[demo.ID] = model.Wallet{UserID: demo.ID, Balance: 1000}
	s.streams["stream-1"] = model.Stream{ID: "stream-1", Name: "Default Stream", Status: "paused", IngestMode: "url", SegmentDurationSec: 4, PlaylistWindowMinutes: 2, PointsRate: 5, MaxConcurrentSessions: 2}
	return s
}

func (s *MemoryStore) FindUserByEmail(email string) (model.User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[email]
	return u, ok
}

func (s *MemoryStore) CreateStream(st model.Stream) model.Stream {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams[st.ID] = st
	return st
}

func (s *MemoryStore) UpdateStream(id string, fn func(*model.Stream)) (model.Stream, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.streams[id]
	if !ok {
		return model.Stream{}, false
	}
	fn(&st)
	s.streams[id] = st
	return st, true
}

func (s *MemoryStore) GetStream(id string) (model.Stream, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.streams[id]
	return st, ok
}

func (s *MemoryStore) ActiveViewerCount(streamID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, ss := range s.sessions {
		if ss.StreamID == streamID && ss.State == "active" {
			count++
		}
	}
	return count
}

func (s *MemoryStore) ActiveUserSessionCount(userID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, ss := range s.sessions {
		if ss.UserID == userID && ss.State == "active" {
			count++
		}
	}
	return count
}

func (s *MemoryStore) GetWallet(userID string) (model.Wallet, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.wallets[userID]
	return w, ok
}

func (s *MemoryStore) CreateSession(userID, streamID, ip, ua string) model.Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	sid := fmt.Sprintf("s_%d", time.Now().UnixNano())
	ss := model.Session{ID: sid, UserID: userID, StreamID: streamID, State: "active", StartedAt: now, LastSeenAt: now, IP: ip, UserAgent: ua}
	s.sessions[sid] = ss
	return ss
}

func (s *MemoryStore) GetSession(sessionID string) (model.Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.sessions[sessionID]
	return ss, ok
}

func (s *MemoryStore) UpdateSessionState(sessionID, state string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.sessions[sessionID]
	if !ok {
		return false
	}
	ss.State = state
	ss.LastSeenAt = time.Now().UTC()
	s.sessions[sessionID] = ss
	return true
}

func (s *MemoryStore) TouchSession(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.sessions[sessionID]
	if !ok {
		return false
	}
	ss.LastSeenAt = time.Now().UTC()
	s.sessions[sessionID] = ss
	return true
}

func (s *MemoryStore) DeductPoints(userID, streamID, sessionID string, points int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	wallet, ok := s.wallets[userID]
	if !ok {
		return 0, fmt.Errorf("wallet not found")
	}
	if wallet.Balance <= 0 {
		return wallet.Balance, fmt.Errorf("insufficient points")
	}
	wallet.Balance -= points
	if wallet.Balance < 0 {
		wallet.Balance = 0
	}
	s.wallets[userID] = wallet
	s.ledger = append(s.ledger, model.LedgerEntry{ID: fmt.Sprintf("l_%d", time.Now().UnixNano()), UserID: userID, Delta: -points, Reason: "heartbeat_deduction", StreamID: streamID, SessionID: sessionID, CreatedAt: time.Now().UTC()})
	return wallet.Balance, nil
}

func (s *MemoryStore) Metrics() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	active := 0
	for _, ss := range s.sessions {
		if ss.State == "active" {
			active++
		}
	}
	return map[string]int{"active_sessions": active, "ledger_entries": len(s.ledger)}
}

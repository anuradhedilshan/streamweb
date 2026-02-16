package store

import "streamweb/api/internal/model"

type Repository interface {
	FindUserByEmail(email string) (model.User, bool)
	CreateStream(st model.Stream) model.Stream
	UpdateStream(id string, fn func(*model.Stream)) (model.Stream, bool)
	GetStream(id string) (model.Stream, bool)
	ActiveViewerCount(streamID string) int
	ActiveUserSessionCount(userID string) int
	GetWallet(userID string) (model.Wallet, bool)
	CreateSession(userID, streamID, ip, ua string) model.Session
	GetSession(sessionID string) (model.Session, bool)
	UpdateSessionState(sessionID, state string) bool
	TouchSession(sessionID string) bool
	DeductPoints(userID, streamID, sessionID string, points int64) (int64, error)
	Metrics() map[string]int
	ErrorSummary() map[string]int
	PointsSpentLastMinute() int64
	IncrementError(kind string)
}

package protocol

import (
	"sync"
	"time"
)

// SessionState 会话状态
type SessionState string

const (
	StateDisconnected    SessionState = "disconnected"
	StateConnecting      SessionState = "connecting"
	StateInitialized     SessionState = "initialized"
	StateOfferSent       SessionState = "offer_sent"
	StateAnswerReceived  SessionState = "answer_received"
	StateWebRTCConnected SessionState = "webrtc_connected"
	StateReady           SessionState = "ready"
	StateActive          SessionState = "active"
)

// Session 会话信息
type Session struct {
	ID        string
	State     SessionState
	CreatedAt time.Time
	UpdatedAt time.Time
	mu        sync.RWMutex
}

// NewSession 创建新会话
func NewSession(sessionID string) *Session {
	now := time.Now()
	return &Session{
		ID:        sessionID,
		State:     StateDisconnected,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetState 设置会话状态
func (s *Session) SetState(state SessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = state
	s.UpdatedAt = time.Now()
}

// GetState 获取会话状态
func (s *Session) GetState() SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// IsActive 检查会话是否活跃
func (s *Session) IsActive() bool {
	state := s.GetState()
	return state == StateReady || state == StateActive
}

// SessionManager 会话管理器
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewSessionManager 创建会话管理器
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// AddSession 添加会话
func (sm *SessionManager) AddSession(session *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[session.ID] = session
}

// GetSession 获取会话
func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	session, exists := sm.sessions[sessionID]
	return session, exists
}

// RemoveSession 移除会话
func (sm *SessionManager) RemoveSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, sessionID)
}

// CleanupInactiveSessions 清理非活跃会话
func (sm *SessionManager) CleanupInactiveSessions(timeout time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for id, session := range sm.sessions {
		if now.Sub(session.UpdatedAt) > timeout {
			delete(sm.sessions, id)
		}
	}
}

// GetAllSessions 获取所有会话
func (sm *SessionManager) GetAllSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

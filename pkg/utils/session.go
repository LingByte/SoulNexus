package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Session represents a server-side session
type Session struct {
	// Session ID
	ID string

	// Creation time
	CreatedAt time.Time

	// Last activity time
	LastActivity time.Time

	// Session data
	Data map[string]interface{}

	// Mutex for concurrent access
	mu sync.RWMutex
}

// NewSession creates a new session
func NewSession() *Session {
	return &Session{
		ID:           generateSessionID(),
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Data:         make(map[string]interface{}),
	}
}

// GetID returns the session ID
func (s *Session) GetID() string {
	return s.ID
}

// GetCreatedAt returns the session creation time
func (s *Session) GetCreatedAt() time.Time {
	return s.CreatedAt
}

// GetLastActivity returns the last activity time
func (s *Session) GetLastActivity() time.Time {
	return s.LastActivity
}

// UpdateActivity updates the last activity time
func (s *Session) UpdateActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActivity = time.Now()
}

// GetData retrieves session data
func (s *Session) GetData(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.Data[key]
	return value, ok
}

// SetData sets session data
func (s *Session) SetData(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[key] = value
}

// SessionManager manages server-side sessions
type SessionManager struct {
	// Session mapping
	sessions map[string]*Session

	// Expiry time in seconds
	expirySeconds int

	// Mutex for concurrent access
	mu sync.RWMutex
}

// NewSessionManager creates a session manager
func NewSessionManager(expirySeconds int) *SessionManager {
	manager := &SessionManager{
		sessions:      make(map[string]*Session),
		expirySeconds: expirySeconds,
	}

	// Start goroutine to clean up expired sessions
	go manager.cleanupExpiredSessions()

	return manager
}

// CreateSession creates a new session
func (m *SessionManager) CreateSession() *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := NewSession()
	m.sessions[session.ID] = session
	return session
}

// GetSession retrieves a session
func (m *SessionManager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[id]
	if ok {
		session.UpdateActivity()
	}
	return session, ok
}

// GetActiveSessions retrieves all active session IDs
func (m *SessionManager) GetActiveSessions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessionIDs := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		sessionIDs = append(sessionIDs, id)
	}
	return sessionIDs
}

// TerminateSession terminates a session
func (m *SessionManager) TerminateSession(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
		return true
	}
	return false
}

// cleanupExpiredSessions cleans up expired sessions
func (m *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for id, session := range m.sessions {
			session.mu.RLock()
			if now.Sub(session.LastActivity) > time.Duration(m.expirySeconds)*time.Second {
				delete(m.sessions, id)
			}
			session.mu.RUnlock()
		}
		m.mu.Unlock()
	}
}

// Generate a session ID
func generateSessionID() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(fmt.Errorf("failed to generate random session ID: %w", err))
	}
	return hex.EncodeToString(bytes)
}

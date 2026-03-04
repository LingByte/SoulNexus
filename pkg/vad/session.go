package vad

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// SessionManager VAD 会话管理器
type SessionManager struct {
	client        *Client
	sessions      map[string]*Session
	mu            sync.RWMutex
	logger        *zap.Logger
	ttl           time.Duration // 会话过期时间
	cleanupTicker *time.Ticker
	stopChan      chan struct{}
}

// Session VAD 会话
type Session struct {
	ID             string
	CreatedAt      time.Time
	LastActivityAt time.Time
	HaveVoice      bool
	VoiceStop      bool
	LastSpeechProb float64
	mu             sync.RWMutex
}

// NewSessionManager 创建新的会话管理器
func NewSessionManager(client *Client, logger *zap.Logger) *SessionManager {
	if logger == nil {
		logger = zap.NewNop()
	}

	sm := &SessionManager{
		client:   client,
		sessions: make(map[string]*Session),
		logger:   logger,
		ttl:      5 * time.Minute, // 默认 5 分钟过期
		stopChan: make(chan struct{}),
	}

	// 启动清理 goroutine
	sm.startCleanup()

	return sm
}

// SetTTL 设置会话过期时间
func (sm *SessionManager) SetTTL(ttl time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.ttl = ttl
}

// GetOrCreateSession 获取或创建会话
func (sm *SessionManager) GetOrCreateSession(sessionID string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.mu.Lock()
		session.LastActivityAt = time.Now()
		session.mu.Unlock()
		return session
	}

	session := &Session{
		ID:             sessionID,
		CreatedAt:      time.Now(),
		LastActivityAt: time.Now(),
	}

	sm.sessions[sessionID] = session
	sm.logger.Debug("session created", zap.String("session_id", sessionID))

	return session
}

// ProcessAudio 处理音频数据
func (sm *SessionManager) ProcessAudio(
	sessionID string,
	audioData []byte,
	format string,
	threshold ...float64,
) (*DetectResponse, error) {
	session := sm.GetOrCreateSession(sessionID)

	// 如果没有提供阈值，使用 0（VAD 服务会使用默认值）
	var th float64 = 0
	if len(threshold) > 0 {
		th = threshold[0]
	}

	// 调用 VAD 服务
	result, err := sm.client.Detect(audioData, format, sessionID, th)
	if err != nil {
		return nil, err
	}

	// 更新会话状态
	session.mu.Lock()
	session.HaveVoice = result.HaveVoice
	session.VoiceStop = result.VoiceStop
	session.LastSpeechProb = result.SpeechProb
	session.LastActivityAt = time.Now()
	session.mu.Unlock()

	return result, nil
}

// GetSession 获取会话
func (sm *SessionManager) GetSession(sessionID string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[sessionID]
}

// ResetSession 重置会话
func (sm *SessionManager) ResetSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.mu.Lock()
		session.HaveVoice = false
		session.VoiceStop = false
		session.LastSpeechProb = 0
		session.mu.Unlock()
	}

	// 调用服务端重置
	return sm.client.ResetSession(sessionID)
}

// DeleteSession 删除会话
func (sm *SessionManager) DeleteSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[sessionID]; exists {
		delete(sm.sessions, sessionID)
		sm.logger.Debug("session deleted", zap.String("session_id", sessionID))
	}

	// 调用服务端重置
	return sm.client.ResetSession(sessionID)
}

// ListSessions 列出所有活跃会话
func (sm *SessionManager) ListSessions() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]string, 0, len(sm.sessions))
	for id := range sm.sessions {
		sessions = append(sessions, id)
	}
	return sessions
}

// startCleanup 启动过期会话清理
func (sm *SessionManager) startCleanup() {
	sm.cleanupTicker = time.NewTicker(1 * time.Minute)

	go func() {
		for {
			select {
			case <-sm.cleanupTicker.C:
				sm.cleanup()
			case <-sm.stopChan:
				sm.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// cleanup 清理过期会话
func (sm *SessionManager) cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	expiredCount := 0

	for id, session := range sm.sessions {
		session.mu.RLock()
		lastActivity := session.LastActivityAt
		session.mu.RUnlock()

		if now.Sub(lastActivity) > sm.ttl {
			delete(sm.sessions, id)
			expiredCount++
			sm.logger.Debug("expired session cleaned up", zap.String("session_id", id))
		}
	}

	if expiredCount > 0 {
		sm.logger.Info("session cleanup completed", zap.Int("expired_count", expiredCount))
	}
}

// Close 关闭会话管理器
func (sm *SessionManager) Close() error {
	close(sm.stopChan)
	return nil
}

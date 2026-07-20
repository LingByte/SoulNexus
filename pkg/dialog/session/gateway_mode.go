package session

import "strings"

// GatewayMode reports whether this session simulates the voicedialog gateway path.
func (s *Session) GatewayMode() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.EqualFold(s.dialogMode, "gateway")
}

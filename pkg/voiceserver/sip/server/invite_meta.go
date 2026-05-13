package server

import (
	"strings"
)

// inviteBrief captures INVITE headers + signaling source for HTTP/dialog bridges (WebSocket meta).
type inviteBrief struct {
	From   string
	To     string
	Remote string
}

func (s *SIPServer) storeInviteBrief(callID, from, to, remote string) {
	if s == nil || strings.TrimSpace(callID) == "" {
		return
	}
	s.inviteBriefMu.Lock()
	defer s.inviteBriefMu.Unlock()
	if s.inviteBrief == nil {
		s.inviteBrief = make(map[string]inviteBrief)
	}
	s.inviteBrief[callID] = inviteBrief{From: from, To: to, Remote: remote}
}

func (s *SIPServer) peekInviteBrief(callID string) (from, to, remote string) {
	if s == nil {
		return "", "", ""
	}
	s.inviteBriefMu.RLock()
	defer s.inviteBriefMu.RUnlock()
	b := s.inviteBrief[callID]
	return b.From, b.To, b.Remote
}

func (s *SIPServer) clearInviteBrief(callID string) {
	if s == nil || strings.TrimSpace(callID) == "" {
		return
	}
	s.inviteBriefMu.Lock()
	defer s.inviteBriefMu.Unlock()
	delete(s.inviteBrief, callID)
}

// (endVoiceDialogBridge is now defined in compat.go which dispatches to
// the registered CallLifecycleObserver / OnTerminate hook.)

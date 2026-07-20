package tts

import (
	"strings"
	"sync"
)

// ResumeBuffer stores TTS text interrupted by barge-in when resumePlay is enabled.
type ResumeBuffer struct {
	mu   sync.Mutex
	text string
}

// Save stores remaining speakable text.
func (r *ResumeBuffer) Save(text string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.text = strings.TrimSpace(text)
	r.mu.Unlock()
}

// Take returns saved text and clears the buffer.
func (r *ResumeBuffer) Take() string {
	if r == nil {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.text
	r.text = ""
	return out
}

// HasPending reports whether resume text is waiting.
func (r *ResumeBuffer) HasPending() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return strings.TrimSpace(r.text) != ""
}

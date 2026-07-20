package local

import (
	"fmt"
	"sync"

	"github.com/GanymedeNil/go-webrtcvad"
)

// WebRTCVAD wraps Google's WebRTC voice-activity detector for uplink PCM.
type WebRTCVAD struct {
	vad        *webrtcvad.VAD
	sampleRate int
	mode       int
	mu         sync.Mutex
}

// NewWebRTCVAD creates a WebRTC VAD instance (mode 0–3, default 2).
func NewWebRTCVAD(sampleRate, mode int) (*WebRTCVAD, error) {
	if mode < 0 || mode > 3 {
		mode = 2
	}
	v, err := webrtcvad.New()
	if err != nil {
		return nil, err
	}
	if err := v.SetMode(mode); err != nil {
		return nil, err
	}
	return &WebRTCVAD{vad: v, sampleRate: sampleRate, mode: mode}, nil
}

// IsSpeech returns true when the frame contains speech.
func (w *WebRTCVAD) IsSpeech(pcm []byte) (bool, error) {
	if w == nil || w.vad == nil || len(pcm) < 2 {
		return false, nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	active, err := w.vad.Process(w.sampleRate, pcm)
	if err != nil {
		return false, fmt.Errorf("webrtc vad: %w", err)
	}
	return active, nil
}

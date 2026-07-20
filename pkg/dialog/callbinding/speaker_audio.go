package callbinding

import (
	"strings"
	"sync"
)

// utteranceAudio holds a sliding window of uplink PCM for on-call voiceprint
// identify. The LLM cannot supply real audioBase64 in a live voice session.
type utteranceAudio struct {
	mu         sync.Mutex
	pcm        []byte
	sampleRate int
}

var callUtteranceAudio sync.Map // callID -> *utteranceAudio

const maxUtterancePCMBytes = 16000 * 2 * 10 // ~10s @ 16 kHz mono s16le

func utteranceSlot(callID string) *utteranceAudio {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return nil
	}
	v, _ := callUtteranceAudio.LoadOrStore(callID, &utteranceAudio{sampleRate: 16000})
	ua, _ := v.(*utteranceAudio)
	return ua
}

// AppendUserUtterancePCM appends uplink PCM into a sliding window for this call.
func AppendUserUtterancePCM(callID string, pcm []byte, sampleRate int) {
	ua := utteranceSlot(callID)
	if ua == nil || len(pcm) == 0 {
		return
	}
	ua.mu.Lock()
	defer ua.mu.Unlock()
	if sampleRate > 0 {
		ua.sampleRate = sampleRate
	}
	ua.pcm = append(ua.pcm, pcm...)
	if len(ua.pcm) > maxUtterancePCMBytes {
		ua.pcm = append([]byte(nil), ua.pcm[len(ua.pcm)-maxUtterancePCMBytes:]...)
		if len(ua.pcm)%2 == 1 {
			ua.pcm = ua.pcm[1:]
		}
	}
}

// GetUserUtterancePCM returns a copy of the latest buffered user PCM for the call.
func GetUserUtterancePCM(callID string) (pcm []byte, sampleRate int, ok bool) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return nil, 0, false
	}
	v, loaded := callUtteranceAudio.Load(callID)
	if !loaded {
		return nil, 0, false
	}
	ua, ok := v.(*utteranceAudio)
	if !ok || ua == nil {
		return nil, 0, false
	}
	ua.mu.Lock()
	defer ua.mu.Unlock()
	if len(ua.pcm) < 2 {
		return nil, 0, false
	}
	sr := ua.sampleRate
	if sr <= 0 {
		sr = 16000
	}
	return append([]byte(nil), ua.pcm...), sr, true
}

// ClearUserUtterancePCM drops the call-scoped utterance buffer.
func ClearUserUtterancePCM(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callUtteranceAudio.Delete(callID)
}

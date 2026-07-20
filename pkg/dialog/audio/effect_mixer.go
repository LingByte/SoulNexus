package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// EffectTrack is one looping/once background layer mixed under TTS.
type EffectTrack struct {
	PCM    []byte
	Volume float64
	Mode   string // loop | once
	cursor int
	done   bool
}

// EffectMixer holds decoded PCM effect tracks for one call leg.
type EffectMixer struct {
	mu     sync.Mutex
	tracks []EffectTrack
}

// LoadEffectMixerFromConfig builds active effect tracks from audioTrackConfig JSON.
func LoadEffectMixerFromConfig(ctx context.Context, raw map[string]any, sampleRate int, lg *zap.Logger) (*EffectMixer, error) {
	if len(raw) == 0 || sampleRate <= 0 {
		return nil, nil
	}
	tracksRaw, ok := raw["effectAudioTracks"].([]any)
	if !ok || len(tracksRaw) == 0 {
		if b, err := json.Marshal(raw["effectAudioTracks"]); err == nil {
			var arr []map[string]any
			if json.Unmarshal(b, &arr) == nil {
				for _, row := range arr {
					tracksRaw = append(tracksRaw, row)
				}
			}
		}
	}
	if len(tracksRaw) == 0 {
		return nil, nil
	}
	m := &EffectMixer{}
	for _, item := range tracksRaw {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		active := true
		if v, ok := row["isActive"].(bool); ok {
			active = v
		}
		if !active {
			continue
		}
		name := strings.TrimSpace(fmt.Sprint(row["filename"]))
		if name == "" {
			continue
		}
		vol := 0.8
		if f, ok := row["volume"].(float64); ok && f > 0 {
			vol = f
		}
		mode := strings.ToLower(strings.TrimSpace(fmt.Sprint(row["mode"])))
		if mode == "" {
			mode = "loop"
		}
		pcm, err := loadEffectPCM(ctx, name, sampleRate)
		if err != nil {
			if lg != nil {
				lg.Warn("effect track load failed", zap.String("filename", name), zap.Error(err))
			}
			continue
		}
		if len(pcm) < 2 {
			continue
		}
		m.tracks = append(m.tracks, EffectTrack{PCM: pcm, Volume: vol, Mode: mode})
		if lg != nil {
			lg.Info("effect track loaded", zap.String("filename", name), zap.Float64("volume", vol), zap.String("mode", mode))
		}
	}
	if len(m.tracks) == 0 {
		return nil, nil
	}
	return m, nil
}

func loadEffectPCM(ctx context.Context, ref string, sampleRate int) ([]byte, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("empty effect filename")
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ref, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("http %d", resp.StatusCode)
		}
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return LoadWAVAsPCM16FromBytes(raw, sampleRate)
	}
	if _, err := os.Stat(ref); err == nil {
		return LoadWAVAsPCM16Mono(ref, sampleRate)
	}
	return LoadWAVAsPCM16Mono(ref, sampleRate)
}

// MixFrame mixes active effect tracks into a TTS PCM frame.
func (m *EffectMixer) MixFrame(tts []byte) []byte {
	if m == nil || len(tts) < 2 {
		return tts
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.tracks) == 0 {
		return tts
	}
	out := make([]byte, len(tts))
	copy(out, tts)
	frameSamples := len(tts) / 2
	for ti := range m.tracks {
		tr := &m.tracks[ti]
		if tr.done || len(tr.PCM) < 2 {
			continue
		}
		effectSamples := len(tr.PCM) / 2
		for i := 0; i < frameSamples; i++ {
			idx := tr.cursor + i
			if tr.Mode == "loop" {
				idx = idx % effectSamples
			} else if idx >= effectSamples {
				tr.done = true
				break
			}
			off := i * 2
			eoff := idx * 2
			main := int16(int(out[off]) | int(out[off+1])<<8)
			eff := int16(int(tr.PCM[eoff]) | int(tr.PCM[eoff+1])<<8)
			mixed := float64(main) + float64(eff)*tr.Volume
			if mixed > 32767 {
				mixed = 32767
			} else if mixed < -32768 {
				mixed = -32768
			}
			v := int16(mixed)
			out[off] = byte(v)
			out[off+1] = byte(v >> 8)
		}
		if !tr.done {
			tr.cursor += frameSamples
			if tr.Mode != "loop" && tr.cursor >= effectSamples {
				tr.done = true
			}
		}
	}
	return out
}

package gateway

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	llmtts "github.com/LingByte/lingllm/protocol/voice/tts"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"
	"go.uber.org/zap"
)

const (
	ttsPrefetchMaxDefault = 1
	ttsPrefetchChanCap    = 64
)

func ttsPrefetchMaxConcurrency() int {
	if s := strings.TrimSpace(os.Getenv("VOICE_TTS_PREFETCH_MAX")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return n
		}
	}
	return ttsPrefetchMaxDefault
}

func gatewayTextSegmenterConfig() llmtts.TextSegmenterConfig {
	cfg := llmtts.DefaultTextSegmenterConfig()
	if cfg.FirstMaxChars > 12 {
		cfg.FirstMaxChars = 12
	}
	if s := strings.TrimSpace(os.Getenv("VOICE_TTS_FIRST_MIN_CHARS")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 2 {
			cfg.FirstMinChars = n
		}
	}
	if s := strings.TrimSpace(os.Getenv("VOICE_TTS_FIRST_MAX_CHARS")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			cfg.FirstMaxChars = n
		}
	}
	if s := strings.TrimSpace(os.Getenv("VOICE_TTS_REST_FORCE_MAX_CHARS")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			cfg.RestForceMaxChars = n
		}
	}
	return cfg
}

func (c *Client) initTTSStreamSegmenter() {
	if c == nil || c.cfg.Attached == nil || c.cfg.Attached.TTS == nil {
		return
	}
	cfg := gatewayTextSegmenterConfig()
	c.ttsStreamSegmenter = llmtts.NewTextSegmenterComponent(cfg, c.onStreamTextSegment)
}

func (c *Client) resetTTSStreamState() {
	if c == nil {
		return
	}
	if c.ttsStreamSegmenter != nil {
		c.ttsStreamSegmenter.Reset()
	}
	c.streamMu.Lock()
	c.streamUtteranceID = ""
	c.streamSegIdx = 0
	c.streamMu.Unlock()
}

func (c *Client) invalidateQueuedTTS() {
	if c == nil {
		return
	}
	c.ttsGenInvalidBefore.Store(c.streamGen.Load())
	c.drainTTSQueue()
}

func (c *Client) onStreamTextSegment(seg llmtts.TextSegment) {
	if c == nil {
		return
	}
	text := tts.SanitizeForSpeech(seg.Text)
	if text == "" {
		return
	}
	c.streamMu.Lock()
	uid := c.streamUtteranceID
	gen := c.streamGen.Load()
	c.streamSegIdx++
	segIdx := c.streamSegIdx
	c.streamMu.Unlock()
	if uid == "" {
		uid = fmt.Sprintf("stream-%d", time.Now().UnixNano())
	}
	utterID := fmt.Sprintf("%s-s%d", uid, segIdx)
	c.enqueueSpeakJob(text, utterID, nil, gen)
}

func (c *Client) handleTTSStream(text, utteranceID string, streamEnd bool) {
	if c == nil {
		return
	}
	utteranceID = strings.TrimSpace(utteranceID)
	if utteranceID == "" {
		utteranceID = fmt.Sprintf("stream-%d", time.Now().UnixNano())
	}

	c.streamMu.Lock()
	if utteranceID != c.streamUtteranceID {
		prev := c.streamUtteranceID
		c.streamUtteranceID = utteranceID
		c.streamSegIdx = 0
		c.streamGen.Add(1)
		c.streamMu.Unlock()

		if prev != "" {
			c.invalidateQueuedTTS()
			if c.cfg.Attached != nil && c.cfg.Attached.TTS != nil {
				c.cfg.Attached.TTS.Interrupt()
			}
		}
		if c.ttsStreamSegmenter != nil {
			c.ttsStreamSegmenter.Reset()
		}
	} else {
		c.streamMu.Unlock()
	}

	if c.ttsStreamSegmenter == nil {
		c.log.Warn("voice/gateway tts.stream dropped (segmenter not ready)",
			zap.String("call_id", c.cfg.CallID))
		return
	}

	c.ttsStreamSegmenter.SetPlayID(utteranceID)
	ctx := context.Background()
	if c.runCtx != nil {
		ctx = c.runCtx
	}
	if strings.TrimSpace(text) != "" {
		if _, _, err := c.ttsStreamSegmenter.Process(ctx, text); err != nil {
			c.log.Warn("voice/gateway tts.stream segmenter",
				zap.String("call_id", c.cfg.CallID),
				zap.Error(err))
		}
	}
	if streamEnd {
		c.ttsStreamSegmenter.OnComplete()
		c.streamMu.Lock()
		c.streamUtteranceID = ""
		c.streamMu.Unlock()
	}
}

func (c *Client) enqueueSpeakJob(text, utter string, meta *CommandMeta, gen uint64) {
	if c == nil || c.ttsQueue == nil {
		return
	}
	text = tts.SanitizeForSpeech(text)
	if text == "" {
		return
	}
	if gen != 0 && gen < c.ttsGenInvalidBefore.Load() {
		_ = c.sendEvent(Event{
			Type:        EvTTSEnded,
			CallID:      c.cfg.CallID,
			UtteranceID: utter,
			OK:          false,
		})
		return
	}

	job := ttsJob{text: text, utter: utter, meta: meta, gen: gen}

	select {
	case c.ttsQueue <- job:
	default:
		job.cancelPrefetch()
		c.log.Warn("voice/gateway tts queue full, dropping",
			zap.String("call_id", c.cfg.CallID),
			zap.String("utter", utter))
		_ = c.sendEvent(Event{
			Type:        EvTTSEnded,
			CallID:      c.cfg.CallID,
			UtteranceID: utter,
			OK:          false,
		})
	}
}

func (c *Client) startSegmentPrefetch(job *ttsJob) {
	if c == nil || job == nil || c.cfg.Attached == nil || c.cfg.Attached.TTS == nil {
		return
	}
	svc := c.cfg.Attached.TTS.Service()
	if svc == nil {
		return
	}
	parent := context.Background()
	if c.runCtx != nil {
		parent = c.runCtx
	}
	ctx, cancel := context.WithCancel(parent)
	job.prefetchCancel = cancel
	ch := make(chan []byte, ttsPrefetchChanCap)
	job.prefetchCh = ch

	if cs, ok := svc.(*tts.CachingService); ok {
		if pcm, hit := cs.CachedPCM(job.text); hit && len(pcm) > 0 {
			go func() {
				defer close(ch)
				cp := make([]byte, len(pcm))
				copy(cp, pcm)
				select {
				case ch <- cp:
				case <-ctx.Done():
				}
			}()
			return
		}
	}

	go func() {
		defer close(ch)
		if c.ttsPrefetchSem != nil {
			select {
			case c.ttsPrefetchSem <- struct{}{}:
				defer func() { <-c.ttsPrefetchSem }()
			case <-ctx.Done():
				return
			}
		}
		err := svc.SynthesizeStream(ctx, job.text, func(pcm []byte) error {
			if len(pcm) == 0 {
				return nil
			}
			cp := make([]byte, len(pcm))
			copy(cp, pcm)
			select {
			case ch <- cp:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		if err != nil && ctx.Err() == nil {
			job.setPrefetchErr(err)
		}
	}()
}

var sessionWarmTexts = []string{"嗯，", "好。"}

func (c *Client) warmTTSEngine() {
	if c == nil || c.cfg.Attached == nil || c.cfg.Attached.TTS == nil {
		return
	}
	svc := c.cfg.Attached.TTS.Service()
	if svc == nil {
		return
	}
	parent := c.runCtx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	for _, text := range sessionWarmTexts {
		if err := svc.SynthesizeStream(ctx, text, func([]byte) error { return nil }); err != nil && ctx.Err() == nil {
			c.log.Debug("voice/gateway tts warm",
				zap.String("call_id", c.cfg.CallID),
				zap.String("text", text),
				zap.Error(err))
		}
	}
}

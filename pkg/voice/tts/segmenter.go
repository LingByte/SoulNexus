package tts

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"unicode"

	llmtts "github.com/LingByte/lingllm/protocol/voice/tts"
)

// SegmenterConfig controls LLM→TTS streaming segmentation (aligned with
// lingllm/protocol/voice/tts TextSegmenterConfig).
//
//   - First segment: low latency — sentence end, comma (≥ FirstMinRunes),
//     or FirstMaxRunes with punctuation-aware boundary.
//   - Later segments: sentence end only, or RestForceMaxChars safety split.
type SegmenterConfig struct {
	FirstMinRunes     int
	FirstMaxRunes     int
	RestForceMaxRunes int
}

// LingllmTextSegmenterConfig maps this package's config to lingllm protocol/voice/tts.
func (c SegmenterConfig) LingllmTextSegmenterConfig() llmtts.TextSegmenterConfig {
	return llmtts.TextSegmenterConfig{
		FirstMinChars:     c.FirstMinRunes,
		FirstMaxChars:     c.FirstMaxRunes,
		RestForceMaxChars: c.RestForceMaxRunes,
	}
}

// DefaultSegmenterConfig returns punctuation-first defaults for the voice TTS pipeline.
//
// FirstMinRunes=8 avoids burning a full TTS round-trip on tiny greetings
// like「您好，」(3 runes). Those used to flush on the first comma, then the
// useful body waited behind ~150–400ms TTFB even when the LLM had already
// streamed the rest. Prefer holding until a meaningful first chunk.
func DefaultSegmenterConfig() SegmenterConfig {
	return SegmenterConfig{
		FirstMinRunes:     8,
		FirstMaxRunes:     8,
		RestForceMaxRunes: 120,
	}
}

// PipelineSegmenterConfigFromEnv overlays TTS_FIRST_* / TTS_REST_* env vars.
func PipelineSegmenterConfigFromEnv() SegmenterConfig {
	cfg := DefaultSegmenterConfig()
	if v := strings.TrimSpace(os.Getenv("TTS_FIRST_MIN_RUNES")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.FirstMinRunes = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("TTS_FIRST_MAX_RUNES")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.FirstMaxRunes = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("TTS_REST_FORCE_MAX_RUNES")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.RestForceMaxRunes = n
		}
	}
	return cfg.normalized()
}

func (c SegmenterConfig) normalized() SegmenterConfig {
	out := c
	if out.FirstMinRunes <= 0 {
		out.FirstMinRunes = DefaultSegmenterConfig().FirstMinRunes
	}
	if out.FirstMaxRunes <= 0 {
		out.FirstMaxRunes = DefaultSegmenterConfig().FirstMaxRunes
	}
	if out.RestForceMaxRunes <= 0 {
		out.RestForceMaxRunes = DefaultSegmenterConfig().RestForceMaxRunes
	}
	return out
}

// Segmenter buffers LLM token stream and emits TTS-ready phrases.
type Segmenter struct {
	cfg SegmenterConfig

	mu           sync.Mutex
	buf          strings.Builder
	firstFlushed bool
	onSegment    func(segment string, final bool)
}

// NewSegmenter builds a segmenter. onSegment is invoked synchronously from Push/Complete.
func NewSegmenter(cfg SegmenterConfig, onSegment func(string, bool)) *Segmenter {
	if onSegment == nil {
		onSegment = func(string, bool) {}
	}
	return &Segmenter{
		cfg:       cfg.normalized(),
		onSegment: onSegment,
	}
}

// Push feeds one LLM delta; may emit 0..N segments.
func (s *Segmenter) Push(piece string) {
	if s == nil || piece == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf.WriteString(piece)
	s.evaluateLocked()
}

// Complete flushes any buffered tail (end of LLM stream).
func (s *Segmenter) Complete() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.buf.Len() > 0 {
		s.flushLocked(true)
	}
}

// Reset drops buffered text without emitting.
func (s *Segmenter) Reset() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.buf.Reset()
	s.firstFlushed = false
	s.mu.Unlock()
}

func (s *Segmenter) evaluateLocked() {
	cur := s.buf.String()
	if cur == "" {
		return
	}
	// First segment prefers early flush (comma / FirstMaxRunes) so TTS can
	// start before a full sentence arrives. Sentence-end flush only after
	// the first segment — otherwise a one-shot LLM reply ending in 。/？
	// would enqueue the entire turn as a single Speak.
	if !s.firstFlushed {
		s.evaluateFirstLocked(cur)
		return
	}
	if endsWithSentencePunct(cur) {
		s.flushLocked(false)
		return
	}
	s.evaluateRestLocked(cur)
}

func (s *Segmenter) evaluateFirstLocked(cur string) {
	cfg := s.cfg
	bufLen := runeLen(cur)

	if endsWithPausePunct(cur) && bufLen >= cfg.FirstMinRunes {
		s.flushLocked(false)
		return
	}
	if cfg.FirstMaxRunes > 0 && bufLen >= cfg.FirstMaxRunes {
		if head, tail, ok := splitAtPunctuationBoundary(cur, cfg.FirstMaxRunes, true); ok {
			// FirstMax used to cut at the first comma inside the window,
			// producing tiny「您好，」heads that burn a full TTS round-trip.
			// Reject undersized pause cuts; wait for sentence end or a
			// longer boundary once the buffer grows.
			if runeLen(head) < cfg.FirstMinRunes {
				if endsWithSentencePunct(cur) {
					s.flushLocked(false)
					return
				}
				wide := cfg.FirstMaxRunes * 2
				if wide < cfg.FirstMinRunes+cfg.FirstMaxRunes {
					wide = cfg.FirstMinRunes + cfg.FirstMaxRunes
				}
				if bufLen >= wide {
					if head2, tail2, ok2 := splitAtPunctuationBoundary(cur, wide, true); ok2 && runeLen(head2) >= cfg.FirstMinRunes {
						s.emitLocked(head2, false)
						s.buf.Reset()
						s.buf.WriteString(tail2)
						if tail2 != "" {
							s.evaluateLocked()
						}
					}
				}
				return
			}
			s.emitLocked(head, false)
			s.buf.Reset()
			s.buf.WriteString(tail)
			if tail != "" {
				s.evaluateLocked()
			}
			return
		}
	}
	// Short complete sentence (below FirstMaxRunes) — flush now.
	if endsWithSentencePunct(cur) {
		s.flushLocked(false)
	}
}

func (s *Segmenter) evaluateRestLocked(cur string) {
	cfg := s.cfg
	if cfg.RestForceMaxRunes <= 0 || runeLen(cur) < cfg.RestForceMaxRunes {
		return
	}
	if head, tail, ok := splitAtPunctuationBoundary(cur, cfg.RestForceMaxRunes, false); ok {
		s.emitLocked(head, false)
		s.buf.Reset()
		s.buf.WriteString(tail)
		if tail != "" {
			s.evaluateLocked()
		}
	}
}

func (s *Segmenter) flushLocked(final bool) {
	text := strings.TrimSpace(s.buf.String())
	s.buf.Reset()
	if text == "" {
		return
	}
	s.emitLocked(text, final)
}

func (s *Segmenter) emitLocked(text string, final bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	s.firstFlushed = true
	s.onSegment(text, final)
}

func endsWithSentencePunct(s string) bool {
	return hasSuffixRune(s, '。', '！', '？', '.', '!', '?', '\n')
}

func endsWithPausePunct(s string) bool {
	return hasSuffixRune(s, '，', '、', ',', ';', '；', '：', ':')
}

func hasSuffixRune(s string, marks ...rune) bool {
	if s == "" {
		return false
	}
	runes := []rune(s)
	last := runes[len(runes)-1]
	for _, m := range marks {
		if last == m {
			return true
		}
	}
	return false
}

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// splitAtPunctuationBoundary splits near maxRunes. allowPause enables comma/colon cuts (first segment only).
func splitAtPunctuationBoundary(text string, maxRunes int, allowPause bool) (head, tail string, ok bool) {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return "", "", false
	}
	search := runes[:maxRunes]
	best := -1
	bestRank := 0

	rank := func(r rune) int {
		switch r {
		case '。', '！', '？', '.', '!', '?':
			return 3
		case '，', '、', ',', ';', '；', '：', ':':
			if allowPause {
				return 2
			}
			return 0
		default:
			return 0
		}
	}

	for i, r := range search {
		if rnk := rank(r); rnk > bestRank {
			bestRank = rnk
			best = i
		}
	}
	if bestRank == 0 {
		for i := len(search) - 1; i >= 0; i-- {
			if unicode.IsSpace(search[i]) {
				best = i
				bestRank = 1
				break
			}
		}
		if bestRank == 0 {
			best = maxRunes - 1
		}
	}
	head = string(runes[:best+1])
	tail = string(runes[best+1:])
	head = strings.TrimSpace(head)
	return head, tail, head != ""
}

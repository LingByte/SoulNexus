package tts

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
)

// TextSegmenterConfig controls LLM→TTS streaming segmentation.
//
// Strategy:
//   - First segment: lower latency — break on sentence end, comma (≥FirstMinChars),
//     or FirstMaxChars with punctuation-aware boundary.
//   - Later segments: semantic priority — only sentence-ending punctuation or
//     stream end; no comma/char chops unless RestForceMaxChars safety triggers.
type TextSegmenterConfig struct {
	DelayTimeout time.Duration

	// First segment (latency-optimized)
	FirstMinChars int // comma/pause flush once buffer has this many runes (default 6)
	FirstMaxChars int // first-chunk safety cap with punctuation-aware cut (default 18, 0=off)

	// Later segments (semantic priority)
	RestForceMaxChars int // emergency split only after this many runes without sentence end (default 120, 0=wait for OnComplete)

	// Deprecated: use FirstMinChars / FirstMaxChars. Kept for backward compatibility.
	MinChars int
	MaxChars int
}

// DefaultTextSegmenterConfig returns punctuation-first segmentation defaults.
func DefaultTextSegmenterConfig() TextSegmenterConfig {
	return TextSegmenterConfig{
		DelayTimeout:      0,
		FirstMinChars:     2,
		FirstMaxChars:     18,
		RestForceMaxChars: 120,
	}
}

func (c TextSegmenterConfig) normalized() TextSegmenterConfig {
	out := c
	if out.FirstMinChars == 0 {
		if out.MinChars > 0 {
			out.FirstMinChars = out.MinChars
		} else {
			out.FirstMinChars = 2
		}
	}
	if out.FirstMaxChars == 0 {
		if out.MaxChars > 0 {
			out.FirstMaxChars = out.MaxChars
		} else {
			out.FirstMaxChars = 18
		}
	}
	if out.RestForceMaxChars == 0 && out.MaxChars > 50 {
		out.RestForceMaxChars = 120
	}
	if out.RestForceMaxChars == 0 {
		out.RestForceMaxChars = 120
	}
	return out
}

// TextSegmenterComponent segments streaming LLM text for TTS synthesis.
type TextSegmenterComponent struct {
	mu            sync.Mutex
	config        TextSegmenterConfig
	buffer        string
	lastUpdate    time.Time
	delayTimer    *time.Timer
	outputFunc    func(TextSegment)
	currentPlayID string
	firstFlushed  bool
	logger        func(string)
}

// NewTextSegmenterComponent creates a new text segmenter component.
func NewTextSegmenterComponent(config TextSegmenterConfig, outputFunc func(TextSegment)) *TextSegmenterComponent {
	if config.DelayTimeout == 0 && config.FirstMinChars == 0 && config.FirstMaxChars == 0 &&
		config.RestForceMaxChars == 0 && config.MinChars == 0 && config.MaxChars == 0 {
		config = DefaultTextSegmenterConfig()
	}
	return &TextSegmenterComponent{
		config:     config.normalized(),
		outputFunc: outputFunc,
	}
}

// Name returns the component name.
func (s *TextSegmenterComponent) Name() string {
	return "text_segmenter"
}

// Process ingests streaming LLM text and emits segments when rules fire.
func (s *TextSegmenterComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	text, ok := data.(string)
	if !ok {
		return data, false, fmt.Errorf("expected string, got %T", data)
	}
	if text == "" {
		return text, true, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.buffer += text
	s.lastUpdate = time.Now()
	s.evaluateLocked()
	return "", true, nil
}

func (s *TextSegmenterComponent) evaluateLocked() {
	if s.buffer == "" {
		return
	}

	if endsWithSentencePunct(s.buffer) {
		s.flushLocked(true)
		return
	}

	if !s.firstFlushed {
		s.evaluateFirstSegmentLocked()
		return
	}

	s.evaluateRestSegmentLocked()
}

func (s *TextSegmenterComponent) evaluateFirstSegmentLocked() {
	bufLen := runeLen(s.buffer)
	cfg := s.config

	if endsWithPausePunct(s.buffer) && bufLen >= cfg.FirstMinChars {
		s.flushLocked(false)
		return
	}

	if cfg.FirstMaxChars > 0 && bufLen >= cfg.FirstMaxChars {
		if head, tail, ok := splitAtPunctuationBoundary(s.buffer, cfg.FirstMaxChars, true); ok {
			s.emitPartialLocked(head, false)
			s.buffer = tail
			return
		}
	}
}

func (s *TextSegmenterComponent) evaluateRestSegmentLocked() {
	cfg := s.config
	if cfg.RestForceMaxChars <= 0 || runeLen(s.buffer) < cfg.RestForceMaxChars {
		return
	}
	// Emergency: long unpunctuated run — prefer sentence/pause boundary, never arbitrary mid-rune chop.
	if head, tail, ok := splitAtPunctuationBoundary(s.buffer, cfg.RestForceMaxChars, false); ok {
		s.emitPartialLocked(head, false)
		s.buffer = tail
	}
}

// OnComplete flushes the tail when the LLM stream ends.
func (s *TextSegmenterComponent) OnComplete() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.delayTimer != nil {
		s.delayTimer.Stop()
		s.delayTimer = nil
	}
	if len(s.buffer) > 0 {
		s.flushLocked(true)
	}
}

func (s *TextSegmenterComponent) flushLocked(isFinal bool) {
	if len(s.buffer) == 0 {
		return
	}
	s.emitPartialLocked(s.buffer, isFinal)
	s.buffer = ""
}

func (s *TextSegmenterComponent) emitPartialLocked(text string, isFinal bool) {
	text = SanitizeSpeech(strings.TrimSpace(text))
	if text == "" {
		return
	}

	if s.delayTimer != nil {
		s.delayTimer.Stop()
		s.delayTimer = nil
	}

	segment := TextSegment{
		Text:      text,
		IsFinal:   isFinal,
		Timestamp: time.Now(),
		PlayID:    s.currentPlayID,
	}

	s.firstFlushed = true

	if s.outputFunc != nil {
		s.outputFunc(segment)
	}
	if s.logger != nil {
		s.logger(fmt.Sprintf("[TextSegmenter] Flushed segment: %q (final=%v)", segment.Text, isFinal))
	}
}

// SetPlayID sets the current play ID for segments.
func (s *TextSegmenterComponent) SetPlayID(playID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if playID != s.currentPlayID {
		s.currentPlayID = playID
		s.firstFlushed = false
	}
}

// SetLogger sets the logging callback.
func (s *TextSegmenterComponent) SetLogger(callback func(string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger = callback
}

// Reset resets the segmenter state.
func (s *TextSegmenterComponent) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buffer = ""
	s.firstFlushed = false
	if s.delayTimer != nil {
		s.delayTimer.Stop()
		s.delayTimer = nil
	}
}

func endsWithSentencePunct(s string) bool {
	return hasSuffixRune(s, '。', '！', '？', '.', '!', '?')
}

func endsWithPausePunct(s string) bool {
	return hasSuffixRune(s, '，', '、', ',', ';', '：', ':', '；')
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
	return len([]rune(s))
}

// splitAtPunctuationBoundary returns head+tail split preferring punctuation near limit.
// allowPause true: comma/colon acceptable (first segment). false: sentence enders only.
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
		// No punctuation — fall back to last space for Latin, else hard cut at maxRunes.
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
	return strings.TrimSpace(head), tail, strings.TrimSpace(head) != ""
}

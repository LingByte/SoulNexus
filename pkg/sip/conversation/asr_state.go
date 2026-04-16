package conversation

// Mirrored from pkg/hardware/sessions/state_manager.go — incremental ASR text without restarting the recognizer.

import (
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	// TextSimilarityThreshold 文本相似度阈值
	TextSimilarityThreshold = 0.85
)

// ASRStateManager tracks cumulative ASR strings and yields incremental sentences / deltas
// (same behavior as hardware sessions.ASRStateManager).
type ASRStateManager struct {
	mu                          sync.RWMutex
	lastASRText                 string
	lastProcessedText           string
	lastProcessedCumulativeText string
	sentenceEndings             []rune
}

// NewASRStateManager creates a state manager for SIP voice ASR.
func NewASRStateManager() *ASRStateManager {
	return &ASRStateManager{
		sentenceEndings: []rune{'。', '！', '？', '.', '!', '?'},
	}
}

// UpdateASRText updates with the latest cumulative ASR string and returns text to send downstream (one or more new sentences, or incremental on final).
func (m *ASRStateManager) UpdateASRText(text string, isFinal bool) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if text == "" {
		return ""
	}
	m.lastASRText = text

	if isFinal {
		if text == m.lastProcessedText {
			return ""
		}
		incremental := m.extractIncremental(text)
		if incremental == "" {
			if m.lastProcessedCumulativeText != "" && text == m.lastProcessedCumulativeText {
				return ""
			}
			incremental = text
		}
		m.lastProcessedText = text
		m.lastProcessedCumulativeText = text
		return incremental
	}

	newSentences := m.extractNewSentences(text)
	if newSentences != "" {
		m.lastProcessedCumulativeText = text
		return newSentences
	}

	if m.lastProcessedCumulativeText != "" {
		normalizedLast := normalizeTextFast(m.lastProcessedCumulativeText)
		normalizedCurrent := normalizeTextFast(text)

		if normalizedCurrent != "" && normalizedLast != "" {
			similarity := calculateSimilarityFast(normalizedCurrent, normalizedLast)
			if similarity > TextSimilarityThreshold {
				m.lastProcessedCumulativeText = text
				return ""
			}
		}
	}

	if m.lastProcessedCumulativeText == "" {
		m.lastProcessedCumulativeText = text
	}

	return ""
}

func (m *ASRStateManager) extractIncremental(current string) string {
	if m.lastProcessedCumulativeText == "" {
		return current
	}

	if current == m.lastProcessedCumulativeText {
		return ""
	}

	normalizedLast := normalizeTextFast(m.lastProcessedCumulativeText)
	normalizedCurrent := normalizeTextFast(current)

	if normalizedCurrent == normalizedLast {
		return ""
	}

	similarity := calculateSimilarityFast(normalizedCurrent, normalizedLast)
	if similarity > TextSimilarityThreshold {
		return ""
	}

	if strings.HasPrefix(current, m.lastProcessedCumulativeText) {
		incremental := current[len(m.lastProcessedCumulativeText):]
		normalizedIncremental := normalizeTextFast(incremental)
		if normalizedIncremental == "" {
			return ""
		}

		if len(normalizedIncremental) < len(normalizedLast)/2 {
			incSimilarity := calculateSimilarityFast(normalizedIncremental, normalizedLast)
			if incSimilarity > TextSimilarityThreshold {
				return ""
			}
		}
		return strings.TrimSpace(incremental)
	}

	lastSentence := m.extractLastSentence(current)
	if lastSentence != "" && lastSentence != m.lastProcessedCumulativeText {
		normalizedLastSentence := normalizeTextFast(lastSentence)
		if normalizedLastSentence != normalizedLast {
			similarity := calculateSimilarityFast(normalizedLastSentence, normalizedLast)
			if similarity <= TextSimilarityThreshold {
				return lastSentence
			}
		}
	}
	return current
}

func (m *ASRStateManager) extractNewSentences(current string) string {
	if current == "" {
		return ""
	}

	lastProcessed := m.lastProcessedCumulativeText
	if lastProcessed == "" {
		lastEndingIdx := m.findLastSentenceEnding(current)
		if lastEndingIdx >= 0 {
			runes := []rune(current)
			return string(runes[:lastEndingIdx+1])
		}
		return ""
	}

	if !strings.HasPrefix(current, lastProcessed) {
		lastEndingIdx := m.findLastSentenceEnding(current)
		if lastEndingIdx >= 0 {
			runes := []rune(current)
			return string(runes[:lastEndingIdx+1])
		}
		return ""
	}

	currentRunes := []rune(current)
	lastProcessedRunes := []rune(lastProcessed)

	if len(currentRunes) <= len(lastProcessedRunes) {
		return ""
	}

	newRunes := currentRunes[len(lastProcessedRunes):]
	newText := string(newRunes)

	if newText == "" {
		return ""
	}

	lastEndingIdx := -1
	for i, r := range newRunes {
		for _, ending := range m.sentenceEndings {
			if r == ending {
				lastEndingIdx = i
			}
		}
	}

	if lastEndingIdx >= 0 {
		return string(newRunes[:lastEndingIdx+1])
	}

	return ""
}

func (m *ASRStateManager) findLastSentenceEnding(text string) int {
	runes := []rune(text)
	for i := len(runes) - 1; i >= 0; i-- {
		for _, ending := range m.sentenceEndings {
			if runes[i] == ending {
				return i
			}
		}
	}
	return -1
}

func (m *ASRStateManager) extractLastSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	lastIndex := m.findLastSentenceEnding(text)
	if lastIndex < 0 {
		return text
	}

	sentenceStart := 0
	for i := lastIndex - 1; i >= 0; i-- {
		r, size := utf8.DecodeRuneInString(text[i:])
		for _, ending := range m.sentenceEndings {
			if r == ending {
				sentenceStart = i + size
				for sentenceStart < len(text) && (text[sentenceStart] == ' ' || text[sentenceStart] == '\t') {
					sentenceStart++
				}
				result := strings.TrimSpace(text[sentenceStart : lastIndex+1])
				if result != "" {
					return result
				}
				return text[sentenceStart : lastIndex+1]
			}
		}
	}

	result := strings.TrimSpace(text[:lastIndex+1])
	if result != "" {
		return result
	}
	return text[:lastIndex+1]
}

// Clear resets state (e.g. new call).
func (m *ASRStateManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastASRText = ""
	m.lastProcessedText = ""
	m.lastProcessedCumulativeText = ""
}

// GetLastProcessedCumulativeText returns the internal cumulative pointer (for tests / debug).
func (m *ASRStateManager) GetLastProcessedCumulativeText() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastProcessedCumulativeText
}

func normalizeTextFast(text string) string {
	if text == "" {
		return ""
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	var result strings.Builder
	result.Grow(len(text))

	var lastChar rune
	var hasLastChar bool

	for _, r := range text {
		if unicode.Is(unicode.Han, r) || unicode.IsLetter(r) || unicode.IsNumber(r) {
			if !hasLastChar || r != lastChar {
				result.WriteRune(r)
				lastChar = r
				hasLastChar = true
			}
		}
	}

	return result.String()
}

func calculateSimilarityFast(text1, text2 string) float64 {
	if text1 == "" && text2 == "" {
		return 1.0
	}
	if text1 == "" || text2 == "" {
		return 0.0
	}
	if text1 == text2 {
		return 1.0
	}

	len1, len2 := len(text1), len(text2)
	maxLen := len1
	if len2 > maxLen {
		maxLen = len2
	}

	if maxLen == 0 {
		return 1.0
	}

	if maxLen <= 3 {
		if text1 == text2 {
			return 1.0
		}
		return 0.0
	}

	if abs(len1-len2) > maxLen/2 {
		return 0.0
	}

	distance := levenshteinDistanceFast(text1, text2)
	return 1.0 - float64(distance)/float64(maxLen)
}

func levenshteinDistanceFast(s1, s2 string) int {
	len1, len2 := len(s1), len(s2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}
	if s1 == s2 {
		return 0
	}

	if len1 > len2 {
		s1, s2 = s2, s1
		len1, len2 = len2, len1
	}

	prevRow := make([]int, len1+1)
	currRow := make([]int, len1+1)

	for i := 0; i <= len1; i++ {
		prevRow[i] = i
	}

	for i := 1; i <= len2; i++ {
		currRow[0] = i
		for j := 1; j <= len1; j++ {
			cost := 0
			if s2[i-1] != s1[j-1] {
				cost = 1
			}

			currRow[j] = min3(
				currRow[j-1]+1,
				prevRow[j]+1,
				prevRow[j-1]+cost,
			)
		}
		prevRow, currRow = currRow, prevRow
	}

	return prevRow[len1]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

package sessions

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

// StateManager 状态管理器接口
type StateManager interface {
	// UpdateASRText 更新 ASR 文本并返回增量文本
	UpdateASRText(text string, isFinal bool) string
	// Clear 清空状态
	Clear()
	// GetLastText 获取最后处理的文本
	GetLastText() string
}

// ASRStateManager ASR 状态管理器
type ASRStateManager struct {
	mu                          sync.RWMutex
	lastASRText                 string // 最后收到的 ASR 文本
	lastProcessedText           string // 最后处理的文本（用于 isFinal=true）
	lastProcessedCumulativeText string // 累积处理的文本（用于句子级处理）
	sentenceEndings             []rune // 句子结束符
}

// NewASRStateManager 创建状态管理器
func NewASRStateManager() *ASRStateManager {
	return &ASRStateManager{
		sentenceEndings: []rune{'。', '！', '？', '.', '!', '?'},
	}
}

// UpdateASRText 更新 ASR 文本并返回增量文本
func (m *ASRStateManager) UpdateASRText(text string, isFinal bool) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if text == "" {
		return ""
	}
	m.lastASRText = text

	// 如果是最终结果，直接返回增量
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

	// 非最终结果：检查是否有完整句子
	// 关键改进：只要有完整句子就立即返回，不要等待 isFinal=true
	newSentences := m.extractNewSentences(text)
	if newSentences != "" {
		// 重要：这里返回的是新增的句子，不是全部累积文本
		// 更新 lastProcessedCumulativeText 为当前全文本
		m.lastProcessedCumulativeText = text
		return newSentences
	}

	// 如果没有新句子，进行相似度检测
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

	// 更新累积文本但不返回
	if m.lastProcessedCumulativeText == "" {
		m.lastProcessedCumulativeText = text
	}

	return ""
}

// extractIncremental 提取增量文本
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

// extractNewSentences 从累积文本中提取新句子（返回所有新增的完整句子）
func (m *ASRStateManager) extractNewSentences(current string) string {
	if current == "" {
		return ""
	}

	lastProcessed := m.lastProcessedCumulativeText
	if lastProcessed == "" {
		// 第一次处理，查找所有完整句子
		lastEndingIdx := m.findLastSentenceEnding(current)
		if lastEndingIdx >= 0 {
			// 使用 rune 切片确保 UTF-8 安全
			runes := []rune(current)
			return string(runes[:lastEndingIdx+1])
		}
		return ""
	}

	if !strings.HasPrefix(current, lastProcessed) {
		// 当前文本不是以上次处理的文本开头，可能是新的对话
		lastEndingIdx := m.findLastSentenceEnding(current)
		if lastEndingIdx >= 0 {
			runes := []rune(current)
			return string(runes[:lastEndingIdx+1])
		}
		return ""
	}

	// 使用 rune 切片确保 UTF-8 安全
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

	// 查找新增部分中的最后一个句子结束符（返回所有完整句子）
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

// isCompleteSentence 检查是否是完整句子
func (m *ASRStateManager) isCompleteSentence(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}

	for _, r := range text {
		for _, ending := range m.sentenceEndings {
			if r == ending {
				return true
			}
		}
	}
	return false
}

// findLastSentenceEnding 查找最后一个句子结束符位置（返回 rune 索引）
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

// extractLastSentence 提取最后一个句子
func (m *ASRStateManager) extractLastSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	lastIndex := m.findLastSentenceEnding(text)
	if lastIndex < 0 {
		return text
	}

	// 查找句子开始位置
	sentenceStart := 0
	for i := lastIndex - 1; i >= 0; i-- {
		r, size := utf8.DecodeRuneInString(text[i:])
		for _, ending := range m.sentenceEndings {
			if r == ending {
				sentenceStart = i + size
				// 跳过空白字符
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

// Clear 清空状态
func (m *ASRStateManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastASRText = ""
	m.lastProcessedText = ""
	m.lastProcessedCumulativeText = ""
}

// GetLastText 获取最后处理的文本
func (m *ASRStateManager) GetLastText() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastProcessedText
}

// GetLastProcessedCumulativeText 获取累积处理的文本（用于调试）
func (m *ASRStateManager) GetLastProcessedCumulativeText() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastProcessedCumulativeText
}

// normalizeTextFast 快速标准化文本用于比较
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

// calculateSimilarityFast 快速计算相似度
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

	// 对于很短的字符串，使用简单比较
	if maxLen <= 3 {
		if text1 == text2 {
			return 1.0
		}
		return 0.0
	}

	// 对于长度差异很大的字符串，返回低相似度
	if abs(len1-len2) > maxLen/2 {
		return 0.0
	}

	distance := levenshteinDistanceFast(text1, text2)
	return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistanceFast 快速计算编辑距离
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

	// 使用单行优化
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
				currRow[j-1]+1,    // 插入
				prevRow[j]+1,      // 删除
				prevRow[j-1]+cost, // 替换
			)
		}
		prevRow, currRow = currRow, prevRow
	}

	return prevRow[len1]
}

// min3 返回三个数中的最小值
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

// abs 返回绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

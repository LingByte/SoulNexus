package state

import (
	"context"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	// TextSimilarityThreshold 文本相似度阈值，超过此值认为文本重复
	TextSimilarityThreshold = 0.85
)

// Manager 状态管理器
type Manager struct {
	mu                          sync.RWMutex
	processing                  bool
	ttsPlaying                  bool
	fatalError                  bool
	lastASRText                 string
	lastProcessedText           string
	lastProcessedCumulativeText string
	asrCompleteTime             time.Time
	ttsCtx                      context.Context
	ttsCancel                   context.CancelFunc
}

// NewManager 创建状态管理器
func NewManager() *Manager {
	return &Manager{}
}

// SetProcessing 设置处理状态
func (m *Manager) SetProcessing(processing bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processing = processing
	if processing {
		m.asrCompleteTime = time.Now()
	}
}

// IsProcessing 检查是否正在处理
func (m *Manager) IsProcessing() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.processing
}

// SetTTSPlaying 设置TTS播放状态
func (m *Manager) SetTTSPlaying(playing bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ttsPlaying = playing
}

// IsTTSPlaying 检查TTS是否正在播放
func (m *Manager) IsTTSPlaying() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ttsPlaying
}

// SetFatalError 设置致命错误状态
func (m *Manager) SetFatalError(fatal bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fatalError = fatal
}

// IsFatalError 检查是否有致命错误
func (m *Manager) IsFatalError() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.fatalError
}

// CanProcess 检查是否可以处理新请求（合并检查，减少锁操作）
// 返回 (canProcess, isFatalError, isProcessing)
func (m *Manager) CanProcess() (bool, bool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	canProcess := !m.fatalError && !m.processing
	return canProcess, m.fatalError, m.processing
}

// UpdateASRText 更新ASR文本并返回增量文本
func (m *Manager) UpdateASRText(text string, isLast bool) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if text == "" {
		return ""
	}

	// 更新最后识别的文本
	m.lastASRText = text

	// 如果是最终结果，提取增量
	if isLast {
		// 检查是否已处理过
		if text == m.lastProcessedText {
			return ""
		}

		// 提取增量文本
		incremental := m.extractIncremental(text)
		if incremental == "" {
			return ""
		}

		// 更新已处理的文本
		m.lastProcessedText = text
		m.lastProcessedCumulativeText = text

		return incremental
	}

	// 中间结果，只更新累积文本
	m.lastASRText = text

	// 检查是否是完整句子
	if isCompleteSentence(text) {
		// 提取增量
		incremental := m.extractIncremental(text)
		if incremental == "" {
			return ""
		}

		// 更新已处理的累积文本
		m.lastProcessedCumulativeText = text
		return incremental
	}

	// 不是完整句子，只累积，不返回
	return ""
}

// extractIncremental 提取增量文本
func (m *Manager) extractIncremental(current string) string {
	if m.lastProcessedCumulativeText == "" {
		return current
	}

	// 归一化文本用于比较
	normalizedLast := normalizeText(m.lastProcessedCumulativeText)
	normalizedCurrent := normalizeText(current)

	// 如果完全相同，返回空
	if normalizedCurrent == normalizedLast {
		return ""
	}

	// 计算相似度
	similarity := calculateSimilarity(normalizedCurrent, normalizedLast)
	if similarity > TextSimilarityThreshold {
		return ""
	}

	// 如果当前文本包含上次的文本（前缀匹配），提取增量
	if strings.HasPrefix(current, m.lastProcessedCumulativeText) {
		incremental := current[len(m.lastProcessedCumulativeText):]
		normalizedIncremental := normalizeText(incremental)
		if normalizedIncremental == "" {
			return ""
		}

		// 检查增量是否只是重复
		if len(normalizedIncremental) < len(normalizedLast)/2 {
			incSimilarity := calculateSimilarity(normalizedIncremental, normalizedLast)
			if incSimilarity > TextSimilarityThreshold {
				return ""
			}
		}

		return strings.TrimSpace(incremental)
	}

	// 如果当前文本不包含上次的文本作为前缀，可能是新的独立句子
	// 尝试提取最后一个完整句子作为增量
	// 例如：lastProcessedCumulativeText = "嗯。", current = "跟我说再见。"
	// 应该返回 "跟我说再见。"

	// 检查当前文本是否包含多个句子，提取最后一个句子
	lastSentence := extractLastSentence(current)
	if lastSentence != "" && lastSentence != m.lastProcessedCumulativeText {
		// 检查这个句子是否与上次处理的文本不同
		normalizedLastSentence := normalizeText(lastSentence)
		if normalizedLastSentence != normalizedLast {
			similarity := calculateSimilarity(normalizedLastSentence, normalizedLast)
			if similarity <= TextSimilarityThreshold {
				return lastSentence
			}
		}
	}

	// 如果无法提取增量，返回整个当前文本（可能是全新的内容）
	return current
}

// extractLastSentence 提取文本中的最后一个完整句子
func extractLastSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	// 句子结束标记
	sentenceEndings := []rune{'。', '！', '？', '.', '!', '?'}

	// 从后往前查找最后一个句子结束标记
	lastIndex := -1
	for _, ending := range sentenceEndings {
		idx := strings.LastIndexFunc(text, func(r rune) bool {
			return r == ending
		})
		if idx > lastIndex {
			lastIndex = idx
		}
	}

	if lastIndex >= 0 {
		// 找到最后一个句子结束标记，向前查找句子开始位置
		sentenceStart := 0
		for i := lastIndex - 1; i >= 0; i-- {
			r := rune(text[i])
			// 检查是否是句子结束标记
			for _, ending := range sentenceEndings {
				if r == ending {
					sentenceStart = i + 1
					// 跳过空白字符
					for sentenceStart < len(text) && (text[sentenceStart] == ' ' || text[sentenceStart] == '\t') {
						sentenceStart++
					}
					lastSentence := strings.TrimSpace(text[sentenceStart : lastIndex+1])
					if lastSentence != "" {
						return lastSentence
					}
					return text[sentenceStart : lastIndex+1]
				}
			}
		}

		// 如果没有找到前一个句子结束标记，返回从开头到最后一个句子结束标记的部分
		lastSentence := strings.TrimSpace(text[:lastIndex+1])
		if lastSentence != "" {
			return lastSentence
		}
	}

	// 如果没有找到句子结束标记，返回整个文本
	return text
}

// SetTTSCtx 设置TTS上下文
func (m *Manager) SetTTSCtx(ctx context.Context, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ttsCancel != nil {
		m.ttsCancel()
	}
	m.ttsCtx = ctx
	m.ttsCancel = cancel
}

// GetTTSCtx 获取TTS上下文
func (m *Manager) GetTTSCtx() context.Context {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ttsCtx
}

// CancelTTS 取消TTS
func (m *Manager) CancelTTS() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ttsCancel != nil {
		m.ttsCancel()
		m.ttsCancel = nil
		m.ttsCtx = nil
	}
}

// GetASRCompleteTime 获取ASR完成时间
func (m *Manager) GetASRCompleteTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.asrCompleteTime
}

// Clear 清空状态
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processing = false
	m.ttsPlaying = false
	m.fatalError = false
	m.lastASRText = ""
	m.lastProcessedText = ""
	m.lastProcessedCumulativeText = ""
	if m.ttsCancel != nil {
		m.ttsCancel()
		m.ttsCancel = nil
		m.ttsCtx = nil
	}
}

// isCompleteSentence 检查是否是完整句子
func isCompleteSentence(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}

	// 检查是否包含句子结束标记
	sentenceEndings := []rune{'。', '！', '？', '.', '!', '?'}
	for _, r := range sentenceEndings {
		if strings.ContainsRune(text, r) {
			return true
		}
	}

	return false
}

// normalizeText 归一化文本用于比较
func normalizeText(text string) string {
	if text == "" {
		return ""
	}

	runes := []rune(strings.TrimSpace(text))
	var result strings.Builder

	for _, r := range runes {
		if unicode.Is(unicode.Han, r) || unicode.IsLetter(r) || unicode.IsNumber(r) {
			result.WriteRune(r)
		}
	}

	normalized := result.String()
	return removeRepeatedChars(normalized)
}

// removeRepeatedChars 去除重复字符
func removeRepeatedChars(text string) string {
	if text == "" {
		return ""
	}

	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}

	var result strings.Builder
	lastChar := runes[0]
	result.WriteRune(lastChar)

	for i := 1; i < len(runes); i++ {
		if runes[i] != lastChar {
			result.WriteRune(runes[i])
			lastChar = runes[i]
		}
	}

	return result.String()
}

// calculateSimilarity 计算文本相似度
func calculateSimilarity(text1, text2 string) float64 {
	if text1 == "" && text2 == "" {
		return 1.0
	}
	if text1 == "" || text2 == "" {
		return 0.0
	}
	if text1 == text2 {
		return 1.0
	}

	distance := levenshteinDistance(text1, text2)
	maxLen := len(text1)
	if len(text2) > maxLen {
		maxLen = len(text2)
	}

	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistance 计算编辑距离
func levenshteinDistance(s1, s2 string) int {
	runes1 := []rune(s1)
	runes2 := []rune(s2)

	rows := len(runes1) + 1
	cols := len(runes2) + 1
	dp := make([][]int, rows)
	for i := range dp {
		dp[i] = make([]int, cols)
	}

	for i := 0; i < rows; i++ {
		dp[i][0] = i
	}
	for j := 0; j < cols; j++ {
		dp[0][j] = j
	}

	for i := 1; i < rows; i++ {
		for j := 1; j < cols; j++ {
			if runes1[i-1] == runes2[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				dp[i][j] = min3(
					dp[i-1][j]+1,
					dp[i][j-1]+1,
					dp[i-1][j-1]+1,
				)
			}
		}
	}

	return dp[rows-1][cols-1]
}

// min3 返回三个数的最小值
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

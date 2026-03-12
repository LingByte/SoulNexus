package filter

import (
	"strings"
	"sync"

	"go.uber.org/zap"
)

// Manager 过滤词管理器
type Manager struct {
	blacklist map[string]bool // 黑名单词集合
	counts    map[string]int  // 被过滤词的累计次数
	mu        sync.RWMutex
	logger    *zap.Logger
}

// NewManager 创建过滤词管理器
func NewManager(logger *zap.Logger) (*Manager, error) {
	m := &Manager{
		blacklist: make(map[string]bool),
		counts:    make(map[string]int),
		logger:    logger,
	}

	// 加载默认黑名单
	m.loadDefaultBlacklist()

	m.logger.Info("过滤词管理器初始化成功，使用默认黑名单",
		zap.Int("count", len(m.blacklist)),
	)

	return m, nil
}

// LoadDictionary 加载字典（已废弃，仅保留接口兼容性）
// 现在只使用默认黑名单
func (m *Manager) LoadDictionary() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 清空现有黑名单
	m.blacklist = make(map[string]bool)

	// 重新加载默认黑名单
	m.loadDefaultBlacklist()

	return nil
}

// loadDefaultBlacklist 加载默认黑名单
func (m *Manager) loadDefaultBlacklist() {
	defaultWords := []string{
		"嗯", "嗯嗯",
		"啊", "啊啊",
		"呃", "呃呃",
		"额", "额额",
		"哦", "哦哦",
		"噢",
		"呀",
		"哈", "哈哈",
		"嘿",
		"喂",
		"哼",
		"唉",
		"哎",
		"诶",
		"欸",
	}

	for _, word := range defaultWords {
		m.blacklist[word] = true
		m.blacklist[strings.ToLower(word)] = true
	}
}

// IsFiltered 检查文本是否应该被过滤
// 注意：只使用精确匹配（==），不使用包含匹配（contains）
func (m *Manager) IsFiltered(text string) bool {
	if text == "" {
		return true
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// 去除首尾空白和标点符号
	cleaned := strings.TrimSpace(text)
	cleaned = strings.Trim(cleaned, "。，、；：？！\"\"''（）【】《》")

	// 只进行精确匹配（==），不进行包含匹配
	// 检查原始文本和小写文本
	if m.blacklist[cleaned] || m.blacklist[strings.ToLower(cleaned)] {
		return true
	}

	return false
}

// RecordFiltered 记录被过滤的词（累计计数）
func (m *Manager) RecordFiltered(text string) {
	if text == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cleaned := strings.TrimSpace(text)
	cleaned = strings.Trim(cleaned, "。，、；：？！\"\"''（）【】《》")

	m.counts[cleaned]++

	m.logger.Debug("记录被过滤的词",
		zap.String("text", cleaned),
		zap.Int("count", m.counts[cleaned]),
	)
}

// GetFilteredCount 获取被过滤词的累计次数
func (m *Manager) GetFilteredCount(text string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cleaned := strings.TrimSpace(text)
	cleaned = strings.Trim(cleaned, "。，、；：？！\"\"''（）【】《》")

	return m.counts[cleaned]
}

// GetAllCounts 获取所有被过滤词的累计次数
func (m *Manager) GetAllCounts() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]int)
	for k, v := range m.counts {
		result[k] = v
	}
	return result
}

// Reload 重新加载字典
func (m *Manager) Reload() error {
	return m.LoadDictionary()
}

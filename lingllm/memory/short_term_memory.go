package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/lingllm/protocol"
)

// ShortTermMemory represents the short-term memory for recent N rounds.
//
// L2: Short-Term Memory (短期会话记忆，最近 N 轮)
// 作用：最近几轮对话的摘要（不是原文），跨轮连贯、不重复、不溢出
// 存储：内存缓存，带 TTL（比如 1 小时）
// 特点：会话级、自动摘要、自动过期、容量可控
type ShortTermMemory struct {
	// In-memory storage of round summaries
	summaries map[string]*RoundSummary

	// In-memory index of round IDs for ordering
	roundIndex []string

	// Configuration
	maxRounds   int
	ttl         time.Duration
	cachePrefix string

	// Optional: L2 disk persistence path
	persistPath string

	mu sync.RWMutex
}

// RoundSummary represents a summary of a conversation round.
type RoundSummary struct {
	RoundID      string                 `json:"round_id"`
	Summary      string                 `json:"summary"`
	KeyPoints    []string               `json:"key_points"`
	Messages     int                    `json:"messages"`
	Thoughts     int                    `json:"thoughts"`
	Actions      int                    `json:"actions"`
	Observations int                    `json:"observations"`
	Timestamp    time.Time              `json:"timestamp"`
	ExpiresAt    time.Time              `json:"expires_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// stmDiskSnapshot represents the disk snapshot format for persistence.
type stmDiskSnapshot struct {
	RoundIndex []string                `json:"round_index"`
	Summaries  map[string]RoundSummary `json:"summaries"`
}

// NewShortTermMemory creates a new short-term memory.
func NewShortTermMemory(maxRounds int, ttl time.Duration) *ShortTermMemory {
	if maxRounds <= 0 {
		maxRounds = 10
	}
	if ttl <= 0 {
		ttl = time.Hour
	}

	return &ShortTermMemory{
		summaries:   make(map[string]*RoundSummary),
		roundIndex:  make([]string, 0, maxRounds),
		maxRounds:   maxRounds,
		ttl:         ttl,
		cachePrefix: "stm:",
	}
}

// AddRoundSummary adds a summary of a completed round.
// When capacity exceeds maxRounds, the oldest round is removed and returned as evicted
// so L4 can consolidate it.
func (s *ShortTermMemory) AddRoundSummary(summary *RoundSummary) (evicted *RoundSummary, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if summary == nil {
		return nil, fmt.Errorf("summary cannot be nil")
	}

	summary.Timestamp = time.Now()
	summary.ExpiresAt = time.Now().Add(s.ttl)

	s.summaries[summary.RoundID] = summary
	s.roundIndex = append(s.roundIndex, summary.RoundID)

	// If exceeds capacity, evict oldest
	if len(s.roundIndex) > s.maxRounds {
		oldRoundID := s.roundIndex[0]
		s.roundIndex = s.roundIndex[1:]
		evicted = s.summaries[oldRoundID]
		delete(s.summaries, oldRoundID)
	}

	if err := s.savePersistLocked(); err != nil {
		return evicted, err
	}
	return evicted, nil
}

// GetRoundSummary retrieves a summary of a specific round.
func (s *ShortTermMemory) GetRoundSummary(roundID string) (*RoundSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary, exists := s.summaries[roundID]
	if !exists {
		return nil, fmt.Errorf("round summary not found: %s", roundID)
	}

	return summary, nil
}

// GetRecentSummaries retrieves the most recent N summaries.
func (s *ShortTermMemory) GetRecentSummaries(count int) []*RoundSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if count <= 0 || count > len(s.roundIndex) {
		count = len(s.roundIndex)
	}

	summaries := make([]*RoundSummary, 0, count)

	// Get from most recent to oldest
	for i := len(s.roundIndex) - 1; i >= 0 && len(summaries) < count; i-- {
		if summary, exists := s.summaries[s.roundIndex[i]]; exists {
			summaries = append(summaries, summary)
		}
	}

	return summaries
}

// GetAllSummaries retrieves all stored summaries.
func (s *ShortTermMemory) GetAllSummaries() []*RoundSummary {
	return s.GetRecentSummaries(s.maxRounds)
}

// GetSummariesOldestFirst returns all L2 summaries in chronological order (for flushing to L4).
func (s *ShortTermMemory) GetSummariesOldestFirst() []*RoundSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*RoundSummary, 0, len(s.roundIndex))
	for _, roundID := range s.roundIndex {
		if summary, exists := s.summaries[roundID]; exists {
			cp := *summary
			out = append(out, &cp)
		}
	}
	return out
}

// BindPersistence enables L2 disk persistence (dataDir/l2_<subjectID>.json).
func (s *ShortTermMemory) BindPersistence(dataDir, subjectID string) error {
	if s == nil {
		return fmt.Errorf("short-term memory is nil")
	}
	subjectID = sanitizeFileToken(subjectID)
	if subjectID == "" || dataDir == "" {
		return fmt.Errorf("data dir and subject id are required")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	s.mu.Lock()
	s.persistPath = filepath.Join(dataDir, "l2_"+subjectID+".json")
	s.mu.Unlock()
	return s.loadFromDisk()
}

// sanitizeFileToken sanitizes a string for use as a filename.
func sanitizeFileToken(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// loadFromDisk loads summaries from disk.
func (s *ShortTermMemory) loadFromDisk() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.persistPath == "" {
		return nil
	}
	data, err := os.ReadFile(s.persistPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var snap stmDiskSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}
	s.roundIndex = snap.RoundIndex
	if s.roundIndex == nil {
		s.roundIndex = make([]string, 0, s.maxRounds)
	}
	now := time.Now()
	for roundID, summary := range snap.Summaries {
		summary.ExpiresAt = now.Add(s.ttl)
		cp := summary
		s.summaries[roundID] = &cp
	}
	return nil
}

// savePersistLocked saves summaries to disk (must be called with lock held).
func (s *ShortTermMemory) savePersistLocked() error {
	if s.persistPath == "" {
		return nil
	}
	snap := stmDiskSnapshot{
		RoundIndex: append([]string(nil), s.roundIndex...),
		Summaries:  make(map[string]RoundSummary, len(s.roundIndex)),
	}
	for roundID, summary := range s.summaries {
		snap.Summaries[roundID] = *summary
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.persistPath, data, 0o644)
}

// removePersistFileLocked removes the persist file (must be called with lock held).
func (s *ShortTermMemory) removePersistFileLocked() {
	if s.persistPath != "" {
		_ = os.Remove(s.persistPath)
	}
}

// BuildContextString builds a context string from recent summaries.
func (s *ShortTermMemory) BuildContextString(count int) string {
	summaries := s.GetRecentSummaries(count)

	if len(summaries) == 0 {
		return ""
	}

	var contextStr strings.Builder
	contextStr.WriteString("=== Recent Conversation History ===\n\n")

	for i := len(summaries) - 1; i >= 0; i-- {
		summary := summaries[i]
		contextStr.WriteString(fmt.Sprintf("Round %s (at %s):\n", summary.RoundID, summary.Timestamp.Format("15:04:05")))
		contextStr.WriteString(fmt.Sprintf("Summary: %s\n", summary.Summary))

		if len(summary.KeyPoints) > 0 {
			contextStr.WriteString("Key Points:\n")
			for _, kp := range summary.KeyPoints {
				contextStr.WriteString(fmt.Sprintf("  - %s\n", kp))
			}
		}

		contextStr.WriteString(fmt.Sprintf("Stats: %d messages, %d thoughts, %d actions, %d observations\n\n",
			summary.Messages, summary.Thoughts, summary.Actions, summary.Observations))
	}

	return contextStr.String()
}

// GenerateSummaryFromWorkingMemory generates a summary from working memory.
func (s *ShortTermMemory) GenerateSummaryFromWorkingMemory(wm *WorkingMemory) *RoundSummary {
	stats := wm.GetStats()
	messages := wm.GetMessages()
	chain := wm.GetReActChain()

	// Extract key points from thoughts
	keyPoints := make([]string, 0)
	for _, thought := range chain.Thoughts {
		if len(thought) > 0 {
			keyPoints = append(keyPoints, thought)
		}
	}

	// Generate summary from messages
	summary := s.generateSummaryText(messages)

	return &RoundSummary{
		RoundID:      stats.RoundID,
		Summary:      summary,
		KeyPoints:    keyPoints,
		Messages:     stats.MessageCount,
		Thoughts:     stats.ThoughtCount,
		Actions:      stats.ActionCount,
		Observations: stats.ObservationCount,
		Timestamp:    time.Now(),
		ExpiresAt:    time.Now().Add(s.ttl),
	}
}

// generateSummaryText generates a summary from messages.
func (s *ShortTermMemory) generateSummaryText(messages []protocol.Message) string {
	if len(messages) == 0 {
		return "No conversation in this round"
	}

	// Extract user and assistant messages
	var userMsgs []string
	var assistantMsgs []string

	for _, msg := range messages {
		if msg.Role == protocol.RoleUser && len(msg.Content) > 0 {
			userMsgs = append(userMsgs, msg.Content)
		} else if msg.Role == protocol.RoleAssistant && len(msg.Content) > 0 {
			assistantMsgs = append(assistantMsgs, msg.Content)
		}
	}

	// Build summary
	var summary string
	if len(userMsgs) > 0 {
		summary += fmt.Sprintf("User asked: %s. ", userMsgs[len(userMsgs)-1])
	}
	if len(assistantMsgs) > 0 {
		summary += fmt.Sprintf("Assistant responded: %s", assistantMsgs[len(assistantMsgs)-1])
	}

	return summary
}

// GetStats returns statistics about short-term memory.
func (s *ShortTermMemory) GetStats() *ShortTermMemoryStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &ShortTermMemoryStats{
		StoredRounds: len(s.roundIndex),
		MaxRounds:    s.maxRounds,
		TTL:          s.ttl,
	}
}

// ShortTermMemoryStats represents statistics about short-term memory.
type ShortTermMemoryStats struct {
	StoredRounds int
	MaxRounds    int
	TTL          time.Duration
}

// Clear clears all stored summaries.
func (s *ShortTermMemory) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.summaries = make(map[string]*RoundSummary)
	s.roundIndex = make([]string, 0, s.maxRounds)
	s.removePersistFileLocked()
	return nil
}

// ToMessages converts short-term memory to messages for LLM context.
func (s *ShortTermMemory) ToMessages(count int) []protocol.Message {
	contextStr := s.BuildContextString(count)

	if contextStr == "" {
		return nil
	}

	return []protocol.Message{
		{
			Role:    protocol.RoleSystem,
			Content: contextStr,
		},
	}
}

package agent

import (
	"time"
)

// AddRecord 添加执行记录
func (h *ExecutionHistory) AddRecord(record ExecutionRecord) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := record.AgentID + ":" + record.TaskType
	if h.records[key] == nil {
		h.records[key] = make([]ExecutionRecord, 0)
	}

	h.records[key] = append(h.records[key], record)

	// 限制历史记录大小
	if len(h.records[key]) > h.maxSize {
		// 保留最近的记录
		h.records[key] = h.records[key][len(h.records[key])-h.maxSize:]
	}
}

// GetAgentRecords 获取Agent的执行记录
func (h *ExecutionHistory) GetAgentRecords(agentID, taskType string) []ExecutionRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	key := agentID + ":" + taskType
	return h.records[key]
}

// GetRecentRecords 获取最近的执行记录
func (h *ExecutionHistory) GetRecentRecords(agentID string, limit int) []ExecutionRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	allRecords := make([]ExecutionRecord, 0)
	for key, records := range h.records {
		if len(key) > len(agentID) && key[:len(agentID)] == agentID {
			allRecords = append(allRecords, records...)
		}
	}

	// 按时间排序（最新的在前）
	for i := 0; i < len(allRecords)-1; i++ {
		for j := i + 1; j < len(allRecords); j++ {
			if allRecords[i].Timestamp.Before(allRecords[j].Timestamp) {
				allRecords[i], allRecords[j] = allRecords[j], allRecords[i]
			}
		}
	}

	if limit > 0 && len(allRecords) > limit {
		return allRecords[:limit]
	}

	return allRecords
}

// GetStats 获取统计信息
func (h *ExecutionHistory) GetStats(agentID, taskType string) *ExecutionStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	records := h.GetAgentRecords(agentID, taskType)
	if len(records) == 0 {
		return &ExecutionStats{
			TotalExecutions: 0,
			SuccessRate:     0,
			AvgDuration:     0,
		}
	}

	successCount := 0
	totalDuration := time.Duration(0)
	for _, record := range records {
		if record.Success {
			successCount++
		}
		totalDuration += record.Duration
	}

	return &ExecutionStats{
		TotalExecutions: len(records),
		SuccessRate:     float64(successCount) / float64(len(records)),
		AvgDuration:     totalDuration / time.Duration(len(records)),
		LastExecution:   records[len(records)-1].Timestamp,
	}
}

// ExecutionStats 执行统计
type ExecutionStats struct {
	TotalExecutions int
	SuccessRate     float64
	AvgDuration     time.Duration
	LastExecution   time.Time
}

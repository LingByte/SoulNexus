package memory

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/LingByte/lingllm/protocol"
)

func TestNewShortTermMemory(t *testing.T) {
	stm := NewShortTermMemory(10, time.Hour)

	if stm.maxRounds != 10 {
		t.Errorf("expected maxRounds=10, got %d", stm.maxRounds)
	}

	if stm.ttl != time.Hour {
		t.Errorf("expected ttl=1h, got %v", stm.ttl)
	}
}

func TestAddRoundSummary(t *testing.T) {
	stm := NewShortTermMemory(3, time.Hour)

	summary := &RoundSummary{
		RoundID:  "round-1",
		Summary:  "Test summary",
		Messages: 5,
	}

	evicted, err := stm.AddRoundSummary(summary)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evicted != nil {
		t.Errorf("expected no eviction on first add")
	}

	retrieved, err := stm.GetRoundSummary("round-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.Summary != "Test summary" {
		t.Errorf("expected 'Test summary', got '%s'", retrieved.Summary)
	}
}

func TestAddRoundSummaryEviction(t *testing.T) {
	stm := NewShortTermMemory(2, time.Hour)

	// Add 3 summaries to trigger eviction
	summary1 := &RoundSummary{RoundID: "round-1", Summary: "Summary 1"}
	summary2 := &RoundSummary{RoundID: "round-2", Summary: "Summary 2"}
	summary3 := &RoundSummary{RoundID: "round-3", Summary: "Summary 3"}

	stm.AddRoundSummary(summary1)
	stm.AddRoundSummary(summary2)
	evicted, _ := stm.AddRoundSummary(summary3)

	if evicted == nil {
		t.Errorf("expected eviction of round-1")
	}

	if evicted.RoundID != "round-1" {
		t.Errorf("expected evicted round-1, got %s", evicted.RoundID)
	}

	// Verify round-1 is no longer accessible
	_, err := stm.GetRoundSummary("round-1")
	if err == nil {
		t.Errorf("expected error for evicted round")
	}
}

func TestGetRecentSummaries(t *testing.T) {
	stm := NewShortTermMemory(10, time.Hour)

	// Add 5 summaries
	for i := 1; i <= 5; i++ {
		summary := &RoundSummary{
			RoundID: "round-" + string(rune(48+i)),
			Summary: "Summary " + string(rune(48+i)),
		}
		stm.AddRoundSummary(summary)
	}

	recent := stm.GetRecentSummaries(3)
	if len(recent) != 3 {
		t.Errorf("expected 3 recent summaries, got %d", len(recent))
	}

	// Should be in reverse order (most recent first)
	if recent[0].RoundID != "round-5" {
		t.Errorf("expected most recent round-5, got %s", recent[0].RoundID)
	}
}

func TestGetAllSummaries(t *testing.T) {
	stm := NewShortTermMemory(5, time.Hour)

	for i := 1; i <= 3; i++ {
		summary := &RoundSummary{
			RoundID: "round-" + string(rune(48+i)),
			Summary: "Summary " + string(rune(48+i)),
		}
		stm.AddRoundSummary(summary)
	}

	all := stm.GetAllSummaries()
	if len(all) != 3 {
		t.Errorf("expected 3 summaries, got %d", len(all))
	}
}

func TestGetSummariesOldestFirst(t *testing.T) {
	stm := NewShortTermMemory(10, time.Hour)

	for i := 1; i <= 3; i++ {
		summary := &RoundSummary{
			RoundID: "round-" + string(rune(48+i)),
			Summary: "Summary " + string(rune(48+i)),
		}
		stm.AddRoundSummary(summary)
	}

	oldest := stm.GetSummariesOldestFirst()
	if len(oldest) != 3 {
		t.Errorf("expected 3 summaries, got %d", len(oldest))
	}

	// Should be in chronological order (oldest first)
	if oldest[0].RoundID != "round-1" {
		t.Errorf("expected oldest round-1, got %s", oldest[0].RoundID)
	}

	if oldest[2].RoundID != "round-3" {
		t.Errorf("expected newest round-3, got %s", oldest[2].RoundID)
	}
}

func TestGenerateSummaryFromWorkingMemory(t *testing.T) {
	stm := NewShortTermMemory(10, time.Hour)
	wm := NewWorkingMemory("round-1")

	wm.AddMessage(protocol.RoleUser, "Hello")
	wm.AddMessage(protocol.RoleAssistant, "Hi there")
	wm.AddThought("User greeted")

	summary := stm.GenerateSummaryFromWorkingMemory(wm)

	if summary.RoundID != "round-1" {
		t.Errorf("expected round-1, got %s", summary.RoundID)
	}

	// AddThought also adds a message, so we have 3 total
	if summary.Messages != 3 {
		t.Errorf("expected 3 messages, got %d", summary.Messages)
	}

	if summary.Thoughts != 1 {
		t.Errorf("expected 1 thought, got %d", summary.Thoughts)
	}

	if summary.Summary == "" {
		t.Errorf("expected non-empty summary")
	}
}

func TestBuildContextString(t *testing.T) {
	stm := NewShortTermMemory(10, time.Hour)

	summary := &RoundSummary{
		RoundID:      "round-1",
		Summary:      "Test summary",
		KeyPoints:    []string{"point 1", "point 2"},
		Messages:     5,
		Thoughts:     2,
		Actions:      1,
		Observations: 1,
	}

	stm.AddRoundSummary(summary)

	contextStr := stm.BuildContextString(1)

	if contextStr == "" {
		t.Errorf("expected non-empty context string")
	}

	if !contains(contextStr, "Recent Conversation History") {
		t.Errorf("expected 'Recent Conversation History' in context")
	}

	if !contains(contextStr, "Test summary") {
		t.Errorf("expected summary in context")
	}

	if !contains(contextStr, "point 1") {
		t.Errorf("expected key points in context")
	}
}

func TestToMessages(t *testing.T) {
	stm := NewShortTermMemory(10, time.Hour)

	summary := &RoundSummary{
		RoundID: "round-1",
		Summary: "Test summary",
	}

	stm.AddRoundSummary(summary)

	messages := stm.ToMessages(1)

	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Role != protocol.RoleSystem {
		t.Errorf("expected system role")
	}

	if messages[0].Content == "" {
		t.Errorf("expected non-empty content")
	}
}

func TestSTMClear(t *testing.T) {
	stm := NewShortTermMemory(10, time.Hour)

	summary := &RoundSummary{RoundID: "round-1", Summary: "Test"}
	stm.AddRoundSummary(summary)

	stm.Clear()

	stats := stm.GetStats()
	if stats.StoredRounds != 0 {
		t.Errorf("expected 0 stored rounds after clear, got %d", stats.StoredRounds)
	}
}

func TestSTMGetStats(t *testing.T) {
	stm := NewShortTermMemory(10, time.Hour)

	summary := &RoundSummary{RoundID: "round-1", Summary: "Test"}
	stm.AddRoundSummary(summary)

	stats := stm.GetStats()

	if stats.StoredRounds != 1 {
		t.Errorf("expected 1 stored round, got %d", stats.StoredRounds)
	}

	if stats.MaxRounds != 10 {
		t.Errorf("expected maxRounds=10, got %d", stats.MaxRounds)
	}

	if stats.TTL != time.Hour {
		t.Errorf("expected ttl=1h, got %v", stats.TTL)
	}
}

func TestBindPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	stm := NewShortTermMemory(10, time.Hour)

	err := stm.BindPersistence(tmpDir, "test-subject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify persistence file path is set
	expectedPath := filepath.Join(tmpDir, "l2_test-subject.json")
	if stm.persistPath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, stm.persistPath)
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	// Create and populate first instance
	stm1 := NewShortTermMemory(10, time.Hour)
	stm1.BindPersistence(tmpDir, "test-subject")

	summary := &RoundSummary{
		RoundID:   "round-1",
		Summary:   "Test summary",
		KeyPoints: []string{"point 1", "point 2"},
		Messages:  5,
	}

	stm1.AddRoundSummary(summary)

	// Create second instance and load from disk
	stm2 := NewShortTermMemory(10, time.Hour)
	stm2.BindPersistence(tmpDir, "test-subject")

	// Verify data was loaded
	retrieved, err := stm2.GetRoundSummary("round-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.Summary != "Test summary" {
		t.Errorf("expected 'Test summary', got '%s'", retrieved.Summary)
	}

	if len(retrieved.KeyPoints) != 2 {
		t.Errorf("expected 2 key points, got %d", len(retrieved.KeyPoints))
	}
}

func TestSanitizeFileToken(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user-123", "user-123"},
		{"user/123", "user_123"},
		{"user\\123", "user_123"},
		{"user:123", "user_123"},
		{"user*123", "user_123"},
		{"user?123", "user_123"},
		{"user\"123", "user_123"},
		{"user<123", "user_123"},
		{"user>123", "user_123"},
		{"user|123", "user_123"},
		{"  user  ", "user"},
		{"", ""},
	}

	for _, test := range tests {
		result := sanitizeFileToken(test.input)
		if result != test.expected {
			t.Errorf("sanitizeFileToken(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestSTMConcurrency(t *testing.T) {
	stm := NewShortTermMemory(10, time.Hour)

	done := make(chan bool)

	// Add summaries concurrently
	for i := 0; i < 5; i++ {
		go func(id int) {
			summary := &RoundSummary{
				RoundID: "round-" + string(rune(48+id)),
				Summary: "Summary " + string(rune(48+id)),
			}
			stm.AddRoundSummary(summary)
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	stats := stm.GetStats()
	if stats.StoredRounds == 0 {
		t.Errorf("expected summaries after concurrent adds")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

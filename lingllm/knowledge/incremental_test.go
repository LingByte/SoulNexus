package knowledge

import (
	"testing"
	"time"
)

func TestComputeHash(t *testing.T) {
	record := &Record{
		ID:      "1",
		Title:   "Test",
		Content: "Content",
		Tags:    []string{"tag1"},
	}

	hash1 := ComputeHash(record)
	hash2 := ComputeHash(record)

	if hash1 != hash2 {
		t.Error("same record should produce same hash")
	}

	// Change content
	record.Content = "Different"
	hash3 := ComputeHash(record)

	if hash1 == hash3 {
		t.Error("different content should produce different hash")
	}
}

func TestIncrementalUpdater_GetChanges(t *testing.T) {
	updater := NewIncrementalUpdater()

	records := []Record{
		{ID: "1", Title: "Doc1", Content: "Content1"},
		{ID: "2", Title: "Doc2", Content: "Content2"},
	}

	// First sync - all should be new
	changed, deleted := updater.GetChanges(records)
	if len(changed) != 2 {
		t.Errorf("expected 2 changed, got %d", len(changed))
	}
	if len(deleted) != 0 {
		t.Errorf("expected 0 deleted, got %d", len(deleted))
	}

	// Second sync - no changes
	changed, deleted = updater.GetChanges(records)
	if len(changed) != 0 {
		t.Errorf("expected 0 changed, got %d", len(changed))
	}
	if len(deleted) != 0 {
		t.Errorf("expected 0 deleted, got %d", len(deleted))
	}

	// Third sync - modify one, delete one
	records[0].Content = "Modified"
	records = records[:1]

	changed, deleted = updater.GetChanges(records)
	if len(changed) != 1 {
		t.Errorf("expected 1 changed, got %d", len(changed))
	}
	if len(deleted) != 1 {
		t.Errorf("expected 1 deleted, got %d", len(deleted))
	}
}

func TestIncrementalUpdater_UpdateHashes(t *testing.T) {
	updater := NewIncrementalUpdater()

	records := []Record{
		{ID: "1", Title: "Doc1", Content: "Content1"},
	}

	updater.UpdateHashes(records)

	hashes := updater.GetHashes()
	if len(hashes) != 1 {
		t.Errorf("expected 1 hash, got %d", len(hashes))
	}

	if _, ok := hashes["1"]; !ok {
		t.Error("expected hash for document 1")
	}
}

func TestIncrementalUpdater_Clear(t *testing.T) {
	updater := NewIncrementalUpdater()

	records := []Record{
		{ID: "1", Title: "Doc1", Content: "Content1"},
	}

	updater.UpdateHashes(records)

	if len(updater.GetHashes()) != 1 {
		t.Error("expected 1 hash before clear")
	}

	updater.Clear()

	if len(updater.GetHashes()) != 0 {
		t.Error("expected 0 hashes after clear")
	}
}

func TestIncrementalUpdater_LastSyncTime(t *testing.T) {
	updater := NewIncrementalUpdater()

	before := time.Now()
	time.Sleep(10 * time.Millisecond)

	records := []Record{
		{ID: "1", Title: "Doc1", Content: "Content1"},
	}
	updater.UpdateHashes(records)

	after := time.Now()
	lastSync := updater.LastSyncTime()

	if lastSync.Before(before) || lastSync.After(after) {
		t.Error("last sync time should be between before and after")
	}
}

func TestIncrementalUpdater_GetSyncStats(t *testing.T) {
	updater := NewIncrementalUpdater()

	records := []Record{
		{ID: "1", Title: "Doc1", Content: "Content1"},
		{ID: "2", Title: "Doc2", Content: "Content2"},
		{ID: "3", Title: "Doc3", Content: "Content3"},
	}

	// Initial sync
	updater.UpdateHashes(records)

	// Modify one, keep one, delete one
	records[0].Content = "Modified"
	records = records[:2]

	stats := updater.GetSyncStats(records)

	if stats.Added != 0 {
		t.Errorf("expected 0 added, got %d", stats.Added)
	}
	if stats.Updated != 1 {
		t.Errorf("expected 1 updated, got %d", stats.Updated)
	}
	if stats.Unchanged != 1 {
		t.Errorf("expected 1 unchanged, got %d", stats.Unchanged)
	}
	if stats.Deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", stats.Deleted)
	}
}

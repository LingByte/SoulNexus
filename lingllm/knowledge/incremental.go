package knowledge

import (
	"context"
	"crypto/md5"
	"fmt"
	"sync"
	"time"
)

// DocumentHash represents a document's hash for change detection
type DocumentHash struct {
	ID        string
	Hash      string
	UpdatedAt time.Time
}

// IncrementalUpdater manages incremental document updates
type IncrementalUpdater struct {
	mu       sync.RWMutex
	hashes   map[string]string // ID -> Hash
	lastSync time.Time
}

// NewIncrementalUpdater creates a new incremental updater
func NewIncrementalUpdater() *IncrementalUpdater {
	return &IncrementalUpdater{
		hashes:   make(map[string]string),
		lastSync: time.Now(),
	}
}

// ComputeHash computes the hash of a document
func ComputeHash(record *Record) string {
	data := fmt.Sprintf("%s:%s:%s:%v:%v",
		record.ID,
		record.Title,
		record.Content,
		record.Tags,
		record.Metadata,
	)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// GetChanges detects which documents have changed
func (iu *IncrementalUpdater) GetChanges(records []Record) ([]Record, []string) {
	iu.mu.Lock()
	defer iu.mu.Unlock()

	var changed []Record
	var deleted []string

	// Check for changed or new documents
	for i := range records {
		currentHash := ComputeHash(&records[i])
		previousHash, exists := iu.hashes[records[i].ID]

		if !exists || previousHash != currentHash {
			changed = append(changed, records[i])
			iu.hashes[records[i].ID] = currentHash
		}
	}

	// Check for deleted documents
	currentIDs := make(map[string]bool)
	for _, r := range records {
		currentIDs[r.ID] = true
	}

	for id := range iu.hashes {
		if !currentIDs[id] {
			deleted = append(deleted, id)
			delete(iu.hashes, id)
		}
	}

	iu.lastSync = time.Now()
	return changed, deleted
}

// UpdateHashes updates the stored hashes
func (iu *IncrementalUpdater) UpdateHashes(records []Record) {
	iu.mu.Lock()
	defer iu.mu.Unlock()

	for i := range records {
		hash := ComputeHash(&records[i])
		iu.hashes[records[i].ID] = hash
	}
	iu.lastSync = time.Now()
}

// GetHashes returns all stored hashes
func (iu *IncrementalUpdater) GetHashes() map[string]string {
	iu.mu.RLock()
	defer iu.mu.RUnlock()

	hashes := make(map[string]string)
	for k, v := range iu.hashes {
		hashes[k] = v
	}
	return hashes
}

// LastSyncTime returns the last sync time
func (iu *IncrementalUpdater) LastSyncTime() time.Time {
	iu.mu.RLock()
	defer iu.mu.RUnlock()
	return iu.lastSync
}

// Clear clears all stored hashes
func (iu *IncrementalUpdater) Clear() {
	iu.mu.Lock()
	defer iu.mu.Unlock()
	iu.hashes = make(map[string]string)
	iu.lastSync = time.Now()
}

// IncrementalAddDocuments adds documents with change detection
func (kb *KnowledgeBase) IncrementalAddDocuments(ctx context.Context, updater *IncrementalUpdater, records []Record) error {
	if updater == nil {
		return fmt.Errorf("incremental updater is required")
	}

	// Detect changes
	changed, deleted := updater.GetChanges(records)

	// Delete removed documents
	if len(deleted) > 0 {
		for _, id := range deleted {
			_ = kb.DeleteDocument(ctx, id)
		}
	}

	// Add/update changed documents
	if len(changed) > 0 {
		opts := &UpsertOptions{
			Namespace: kb.namespace,
		}

		if err := kb.handler.Upsert(ctx, changed, opts); err != nil {
			return fmt.Errorf("failed to upsert changed documents: %w", err)
		}

		// Update search engine if available
		if kb.searcher != nil {
			searchDocs := make([]interface{}, 0, len(changed))
			for _, record := range changed {
				searchDocs = append(searchDocs, map[string]interface{}{
					"id":       record.ID,
					"title":    record.Title,
					"content":  record.Content,
					"source":   record.Source,
					"metadata": record.Metadata,
				})
			}
		}
	}

	// Update hashes for next sync
	updater.UpdateHashes(records)

	return nil
}

// SyncStats represents synchronization statistics
type SyncStats struct {
	Added      int
	Updated    int
	Deleted    int
	Unchanged  int
	TotalTime  time.Duration
	LastSyncAt time.Time
}

// GetSyncStats returns synchronization statistics
func (iu *IncrementalUpdater) GetSyncStats(records []Record) SyncStats {
	iu.mu.RLock()
	defer iu.mu.RUnlock()

	stats := SyncStats{
		LastSyncAt: iu.lastSync,
	}

	currentIDs := make(map[string]bool)
	for _, r := range records {
		currentIDs[r.ID] = true
		hash := ComputeHash(&r)
		previousHash, exists := iu.hashes[r.ID]

		if !exists {
			stats.Added++
		} else if previousHash != hash {
			stats.Updated++
		} else {
			stats.Unchanged++
		}
	}

	for id := range iu.hashes {
		if !currentIDs[id] {
			stats.Deleted++
		}
	}

	return stats
}

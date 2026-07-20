package knowledge_search

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	llmsearch "github.com/LingByte/lingllm/search"
)

// SearchIndexManager holds one Bleve full-text index per knowledge namespace (collection).
type SearchIndexManager struct {
	rootDir string
	timeout time.Duration
	mu      sync.RWMutex
	indexes map[string]llmsearch.Engine
}

func NewSearchIndexManager(rootDir string, queryTimeoutSec int) (*SearchIndexManager, error) {
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		rootDir = "./data/knowledge-search"
	}
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, fmt.Errorf("create search index dir: %w", err)
	}
	timeout := time.Duration(queryTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &SearchIndexManager{
		rootDir: rootDir,
		timeout: timeout,
		indexes: make(map[string]llmsearch.Engine),
	}, nil
}

func (m *SearchIndexManager) indexPath(namespace string) string {
	safe := sanitizeIndexName(namespace)
	return filepath.Join(m.rootDir, safe)
}

func sanitizeIndexName(namespace string) string {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range ns {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "default"
	}
	return out
}

// ForNamespace opens or creates the Bleve index for a knowledge namespace collection.
func (m *SearchIndexManager) ForNamespace(namespace string) (llmsearch.Engine, error) {
	if m == nil {
		return nil, fmt.Errorf("search index manager is not initialized")
	}
	key := strings.TrimSpace(namespace)
	if key == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	m.mu.RLock()
	if eng, ok := m.indexes[key]; ok {
		m.mu.RUnlock()
		return eng, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.indexes == nil {
		m.indexes = make(map[string]llmsearch.Engine)
	}
	if eng, ok := m.indexes[key]; ok {
		return eng, nil
	}

	cfg := llmsearch.Config{
		IndexPath:           m.indexPath(key),
		DefaultAnalyzer:     "standard",
		DefaultSearchFields: []string{"title", "content", "body"},
		QueryTimeout:        m.timeout,
		BatchSize:           200,
	}
	mapping := llmsearch.BuildIndexMapping("standard")
	eng, err := llmsearch.New(cfg, mapping)
	if err != nil {
		return nil, fmt.Errorf("open search index for %q: %w", key, err)
	}
	m.indexes[key] = eng
	return eng, nil
}

// DeleteNamespace closes and removes the on-disk index for a collection.
func (m *SearchIndexManager) DeleteNamespace(namespace string) error {
	if m == nil {
		return nil
	}
	key := strings.TrimSpace(namespace)
	if key == "" {
		return nil
	}

	m.mu.Lock()
	eng, ok := m.indexes[key]
	if ok {
		delete(m.indexes, key)
	}
	m.mu.Unlock()

	if ok && eng != nil {
		_ = eng.Close()
	}
	path := m.indexPath(key)
	if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Close closes all open indexes.
func (m *SearchIndexManager) Close() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var first error
	for key, eng := range m.indexes {
		if eng != nil {
			if err := eng.Close(); err != nil && first == nil {
				first = err
			}
		}
		delete(m.indexes, key)
	}
	return first
}

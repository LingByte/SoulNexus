package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
)

/*
Knowledge Base Factory Pattern Usage Guide

This package provides a unified knowledge base interface supporting multiple knowledge base providers:
- aliyun: Aliyun Bailian Knowledge Base
- milvus/zilliz: Milvus Vector Database
- qdrant: Qdrant Vector Database
- elasticsearch: Elasticsearch Full-Text Search Engine
- pinecone: Pinecone Vector Database

Usage:

1. Basic usage (read from config file):
   ```go
   import "github.com/LingByte/SoulNexus/pkg/knowledge"

   // Read provider and config from config file
   kb, err := knowledge.GetKnowledgeBaseByProvider("aliyun", config)
   if err != nil {
       return err
   }

   // Use knowledge base
   results, err := kb.Search(ctx, "knowledge_key", knowledge.SearchOptions{
       Query: "query text",
       TopK:  10,
   })
   ```

2. Read config from database (recommended):
   ```go
   import (
       "github.com/LingByte/SoulNexus/internal/models"
       "github.com/LingByte/SoulNexus/pkg/knowledge"
   )

   // 1. Get knowledge base info from database
   k, err := models.GetKnowledge(db, "knowledge_key")
   if err != nil {
       return err
   }

   // 2. Parse config
   var config map[string]interface{}
   if k.Config != "" {
       json.Unmarshal([]byte(k.Config), &config)
   }

   // 3. Create knowledge base instance (will be cached automatically)
   kb, err := knowledge.GetKnowledgeBaseByProvider(k.Provider, config)
   if err != nil {
       return err
   }

   // 4. Use knowledge base
   results, err := kb.Search(ctx, k.KnowledgeKey, knowledge.SearchOptions{
       Query: "query text",
       TopK:  10,
   })
   ```

3. Create knowledge base index:
   ```go
   indexName, err := kb.CreateIndex(ctx, "my_index", map[string]interface{}{
       "collection_name": "my_collection",
       "dimension": 768,
   })
   ```

4. Upload document:
   ```go
   file, header, _ := c.Request.FormFile("file")
   err := kb.UploadDocument(ctx, "knowledge_key", file, header, map[string]interface{}{
       "metadata": map[string]string{"source": "upload"},
   })
   ```

5. Delete knowledge base:
   ```go
   err := kb.DeleteIndex(ctx, "knowledge_key")
   ```

Notes:
- Instances are automatically cached, same provider + same config will reuse the same instance
- Config information needs to provide corresponding parameters according to different providers
- All operations require context.Context, if nil will use context.Background()
*/

// manager knowledge base manager implementation
type manager struct {
	providers map[string]func(config map[string]interface{}) (KnowledgeBase, error)
	instances map[string]KnowledgeBase // cached instances: key = provider + config hash
	mu        sync.RWMutex
}

var defaultManager = &manager{
	providers: make(map[string]func(config map[string]interface{}) (KnowledgeBase, error)),
	instances: make(map[string]KnowledgeBase),
}

// GetManager gets the default knowledge base manager
func GetManager() Manager {
	return defaultManager
}

// getConfigHash generates hash value of config for cache key
func getConfigHash(config map[string]interface{}) string {
	if config == nil {
		return "nil"
	}
	// Serialize config to JSON and calculate hash
	jsonBytes, err := json.Marshal(config)
	if err != nil {
		// If serialization fails, use empty string
		return "error"
	}
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}

// GetKnowledgeBase gets knowledge base instance by provider type (with cache)
func (m *manager) GetKnowledgeBase(provider string, config map[string]interface{}) (KnowledgeBase, error) {
	m.mu.RLock()
	factory, exists := m.providers[provider]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf(ErrUnsupportedProvider, provider)
	}

	// Generate cache key: provider + config hash
	cacheKey := provider + ":" + getConfigHash(config)

	// Check cache
	m.mu.RLock()
	if instance, cached := m.instances[cacheKey]; cached {
		m.mu.RUnlock()
		return instance, nil
	}
	m.mu.RUnlock()

	// Create new instance
	instance, err := factory(config)
	if err != nil {
		return nil, err
	}

	// Cache instance
	m.mu.Lock()
	m.instances[cacheKey] = instance
	m.mu.Unlock()

	return instance, nil
}

// ClearCache clears all cached instances
func (m *manager) ClearCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instances = make(map[string]KnowledgeBase)
}

// RemoveCachedInstance removes cached instance for specified provider and config
func (m *manager) RemoveCachedInstance(provider string, config map[string]interface{}) {
	cacheKey := provider + ":" + getConfigHash(config)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.instances, cacheKey)
}

// RegisterProvider registers a new knowledge base provider
func (m *manager) RegisterProvider(name string, factory func(config map[string]interface{}) (KnowledgeBase, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[name] = factory
}

// ListProviders lists all registered providers
func (m *manager) ListProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]string, 0, len(m.providers))
	for name := range m.providers {
		providers = append(providers, name)
	}
	return providers
}

// GetKnowledgeBaseByProvider convenience method: gets knowledge base instance by provider name
func GetKnowledgeBaseByProvider(provider string, config map[string]interface{}) (KnowledgeBase, error) {
	return defaultManager.GetKnowledgeBase(provider, config)
}

// RegisterKnowledgeBaseProvider convenience method: registers knowledge base provider
func RegisterKnowledgeBaseProvider(name string, factory func(config map[string]interface{}) (KnowledgeBase, error)) {
	defaultManager.RegisterProvider(name, factory)
}

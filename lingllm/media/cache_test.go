package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalMediaCache_BuildKey(t *testing.T) {
	cache := &LocalMediaCache{}

	tests := []struct {
		name   string
		params []string
	}{
		{
			name:   "single param",
			params: []string{"test"},
		},
		{
			name:   "multiple params",
			params: []string{"test", "data"},
		},
		{
			name:   "empty params",
			params: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cache.BuildKey(tt.params...)
			// Just verify it returns a non-empty string
			if got == "" {
				t.Error("BuildKey() should return non-empty string")
			}
			// Verify it's a valid hex string (MD5 is 32 hex chars)
			if len(got) != 32 {
				t.Errorf("BuildKey() returned %d chars, want 32", len(got))
			}
		})
	}
}

func TestLocalMediaCache_StoreAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &LocalMediaCache{
		Disabled:  false,
		CacheRoot: tmpDir,
	}

	key := "test_key"
	data := []byte("test data")

	// Test Store
	err := cache.Store(key, data)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Test Get
	got, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(got) != string(data) {
		t.Errorf("Get() = %v, want %v", string(got), string(data))
	}
}

func TestLocalMediaCache_GetNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &LocalMediaCache{
		Disabled:  false,
		CacheRoot: tmpDir,
	}

	_, err := cache.Get("nonexistent")
	if err == nil {
		t.Error("Get() should return error for nonexistent key")
	}
}

func TestLocalMediaCache_Disabled(t *testing.T) {
	cache := &LocalMediaCache{
		Disabled: true,
	}

	err := cache.Store("key", []byte("data"))
	if err != nil {
		t.Errorf("Store() on disabled cache should return nil, got %v", err)
	}

	_, err = cache.Get("key")
	if err == nil {
		t.Error("Get() on disabled cache should return error")
	}
}

func TestLocalMediaCache_StoreExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &LocalMediaCache{
		Disabled:  false,
		CacheRoot: tmpDir,
	}

	key := "test_key"
	data := []byte("test data")

	// Store first time
	err := cache.Store(key, data)
	if err != nil {
		t.Fatalf("First Store() error = %v", err)
	}

	// Store again (should succeed, overwrite)
	err = cache.Store(key, []byte("new data"))
	if err != nil {
		t.Fatalf("Second Store() error = %v", err)
	}
}

func TestLocalMediaCache_StoreDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &LocalMediaCache{
		Disabled:  false,
		CacheRoot: tmpDir,
	}

	// Create a directory with the key name
	keyPath := filepath.Join(tmpDir, "dir_key")
	os.Mkdir(keyPath, 0755)

	// Try to store with the same key
	err := cache.Store("dir_key", []byte("data"))
	if err == nil {
		t.Error("Store() should return error when key is a directory")
	}
}

func TestMediaCache_SingletonInstance(t *testing.T) {
	cache1 := MediaCache()
	cache2 := MediaCache()

	if cache1 != cache2 {
		t.Error("MediaCache() should return the same instance")
	}
}

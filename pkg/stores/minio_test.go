package stores

import (
	"os"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

func TestNewMinioStore(t *testing.T) {
	envKeys := []string{"MINIO_ENDPOINT", "MINIO_ACCESS_KEY", "MINIO_SECRET_KEY", "MINIO_BUCKET", "MINIO_USE_SSL", "MINIO_PUBLIC_BASE"}
	orig := make(map[string]string, len(envKeys))
	for _, k := range envKeys {
		orig[k] = os.Getenv(k)
	}
	defer func() {
		for _, k := range envKeys {
			if orig[k] != "" {
				os.Setenv(k, orig[k])
			} else {
				os.Unsetenv(k)
			}
		}
		utils.PurgeEnvCacheForTest()
	}()

	os.Setenv("MINIO_ENDPOINT", "localhost:9000")
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin")
	os.Setenv("MINIO_BUCKET", "test")
	os.Setenv("MINIO_USE_SSL", "true")
	os.Setenv("MINIO_PUBLIC_BASE", "https://cdn.example.com")

	store := NewMinioStore().(*MinioStore)
	if store.Endpoint != "localhost:9000" {
		t.Fatalf("Endpoint mismatch: %s", store.Endpoint)
	}
	if store.AccessKey != "minioadmin" {
		t.Fatalf("AccessKey mismatch: %s", store.AccessKey)
	}
	if store.SecretKey != "minioadmin" {
		t.Fatalf("SecretKey mismatch")
	}
	if store.Bucket != "test" {
		t.Fatalf("Bucket mismatch: %s", store.Bucket)
	}
	if !store.UseSSL {
		t.Fatalf("UseSSL should be true")
	}
	if store.BaseURL != "https://cdn.example.com" {
		t.Fatalf("BaseURL mismatch: %s", store.BaseURL)
	}
}

func TestNewMinioStore_Defaults(t *testing.T) {
	envKeys := []string{"MINIO_ENDPOINT", "MINIO_ACCESS_KEY", "MINIO_SECRET_KEY", "MINIO_BUCKET", "MINIO_USE_SSL", "MINIO_PUBLIC_BASE"}
	orig := make(map[string]string, len(envKeys))
	for _, k := range envKeys {
		orig[k] = os.Getenv(k)
	}
	defer func() {
		for _, k := range envKeys {
			if orig[k] != "" {
				os.Setenv(k, orig[k])
			} else {
				os.Unsetenv(k)
			}
		}
		utils.PurgeEnvCacheForTest()
	}()

	// Explicitly unset to test defaults
	os.Unsetenv("MINIO_ENDPOINT")
	os.Unsetenv("MINIO_ACCESS_KEY")
	os.Unsetenv("MINIO_SECRET_KEY")
	os.Unsetenv("MINIO_BUCKET")
	os.Unsetenv("MINIO_USE_SSL")
	os.Unsetenv("MINIO_PUBLIC_BASE")
	utils.PurgeEnvCacheForTest()

	store := NewMinioStore().(*MinioStore)
	if store.UseSSL {
		t.Fatalf("UseSSL should default to false")
	}
	if store.BaseURL != "" {
		t.Fatalf("BaseURL should default to empty, got: %s", store.BaseURL)
	}
}

func TestNewMinioStore_UseSSL_WithOne(t *testing.T) {
	orig := os.Getenv("MINIO_USE_SSL")
	origBase := os.Getenv("MINIO_PUBLIC_BASE")
	defer func() {
		if orig != "" {
			os.Setenv("MINIO_USE_SSL", orig)
		} else {
			os.Unsetenv("MINIO_USE_SSL")
		}
		if origBase != "" {
			os.Setenv("MINIO_PUBLIC_BASE", origBase)
		} else {
			os.Unsetenv("MINIO_PUBLIC_BASE")
		}
		utils.PurgeEnvCacheForTest()
	}()

	os.Setenv("MINIO_USE_SSL", "1")
	os.Unsetenv("MINIO_PUBLIC_BASE")
	utils.PurgeEnvCacheForTest()
	store := NewMinioStore().(*MinioStore)
	if !store.UseSSL {
		t.Fatalf("UseSSL should be true when MINIO_USE_SSL=1")
	}
}

func TestMinioStore_PublicURL(t *testing.T) {
	// With custom BaseURL
	m1 := &MinioStore{
		Endpoint: "localhost:9000",
		Bucket:   "test-bucket",
		BaseURL:  "https://cdn.example.com",
		UseSSL:   false,
	}
	if url := m1.PublicURL("path/file.txt"); url != "https://cdn.example.com/test-bucket/path/file.txt" {
		t.Fatalf("PublicURL with BaseURL got %q", url)
	}

	// Without BaseURL, UseSSL=false
	m2 := &MinioStore{
		Endpoint: "localhost:9000",
		Bucket:   "test-bucket",
		UseSSL:   false,
	}
	url2 := m2.PublicURL("file.txt")
	if !strings.HasPrefix(url2, "http://") {
		t.Fatalf("PublicURL without SSL should use http://, got: %s", url2)
	}
	if !strings.Contains(url2, "test-bucket/file.txt") {
		t.Fatalf("PublicURL should contain bucket and key, got: %s", url2)
	}

	// Without BaseURL, UseSSL=true
	m3 := &MinioStore{
		Endpoint: "s3.example.com",
		Bucket:   "test-bucket",
		UseSSL:   true,
	}
	url3 := m3.PublicURL("file.txt")
	if !strings.HasPrefix(url3, "https://") {
		t.Fatalf("PublicURL with SSL should use https://, got: %s", url3)
	}
	if !strings.Contains(url3, "test-bucket/file.txt") {
		t.Fatalf("PublicURL should contain bucket and key, got: %s", url3)
	}
}

func TestMinioStore_SetBaseURL(t *testing.T) {
	m := &MinioStore{}
	m.SetBaseURL("https://new-cdn.example.com")
	if m.BaseURL != "https://new-cdn.example.com" {
		t.Fatalf("SetBaseURL failed, got: %s", m.BaseURL)
	}

	m.SetBaseURL("")
	if m.BaseURL != "" {
		t.Fatalf("SetBaseURL to empty failed, got: %s", m.BaseURL)
	}
}

func TestMinioStore_ImplementsStore(t *testing.T) {
	assertImplementsStore(t, &MinioStore{})
}

package stores

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// s3IntegrationEnvReady checks if S3 integration test environment variables are set
func s3IntegrationEnvReady() bool {
	_, a := os.LookupEnv("S3_ACCESS_KEY_ID")
	_, b := os.LookupEnv("S3_SECRET_ACCESS_KEY")
	_, c := os.LookupEnv("S3_BUCKET")
	_, d := os.LookupEnv("S3_REGION")
	return a && b && c && d
}

// s3MustEnv gets environment variable or panics if not set
func s3MustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		panic("missing env: " + k)
	}
	return v
}

// s3ReadAll reads all data from reader
func s3ReadAll(t *testing.T, r io.Reader) string {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("readAll err: %v", err)
	}
	return string(b)
}

// TestIntegration_S3_CRUD performs integration test for S3 storage
// Required environment variables: S3_ACCESS_KEY_ID, S3_SECRET_ACCESS_KEY, S3_BUCKET, S3_REGION
// Optional: S3_ENDPOINT, S3_USE_PATH_STYLE, S3_DOMAIN
// This test will actually upload/query/download/delete a temporary key: test-go-lingecho/<timestamp>.txt
func TestIntegration_S3_CRUD(t *testing.T) {
	if !s3IntegrationEnvReady() {
		t.Skip("skip integration test: S3_* env not fully set")
	}

	usePathStyle := strings.ToLower(os.Getenv("S3_USE_PATH_STYLE")) == "true" || os.Getenv("S3_USE_PATH_STYLE") == "1"
	store := &S3Store{
		Region:          s3MustEnv("S3_REGION"),
		AccessKeyID:     s3MustEnv("S3_ACCESS_KEY_ID"),
		AccessKeySecret: s3MustEnv("S3_SECRET_ACCESS_KEY"),
		BucketName:      s3MustEnv("S3_BUCKET"),
		Endpoint:        os.Getenv("S3_ENDPOINT"),
		UsePathStyle:    usePathStyle,
		Domain:          os.Getenv("S3_DOMAIN"),
	}

	key := "test-go-lingecho/" + time.Now().Format("20060102-150405") + ".txt"
	content := "hello-from-s3-integration-test"

	// 1) Write
	if err := store.Write(key, bytes.NewBufferString(content)); err != nil {
		t.Fatalf("Write err: %v", err)
	}

	// 2) Exists should be true
	ok, err := store.Exists(key)
	if err != nil {
		t.Fatalf("Exists err: %v", err)
	}
	if !ok {
		t.Fatalf("Exists returned false after write")
	}

	// 3) Read
	rc, size, err := store.Read(key)
	if err != nil {
		t.Fatalf("Read err: %v", err)
	}
	defer rc.Close()

	if size != int64(len(content)) {
		t.Fatalf("Read size mismatch, got: %d, want: %d", size, len(content))
	}

	data := s3ReadAll(t, rc)
	if data != content {
		t.Fatalf("Read content mismatch, got: %q, want: %q", data, content)
	}

	// 4) PublicURL
	u := store.PublicURL(key)
	if u == "" {
		t.Fatalf("PublicURL returned empty string")
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		t.Fatalf("PublicURL should start with http:// or https://, got: %s", u)
	}
	if !strings.Contains(u, key) {
		t.Fatalf("PublicURL should contain key, got: %s", u)
	}

	// 5) Delete
	if err := store.Delete(key); err != nil {
		t.Fatalf("Delete err: %v", err)
	}

	// 6) Exists should be false
	ok, err = store.Exists(key)
	if err != nil {
		t.Fatalf("Exists after delete err: %v", err)
	}
	if ok {
		t.Fatalf("Exists should be false after delete")
	}
}

// TestS3Store_PublicURL tests PublicURL generation with different configurations
func TestS3Store_PublicURL(t *testing.T) {
	// Test with custom domain
	store1 := &S3Store{
		Region:     "us-east-1",
		BucketName: "test-bucket",
		Domain:     "https://cdn.example.com",
	}
	u1 := store1.PublicURL("path/to/file.txt")
	expected1 := "https://cdn.example.com/path/to/file.txt"
	if u1 != expected1 {
		t.Fatalf("PublicURL with custom domain got %q, want %q", u1, expected1)
	}

	// Test with domain without protocol
	store2 := &S3Store{
		Region:     "us-east-1",
		BucketName: "test-bucket",
		Domain:     "cdn.example.com",
	}
	u2 := store2.PublicURL("file.txt")
	if !strings.HasPrefix(u2, "https://") {
		t.Fatalf("PublicURL should add https:// prefix, got: %s", u2)
	}

	// Test with custom endpoint and path style
	store3 := &S3Store{
		Region:       "us-east-1",
		BucketName:   "test-bucket",
		Endpoint:     "https://s3.example.com",
		UsePathStyle: true,
	}
	u3 := store3.PublicURL("path/file.txt")
	expected3 := "https://s3.example.com/test-bucket/path/file.txt"
	if u3 != expected3 {
		t.Fatalf("PublicURL with custom endpoint and path style got %q, want %q", u3, expected3)
	}

	// Test with custom endpoint without path style
	store4 := &S3Store{
		Region:       "us-east-1",
		BucketName:   "test-bucket",
		Endpoint:     "https://s3.example.com",
		UsePathStyle: false,
	}
	u4 := store4.PublicURL("file.txt")
	expected4 := "https://s3.example.com/file.txt"
	if u4 != expected4 {
		t.Fatalf("PublicURL with custom endpoint without path style got %q, want %q", u4, expected4)
	}

	// Test standard S3 URL
	store5 := &S3Store{
		Region:     "us-east-1",
		BucketName: "test-bucket",
	}
	u5 := store5.PublicURL("path/file.txt")
	expected5 := "https://test-bucket.s3.us-east-1.amazonaws.com/path/file.txt"
	if u5 != expected5 {
		t.Fatalf("PublicURL standard S3 got %q, want %q", u5, expected5)
	}
}

// TestNewS3Store tests S3 store creation from environment variables
func TestNewS3Store(t *testing.T) {
	// Save original env
	origRegion := os.Getenv("S3_REGION")
	origKeyID := os.Getenv("S3_ACCESS_KEY_ID")
	origSecret := os.Getenv("S3_SECRET_ACCESS_KEY")
	origBucket := os.Getenv("S3_BUCKET")
	origEndpoint := os.Getenv("S3_ENDPOINT")
	origPathStyle := os.Getenv("S3_USE_PATH_STYLE")
	origDomain := os.Getenv("S3_DOMAIN")

	// Restore env after test
	defer func() {
		if origRegion != "" {
			os.Setenv("S3_REGION", origRegion)
		} else {
			os.Unsetenv("S3_REGION")
		}
		if origKeyID != "" {
			os.Setenv("S3_ACCESS_KEY_ID", origKeyID)
		} else {
			os.Unsetenv("S3_ACCESS_KEY_ID")
		}
		if origSecret != "" {
			os.Setenv("S3_SECRET_ACCESS_KEY", origSecret)
		} else {
			os.Unsetenv("S3_SECRET_ACCESS_KEY")
		}
		if origBucket != "" {
			os.Setenv("S3_BUCKET", origBucket)
		} else {
			os.Unsetenv("S3_BUCKET")
		}
		if origEndpoint != "" {
			os.Setenv("S3_ENDPOINT", origEndpoint)
		} else {
			os.Unsetenv("S3_ENDPOINT")
		}
		if origPathStyle != "" {
			os.Setenv("S3_USE_PATH_STYLE", origPathStyle)
		} else {
			os.Unsetenv("S3_USE_PATH_STYLE")
		}
		if origDomain != "" {
			os.Setenv("S3_DOMAIN", origDomain)
		} else {
			os.Unsetenv("S3_DOMAIN")
		}
	}()

	// Test with path style enabled
	os.Setenv("S3_REGION", "us-east-1")
	os.Setenv("S3_ACCESS_KEY_ID", "test-key-id")
	os.Setenv("S3_SECRET_ACCESS_KEY", "test-secret")
	os.Setenv("S3_BUCKET", "test-bucket")
	os.Setenv("S3_ENDPOINT", "https://s3.example.com")
	os.Setenv("S3_USE_PATH_STYLE", "true")
	os.Setenv("S3_DOMAIN", "https://cdn.example.com")

	store := NewS3Store().(*S3Store)
	if store.Region != "us-east-1" {
		t.Fatalf("Region mismatch: %s", store.Region)
	}
	if store.AccessKeyID != "test-key-id" {
		t.Fatalf("AccessKeyID mismatch: %s", store.AccessKeyID)
	}
	if store.AccessKeySecret != "test-secret" {
		t.Fatalf("AccessKeySecret mismatch")
	}
	if store.BucketName != "test-bucket" {
		t.Fatalf("BucketName mismatch: %s", store.BucketName)
	}
	if store.Endpoint != "https://s3.example.com" {
		t.Fatalf("Endpoint mismatch: %s", store.Endpoint)
	}
	if !store.UsePathStyle {
		t.Fatalf("UsePathStyle should be true")
	}
	if store.Domain != "https://cdn.example.com" {
		t.Fatalf("Domain mismatch: %s", store.Domain)
	}

	// Test with path style disabled
	os.Setenv("S3_USE_PATH_STYLE", "false")
	store2 := NewS3Store().(*S3Store)
	if store2.UsePathStyle {
		t.Fatalf("UsePathStyle should be false")
	}

	// Test with "1" as true
	os.Setenv("S3_USE_PATH_STYLE", "1")
	store3 := NewS3Store().(*S3Store)
	if !store3.UsePathStyle {
		t.Fatalf("UsePathStyle should be true when S3_USE_PATH_STYLE=1")
	}
}

// TestS3Store_Exists_NonExistent tests Exists for non-existent keys
func TestS3Store_Exists_NonExistent(t *testing.T) {
	if !s3IntegrationEnvReady() {
		t.Skip("skip integration test: S3_* env not fully set")
	}

	store := &S3Store{
		Region:          s3MustEnv("S3_REGION"),
		AccessKeyID:     s3MustEnv("S3_ACCESS_KEY_ID"),
		AccessKeySecret: s3MustEnv("S3_SECRET_ACCESS_KEY"),
		BucketName:      s3MustEnv("S3_BUCKET"),
		Endpoint:        os.Getenv("S3_ENDPOINT"),
		UsePathStyle:    strings.ToLower(os.Getenv("S3_USE_PATH_STYLE")) == "true" || os.Getenv("S3_USE_PATH_STYLE") == "1",
	}

	// Test with a key that definitely doesn't exist
	nonExistentKey := "test-go-lingecho/non-existent-" + time.Now().Format("20060102-150405-999999") + ".txt"
	ok, err := store.Exists(nonExistentKey)
	if err != nil {
		t.Fatalf("Exists for non-existent key should not return error, got: %v", err)
	}
	if ok {
		t.Fatalf("Exists should return false for non-existent key")
	}
}

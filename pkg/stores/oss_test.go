package stores

import (
	"os"
	"strings"
	"testing"
)

func TestNewOSSStore(t *testing.T) {
	envKeys := []string{"OSS_ACCESS_KEY_ID", "OSS_ACCESS_KEY_SECRET", "OSS_ENDPOINT", "OSS_BUCKET_NAME"}
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
	}()

	os.Setenv("OSS_ACCESS_KEY_ID", "test-key-id")
	os.Setenv("OSS_ACCESS_KEY_SECRET", "test-secret")
	os.Setenv("OSS_ENDPOINT", "oss-cn-hangzhou.aliyuncs.com")
	os.Setenv("OSS_BUCKET_NAME", "test-bucket")

	store := NewOSSStore().(*OSSStore)
	if store.AccessKeyID != "test-key-id" {
		t.Fatalf("AccessKeyID mismatch: %s", store.AccessKeyID)
	}
	if store.AccessKeySecret != "test-secret" {
		t.Fatalf("AccessKeySecret mismatch")
	}
	if store.Endpoint != "oss-cn-hangzhou.aliyuncs.com" {
		t.Fatalf("Endpoint mismatch: %s", store.Endpoint)
	}
	if store.BucketName != "test-bucket" {
		t.Fatalf("BucketName mismatch: %s", store.BucketName)
	}
	if store.baseURL != "" {
		t.Fatalf("baseURL should default to empty, got: %s", store.baseURL)
	}
}

func TestOSSStore_PublicURL(t *testing.T) {
	// With baseURL set via SetBaseURL
	o1 := &OSSStore{
		Endpoint:   "oss-cn-hangzhou.aliyuncs.com",
		BucketName: "test-bucket",
		baseURL:    "https://cdn.example.com",
	}
	if url := o1.PublicURL("path/file.txt"); url != "https://cdn.example.com/path/file.txt" {
		t.Fatalf("PublicURL with baseURL got %q", url)
	}

	// Without baseURL, standard OSS format
	o2 := &OSSStore{
		Endpoint:   "oss-cn-hangzhou.aliyuncs.com",
		BucketName: "test-bucket",
	}
	url2 := o2.PublicURL("file.txt")
	expected := "https://test-bucket.oss-cn-hangzhou.aliyuncs.com/file.txt"
	if url2 != expected {
		t.Fatalf("PublicURL standard OSS got %q, want %q", url2, expected)
	}

	// Endpoint with https:// prefix
	o3 := &OSSStore{
		Endpoint:   "https://oss-cn-beijing.aliyuncs.com",
		BucketName: "my-bucket",
	}
	url3 := o3.PublicURL("data/file.bin")
	expected3 := "https://my-bucket.oss-cn-beijing.aliyuncs.com/data/file.bin"
	if url3 != expected3 {
		t.Fatalf("PublicURL with https:// endpoint got %q, want %q", url3, expected3)
	}

	// Endpoint with http:// prefix
	o4 := &OSSStore{
		Endpoint:   "http://oss-cn-shanghai.aliyuncs.com",
		BucketName: "bucket-4",
	}
	url4 := o4.PublicURL("key")
	expected4 := "https://bucket-4.oss-cn-shanghai.aliyuncs.com/key"
	if url4 != expected4 {
		t.Fatalf("PublicURL with http:// endpoint got %q, want %q", url4, expected4)
	}
}

func TestOSSStore_SetBaseURL(t *testing.T) {
	o := &OSSStore{}
	o.SetBaseURL("https://my-cdn.com")
	if o.baseURL != "https://my-cdn.com" {
		t.Fatalf("SetBaseURL failed, got: %s", o.baseURL)
	}

	// Empty baseURL
	o.SetBaseURL("")
	if o.baseURL != "" {
		t.Fatalf("SetBaseURL to empty failed, got: %s", o.baseURL)
	}
}

func TestOSSStore_Delete_CredentialNotConfigured(t *testing.T) {
	o := &OSSStore{}
	err := o.Delete("test-key")
	if err == nil {
		t.Fatal("Delete should return error when credentials not configured")
	}
	if !strings.Contains(err.Error(), "credentials not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOSSStore_Exists_CredentialNotConfigured(t *testing.T) {
	o := &OSSStore{}
	ok, err := o.Exists("test-key")
	if err == nil {
		t.Fatal("Exists should return error when credentials not configured")
	}
	if ok {
		t.Fatal("Exists should return false when credentials not configured")
	}
}

func TestOSSStore_Read_CredentialNotConfigured(t *testing.T) {
	o := &OSSStore{}
	rc, size, err := o.Read("test-key")
	if err == nil {
		t.Fatal("Read should return error when credentials not configured")
	}
	if rc != nil {
		t.Fatal("ReadCloser should be nil when credentials not configured")
	}
	if size != 0 {
		t.Fatalf("size should be 0, got: %d", size)
	}
}

func TestOSSStore_Write_CredentialNotConfigured(t *testing.T) {
	o := &OSSStore{}
	err := o.Write("test-key", strings.NewReader("data"))
	if err == nil {
		t.Fatal("Write should return error when credentials not configured")
	}
}

func TestOSSStore_ImplementsStore(t *testing.T) {
	assertImplementsStore(t, &OSSStore{})
}

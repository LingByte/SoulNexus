package stores

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCosStore(t *testing.T) {
	store := NewCosStore().(*CosStore)

	// 检查配置
	if store.SecretID == "" || store.SecretKey == "" || store.Region == "" || store.BucketName == "" {
		t.Skipf("skipping cos test: missing credentials - SecretID=%v, SecretKey=%v, Region=%v, BucketName=%v",
			store.SecretID != "", store.SecretKey != "", store.Region != "", store.BucketName != "")
	}

	assert.NotNil(t, store)

	bucketName := store.BucketName
	fname := "test.txt"
	testContent := "hello world"

	t.Logf("Testing with bucket: %s, region: %s", bucketName, store.Region)

	// 测试文件不存在
	ok, err := store.Exists(fname)
	if err != nil {
		t.Fatalf("Exists check failed: %v", err)
	}
	assert.False(t, ok)

	// 测试写入文件
	err = store.Write(fname, bytes.NewReader([]byte(testContent)))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// 测试文件存在
	ok, err = store.Exists(fname)
	if err != nil {
		t.Fatalf("Exists check after write failed: %v", err)
	}
	assert.True(t, ok)

	// 测试读取文件
	r, size, err := store.Read(fname)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	assert.True(t, size > 0)

	content, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	assert.Equal(t, testContent, string(content))
	r.Close()

	// 测试公开URL
	url := store.PublicURL(fname)
	assert.True(t, strings.Contains(url, bucketName))
	assert.True(t, strings.Contains(url, fname))
	t.Logf("Public URL: %s", url)

	// 测试删除文件
	err = store.Delete(fname)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 验证文件已删除
	ok, err = store.Exists(fname)
	if err != nil {
		t.Fatalf("Exists check after delete failed: %v", err)
	}
	assert.False(t, ok)
	assert.NoError(t, err)

	// 验证文件已删除
	ok, err = store.Exists(fname)
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestCosStore_PublicURL(t *testing.T) {
	c := &CosStore{
		BucketName: "my-bucket",
		Region:     "ap-guangzhou",
	}
	url := c.PublicURL("path/to/file.txt")
	expected := "https://my-bucket.cos.ap-guangzhou.myqcloud.com/path/to/file.txt"
	if url != expected {
		t.Fatalf("PublicURL got %q, want %q", url, expected)
	}
}

func TestInitCos(t *testing.T) {
	c := &CosStore{
		BucketName: "test-bucket",
		Region:     "ap-guangzhou",
	}
	// InitCos with explicit bucket name
	client := InitCos("override-bucket", c)
	if client == nil {
		t.Fatal("InitCos returned nil client")
	}

	// InitCos with empty bucket name (falls back to CosStore.BucketName)
	client2 := InitCos("", c)
	if client2 == nil {
		t.Fatal("InitCos with empty bucket returned nil client")
	}
}

func TestCosStore_Delete_CredentialNotConfigured(t *testing.T) {
	c := &CosStore{}
	err := c.Delete("test-key")
	if err == nil {
		t.Fatal("Delete should return error when credentials not configured")
	}
	if !strings.Contains(err.Error(), "credentials not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCosStore_Exists_CredentialNotConfigured(t *testing.T) {
	c := &CosStore{}
	ok, err := c.Exists("test-key")
	if err == nil {
		t.Fatal("Exists should return error when credentials not configured")
	}
	if ok {
		t.Fatal("Exists should return false when credentials not configured")
	}
}

func TestCosStore_Read_CredentialNotConfigured(t *testing.T) {
	c := &CosStore{}
	rc, size, err := c.Read("test-key")
	if err == nil {
		t.Fatal("Read should return error when credentials not configured")
	}
	if rc != nil {
		t.Fatal("ReadCloser should be nil")
	}
	if size != 0 {
		t.Fatalf("size should be 0, got: %d", size)
	}
}

func TestCosStore_Write_CredentialNotConfigured(t *testing.T) {
	c := &CosStore{}
	err := c.Write("test-key", strings.NewReader("data"))
	if err == nil {
		t.Fatal("Write should return error when credentials not configured")
	}
}

func TestCosStore_ImplementsStore(t *testing.T) {
	assertImplementsStore(t, &CosStore{})
}

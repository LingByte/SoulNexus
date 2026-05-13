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

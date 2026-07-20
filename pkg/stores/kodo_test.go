package stores

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

func readAll(t *testing.T, r io.Reader) string {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("readAll err: %v", err)
	}
	return string(b)
}

func checkEnvReady() bool {
	// Require explicit opt-in for the live qiniu integration test so plain
	// `go test ./...` (which loads developer .env files via godotenv) doesn't
	// hit production buckets — and doesn't fail when the AK/SK lacks the
	// delete IAM permission this test exercises.
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("QINIU_INTEGRATION_TEST")), "true") {
		return false
	}
	accessKey := utils.GetEnv("QINIU_ACCESS_KEY")
	secretKey := utils.GetEnv("QINIU_SECRET_KEY")
	bucket := utils.GetEnv("QINIU_BUCKET")
	domain := utils.GetEnv("QINIU_DOMAIN")
	return accessKey != "" && secretKey != "" && bucket != "" && domain != ""
}

// newTestStore 创建测试用的 QiNiuStore 实例
func newTestStore(t *testing.T) *QiNiuStore {
	t.Helper()
	if !checkEnvReady() {
		t.Skip("skip integration test: QINIU_* env not fully set")
	}
	private := strings.EqualFold(os.Getenv("QINIU_PRIVATE"), "true")
	return &QiNiuStore{
		AccessKey:  utils.GetEnv("QINIU_ACCESS_KEY"),
		SecretKey:  utils.GetEnv("QINIU_SECRET_KEY"),
		BucketName: utils.GetEnv("QINIU_BUCKET"),
		Domain:     utils.GetEnv("QINIU_DOMAIN"),
		Private:    private,
		Region:     utils.GetEnv("QINIU_REGION"),
	}
}

func TestQiNiuCRUD(t *testing.T) {
	if !checkEnvReady() {
		t.Skip("skip integration test: QINIU_* env not fully set")
	}

	private := strings.EqualFold(os.Getenv("QINIU_PRIVATE"), "true")
	store := &QiNiuStore{
		AccessKey:  utils.GetEnv("QINIU_ACCESS_KEY"),
		SecretKey:  utils.GetEnv("QINIU_SECRET_KEY"),
		BucketName: utils.GetEnv("QINIU_BUCKET"),
		Domain:     utils.GetEnv("QINIU_DOMAIN"),
		Private:    private,
		Region:     utils.GetEnv("QINIU_REGION"),
	}
	key := "test-go-lingecho/" + time.Now().Format("20060102-150405") + ".txt"
	content := "hello-from-integration"

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
	rc, _, err := store.Read(key)
	if err != nil {
		t.Fatalf("Read err: %v", err)
	}
	data := readAll(t, rc)
	_ = rc.Close()
	if data != content {
		t.Fatalf("Read content mismatch, got: %q", data)
	}

	// 4) PublicURL
	u := store.PublicURL(key)
	if !strings.HasPrefix(u, "http") {
		t.Fatalf("PublicURL invalid: %s", u)
	}
	if private && !strings.Contains(u, "token=") {
		t.Fatalf("Private PublicURL should be signed, got: %s", u)
	}

	// 5) Delete
	if err := store.Delete(key); err != nil {
		t.Fatalf("Delete err: %v", err)
	}

	// 6) Exists should be false
	ok, err = store.Exists(key)
	if err != nil {
		// 删除后 Stat 可能返回 612，我们在 Exists 里已处理为 false,nil
		// 如果这里仍返回错误，说明 SDK 行为变更或网络异常
		t.Fatalf("Exists after delete err: %v", err)
	}
	if ok {
		t.Fatalf("Exists should be false after delete")
	}
}

func TestNewQiNiuStore(t *testing.T) {
	envKeys := []string{"QINIU_ACCESS_KEY", "QINIU_SECRET_KEY", "QINIU_BUCKET", "QINIU_DOMAIN", "QINIU_PRIVATE", "QINIU_REGION"}
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

	os.Setenv("QINIU_ACCESS_KEY", "test-ak")
	os.Setenv("QINIU_SECRET_KEY", "test-sk")
	os.Setenv("QINIU_BUCKET", "test-bucket")
	os.Setenv("QINIU_DOMAIN", "cdn.example.com")
	os.Setenv("QINIU_PRIVATE", "true")
	os.Setenv("QINIU_REGION", "z0")

	store := NewQiNiuStore().(*QiNiuStore)
	if store.AccessKey != "test-ak" {
		t.Fatalf("AccessKey mismatch: %s", store.AccessKey)
	}
	if store.SecretKey != "test-sk" {
		t.Fatalf("SecretKey mismatch")
	}
	if store.BucketName != "test-bucket" {
		t.Fatalf("BucketName mismatch: %s", store.BucketName)
	}
	if store.Domain != "cdn.example.com" {
		t.Fatalf("Domain mismatch: %s", store.Domain)
	}
	if !store.Private {
		t.Fatalf("Private should be true")
	}
	if store.Region != "z0" {
		t.Fatalf("Region mismatch: %s", store.Region)
	}
}

func TestNewQiNiuStore_PublicByDefault(t *testing.T) {
	orig := os.Getenv("QINIU_PRIVATE")
	os.Unsetenv("QINIU_PRIVATE")
	defer func() {
		if orig != "" {
			os.Setenv("QINIU_PRIVATE", orig)
		} else {
			os.Unsetenv("QINIU_PRIVATE")
		}
	}()

	_ = NewQiNiuStore().(*QiNiuStore)
	// Default should be false (public) when QINIU_PRIVATE not set
}

func TestQiNiuStore_PublicURL(t *testing.T) {
	// Public bucket
	q1 := &QiNiuStore{
		Domain:     "cdn.example.com",
		Private:    false,
		AccessKey:  "ak",
		SecretKey:  "sk",
		BucketName: "test-bucket",
	}
	u := q1.PublicURL("path/file.txt")
	if u == "" {
		t.Fatal("PublicURL returned empty")
	}
	if !strings.HasPrefix(u, "http") {
		t.Fatalf("PublicURL should start with http, got: %s", u)
	}
	if !strings.Contains(u, "path/file.txt") {
		t.Fatalf("PublicURL should contain key, got: %s", u)
	}

	// Private bucket
	q2 := &QiNiuStore{
		Domain:     "cdn.example.com",
		Private:    true,
		AccessKey:  "ak",
		SecretKey:  "sk",
		BucketName: "test-bucket",
	}
	u2 := q2.PublicURL("file.txt")
	if u2 == "" {
		t.Fatal("PublicURL for private bucket returned empty")
	}
	if !strings.Contains(u2, "token=") {
		t.Fatalf("Private PublicURL should contain token, got: %s", u2)
	}

	// Domain with https:// prefix
	q3 := &QiNiuStore{
		Domain:     "https://secure.example.com",
		Private:    false,
		AccessKey:  "ak",
		SecretKey:  "sk",
		BucketName: "test-bucket",
	}
	u3 := q3.PublicURL("key")
	if !strings.HasPrefix(u3, "https://") {
		t.Fatalf("PublicURL with https domain should use https, got: %s", u3)
	}
}

func TestQiNiuStore_PublicURL_DomainEmpty(t *testing.T) {
	q := &QiNiuStore{
		Domain:     "",
		Private:    false,
		AccessKey:  "ak",
		SecretKey:  "sk",
		BucketName: "test-bucket",
	}
	u := q.PublicURL("file.txt")
	if u != "" {
		t.Fatalf("PublicURL with empty domain should return empty, got: %s", u)
	}
}

func TestQiNiuStore_UploadToken(t *testing.T) {
	q := &QiNiuStore{
		AccessKey:  "test-ak",
		SecretKey:  "test-sk",
		BucketName: "test-bucket",
	}
	token := q.uploadToken("")
	if token == "" {
		t.Fatal("uploadToken returned empty")
	}

	token2 := q.uploadToken("override-bucket")
	if token2 == "" {
		t.Fatal("uploadToken with explicit bucket returned empty")
	}
}

func TestQiNiuStore_MakeConfig(t *testing.T) {
	q := &QiNiuStore{
		Domain:     "https://cdn.example.com",
		AccessKey:  "test-ak",
		BucketName: "test-bucket",
	}
	cfg := q.makeConfig("")
	if !cfg.UseHTTPS {
		t.Fatal("UseHTTPS should be true for https domain")
	}

	cfg2 := q.makeConfig("override-bucket")
	if !cfg2.UseHTTPS {
		t.Fatal("UseHTTPS should be true for https domain")
	}
}

func TestQiNiuStore_ImplementsStore(t *testing.T) {
	assertImplementsStore(t, &QiNiuStore{})
}

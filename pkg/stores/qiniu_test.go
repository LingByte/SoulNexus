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

package stores

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func tosIntegrationEnvReady() bool {
	_, a := os.LookupEnv("TOS_ACCESS_KEY_ID")
	_, b := os.LookupEnv("TOS_SECRET_ACCESS_KEY")
	_, c := os.LookupEnv("TOS_BUCKET")
	_, d := os.LookupEnv("TOS_REGION")
	_, e := os.LookupEnv("TOS_ENDPOINT")
	return a && b && c && d && e
}

func tosMustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		panic("missing env: " + k)
	}
	return v
}

func TestIntegration_Tos_CRUD(t *testing.T) {
	if !tosIntegrationEnvReady() {
		t.Skip("skip integration test: TOS_* env not fully set")
	}

	store := &TosStore{
		Endpoint:        tosMustEnv("TOS_ENDPOINT"),
		Region:          tosMustEnv("TOS_REGION"),
		AccessKeyID:     tosMustEnv("TOS_ACCESS_KEY_ID"),
		AccessKeySecret: tosMustEnv("TOS_SECRET_ACCESS_KEY"),
		BucketName:      tosMustEnv("TOS_BUCKET"),
		Domain:          os.Getenv("TOS_DOMAIN"),
	}

	key := "test-go-lingecho/" + time.Now().Format("20060102-150405") + ".txt"
	content := "hello-from-tos-integration-test"

	if err := store.Write(key, bytes.NewBufferString(content)); err != nil {
		t.Fatalf("Write err: %v", err)
	}

	ok, err := store.Exists(key)
	if err != nil {
		t.Fatalf("Exists err: %v", err)
	}
	if !ok {
		t.Fatalf("Exists returned false after write")
	}

	rc, size, err := store.Read(key)
	if err != nil {
		t.Fatalf("Read err: %v", err)
	}
	defer rc.Close()

	if size != int64(len(content)) {
		t.Fatalf("Read size mismatch, got: %d, want: %d", size, len(content))
	}

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll err: %v", err)
	}
	if string(data) != content {
		t.Fatalf("Read content mismatch, got: %q, want: %q", string(data), content)
	}

	u := store.PublicURL(key)
	if u == "" {
		t.Fatalf("PublicURL returned empty string")
	}
	if !strings.Contains(u, key) {
		t.Fatalf("PublicURL should contain key, got: %s", u)
	}

	if err := store.Delete(key); err != nil {
		t.Fatalf("Delete err: %v", err)
	}

	ok, err = store.Exists(key)
	if err != nil {
		t.Fatalf("Exists after delete err: %v", err)
	}
	if ok {
		t.Fatalf("Exists should be false after delete")
	}
}

func TestTosStore_PublicURL(t *testing.T) {
	store1 := &TosStore{
		Endpoint:   "tos-cn-beijing.volces.com",
		BucketName: "test-bucket",
		Domain:     "https://cdn.example.com",
	}
	u1 := store1.PublicURL("path/to/file.txt")
	expected1 := "https://cdn.example.com/path/to/file.txt"
	if u1 != expected1 {
		t.Fatalf("PublicURL with custom domain got %q, want %q", u1, expected1)
	}

	store2 := &TosStore{
		Endpoint:   "https://tos-cn-beijing.volces.com",
		BucketName: "test-bucket",
	}
	u2 := store2.PublicURL("file.txt")
	expected2 := "https://test-bucket.tos-cn-beijing.volces.com/file.txt"
	if u2 != expected2 {
		t.Fatalf("PublicURL standard TOS got %q, want %q", u2, expected2)
	}
}

func TestNewTosStore(t *testing.T) {
	envKeys := []string{"TOS_ENDPOINT", "TOS_REGION", "TOS_ACCESS_KEY_ID", "TOS_SECRET_ACCESS_KEY", "TOS_BUCKET", "TOS_DOMAIN"}
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

	os.Setenv("TOS_ENDPOINT", "tos-cn-beijing.volces.com")
	os.Setenv("TOS_REGION", "cn-beijing")
	os.Setenv("TOS_ACCESS_KEY_ID", "test-key-id")
	os.Setenv("TOS_SECRET_ACCESS_KEY", "test-secret")
	os.Setenv("TOS_BUCKET", "test-bucket")
	os.Setenv("TOS_DOMAIN", "https://cdn.example.com")

	store := NewTosStore().(*TosStore)
	if store.Endpoint != "tos-cn-beijing.volces.com" {
		t.Fatalf("Endpoint mismatch: %s", store.Endpoint)
	}
	if store.Region != "cn-beijing" {
		t.Fatalf("Region mismatch: %s", store.Region)
	}
	if store.BucketName != "test-bucket" {
		t.Fatalf("BucketName mismatch: %s", store.BucketName)
	}
	if store.Domain != "https://cdn.example.com" {
		t.Fatalf("Domain mismatch: %s", store.Domain)
	}
}

func TestTosStore_PublicURL_DomainWithoutProtocol(t *testing.T) {
	s := &TosStore{
		Endpoint:   "tos-cn-beijing.volces.com",
		BucketName: "test-bucket",
		Domain:     "files.mycdn.com",
	}
	url := s.PublicURL("data/file.bin")
	expected := "https://files.mycdn.com/data/file.bin"
	if url != expected {
		t.Fatalf("PublicURL with domain without protocol got %q, want %q", url, expected)
	}
}

func TestTosStore_PublicURL_LeadingSlashTrim(t *testing.T) {
	s := &TosStore{
		Endpoint:   "tos-cn-beijing.volces.com",
		BucketName: "test-bucket",
		Domain:     "https://cdn.example.com",
	}
	url := s.PublicURL("/path/to/file.txt")
	expected := "https://cdn.example.com/path/to/file.txt"
	if url != expected {
		t.Fatalf("PublicURL should trim leading slash, got %q", url)
	}
}

func TestTosStore_ImplementsStore(t *testing.T) {
	assertImplementsStore(t, &TosStore{})
}

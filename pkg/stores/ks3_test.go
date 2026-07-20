package stores

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func ks3IntegrationEnvReady() bool {
	_, a := os.LookupEnv("KS3_ACCESS_KEY_ID")
	_, b := os.LookupEnv("KS3_SECRET_ACCESS_KEY")
	_, c := os.LookupEnv("KS3_BUCKET")
	_, d := os.LookupEnv("KS3_REGION")
	_, e := os.LookupEnv("KS3_ENDPOINT")
	return a && b && c && d && e
}

func ks3MustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		panic("missing env: " + k)
	}
	return v
}

func TestIntegration_Ks3_CRUD(t *testing.T) {
	if !ks3IntegrationEnvReady() {
		t.Skip("skip integration test: KS3_* env not fully set")
	}

	store := &Ks3Store{
		Endpoint:        ks3MustEnv("KS3_ENDPOINT"),
		Region:          ks3MustEnv("KS3_REGION"),
		AccessKeyID:     ks3MustEnv("KS3_ACCESS_KEY_ID"),
		AccessKeySecret: ks3MustEnv("KS3_SECRET_ACCESS_KEY"),
		BucketName:      ks3MustEnv("KS3_BUCKET"),
		Domain:          os.Getenv("KS3_DOMAIN"),
	}

	key := "test-go-lingecho/" + time.Now().Format("20060102-150405") + ".txt"
	content := "hello-from-ks3-integration-test"

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

func TestKs3Store_PublicURL(t *testing.T) {
	store1 := &Ks3Store{
		Endpoint:   "ks3-cn-beijing.ksyuncs.com",
		BucketName: "test-bucket",
		Domain:     "https://cdn.example.com",
	}
	u1 := store1.PublicURL("path/to/file.txt")
	expected1 := "https://cdn.example.com/path/to/file.txt"
	if u1 != expected1 {
		t.Fatalf("PublicURL with custom domain got %q, want %q", u1, expected1)
	}

	store2 := &Ks3Store{
		Endpoint:   "https://ks3-cn-beijing.ksyuncs.com",
		BucketName: "test-bucket",
	}
	u2 := store2.PublicURL("file.txt")
	expected2 := "https://test-bucket.ks3-cn-beijing.ksyuncs.com/file.txt"
	if u2 != expected2 {
		t.Fatalf("PublicURL standard KS3 got %q, want %q", u2, expected2)
	}
}

func TestNewKs3Store(t *testing.T) {
	envKeys := []string{"KS3_ENDPOINT", "KS3_REGION", "KS3_ACCESS_KEY_ID", "KS3_SECRET_ACCESS_KEY", "KS3_BUCKET", "KS3_DOMAIN"}
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

	os.Setenv("KS3_ENDPOINT", "ks3-cn-beijing.ksyuncs.com")
	os.Setenv("KS3_REGION", "BEIJING")
	os.Setenv("KS3_ACCESS_KEY_ID", "test-key-id")
	os.Setenv("KS3_SECRET_ACCESS_KEY", "test-secret")
	os.Setenv("KS3_BUCKET", "test-bucket")
	os.Setenv("KS3_DOMAIN", "https://cdn.example.com")

	store := NewKs3Store().(*Ks3Store)
	if store.Endpoint != "ks3-cn-beijing.ksyuncs.com" {
		t.Fatalf("Endpoint mismatch: %s", store.Endpoint)
	}
	if store.Region != "BEIJING" {
		t.Fatalf("Region mismatch: %s", store.Region)
	}
	if store.BucketName != "test-bucket" {
		t.Fatalf("BucketName mismatch: %s", store.BucketName)
	}
	if store.Domain != "https://cdn.example.com" {
		t.Fatalf("Domain mismatch: %s", store.Domain)
	}
}

func TestKs3Store_PublicURL_DomainWithoutProtocol(t *testing.T) {
	s := &Ks3Store{
		Endpoint:   "ks3-cn-beijing.ksyuncs.com",
		BucketName: "test-bucket",
		Domain:     "files.mycdn.com",
	}
	url := s.PublicURL("data/file.bin")
	expected := "https://files.mycdn.com/data/file.bin"
	if url != expected {
		t.Fatalf("PublicURL with domain without protocol got %q, want %q", url, expected)
	}
}

func TestKs3Store_PublicURL_LeadingSlashTrim(t *testing.T) {
	s := &Ks3Store{
		Endpoint:   "ks3-cn-beijing.ksyuncs.com",
		BucketName: "test-bucket",
		Domain:     "https://cdn.example.com",
	}
	url := s.PublicURL("/path/to/file.txt")
	expected := "https://cdn.example.com/path/to/file.txt"
	if url != expected {
		t.Fatalf("PublicURL should trim leading slash, got %q", url)
	}
}

func TestCheckKs3Connectivity_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := CheckKs3Connectivity(ctx, "ks3-cn-beijing.ksyuncs.com", "BEIJING", "key", "secret", "bucket")
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestCheckKs3Connectivity_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()

	err := CheckKs3Connectivity(ctx, "ks3-cn-beijing.ksyuncs.com", "BEIJING", "key", "secret", "bucket")
	if err == nil {
		t.Fatal("expected timeout error with invalid credentials, got nil")
	}
}

func TestKs3Store_ImplementsStore(t *testing.T) {
	assertImplementsStore(t, &Ks3Store{})
}

package stores

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func obsIntegrationEnvReady() bool {
	_, a := os.LookupEnv("OBS_ACCESS_KEY_ID")
	_, b := os.LookupEnv("OBS_SECRET_ACCESS_KEY")
	_, c := os.LookupEnv("OBS_BUCKET")
	_, d := os.LookupEnv("OBS_REGION")
	_, e := os.LookupEnv("OBS_ENDPOINT")
	return a && b && c && d && e
}

func obsMustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		panic("missing env: " + k)
	}
	return v
}

func TestIntegration_Obs_CRUD(t *testing.T) {
	if !obsIntegrationEnvReady() {
		t.Skip("skip integration test: OBS_* env not fully set")
	}

	store := &ObsStore{
		Endpoint:        obsMustEnv("OBS_ENDPOINT"),
		Region:          obsMustEnv("OBS_REGION"),
		AccessKeyID:     obsMustEnv("OBS_ACCESS_KEY_ID"),
		AccessKeySecret: obsMustEnv("OBS_SECRET_ACCESS_KEY"),
		BucketName:      obsMustEnv("OBS_BUCKET"),
		ProxyDomain:     os.Getenv("OBS_PROXY_DOMAIN"),
	}

	key := "test-go-lingecho/" + time.Now().Format("20060102-150405") + ".txt"
	content := "hello-from-obs-integration-test"

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

func TestObsStore_PublicURL(t *testing.T) {
	store1 := &ObsStore{
		Endpoint:    "https://obs.cn-north-4.myhuaweicloud.com",
		BucketName:  "test-bucket",
		ProxyDomain: "https://cdn.example.com",
	}
	u1 := store1.PublicURL("path/to/file.txt")
	expected1 := "https://cdn.example.com/path/to/file.txt"
	if u1 != expected1 {
		t.Fatalf("PublicURL with proxy domain got %q, want %q", u1, expected1)
	}

	store2 := &ObsStore{
		Endpoint:   "https://obs.cn-north-4.myhuaweicloud.com",
		BucketName: "test-bucket",
	}
	u2 := store2.PublicURL("file.txt")
	expected2 := "https://obs.cn-north-4.myhuaweicloud.com/test-bucket/file.txt"
	if u2 != expected2 {
		t.Fatalf("PublicURL standard OBS got %q, want %q", u2, expected2)
	}
}

func TestNewObsStore(t *testing.T) {
	envKeys := []string{"OBS_ENDPOINT", "OBS_REGION", "OBS_ACCESS_KEY_ID", "OBS_SECRET_ACCESS_KEY", "OBS_BUCKET", "OBS_PROXY_DOMAIN"}
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

	os.Setenv("OBS_ENDPOINT", "https://obs.cn-north-4.myhuaweicloud.com")
	os.Setenv("OBS_REGION", "cn-north-4")
	os.Setenv("OBS_ACCESS_KEY_ID", "test-key-id")
	os.Setenv("OBS_SECRET_ACCESS_KEY", "test-secret")
	os.Setenv("OBS_BUCKET", "test-bucket")
	os.Setenv("OBS_PROXY_DOMAIN", "https://cdn.example.com")

	store := NewObsStore().(*ObsStore)
	if store.Endpoint != "https://obs.cn-north-4.myhuaweicloud.com" {
		t.Fatalf("Endpoint mismatch: %s", store.Endpoint)
	}
	if store.Region != "cn-north-4" {
		t.Fatalf("Region mismatch: %s", store.Region)
	}
	if store.BucketName != "test-bucket" {
		t.Fatalf("BucketName mismatch: %s", store.BucketName)
	}
	if store.ProxyDomain != "https://cdn.example.com" {
		t.Fatalf("ProxyDomain mismatch: %s", store.ProxyDomain)
	}
}

func TestObsStore_PublicURL_LeadingSlashTrim(t *testing.T) {
	o := &ObsStore{
		Endpoint:    "https://obs.cn-north-4.myhuaweicloud.com",
		BucketName:  "test-bucket",
		ProxyDomain: "https://cdn.example.com",
	}
	url := o.PublicURL("/path/to/file.txt")
	expected := "https://cdn.example.com/path/to/file.txt"
	if url != expected {
		t.Fatalf("PublicURL should trim leading slash, got %q", url)
	}
}

func TestObsStore_ImplementsStore(t *testing.T) {
	assertImplementsStore(t, &ObsStore{})
}

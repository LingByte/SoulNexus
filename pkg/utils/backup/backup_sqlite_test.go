package backup

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/constants"
)

// memStore is an in-memory stores.Store for backup tests.
type memStore struct {
	files map[string][]byte
}

func (m *memStore) Read(key string) (io.ReadCloser, int64, error) { return nil, 0, os.ErrNotExist }
func (m *memStore) Write(key string, r io.Reader) error {
	if m.files == nil {
		m.files = make(map[string][]byte)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.files[key] = b
	return nil
}
func (m *memStore) Delete(key string) error {
	delete(m.files, key)
	return nil
}
func (m *memStore) Exists(key string) (bool, error) {
	_, ok := m.files[key]
	return ok, nil
}
func (m *memStore) PublicURL(key string) string { return "mem://" + key }

func TestBackupSQLiteToStore(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	if err := os.WriteFile(dbPath, []byte("sqlite-bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(constants.ENV_BACKUP_PATH, dir)

	store := &memStore{}
	if err := backupSQLiteToStore("file:"+dbPath, store); err != nil {
		t.Fatal(err)
	}
	if len(store.files) != 1 {
		t.Fatalf("files %v", store.files)
	}
	for k, v := range store.files {
		if !strings.HasPrefix(k, "backup/sys_backup_") || !strings.HasSuffix(k, ".db") {
			t.Fatalf("key %q", k)
		}
		if !bytes.Equal(v, []byte("sqlite-bytes")) {
			t.Fatalf("content %q", v)
		}
	}
	mpath := manifestPath()
	data, err := os.ReadFile(mpath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "sqlite-bytes") && !strings.Contains(string(data), "backup/") {
		t.Fatalf("manifest %s", data)
	}
}

func TestBackupSQLiteToStoreMissingFile(t *testing.T) {
	t.Setenv(constants.ENV_BACKUP_PATH, t.TempDir())
	err := backupSQLiteToStore("file:"+filepath.Join(t.TempDir(), "nope.db"), &memStore{})
	if err == nil {
		t.Fatal("expected read error")
	}
}

func TestBackupSQLiteDSNTrim(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared.db")
	_ = os.WriteFile(dbPath, []byte("x"), 0644)
	t.Setenv(constants.ENV_BACKUP_PATH, dir)
	store := &memStore{}
	if err := backupSQLiteToStore("file:"+dbPath+"?cache=shared", store); err != nil {
		t.Fatal(err)
	}
	if len(store.files) != 1 {
		t.Fatal("expected one backup file")
	}
}

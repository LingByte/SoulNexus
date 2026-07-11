package petproject_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/petproject"
)

func TestPublishZip_stagesExtractsAndDeletesZip(t *testing.T) {
	dir := t.TempDir()
	store := &stagingTestStore{root: dir}

	zipBytes, err := petproject.PackZip(map[string]string{
		"manifest.json": `{"name":"zip-pet"}`,
		"pet.js":        "window.__PET_SPRITE__={}",
	})
	if err != nil {
		t.Fatal(err)
	}

	prefix := petproject.DefaultPrefix("pkg-1")
	files, err := petproject.PublishZip(store, "pkg-1", zipBytes, prefix)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	staging := filepath.Join(dir, filepath.FromSlash(petproject.StagingZipKey("pkg-1")))
	if _, err := os.Stat(staging); !os.IsNotExist(err) {
		t.Fatalf("staging zip should be deleted, stat err=%v", err)
	}

	manifest := filepath.Join(dir, filepath.FromSlash(prefix), "manifest.json")
	if _, err := os.Stat(manifest); err != nil {
		t.Fatalf("extracted manifest missing: %v", err)
	}
}

type stagingTestStore struct{ root string }

func (s *stagingTestStore) Write(key string, r io.Reader) error {
	fname := filepath.Join(s.root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(fname), 0o755); err != nil {
		return err
	}
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (s *stagingTestStore) Read(key string) (io.ReadCloser, int64, error) {
	fname := filepath.Join(s.root, filepath.FromSlash(key))
	st, err := os.Stat(fname)
	if err != nil {
		return nil, 0, err
	}
	f, err := os.Open(fname)
	if err != nil {
		return nil, 0, err
	}
	return f, st.Size(), nil
}

func (s *stagingTestStore) Delete(key string) error {
	fname := filepath.Join(s.root, filepath.FromSlash(key))
	err := os.Remove(fname)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

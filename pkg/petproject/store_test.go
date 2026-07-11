package petproject_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/petproject"
)

func TestWriteAndReadFilesLocal(t *testing.T) {
	dir := t.TempDir()
	store := &localTestStore{root: dir}
	prefix := petproject.DefaultPrefix("test-template-id")
	files := map[string]string{
		"manifest.json": `{"name":"test"}`,
		"pet.js":        "console.log('ok')",
	}

	if err := petproject.WriteFiles(store, prefix, files); err != nil {
		t.Fatal(err)
	}

	for name, want := range files {
		path := filepath.Join(dir, prefix, name)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("missing file %s: %v", path, err)
		}
		if string(b) != want {
			t.Fatalf("file %s: got %q want %q", name, b, want)
		}
	}

	got, err := petproject.ReadFiles(store, prefix, []string{"manifest.json", "pet.js"})
	if err != nil {
		t.Fatal(err)
	}
	if got["pet.js"] != files["pet.js"] {
		t.Fatalf("read back pet.js: %q", got["pet.js"])
	}
}

type localTestStore struct{ root string }

func (l *localTestStore) Write(key string, r io.Reader) error {
	fname := filepath.Join(l.root, filepath.FromSlash(key))
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

func (l *localTestStore) Read(key string) (io.ReadCloser, int64, error) {
	fname := filepath.Join(l.root, filepath.FromSlash(key))
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

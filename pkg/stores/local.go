package stores

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

var UploadDir = "./uploads"

type LocalStore struct {
	Root       string
	NewDirPerm os.FileMode
}

// Delete implements Store.
func (l *LocalStore) Delete(key string) error {
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return err
	}

	fname := filepath.Clean(filepath.Join(root, key))
	if !strings.HasPrefix(fname, root) {
		return ErrInvalidPath
	}
	return os.Remove(fname)
}

// Exists implements Store.
func (l *LocalStore) Exists(key string) (bool, error) {
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return false, err
	}

	fname := filepath.Clean(filepath.Join(root, key))
	if !strings.HasPrefix(fname, root) {
		return false, ErrInvalidPath
	}
	_, err = os.Stat(fname)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Read implements Store.
func (l *LocalStore) Read(key string) (io.ReadCloser, int64, error) {
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return nil, 0, err
	}
	fname := filepath.Clean(filepath.Join(root, key))
	if !strings.HasPrefix(fname, root) {
		return nil, 0, ErrInvalidPath
	}
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

// Write implements Store.
func (l *LocalStore) Write(key string, r io.Reader) error {
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return err
	}

	fname := filepath.Clean(filepath.Join(root, key))
	if !strings.HasPrefix(fname, root) {
		return ErrInvalidPath
	}
	dir := filepath.Dir(fname)
	err = os.MkdirAll(dir, l.NewDirPerm)
	if err != nil {
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

func (l *LocalStore) PublicURL(key string) string {
	mediaPrefix := strings.TrimSuffix(l.Root, "/")
	key = strings.TrimPrefix(key, "/")
	relativePath := path.Join("/", mediaPrefix, key)
	return relativePath
}

func NewLocalStore() Store {
	uploadDir := utils.GetEnv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = UploadDir
	}
	s := &LocalStore{
		Root:       uploadDir,
		NewDirPerm: 0755,
	}
	return s
}

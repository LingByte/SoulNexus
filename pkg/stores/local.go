package stores

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

var UploadDir = pkgconst.DefaultUploadDir

type LocalStore struct {
	Root       string
	NewDirPerm os.FileMode
}

// Delete implements Store.
func (l *LocalStore) Delete(key string) error {
	fname, err := l.resolveKey(key)
	if err != nil {
		return err
	}
	// RemoveAll clears a regular file or a mistaken directory at this key.
	return os.RemoveAll(fname)
}

// Exists implements Store.
func (l *LocalStore) Exists(key string) (bool, error) {
	fname, err := l.resolveKey(key)
	if err != nil {
		return false, err
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
	fname, err := l.resolveKey(key)
	if err != nil {
		return nil, 0, err
	}
	st, err := os.Stat(fname)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, utils.ErrAttachmentNotExist
		}
		return nil, 0, err
	}
	f, err := os.Open(fname)
	if err != nil {
		return nil, 0, err
	}
	return f, st.Size(), nil
}

func (l *LocalStore) resolveKey(key string) (string, error) {
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return "", err
	}
	fname := filepath.Clean(filepath.Join(root, key))
	if !strings.HasPrefix(fname, root+string(filepath.Separator)) && fname != root {
		return "", ErrInvalidPath
	}
	return fname, nil
}

// Write implements Store.
func (l *LocalStore) Write(key string, r io.Reader) error {
	fname, err := l.resolveKey(key)
	if err != nil {
		return err
	}
	dir := filepath.Dir(fname)
	if err := os.MkdirAll(dir, l.NewDirPerm); err != nil {
		return err
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	// Drop any previous file or mistaken directory at this key, then write fresh bytes.
	_ = os.RemoveAll(fname)
	return os.WriteFile(fname, body, 0644)
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

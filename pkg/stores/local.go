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

// DefaultLocalURLPrefix is the HTTP path prefix used by LocalStore.PublicURL
// when LOCAL_MEDIA_URL_PREFIX is not set. Pair this with a matching
// http.FileServer mount (see cmd/voiceserver/http_listener.go) so the
// returned URLs are actually fetchable.
const DefaultLocalURLPrefix = "/media"

type LocalStore struct {
	Root       string
	NewDirPerm os.FileMode
	// URLPrefix is the HTTP path prefix served by the file server that
	// exposes Root. PublicURL returns URLPrefix + "/" + bucket + "/" + key.
	// Empty falls back to DefaultLocalURLPrefix.
	URLPrefix string
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

// PublicURL returns a path that the local /media file server can resolve.
// It is intentionally rooted at a public URL prefix (default /media) rather
// than at the on-disk Root, so the path stays stable even when UPLOAD_DIR is
// STORAGE_PUBLIC_BASE_URL when an absolute URL is required.
func (l *LocalStore) PublicURL(key string) string {
	prefix := strings.TrimSpace(l.URLPrefix)
	if prefix == "" {
		prefix = strings.TrimSpace(utils.GetEnv("LOCAL_MEDIA_URL_PREFIX"))
	}
	if prefix == "" {
		prefix = DefaultLocalURLPrefix
	}
	prefix = "/" + strings.Trim(prefix, "/")
	// LocalStore's Write/Read/Delete/Exists all ignore the bucket name and
	// resolve keys directly under Root, so PublicURL must follow suit.
	// Cloud backends overlay the bucket in their host name (e.g. S3) so this
	// asymmetry is invisible to callers as long as they go through PublicURL.
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	return path.Join(prefix, key)
}

func NewLocalStore() Store {
	uploadDir := utils.GetEnv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = UploadDir
	}
	s := &LocalStore{
		Root:       uploadDir,
		NewDirPerm: 0755,
		URLPrefix:  strings.TrimSpace(utils.GetEnv("LOCAL_MEDIA_URL_PREFIX")),
	}
	return s
}

package mediagen

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/google/uuid"
)

type Storage struct {
	cfg   Config
	http  *http.Client
	store stores.Store
}

func NewStorage(cfg Config) *Storage {
	return &Storage{
		cfg:   cfg,
		store: stores.Default(),
		http: &http.Client{
			Timeout: 3 * time.Minute,
		},
	}
}

func (s *Storage) DownloadAndSave(remoteURL, subdir, ext, contentType string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, remoteURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载远程资源失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("下载远程资源失败: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取远程资源失败: %w", err)
	}
	if ext == "" {
		ext = ".bin"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	fileName := uuid.NewString() + ext
	return s.SaveBytes(data, subdir, fileName, contentType)
}

func (s *Storage) SaveBytes(data []byte, subdir, fileName, _ string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("空文件无法保存")
	}
	cleanSub := filepath.ToSlash(filepath.Clean(strings.Trim(subdir, "/")))
	name := filepath.Base(fileName)
	if name == "" || name == "." || name == ".." {
		name = uuid.NewString() + ".bin"
	}
	key := filepath.ToSlash(filepath.Join(cleanSub, name))
	store := s.store
	if store == nil {
		store = stores.Default()
	}
	if err := store.Write(key, bytes.NewReader(data)); err != nil {
		return "", fmt.Errorf("对象存储写入失败: %w", err)
	}
	return key, nil
}

func (s *Storage) PublicURL(key string) string {
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	if key == "" {
		return ""
	}
	store := s.store
	if store == nil {
		store = stores.Default()
	}
	return strings.TrimSpace(store.PublicURL(key))
}

func (s *Storage) ReadBytes(key string) ([]byte, error) {
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	if key == "" {
		return nil, fmt.Errorf("empty storage key")
	}
	store := s.store
	if store == nil {
		store = stores.Default()
	}
	r, _, err := store.Read(key)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func ContentTypeForExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".mp4":
		return "video/mp4"
	default:
		return "image/png"
	}
}

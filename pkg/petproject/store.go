package petproject

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
)

const (
	StorageObject = "object"
	MetaVersion   = 2
	RootPrefix    = "pet-projects"
)

// Meta is persisted in js_templates.content — points at object storage, not source code.
type Meta struct {
	V       int      `json:"v"`
	Storage string   `json:"storage"`
	Prefix  string   `json:"prefix"`
	Entry   string   `json:"entry"`
	Files   []string `json:"files,omitempty"`
}

// DefaultEntry is the script served by loader.js when storage is object-backed.
const DefaultEntry = "pet.js"

func DefaultPrefix(templateID string) string {
	return path.Join(RootPrefix, templateID) + "/"
}

// ParseMeta detects object-storage pointer vs legacy inline content.
func ParseMeta(content string) (meta *Meta, legacyInline bool) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil, false
	}
	if !strings.HasPrefix(trimmed, "{") {
		return nil, true
	}
	var m Meta
	if err := json.Unmarshal([]byte(trimmed), &m); err != nil {
		return nil, true
	}
	if m.V == MetaVersion && m.Storage == StorageObject && m.Prefix != "" {
		if m.Entry == "" {
			m.Entry = DefaultEntry
		}
		return &m, false
	}
	// v1 inline project JSON { v:1, files: {...} }
	var inline struct {
		V     int               `json:"v"`
		Files map[string]string `json:"files"`
	}
	if err := json.Unmarshal([]byte(trimmed), &inline); err == nil && inline.V == 1 && len(inline.Files) > 0 {
		return nil, true
	}
	return nil, true
}

func MarshalMeta(prefix, entry string, files []string) (string, error) {
	if entry == "" {
		entry = DefaultEntry
	}
	b, err := json.Marshal(Meta{
		V:       MetaVersion,
		Storage: StorageObject,
		Prefix:  prefix,
		Entry:   entry,
		Files:   files,
	})
	return string(b), err
}

// ResolvePaths returns stored file list or legacy defaults.
func (m *Meta) ResolvePaths() []string {
	if len(m.Files) > 0 {
		return m.Files
	}
	return StandardPaths
}

func objectKey(prefix, rel string) (string, error) {
	return ObjectKey(prefix, rel)
}

// ObjectKey joins object-storage prefix with a relative project path.
func ObjectKey(prefix, rel string) (string, error) {
	rel = filepath.ToSlash(strings.TrimPrefix(strings.TrimSpace(rel), "/"))
	if rel == "" || strings.Contains(rel, "..") {
		return "", fmt.Errorf("invalid project path: %q", rel)
	}
	prefix = strings.TrimSuffix(prefix, "/")
	return prefix + "/" + rel, nil
}

func ValidateFiles(files map[string]string) error {
	if len(files) == 0 {
		return fmt.Errorf("project has no files")
	}
	for name := range files {
		if _, err := objectKey("x", name); err != nil {
			return err
		}
	}
	return nil
}

// WriteFiles uploads all project files under prefix.
func WriteFiles(w Writer, prefix string, files map[string]string) error {
	if err := ValidateFiles(files); err != nil {
		return err
	}
	for name, body := range files {
		key, err := objectKey(prefix, name)
		if err != nil {
			return err
		}
		data, err := DecodeFileContent(body)
		if err != nil {
			return fmt.Errorf("decode %s: %w", name, err)
		}
		if err := w.Write(key, bytes.NewReader(data)); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

// ReadFiles loads project files from object storage by prefix and relative paths.
func ReadFiles(r Reader, prefix string, paths []string) (map[string]string, error) {
	out := make(map[string]string, len(paths))
	for _, name := range paths {
		key, err := objectKey(prefix, name)
		if err != nil {
			return nil, err
		}
		rc, _, err := r.Read(key)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		out[name] = EncodeFileForAPI(name, b)
	}
	return out, nil
}

// EntryScript returns loader.js body for object-backed or legacy inline projects.
func EntryScript(r Reader, templateID, content string) (string, error) {
	meta, legacy := ParseMeta(content)
	if meta != nil {
		key, err := objectKey(meta.Prefix, meta.Entry)
		if err != nil {
			return "", err
		}
		rc, _, err := r.Read(key)
		if err != nil {
			return "", err
		}
		defer rc.Close()
		b, err := io.ReadAll(rc)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	if legacy {
		// Plain JS or v1 inline JSON
		trimmed := strings.TrimSpace(content)
		if strings.HasPrefix(trimmed, "{") {
			var inline struct {
				V     int               `json:"v"`
				Entry string            `json:"entry"`
				Files map[string]string `json:"files"`
			}
			if err := json.Unmarshal([]byte(trimmed), &inline); err == nil && len(inline.Files) > 0 {
				entry := inline.Entry
				if entry == "" {
					entry = DefaultEntry
				}
				if script, ok := inline.Files[entry]; ok {
					return script, nil
				}
			}
		}
		return content, nil
	}
	return "", fmt.Errorf("empty project for template %s", templateID)
}

// Writer reads stores.Store Write surface.
type Writer interface {
	Write(key string, r io.Reader) error
}

// Reader reads stores.Store Read surface.
type Reader interface {
	Read(key string) (io.ReadCloser, int64, error)
}

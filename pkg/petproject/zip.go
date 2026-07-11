package petproject

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// PackZip builds a .soulpet.zip from project files (paths use forward slashes).
func PackZip(files map[string]string) ([]byte, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no files to pack")
	}
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	// stable order: standard files first
	sortProjectPaths(names)

	for _, name := range names {
		body := files[name]
		name = filepath.ToSlash(strings.TrimPrefix(name, "/"))
		if name == "" || strings.Contains(name, "..") {
			return nil, fmt.Errorf("invalid path: %q", name)
		}
		data, err := DecodeFileContent(body)
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", name, err)
		}
		hdr := &zip.FileHeader{
			Name:   name,
			Method: zip.Deflate,
		}
		f, err := w.CreateHeader(hdr)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(data); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnpackZip reads a zip archive into project files map (API encoding for binaries).
func UnpackZip(raw []byte) (map[string]string, error) {
	r, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := normalizeZipEntryPath(f.Name)
		if shouldSkipZipEntry(name) {
			continue
		}
		if strings.Contains(name, "..") {
			return nil, fmt.Errorf("invalid zip path: %q", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		out[name] = EncodeFileForAPI(name, data)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("zip 包为空")
	}
	out = stripSingleZipRoot(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("zip 包为空")
	}
	return out, nil
}

func shouldSkipZipEntry(name string) bool {
	if name == "" {
		return true
	}
	if strings.HasPrefix(name, "__MACOSX/") {
		return true
	}
	if filepath.Base(name) == ".DS_Store" {
		return true
	}
	return false
}

// stripSingleZipRoot removes one shared top-level folder (e.g. 我的桌宠.soulpet/manifest.json → manifest.json).
func stripSingleZipRoot(files map[string]string) map[string]string {
	if len(files) == 0 {
		return files
	}
	var root string
	for path := range files {
		i := strings.Index(path, "/")
		if i <= 0 {
			return files
		}
		top := path[:i]
		if root == "" {
			root = top
		} else if root != top {
			return files
		}
	}
	if root == "" {
		return files
	}
	prefix := root + "/"
	out := make(map[string]string, len(files))
	for path, content := range files {
		rel := strings.TrimPrefix(path, prefix)
		if rel == "" || strings.Contains(rel, "..") {
			continue
		}
		out[rel] = content
	}
	if len(out) == 0 {
		return files
	}
	return out
}

func normalizeZipEntryPath(name string) string {
	name = filepath.ToSlash(name)
	name = strings.TrimPrefix(name, "./")
	return strings.TrimPrefix(name, "/")
}

func sortProjectPaths(paths []string) {
	priority := map[string]int{
		SoulpetYamlFile: 0,
		ManifestFile:    1,
		DefaultEntry:    2,
		"style.css":     3,
		"README.md":     4,
	}
	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			pi := priority[paths[i]]
			pj := priority[paths[j]]
			if pi > pj || (pi == pj && paths[i] > paths[j]) {
				paths[i], paths[j] = paths[j], paths[i]
			}
		}
	}
}

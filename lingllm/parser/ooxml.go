package parser

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func openZipFromRequest(req *ParseRequest) (*zip.Reader, string, error) {
	fileName := req.FileName
	if fileName == "" {
		fileName = req.Path
	}

	var b []byte
	switch {
	case len(req.Content) > 0:
		b = req.Content
	case req.Reader != nil:
		data, err := io.ReadAll(req.Reader)
		if err != nil {
			return nil, fileName, err
		}
		b = data
	case strings.TrimSpace(req.Path) != "":
		data, err := os.ReadFile(req.Path)
		if err != nil {
			return nil, fileName, err
		}
		b = data
	default:
		return nil, fileName, ErrEmptyInput
	}
	if len(b) == 0 {
		return nil, fileName, ErrEmptyInput
	}

	r, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, fileName, err
	}
	return r, fileName, nil
}

func readZipFile(z *zip.Reader, name string) ([]byte, bool, error) {
	name = filepath.ToSlash(strings.TrimPrefix(name, "/"))
	for _, f := range z.File {
		if filepath.ToSlash(f.Name) != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, false, err
		}
		defer rc.Close()
		b, err := io.ReadAll(rc)
		if err != nil {
			return nil, false, err
		}
		return b, true, nil
	}
	return nil, false, nil
}

func listZipFilesWithPrefix(z *zip.Reader, prefix string) []string {
	prefix = filepath.ToSlash(strings.TrimPrefix(prefix, "/"))
	out := make([]string, 0)
	for _, f := range z.File {
		name := filepath.ToSlash(f.Name)
		if strings.HasPrefix(name, prefix) {
			out = append(out, name)
		}
	}
	return out
}

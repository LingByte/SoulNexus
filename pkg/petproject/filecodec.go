package petproject

import (
	"encoding/base64"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const BinaryPrefix = "base64:"

var binaryExtensions = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {}, ".gif": {},
	".moc3": {}, ".wav": {}, ".mp3": {}, ".bin": {},
}

func IsBinaryPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := binaryExtensions[ext]
	return ok
}

func EncodeFileForAPI(path string, content []byte) string {
	if IsBinaryPath(path) || !utf8.Valid(content) {
		return BinaryPrefix + base64.StdEncoding.EncodeToString(content)
	}
	return string(content)
}

func DecodeFileContent(s string) ([]byte, error) {
	if strings.HasPrefix(s, BinaryPrefix) {
		return base64.StdEncoding.DecodeString(strings.TrimPrefix(s, BinaryPrefix))
	}
	return []byte(s), nil
}

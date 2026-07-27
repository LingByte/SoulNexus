package mediagen

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

var allowedImageContentTypes = map[string]struct{}{
	"image/png":                {},
	"image/jpeg":               {},
	"image/jpg":                {},
	"image/webp":               {},
	"image/gif":                {},
	"application/octet-stream": {},
}

type PreparedReference struct {
	DataURI      string
	PublicURL    string
	ImageHash    string
	ReferenceKey string
}

func PrepareReferenceImage(storage *Storage, file *multipart.FileHeader) (*PreparedReference, error) {
	if file == nil {
		return nil, ErrInvalidImage
	}
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("读取参考图失败: %w", err)
	}
	defer src.Close()
	bytes, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("读取参考图失败: %w", err)
	}
	if len(bytes) == 0 {
		return nil, ErrInvalidImage
	}
	contentType := detectImageContentType(bytes, file.Filename, file.Header.Get("Content-Type"))
	if !isAllowedImage(contentType, file.Filename, bytes) {
		return nil, ErrInvalidImage
	}
	ext := extensionForContentType(contentType, file.Filename)
	fileName := uuid.NewString() + ext
	key, err := storage.SaveBytes(bytes, storage.cfg.ReferencesSubdir, fileName, contentType)
	if err != nil {
		return nil, err
	}
	dataURI := "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(bytes)
	publicURL := storage.PublicURL(key)
	return &PreparedReference{
		DataURI:      dataURI,
		PublicURL:    publicURL,
		ReferenceKey: key,
	}, nil
}

// ImageURLForProvider prefers a durable public URL; falls back to data URI.
func (r *PreparedReference) ImageURLForProvider() string {
	if r == nil {
		return ""
	}
	if strings.TrimSpace(r.PublicURL) != "" {
		return strings.TrimSpace(r.PublicURL)
	}
	return r.DataURI
}

func isAllowedImage(contentType, filename string, header []byte) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if _, ok := allowedImageContentTypes[ct]; ok && strings.HasPrefix(ct, "image/") {
		return true
	}
	if ct == "application/octet-stream" && detectImageContentType(header, filename, "") != "" {
		return true
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		return true
	}
	return detectImageContentType(header, filename, "") != ""
}

func detectImageContentType(data []byte, filename, headerContentType string) string {
	ct := strings.ToLower(strings.TrimSpace(headerContentType))
	if strings.HasPrefix(ct, "image/") {
		if ct == "image/jpg" {
			return "image/jpeg"
		}
		return ct
	}
	if len(data) >= 4 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}
	if len(data) >= 3 && data[0] == 'G' && data[1] == 'I' && data[2] == 'F' {
		return "image/gif"
	}
	if len(data) >= 12 && data[0] == 'R' && data[1] == 'I' && data[2] == 'F' && data[3] == 'F' &&
		data[8] == 'W' && data[9] == 'E' && data[10] == 'B' && data[11] == 'P' {
		return "image/webp"
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".png":
		return "image/png"
	}
	return "image/png"
}

func extensionForContentType(contentType, filename string) string {
	if ext := strings.ToLower(filepath.Ext(filename)); ext != "" {
		return ext
	}
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".png"
	}
}

func EnrichPrompt(prompt, style, negative string) string {
	prompt = strings.TrimSpace(prompt)
	styleHints := map[string]string{
		"pixel":     "像素风格",
		"cartoon":   "卡通风格",
		"realistic": "写实风格",
		"anime":     "二次元风格",
	}
	if hint, ok := styleHints[strings.TrimSpace(style)]; ok && hint != "" {
		if prompt == "" {
			prompt = hint
		} else {
			prompt = prompt + "，" + hint
		}
	}
	negative = strings.TrimSpace(negative)
	if negative != "" {
		if prompt == "" {
			prompt = "避免：" + negative
		} else {
			prompt = prompt + "。避免：" + negative
		}
	}
	return prompt
}

func EnrichVideoPrompt(prompt, motion, fps string) string {
	prompt = strings.TrimSpace(prompt)
	motionHints := map[string]string{
		"low":    "轻微运动",
		"medium": "中等运动幅度",
		"high":   "强烈运动",
	}
	if hint, ok := motionHints[strings.TrimSpace(motion)]; ok && hint != "" {
		prompt = appendHint(prompt, hint)
	}
	fps = strings.TrimSpace(fps)
	if fps != "" {
		prompt = appendHint(prompt, fps+" FPS")
	}
	return prompt
}

func appendHint(prompt, hint string) string {
	if prompt == "" {
		return hint
	}
	return prompt + "，" + hint
}

func ParseDurationSeconds(label string) int {
	n := 0
	for _, ch := range label {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		}
	}
	if n <= 0 {
		n = 5
	}
	if n < 5 {
		n = 5
	}
	if n > 10 {
		n = 10
	}
	return n
}

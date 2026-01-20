package utils

import (
	"testing"
)

func TestGetContentType(t *testing.T) {
	tests := []struct {
		name     string
		suffix   string
		expected string
	}{
		{
			name:     "HTML file",
			suffix:   ".html",
			expected: "text/html",
		},
		{
			name:     "HTML file uppercase",
			suffix:   ".HTML",
			expected: "text/html",
		},
		{
			name:     "JPEG file",
			suffix:   ".jpg",
			expected: "image/jpeg",
		},
		{
			name:     "JPEG file uppercase",
			suffix:   ".JPG",
			expected: "image/jpeg",
		},
		{
			name:     "PNG file",
			suffix:   ".png",
			expected: "image/png",
		},
		{
			name:     "PDF file",
			suffix:   ".pdf",
			expected: "application/pdf",
		},
		{
			name:     "Text file",
			suffix:   ".txt",
			expected: "text/plain",
		},
		{
			name:     "MP3 file",
			suffix:   ".mp3",
			expected: "audio/mp3",
		},
		{
			name:     "MP4 file",
			suffix:   ".mp4",
			expected: "video/mp4",
		},
		{
			name:     "CSS file",
			suffix:   ".css",
			expected: "text/css",
		},
		{
			name:     "JavaScript file",
			suffix:   ".js",
			expected: "application/x-javascript",
		},
		{
			name:     "Unknown file type",
			suffix:   ".unknown",
			expected: "application/octet-stream",
		},
		{
			name:     "Empty suffix",
			suffix:   "",
			expected: "application/octet-stream",
		},
		{
			name:     "No dot in suffix",
			suffix:   "jpg",
			expected: "application/octet-stream",
		},
		{
			name:     "Double extension",
			suffix:   ".tar.gz",
			expected: "application/octet-stream",
		},
		{
			name:     "Wildcard match",
			suffix:   ".*",
			expected: "application/octet-stream",
		},
		{
			name:     "Word document",
			suffix:   ".doc",
			expected: "application/msword",
		},
		{
			name:     "Excel spreadsheet",
			suffix:   ".xls",
			expected: "application/x-xls",
		},
		{
			name:     "PowerPoint presentation",
			suffix:   ".ppt",
			expected: "applications-powerpoint",
		},
		{
			name:     "XML file",
			suffix:   ".xml",
			expected: "text/xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetContentType(tt.suffix)
			if got != tt.expected {
				t.Errorf("GetContentType(%q) = %q, want %q", tt.suffix, got, tt.expected)
			}
		})
	}
}

func TestGetContentTypeCaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		suffix   string
		expected string
	}{
		{
			name:     "Uppercase extension",
			suffix:   ".PDF",
			expected: "application/pdf",
		},
		{
			name:     "Mixed case extension",
			suffix:   ".PdF",
			expected: "application/pdf",
		},
		{
			name:     "Lowercase extension",
			suffix:   ".pdf",
			expected: "application/pdf",
		},
		{
			name:     "Uppercase extension without dot",
			suffix:   "PDF",
			expected: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetContentType(tt.suffix)
			if got != tt.expected {
				t.Errorf("GetContentType(%q) = %q, want %q", tt.suffix, got, tt.expected)
			}
		})
	}
}

func TestGetContentTypeCommonExtensions(t *testing.T) {
	// 测试一些常见的文件扩展名
	commonExtensions := []struct {
		extension string
		mimeType  string
	}{
		{".gif", "image/gif"},
		{".bmp", "application/x-bmp"},
		{".ico", "image/x-icon"},
		{".tif", "image/tiff"},
		{".tiff", "image/tiff"},
		{".zip", "application/octet-stream"}, // 未在映射中定义
		{".rar", "application/octet-stream"}, // 未在映射中定义
		{".wav", "audio/wav"},
		{".avi", "video/avi"},
		{".mov", "application/octet-stream"}, // 未在映射中定义
		{".wmv", "video/x-ms-wmv"},
	}

	for _, ext := range commonExtensions {
		t.Run(ext.extension, func(t *testing.T) {
			got := GetContentType(ext.extension)
			if got != ext.mimeType {
				t.Errorf("GetContentType(%q) = %q, want %q", ext.extension, got, ext.mimeType)
			}
		})
	}
}

func TestGetContentTypeSpecialCases(t *testing.T) {
	tests := []struct {
		name     string
		suffix   string
		expected string
	}{
		{
			name:     "Wildcard pattern",
			suffix:   ".*",
			expected: "application/octet-stream",
		},
		{
			name:     "Java files",
			suffix:   ".java",
			expected: "java/*",
		},
		{
			name:     "Class files",
			suffix:   ".class",
			expected: "java/*",
		},
		{
			name:     "Without dot",
			suffix:   "html",
			expected: "application/octet-stream",
		},
		{
			name:     "Empty string",
			suffix:   "",
			expected: "application/octet-stream",
		},
		{
			name:     "Only dot",
			suffix:   ".",
			expected: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetContentType(tt.suffix)
			if got != tt.expected {
				t.Errorf("GetContentType(%q) = %q, want %q", tt.suffix, got, tt.expected)
			}
		})
	}
}

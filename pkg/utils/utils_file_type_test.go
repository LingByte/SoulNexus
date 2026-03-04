package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytesToHexString(t *testing.T) {
	tests := []struct {
		name     string
		src      []byte
		expected string
	}{
		{
			name:     "Test empty byte slice",
			src:      []byte{},
			expected: "",
		},
		{
			name:     "Test single byte slice",
			src:      []byte{255},
			expected: "ff",
		},
		{
			name:     "Test multiple byte slice",
			src:      []byte{255, 0, 128},
			expected: "ff0080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := bytesToHexString(tt.src)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetFileType(t *testing.T) {
	tests := []struct {
		name     string
		buf      []byte
		expected string
	}{
		{
			name:     "Test JPEG file type",
			buf:      []byte{0xFF, 0xD8, 0xFF, 0xE0},
			expected: "jpg",
		},
		{
			name:     "Test PNG file type",
			buf:      []byte{0x89, 0x50, 0x4E, 0x47},
			expected: "png",
		},
		{
			name:     "Test unsupported file type",
			buf:      []byte{0x00, 0x00, 0x00, 0x00},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := GetFileType(tt.buf)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetFileTypeBySuffix(t *testing.T) {
	tests := []struct {
		name     string
		suffix   string
		expected int32
	}{
		{
			name:     "Test image file type",
			suffix:   "jpg",
			expected: FILE_TYPE_IMAGE,
		},
		{
			name:     "Test audio file type",
			suffix:   "mp3",
			expected: FILE_TYPE_AUDIO,
		},
		{
			name:     "Test media file type",
			suffix:   "mp4",
			expected: FILE_TYPE_MEDIA,
		},
		{
			name:     "Test other file type",
			suffix:   "txt",
			expected: FILE_TYPE_FILE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := GetFileTypeBySuffix(tt.suffix)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestSetFileType(t *testing.T) {
	// As setFileType is invoked in init function, we can test it indirectly
	// Test if the fileTypes map is populated correctly by checking the known file signatures
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "Test JPEG file type",
			key:      "ffd8ffe000104a464946",
			expected: "jpg",
		},
		{
			name:     "Test PNG file type",
			key:      "89504e470d0a1a0a0000",
			expected: "png",
		},
		{
			name:     "Test non-existent file type",
			key:      "00000000000000000000",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := fileTypes[tt.key]
			assert.Equal(t, tt.expected, actual)
		})
	}
}

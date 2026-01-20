package utils

import (
	"bytes"
	"compress/zlib"
	"testing"
)

func TestZlib(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
	}{
		{
			name: "Empty input",
			in:   []byte{},
		},
		{
			name: "Simple text",
			in:   []byte("hello world"),
		},
		{
			name: "Long text",
			in:   []byte("this is a longer text to test zlib compression functionality with more data"),
		},
		{
			name: "Binary data",
			in:   []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0xFF, 0xFE, 0xFD},
		},
		{
			name: "Repeated data",
			in:   bytes.Repeat([]byte("a"), 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Zlib(tt.in)

			// Verify that we can decompress and get the original data back
			r := bytes.NewReader(got)
			rc, err := zlib.NewReader(r)
			if err != nil {
				t.Fatalf("Failed to create zlib reader: %v", err)
			}
			defer rc.Close()

			var buf bytes.Buffer
			_, err = buf.ReadFrom(rc)
			if err != nil {
				t.Fatalf("Failed to read from zlib reader: %v", err)
			}

			if !bytes.Equal(buf.Bytes(), tt.in) {
				t.Errorf("Zlib() = %v, want %v", buf.Bytes(), tt.in)
			}
		})
	}
}

func TestUnZlib(t *testing.T) {
	tests := []struct {
		name     string
		origData []byte
	}{
		{
			name:     "Empty input",
			origData: []byte{},
		},
		{
			name:     "Simple text",
			origData: []byte("hello world"),
		},
		{
			name:     "Long text",
			origData: []byte("this is a longer text to test zlib decompression functionality with more data"),
		},
		{
			name:     "Binary data",
			origData: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0xFF, 0xFE, 0xFD},
		},
		{
			name:     "Repeated data",
			origData: bytes.Repeat([]byte("b"), 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First compress the data using standard zlib
			var compressed bytes.Buffer
			w := zlib.NewWriter(&compressed)
			w.Write(tt.origData)
			w.Close()

			// Then test our UnZlib function
			got := UnZlib(compressed.Bytes())

			if !bytes.Equal(got, tt.origData) {
				t.Errorf("UnZlib() = %v, want %v", got, tt.origData)
			}
		})
	}
}

func TestZlibUnZlibRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
	}{
		{
			name: "Round trip test with text",
			in:   []byte("round trip test data for zlib"),
		},
		{
			name: "Round trip test with binary",
			in:   []byte{0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80},
		},
		{
			name: "Round trip with repeated data",
			in:   bytes.Repeat([]byte("round"), 50),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compress with our Zlib function
			compressed := Zlib(tt.in)

			// Decompress with our UnZlib function
			decompressed := UnZlib(compressed)

			if !bytes.Equal(decompressed, tt.in) {
				t.Errorf("Zlib->UnZlib round trip failed, got %v, want %v", decompressed, tt.in)
			}
		})
	}
}

func TestUnZlibWithInvalidData(t *testing.T) {
	// Test with invalid zlib data
	invalidData := []byte("this is not valid zlib data")
	result := UnZlib(invalidData)

	// With invalid data, we expect an empty result since we fixed the function to handle errors
	if len(result) != 0 {
		t.Errorf("UnZlib with invalid data should return empty slice, got %v", result)
	}
}

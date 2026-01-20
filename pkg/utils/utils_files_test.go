package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateFileHash(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test_hash_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write test content
	testContent := "Hello, World!"
	_, err = tmpFile.WriteString(testContent)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Test hash calculation
	hash, err := CalculateFileHash(tmpFile.Name())
	if err != nil {
		t.Fatalf("CalculateFileHash error: %v", err)
	}
	if hash == "" {
		t.Fatalf("CalculateFileHash returned empty hash")
	}
	if len(hash) != 40 { // SHA1 produces 40 character hex string
		t.Fatalf("CalculateFileHash returned hash of length %d, want 40", len(hash))
	}
}

func TestCalculateFileHash_FileNotExists(t *testing.T) {
	_, err := CalculateFileHash("/nonexistent/file/path.txt")
	if err == nil {
		t.Fatalf("CalculateFileHash expected error for non-existent file")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := GenerateID()
	id2 := GenerateID()

	if id1 == "" {
		t.Fatalf("GenerateID returned empty ID")
	}
	if len(id1) != 16 { // SHA1[:8] produces 16 character hex string
		t.Fatalf("GenerateID returned ID of length %d, want 16", len(id1))
	}
	// IDs should be different (very unlikely to be same)
	if id1 == id2 {
		t.Fatalf("GenerateID returned same ID twice: %s", id1)
	}
}

func TestIsDirectory(t *testing.T) {
	// Test with actual directory
	tmpDir := os.TempDir()
	if !IsDirectory(tmpDir) {
		t.Fatalf("IsDirectory(%s) = false, want true", tmpDir)
	}

	// Test with file
	tmpFile, err := os.CreateTemp("", "test_file_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	if IsDirectory(tmpFile.Name()) {
		t.Fatalf("IsDirectory(%s) = true, want false", tmpFile.Name())
	}

	// Test with non-existent path
	if IsDirectory("/nonexistent/path") {
		t.Fatalf("IsDirectory(nonexistent) = true, want false")
	}
}

func TestIsFile(t *testing.T) {
	// Test with file
	tmpFile, err := os.CreateTemp("", "test_file_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	if !IsFile(tmpFile.Name()) {
		t.Fatalf("IsFile(%s) = false, want true", tmpFile.Name())
	}

	// Test with directory
	tmpDir := os.TempDir()
	if IsFile(tmpDir) {
		t.Fatalf("IsFile(%s) = true, want false", tmpDir)
	}

	// Test with non-existent path
	if IsFile("/nonexistent/path.txt") {
		t.Fatalf("IsFile(nonexistent) = true, want false")
	}
}

func TestFileExists(t *testing.T) {
	// Test with existing file
	tmpFile, err := os.CreateTemp("", "test_file_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	if !FileExists(tmpFile.Name()) {
		t.Fatalf("FileExists(%s) = false, want true", tmpFile.Name())
	}

	// Test with non-existent file
	if FileExists("/nonexistent/file/path.txt") {
		t.Fatalf("FileExists(nonexistent) = true, want false")
	}
}

func TestGetFileSize(t *testing.T) {
	// Create a temporary file with known size
	tmpFile, err := os.CreateTemp("", "test_size_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testContent := "Hello, World!"
	expectedSize := int64(len(testContent))
	_, err = tmpFile.WriteString(testContent)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	size, err := GetFileSize(tmpFile.Name())
	if err != nil {
		t.Fatalf("GetFileSize error: %v", err)
	}
	if size != expectedSize {
		t.Fatalf("GetFileSize = %d, want %d", size, expectedSize)
	}
}

func TestGetFileSize_FileNotExists(t *testing.T) {
	_, err := GetFileSize("/nonexistent/file/path.txt")
	if err == nil {
		t.Fatalf("GetFileSize expected error for non-existent file")
	}
}

func TestCopyFile(t *testing.T) {
	// Create source file
	srcFile, err := os.CreateTemp("", "test_copy_src_*.txt")
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	defer os.Remove(srcFile.Name())

	testContent := "Test content for copying"
	_, err = srcFile.WriteString(testContent)
	if err != nil {
		t.Fatalf("Failed to write to source file: %v", err)
	}
	srcFile.Close()

	// Create destination path
	dstFile := filepath.Join(os.TempDir(), "test_copy_dst.txt")
	defer os.Remove(dstFile)

	// Test copy
	err = CopyFile(srcFile.Name(), dstFile)
	if err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	// Verify destination file exists and has correct content
	if !FileExists(dstFile) {
		t.Fatalf("CopyFile destination file does not exist")
	}

	content, err := ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}
	if string(content) != testContent {
		t.Fatalf("Copied file content = %s, want %s", string(content), testContent)
	}
}

func TestCopyFile_SourceNotExists(t *testing.T) {
	dstFile := filepath.Join(os.TempDir(), "test_copy_dst.txt")
	defer os.Remove(dstFile)

	err := CopyFile("/nonexistent/source.txt", dstFile)
	if err == nil {
		t.Fatalf("CopyFile expected error for non-existent source file")
	}
}

func TestEnsureDirectory(t *testing.T) {
	// Test creating new directory
	testDir := filepath.Join(os.TempDir(), "test_ensure_dir")
	defer os.RemoveAll(testDir)

	err := EnsureDirectory(testDir)
	if err != nil {
		t.Fatalf("EnsureDirectory error: %v", err)
	}

	if !IsDirectory(testDir) {
		t.Fatalf("EnsureDirectory did not create directory")
	}

	// Test creating nested directories
	nestedDir := filepath.Join(testDir, "nested", "deep")
	err = EnsureDirectory(nestedDir)
	if err != nil {
		t.Fatalf("EnsureDirectory error for nested dir: %v", err)
	}

	if !IsDirectory(nestedDir) {
		t.Fatalf("EnsureDirectory did not create nested directory")
	}
}

func TestRemoveFile(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test_remove_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	filePath := tmpFile.Name()

	// Verify file exists
	if !FileExists(filePath) {
		t.Fatalf("Test file does not exist before removal")
	}

	// Test removal
	err = RemoveFile(filePath)
	if err != nil {
		t.Fatalf("RemoveFile error: %v", err)
	}

	// Verify file no longer exists
	if FileExists(filePath) {
		t.Fatalf("RemoveFile did not remove file")
	}
}

func TestRemoveDirectory(t *testing.T) {
	// Create a test directory with files
	testDir := filepath.Join(os.TempDir(), "test_remove_dir")
	err := EnsureDirectory(testDir)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a file inside
	testFile := filepath.Join(testDir, "test.txt")
	err = WriteFile(testFile, []byte("test"))
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test removal
	err = RemoveDirectory(testDir)
	if err != nil {
		t.Fatalf("RemoveDirectory error: %v", err)
	}

	// Verify directory no longer exists
	if FileExists(testDir) {
		t.Fatalf("RemoveDirectory did not remove directory")
	}
}

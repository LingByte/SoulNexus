package knowledge

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestExportJSON(t *testing.T) {
	data := ExportData{
		Metadata: ExportMetadata{
			ExportedAt:   time.Now(),
			TotalRecords: 2,
			Handler:      "test",
			Namespace:    "default",
			Version:      "1.0",
		},
		Records: []Record{
			{ID: "1", Title: "Doc1", Content: "Content1"},
			{ID: "2", Title: "Doc2", Content: "Content2"},
		},
	}

	buf := &bytes.Buffer{}
	err := exportJSON(buf, data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's valid JSON
	var result ExportData
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("exported data is not valid JSON: %v", err)
	}

	if result.Metadata.TotalRecords != 2 {
		t.Errorf("expected 2 records, got %d", result.Metadata.TotalRecords)
	}
}

func TestExportJSONL(t *testing.T) {
	data := ExportData{
		Metadata: ExportMetadata{
			ExportedAt:   time.Now(),
			TotalRecords: 2,
			Handler:      "test",
			Namespace:    "default",
			Version:      "1.0",
		},
		Records: []Record{
			{ID: "1", Title: "Doc1", Content: "Content1"},
			{ID: "2", Title: "Doc2", Content: "Content2"},
		},
	}

	buf := &bytes.Buffer{}
	err := exportJSONL(buf, data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's valid JSONL
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	if len(lines) < 3 {
		t.Errorf("expected at least 3 lines, got %d", len(lines))
	}

	// First line should be metadata
	var metaLine map[string]interface{}
	if err := json.Unmarshal(lines[0], &metaLine); err != nil {
		t.Fatalf("first line is not valid JSON: %v", err)
	}

	if _, ok := metaLine["_metadata"]; !ok {
		t.Error("first line should contain _metadata")
	}
}

func TestImportJSONL(t *testing.T) {
	// Create JSONL data
	buf := &bytes.Buffer{}
	metaLine := map[string]interface{}{
		"_metadata": map[string]interface{}{
			"total_records": 2,
		},
	}
	metaBytes, _ := json.Marshal(metaLine)
	buf.Write(append(metaBytes, '\n'))

	record1 := Record{ID: "1", Title: "Doc1", Content: "Content1"}
	record1Bytes, _ := json.Marshal(record1)
	buf.Write(append(record1Bytes, '\n'))

	record2 := Record{ID: "2", Title: "Doc2", Content: "Content2"}
	record2Bytes, _ := json.Marshal(record2)
	buf.Write(append(record2Bytes, '\n'))

	// Import
	records, err := importJSONL(buf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}

	if records[0].ID != "1" {
		t.Errorf("expected first record ID to be 1, got %s", records[0].ID)
	}
}

func TestBackupConfig(t *testing.T) {
	cfg := BackupConfig{
		FilePath:  "/tmp/test_backup",
		Format:    ExportFormatJSON,
		Timestamp: true,
	}

	if cfg.FilePath == "" {
		t.Error("FilePath should not be empty")
	}

	if cfg.Format != ExportFormatJSON {
		t.Errorf("expected format JSON, got %s", cfg.Format)
	}

	if !cfg.Timestamp {
		t.Error("Timestamp should be true")
	}
}

func TestExportMetadata(t *testing.T) {
	metadata := ExportMetadata{
		ExportedAt:   time.Now(),
		TotalRecords: 10,
		Handler:      "qdrant",
		Namespace:    "default",
		Version:      "1.0",
	}

	bytes, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("failed to marshal metadata: %v", err)
	}

	var result ExportMetadata
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("failed to unmarshal metadata: %v", err)
	}

	if result.Handler != "qdrant" {
		t.Errorf("expected handler qdrant, got %s", result.Handler)
	}

	if result.TotalRecords != 10 {
		t.Errorf("expected 10 records, got %d", result.TotalRecords)
	}
}

func TestExportData(t *testing.T) {
	exportData := ExportData{
		Metadata: ExportMetadata{
			ExportedAt:   time.Now(),
			TotalRecords: 1,
			Handler:      "test",
			Namespace:    "default",
			Version:      "1.0",
		},
		Records: []Record{
			{ID: "1", Title: "Test", Content: "Content"},
		},
	}

	bytes, err := json.Marshal(exportData)
	if err != nil {
		t.Fatalf("failed to marshal export data: %v", err)
	}

	var result ExportData
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("failed to unmarshal export data: %v", err)
	}

	if len(result.Records) != 1 {
		t.Errorf("expected 1 record, got %d", len(result.Records))
	}
}

func TestExportFormatConstants(t *testing.T) {
	if ExportFormatJSON != "json" {
		t.Errorf("expected JSON format to be 'json', got %s", ExportFormatJSON)
	}

	if ExportFormatJSONL != "jsonl" {
		t.Errorf("expected JSONL format to be 'jsonl', got %s", ExportFormatJSONL)
	}
}

// Helper function to clean up test files
func cleanupTestFile(filepath string) {
	os.Remove(filepath)
}

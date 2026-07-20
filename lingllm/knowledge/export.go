package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// ExportFormat defines the export format
type ExportFormat string

const (
	ExportFormatJSON  ExportFormat = "json"
	ExportFormatJSONL ExportFormat = "jsonl"
)

// ExportMetadata contains metadata about the export
type ExportMetadata struct {
	ExportedAt   time.Time `json:"exported_at"`
	TotalRecords int       `json:"total_records"`
	Handler      string    `json:"handler"`
	Namespace    string    `json:"namespace"`
	Version      string    `json:"version"`
}

// ExportData represents the complete export data
type ExportData struct {
	Metadata ExportMetadata `json:"metadata"`
	Records  []Record       `json:"records"`
}

// ExportToFile exports knowledge base to a file
func (kb *KnowledgeBase) ExportToFile(ctx context.Context, filepath string, format ExportFormat) error {
	if filepath == "" {
		return fmt.Errorf("filepath is required")
	}

	// Get all records from handler
	records, err := kb.getAllRecords(ctx)
	if err != nil {
		return fmt.Errorf("failed to get records: %w", err)
	}

	// Create export data
	exportData := ExportData{
		Metadata: ExportMetadata{
			ExportedAt:   time.Now(),
			TotalRecords: len(records),
			Handler:      kb.handler.Provider(),
			Namespace:    kb.namespace,
			Version:      "1.0",
		},
		Records: records,
	}

	// Write to file based on format
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	switch format {
	case ExportFormatJSON:
		return exportJSON(file, exportData)
	case ExportFormatJSONL:
		return exportJSONL(file, exportData)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// ExportToWriter exports knowledge base to a writer
func (kb *KnowledgeBase) ExportToWriter(ctx context.Context, w io.Writer, format ExportFormat) error {
	// Get all records from handler
	records, err := kb.getAllRecords(ctx)
	if err != nil {
		return fmt.Errorf("failed to get records: %w", err)
	}

	// Create export data
	exportData := ExportData{
		Metadata: ExportMetadata{
			ExportedAt:   time.Now(),
			TotalRecords: len(records),
			Handler:      kb.handler.Provider(),
			Namespace:    kb.namespace,
			Version:      "1.0",
		},
		Records: records,
	}

	// Write based on format
	switch format {
	case ExportFormatJSON:
		return exportJSON(w, exportData)
	case ExportFormatJSONL:
		return exportJSONL(w, exportData)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// ImportFromFile imports knowledge base from a file
func (kb *KnowledgeBase) ImportFromFile(ctx context.Context, filepath string, format ExportFormat) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return kb.ImportFromReader(ctx, file, format)
}

// ImportFromReader imports knowledge base from a reader
func (kb *KnowledgeBase) ImportFromReader(ctx context.Context, r io.Reader, format ExportFormat) error {
	var exportData ExportData

	switch format {
	case ExportFormatJSON:
		if err := json.NewDecoder(r).Decode(&exportData); err != nil {
			return fmt.Errorf("failed to decode JSON: %w", err)
		}
	case ExportFormatJSONL:
		records, err := importJSONL(r)
		if err != nil {
			return err
		}
		exportData.Records = records
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	// Upsert all records
	if len(exportData.Records) > 0 {
		opts := &UpsertOptions{
			Namespace: kb.namespace,
		}
		if err := kb.handler.Upsert(ctx, exportData.Records, opts); err != nil {
			return fmt.Errorf("failed to upsert records: %w", err)
		}
	}

	return nil
}

// exportJSON exports data in JSON format
func exportJSON(w io.Writer, data ExportData) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// exportJSONL exports data in JSONL format (one record per line)
func exportJSONL(w io.Writer, data ExportData) error {
	// Write metadata as first line
	metaBytes, err := json.Marshal(map[string]interface{}{
		"_metadata": data.Metadata,
	})
	if err != nil {
		return err
	}
	if _, err := w.Write(append(metaBytes, '\n')); err != nil {
		return err
	}

	// Write each record on a separate line
	for _, record := range data.Records {
		recordBytes, err := json.Marshal(record)
		if err != nil {
			return err
		}
		if _, err := w.Write(append(recordBytes, '\n')); err != nil {
			return err
		}
	}

	return nil
}

// importJSONL imports data from JSONL format
func importJSONL(r io.Reader) ([]Record, error) {
	var records []Record
	decoder := json.NewDecoder(r)

	for decoder.More() {
		var data map[string]interface{}
		if err := decoder.Decode(&data); err != nil {
			return nil, fmt.Errorf("failed to decode JSONL: %w", err)
		}

		// Skip metadata line
		if _, ok := data["_metadata"]; ok {
			continue
		}

		// Convert to Record
		recordBytes, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}

		var record Record
		if err := json.Unmarshal(recordBytes, &record); err != nil {
			return nil, err
		}

		records = append(records, record)
	}

	return records, nil
}

// getAllRecords retrieves all records from the handler
func (kb *KnowledgeBase) getAllRecords(ctx context.Context) ([]Record, error) {
	var allRecords []Record

	// Try to get all records using List operation
	opts := &ListOptions{
		Namespace: kb.namespace,
		Limit:     1000,
		Offset:    "",
	}

	for {
		result, err := kb.handler.List(ctx, opts)
		if err != nil {
			// If List is not supported, return empty
			return allRecords, nil
		}

		if result == nil || len(result.Records) == 0 {
			break
		}

		allRecords = append(allRecords, result.Records...)

		// Check if there are more records
		if result.NextOffset == "" {
			break
		}

		opts.Offset = result.NextOffset
	}

	return allRecords, nil
}

// BackupConfig represents backup configuration
type BackupConfig struct {
	FilePath  string
	Format    ExportFormat
	Timestamp bool // Add timestamp to filename
}

// Backup creates a backup of the knowledge base
func (kb *KnowledgeBase) Backup(ctx context.Context, cfg BackupConfig) (string, error) {
	filepath := cfg.FilePath
	if cfg.Timestamp {
		timestamp := time.Now().Format("20060102_150405")
		filepath = fmt.Sprintf("%s_%s", cfg.FilePath, timestamp)
	}

	if err := kb.ExportToFile(ctx, filepath, cfg.Format); err != nil {
		return "", err
	}

	return filepath, nil
}

// Restore restores the knowledge base from a backup
func (kb *KnowledgeBase) Restore(ctx context.Context, filepath string, format ExportFormat) error {
	return kb.ImportFromFile(ctx, filepath, format)
}

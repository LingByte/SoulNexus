package knowledge

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/LingByte/SoulNexus/pkg/knowledge/constants"
	"github.com/LingByte/SoulNexus/pkg/knowledge/models"
	"gorm.io/gorm"
	"strings"
	"sync"
	"time"
)

var (
	syncEnqueueMu sync.RWMutex
	syncEnqueueFn func(sourceID uint)
)

// RegisterSyncEnqueue wires URL sync jobs to the document worker (API process).
func RegisterSyncEnqueue(fn func(sourceID uint)) {
	syncEnqueueMu.Lock()
	syncEnqueueFn = fn
	syncEnqueueMu.Unlock()
}

// StartSyncCron enqueues due knowledge sync sources on a fixed interval.
func StartSyncCron(db *gorm.DB, interval time.Duration) {
	if db == nil || interval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			EnqueueDueSyncSources(db)
		}
	}()
}

// EnqueueDueSyncSources finds active sources past their interval and enqueues sync jobs.
func EnqueueDueSyncSources(db *gorm.DB) {
	if db == nil {
		return
	}
	syncEnqueueMu.RLock()
	enqueue := syncEnqueueFn
	syncEnqueueMu.RUnlock()
	if enqueue == nil {
		return
	}
	var sources []models.KnowledgeSyncSource
	now := time.Now()
	if err := db.Where("status = ?", constants.KnowledgeSyncStatusActive).Find(&sources).Error; err != nil {
		return
	}
	for _, src := range sources {
		if src.IntervalMinutes <= 0 {
			continue
		}
		if src.LastSyncAt != nil && now.Sub(*src.LastSyncAt) < time.Duration(src.IntervalMinutes)*time.Minute {
			continue
		}
		enqueue(src.ID)
	}
}

// TableSyncConfig configures column-based table ingestion.
type TableSyncConfig struct {
	IndexColumns []string `json:"indexColumns"`
	TitleColumn  string   `json:"titleColumn"`
	KeyColumn    string   `json:"keyColumn"`
	Format       string   `json:"format"` // csv, tsv, json
}

// TableSyncRow is one indexed table row.
type TableSyncRow struct {
	Key     string
	Title   string
	Content string
}

// ParseTableSyncConfig reads chunkConfig JSON from a sync source.
func ParseTableSyncConfig(raw []byte) TableSyncConfig {
	cfg := TableSyncConfig{Format: "csv"}
	if len(raw) == 0 {
		return cfg
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return cfg
	}
	if cols, ok := m["indexColumns"].([]any); ok {
		for _, c := range cols {
			if s := strings.TrimSpace(fmt.Sprint(c)); s != "" {
				cfg.IndexColumns = append(cfg.IndexColumns, s)
			}
		}
	}
	if s := strings.TrimSpace(fmt.Sprint(m["titleColumn"])); s != "" && s != "<nil>" {
		cfg.TitleColumn = s
	}
	if s := strings.TrimSpace(fmt.Sprint(m["keyColumn"])); s != "" && s != "<nil>" {
		cfg.KeyColumn = s
	}
	if s := strings.TrimSpace(fmt.Sprint(m["format"])); s != "" && s != "<nil>" {
		cfg.Format = strings.ToLower(s)
	}
	if len(cfg.IndexColumns) == 0 {
		if s := strings.TrimSpace(fmt.Sprint(m["indexColumns"])); s != "" && s != "<nil>" {
			for _, part := range strings.Split(s, ",") {
				if col := strings.TrimSpace(part); col != "" {
					cfg.IndexColumns = append(cfg.IndexColumns, col)
				}
			}
		}
	}
	return cfg
}

// ParseTableSyncBody parses fetched bytes into rows according to config.
func ParseTableSyncBody(body []byte, cfg TableSyncConfig) ([]TableSyncRow, error) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, fmt.Errorf("empty table body")
	}
	if len(cfg.IndexColumns) == 0 {
		return nil, fmt.Errorf("indexColumns is required for table sync")
	}
	switch cfg.Format {
	case "json":
		return parseTableJSON(body, cfg)
	default:
		return parseTableCSV(body, cfg, cfg.Format == "tsv")
	}
}

func parseTableJSON(body []byte, cfg TableSyncConfig) ([]TableSyncRow, error) {
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		var wrapped map[string]any
		if err2 := json.Unmarshal(body, &wrapped); err2 != nil {
			return nil, fmt.Errorf("parse json table: %w", err)
		}
		for _, k := range []string{"data", "rows", "items", "records"} {
			if v, ok := wrapped[k].([]any); ok {
				for _, item := range v {
					if m, ok := item.(map[string]any); ok {
						rows = append(rows, m)
					}
				}
				break
			}
		}
		if len(rows) == 0 {
			return nil, fmt.Errorf("json table: no row array found")
		}
	}
	out := make([]TableSyncRow, 0, len(rows))
	for i, row := range rows {
		tr, err := mapToTableRow(row, cfg, i)
		if err != nil {
			continue
		}
		out = append(out, tr)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid table rows")
	}
	return out, nil
}

func parseTableCSV(body []byte, cfg TableSyncConfig, tab bool) ([]TableSyncRow, error) {
	r := csv.NewReader(bytes.NewReader(body))
	if tab {
		r.Comma = '\t'
	}
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse csv table: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("table needs header + at least one data row")
	}
	headers := records[0]
	colIndex := map[string]int{}
	for i, h := range headers {
		colIndex[strings.TrimSpace(h)] = i
	}
	for _, col := range cfg.IndexColumns {
		if _, ok := colIndex[col]; !ok {
			return nil, fmt.Errorf("column %q not found in header", col)
		}
	}
	if cfg.TitleColumn != "" {
		if _, ok := colIndex[cfg.TitleColumn]; !ok {
			return nil, fmt.Errorf("title column %q not found", cfg.TitleColumn)
		}
	}
	if cfg.KeyColumn != "" {
		if _, ok := colIndex[cfg.KeyColumn]; !ok {
			return nil, fmt.Errorf("key column %q not found", cfg.KeyColumn)
		}
	}
	out := make([]TableSyncRow, 0, len(records)-1)
	for i, rec := range records[1:] {
		row := map[string]any{}
		for name, idx := range colIndex {
			if idx < len(rec) {
				row[name] = rec[idx]
			}
		}
		tr, err := mapToTableRow(row, cfg, i)
		if err != nil {
			continue
		}
		out = append(out, tr)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid table rows")
	}
	return out, nil
}

func mapToTableRow(row map[string]any, cfg TableSyncConfig, rowIndex int) (TableSyncRow, error) {
	key := ""
	if cfg.KeyColumn != "" {
		key = strings.TrimSpace(fmt.Sprint(row[cfg.KeyColumn]))
	}
	if key == "" {
		key = fmt.Sprintf("row-%d", rowIndex+1)
	}
	title := ""
	if cfg.TitleColumn != "" {
		title = strings.TrimSpace(fmt.Sprint(row[cfg.TitleColumn]))
	}
	var b strings.Builder
	for i, col := range cfg.IndexColumns {
		val := strings.TrimSpace(fmt.Sprint(row[col]))
		if val == "" || val == "<nil>" {
			continue
		}
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(col)
		b.WriteString("：")
		b.WriteString(val)
	}
	content := strings.TrimSpace(b.String())
	if content == "" {
		return TableSyncRow{}, fmt.Errorf("empty row")
	}
	if title == "" {
		title = key
	}
	return TableSyncRow{Key: key, Title: title, Content: content}, nil
}

// TableRowRecordID builds a stable vector point id for a table row.
func TableRowRecordID(sourceID uint, rowKey string) string {
	rowKey = strings.TrimSpace(rowKey)
	if rowKey == "" {
		rowKey = "unknown"
	}
	return fmt.Sprintf("table-%d-%s", sourceID, sanitizeRecordKey(rowKey))
}

func sanitizeRecordKey(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "row"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "row"
	}
	if len(out) > 120 {
		return out[:120]
	}
	return out
}

package eval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Sample is one labeled retrieval query for offline evaluation.
type Sample struct {
	Query       string   `json:"query"`
	Namespace   string   `json:"namespace,omitempty"`
	RelevantIDs []string `json:"relevant_ids"`
	GoldAnswer  string   `json:"gold_answer,omitempty"`
}

// LoadJSONL reads evaluation samples from JSON Lines (.jsonl).
// Blank lines and lines starting with # are ignored.
func LoadJSONL(r io.Reader) ([]Sample, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var out []Sample
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var s Sample
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			return nil, fmt.Errorf("eval: line %d: %w", lineNo, err)
		}
		s.Query = strings.TrimSpace(s.Query)
		if s.Query == "" {
			return nil, fmt.Errorf("eval: line %d: query is required", lineNo)
		}
		if len(s.RelevantIDs) == 0 {
			return nil, fmt.Errorf("eval: line %d: relevant_ids is required", lineNo)
		}
		out = append(out, s)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("eval: read jsonl: %w", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("eval: dataset is empty")
	}
	return out, nil
}

// LoadJSONLFile loads samples from a file path.
func LoadJSONLFile(path string) ([]Sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return LoadJSONL(f)
}

// NormalizeSamples fills empty per-sample namespaces with defaultNamespace.
func NormalizeSamples(samples []Sample, defaultNamespace string) []Sample {
	defaultNamespace = strings.TrimSpace(defaultNamespace)
	out := make([]Sample, len(samples))
	for i, s := range samples {
		out[i] = s
		if strings.TrimSpace(out[i].Namespace) == "" {
			out[i].Namespace = defaultNamespace
		}
	}
	return out
}

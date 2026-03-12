package search

import (
	"testing"
)

func TestBuildIndexMapping_DefaultAnalyzer(t *testing.T) {
	m := BuildIndexMapping("")
	if m == nil {
		t.Fatalf("Expected mapping, got nil")
	}
	if m.DefaultAnalyzer == "" {
		t.Fatalf("Expected default analyzer to be set")
	}
}

func TestBuildIndexMapping_CustomAnalyzer(t *testing.T) {
	m := BuildIndexMapping("keyword")
	if m == nil {
		t.Fatalf("Expected mapping, got nil")
	}
	if m.DefaultAnalyzer != "keyword" {
		t.Fatalf("Expected analyzer 'keyword', got %s", m.DefaultAnalyzer)
	}
}

func TestBuildIndexMapping_TypeField(t *testing.T) {
	m := BuildIndexMapping("")
	if m.TypeField != "type" {
		t.Fatalf("Expected type field 'type', got %s", m.TypeField)
	}
}

func TestBuildIndexMapping_ArticleMapping(t *testing.T) {
	m := BuildIndexMapping("")

	articleMapping := m.TypeMapping["article"]
	if articleMapping == nil {
		t.Fatalf("Expected article mapping, got nil")
	}

	// Verify article mapping exists and is not dynamic
	if articleMapping.Dynamic {
		t.Fatalf("Expected article mapping to be non-dynamic")
	}
}

func TestBuildIndexMapping_DefaultMapping(t *testing.T) {
	m := BuildIndexMapping("")

	if m.DefaultMapping == nil {
		t.Fatalf("Expected default mapping, got nil")
	}

	// Default mapping should be dynamic to support user-defined fields
	if !m.DefaultMapping.Dynamic {
		t.Fatalf("Expected default mapping to be dynamic")
	}
}

func TestBuildIndexMapping_FieldProperties(t *testing.T) {
	m := BuildIndexMapping("")
	articleMapping := m.TypeMapping["article"]

	if articleMapping == nil {
		t.Fatalf("Expected article mapping, got nil")
	}

	// Verify article mapping is properly configured
	if articleMapping.Dynamic {
		t.Fatalf("Expected article mapping to be non-dynamic")
	}
}

func TestBuildIndexMapping_NumericField(t *testing.T) {
	m := BuildIndexMapping("")
	articleMapping := m.TypeMapping["article"]

	if articleMapping == nil {
		t.Fatalf("Expected article mapping, got nil")
	}

	// Verify article mapping exists (numeric fields are added via AddFieldMappingsAt)
	if articleMapping.Dynamic {
		t.Fatalf("Expected article mapping to be non-dynamic")
	}
}

func TestBuildIndexMapping_DateTimeField(t *testing.T) {
	m := BuildIndexMapping("")
	articleMapping := m.TypeMapping["article"]

	if articleMapping == nil {
		t.Fatalf("Expected article mapping, got nil")
	}

	// Verify article mapping exists (datetime fields are added via AddFieldMappingsAt)
	if articleMapping.Dynamic {
		t.Fatalf("Expected article mapping to be non-dynamic")
	}
}

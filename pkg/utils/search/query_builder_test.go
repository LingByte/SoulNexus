package search

import (
	"testing"
	"time"
)

func TestBuildQuery_Keyword(t *testing.T) {
	req := SearchRequest{
		Keyword:      "test",
		SearchFields: []string{"title", "body"},
	}

	query := buildQuery(req, []string{"title"})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_Keyword_EmptyFields(t *testing.T) {
	req := SearchRequest{
		Keyword: "test",
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_Keyword_WithDefaultFields(t *testing.T) {
	req := SearchRequest{
		Keyword: "test",
	}

	defaultFields := []string{"title", "body"}
	query := buildQuery(req, defaultFields)
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_QueryString(t *testing.T) {
	boost := 2.0
	req := SearchRequest{
		QueryString: &ClauseQueryString{
			Query:  "test query",
			Fields: []string{"title"},
			Boost:  &boost,
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_MustTerms(t *testing.T) {
	req := SearchRequest{
		MustTerms: map[string][]string{
			"status": {"active"},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_MustTerms_Multiple(t *testing.T) {
	req := SearchRequest{
		MustTerms: map[string][]string{
			"status": {"active", "published"},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_MustNotTerms(t *testing.T) {
	req := SearchRequest{
		MustNotTerms: map[string][]string{
			"status": {"deleted"},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_ShouldTerms(t *testing.T) {
	req := SearchRequest{
		ShouldTerms: map[string][]string{
			"tags": {"go", "python"},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_Matches(t *testing.T) {
	boost := 1.5
	req := SearchRequest{
		Matches: []ClauseMatch{
			{
				Field:    "title",
				Query:    "test",
				Boost:    &boost,
				Operator: "and",
			},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_Matches_OrOperator(t *testing.T) {
	req := SearchRequest{
		Matches: []ClauseMatch{
			{
				Field:    "title",
				Query:    "test",
				Operator: "or",
			},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_Phrases(t *testing.T) {
	boost := 2.0
	req := SearchRequest{
		Phrases: []ClausePhrase{
			{
				Field:  "body",
				Phrase: "exact phrase",
				Slop:   2,
				Boost:  &boost,
			},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_Prefixes(t *testing.T) {
	boost := 1.5
	req := SearchRequest{
		Prefixes: []ClausePrefix{
			{
				Field:  "title",
				Prefix: "test",
				Boost:  &boost,
			},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_Wildcards(t *testing.T) {
	boost := 1.5
	req := SearchRequest{
		Wildcards: []ClauseWildcard{
			{
				Field:   "title",
				Pattern: "test*",
				Boost:   &boost,
			},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_Regexps(t *testing.T) {
	boost := 1.5
	req := SearchRequest{
		Regexps: []ClauseRegex{
			{
				Field:   "title",
				Pattern: "test.*",
				Boost:   &boost,
			},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_Fuzzies(t *testing.T) {
	boost := 1.5
	req := SearchRequest{
		Fuzzies: []ClauseFuzzy{
			{
				Field:     "title",
				Term:      "test",
				Fuzziness: 2,
				Prefix:    1,
				Boost:     &boost,
			},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_NumericRanges(t *testing.T) {
	min := 10.0
	max := 100.0
	req := SearchRequest{
		NumericRanges: []NumericRangeFilter{
			{
				Field: "price",
				GTE:   &min,
				LTE:   &max,
			},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_NumericRanges_GT_LT(t *testing.T) {
	gt := 10.0
	lt := 100.0
	req := SearchRequest{
		NumericRanges: []NumericRangeFilter{
			{
				Field: "price",
				GT:    &gt,
				LT:    &lt,
			},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_TimeRanges(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	req := SearchRequest{
		TimeRanges: []TimeRangeFilter{
			{
				Field:   "createdAt",
				From:    &past,
				To:      &now,
				IncFrom: true,
				IncTo:   true,
			},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_TimeRanges_OnlyFrom(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	req := SearchRequest{
		TimeRanges: []TimeRangeFilter{
			{
				Field:   "createdAt",
				From:    &past,
				IncFrom: true,
			},
		},
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_MinShould(t *testing.T) {
	req := SearchRequest{
		ShouldTerms: map[string][]string{
			"tags": {"go", "python"},
		},
		MinShould: 1,
	}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_Complex(t *testing.T) {
	boost := 2.0
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	min := 10.0

	req := SearchRequest{
		Keyword: "test",
		MustTerms: map[string][]string{
			"status": {"active"},
		},
		ShouldTerms: map[string][]string{
			"tags": {"go"},
		},
		Matches: []ClauseMatch{
			{
				Field: "title",
				Query: "test",
				Boost: &boost,
			},
		},
		NumericRanges: []NumericRangeFilter{
			{
				Field: "price",
				GTE:   &min,
			},
		},
		TimeRanges: []TimeRangeFilter{
			{
				Field:   "createdAt",
				From:    &past,
				To:      &now,
				IncFrom: true,
			},
		},
		MinShould: 1,
	}

	query := buildQuery(req, []string{"title", "body"})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestBuildQuery_EmptyRequest(t *testing.T) {
	req := SearchRequest{}

	query := buildQuery(req, []string{})
	if query == nil {
		t.Fatalf("Expected query, got nil")
	}
}

func TestRMin(t *testing.T) {
	gt := 10.0
	gte := 20.0

	// Test GT
	filter := NumericRangeFilter{GT: &gt}
	result := rMin(filter)
	if result == nil || *result != 10.0 {
		t.Fatalf("Expected 10.0, got %v", result)
	}

	// Test GTE when GT is nil
	filter = NumericRangeFilter{GTE: &gte}
	result = rMin(filter)
	if result == nil || *result != 20.0 {
		t.Fatalf("Expected 20.0, got %v", result)
	}

	// Test nil when both are nil
	filter = NumericRangeFilter{}
	result = rMin(filter)
	if result != nil {
		t.Fatalf("Expected nil, got %v", result)
	}
}

func TestRMax(t *testing.T) {
	lt := 100.0
	lte := 200.0

	// Test LT
	filter := NumericRangeFilter{LT: &lt}
	result := rMax(filter)
	if result == nil || *result != 100.0 {
		t.Fatalf("Expected 100.0, got %v", result)
	}

	// Test LTE when LT is nil
	filter = NumericRangeFilter{LTE: &lte}
	result = rMax(filter)
	if result == nil || *result != 200.0 {
		t.Fatalf("Expected 200.0, got %v", result)
	}

	// Test nil when both are nil
	filter = NumericRangeFilter{}
	result = rMax(filter)
	if result != nil {
		t.Fatalf("Expected nil, got %v", result)
	}
}

func TestBoolPtr(t *testing.T) {
	result := boolPtr(true)
	if result == nil || *result != true {
		t.Fatalf("Expected true, got %v", result)
	}

	result = boolPtr(false)
	if result == nil || *result != false {
		t.Fatalf("Expected false, got %v", result)
	}
}

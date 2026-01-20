package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoin(t *testing.T) {
	tests := []struct {
		name     string
		arg      []any
		str      string
		expected string
	}{
		{
			name:     "Join with integers",
			arg:      []any{1, 2, 3},
			str:      ",",
			expected: "1,2,3",
		},
		{
			name:     "Join with strings",
			arg:      []any{"a", "b", "c"},
			str:      "-",
			expected: "a-b-c",
		},
		{
			name:     "Join with empty array",
			arg:      []any{},
			str:      ",",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := Join(tt.arg, tt.str)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int
		expected []int
	}{
		{
			name:     "Unique with duplicate integers",
			slice:    []int{1, 2, 2, 3, 3, 3},
			expected: []int{1, 2, 3},
		},
		{
			name:     "Unique with no duplicates",
			slice:    []int{1, 2, 3},
			expected: []int{1, 2, 3},
		},
		{
			name:     "Unique with empty slice",
			slice:    []int{},
			expected: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := Unique(tt.slice)
			assert.ElementsMatch(t, tt.expected, actual)
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name     string
		slice1   []int
		slice2   []int
		expected []int
	}{
		{
			name:     "Merge two integer slices",
			slice1:   []int{1, 2, 3},
			slice2:   []int{3, 4, 5},
			expected: []int{1, 2, 3, 3, 4, 5},
		},
		{
			name:     "Merge with empty slice1",
			slice1:   []int{},
			slice2:   []int{1, 2},
			expected: []int{1, 2},
		},
		{
			name:     "Merge with empty slice2",
			slice1:   []int{1, 2},
			slice2:   []int{},
			expected: []int{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := Merge(tt.slice1, tt.slice2)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestIntersect(t *testing.T) {
	tests := []struct {
		name     string
		slice1   []int
		slice2   []int
		expected []int
	}{
		{
			name:     "Intersect of two integer slices",
			slice1:   []int{1, 2, 3},
			slice2:   []int{2, 3, 4},
			expected: []int{2, 3},
		},
		{
			name:     "Intersect with no common elements",
			slice1:   []int{1, 2},
			slice2:   []int{3, 4},
			expected: []int{},
		},
		{
			name:     "Intersect with empty slice1",
			slice1:   []int{},
			slice2:   []int{1, 2},
			expected: []int{},
		},
		{
			name:     "Intersect with empty slice2",
			slice1:   []int{1, 2},
			slice2:   []int{},
			expected: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := Intersect(tt.slice1, tt.slice2)
			assert.ElementsMatch(t, tt.expected, actual)
		})
	}
}

func TestDifference(t *testing.T) {
	tests := []struct {
		name     string
		slice1   []int
		slice2   []int
		expected []int
	}{
		{
			name:     "Difference of two integer slices",
			slice1:   []int{1, 2, 3},
			slice2:   []int{2, 3, 4},
			expected: []int{1},
		},
		{
			name:     "Difference with no common elements",
			slice1:   []int{1, 2},
			slice2:   []int{3, 4},
			expected: []int{1, 2},
		},
		{
			name:     "Difference with empty slice1",
			slice1:   []int{},
			slice2:   []int{1, 2},
			expected: []int{},
		},
		{
			name:     "Difference with empty slice2",
			slice1:   []int{1, 2},
			slice2:   []int{},
			expected: []int{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := Difference(tt.slice1, tt.slice2)
			assert.ElementsMatch(t, tt.expected, actual)
		})
	}
}

func TestInArray(t *testing.T) {
	tests := []struct {
		name     string
		needle   int
		haystack []int
		expected bool
	}{
		{
			name:     "Element is in array",
			needle:   2,
			haystack: []int{1, 2, 3},
			expected: true,
		},
		{
			name:     "Element is not in array",
			needle:   4,
			haystack: []int{1, 2, 3},
			expected: false,
		},
		{
			name:     "Empty array",
			needle:   1,
			haystack: []int{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := InArray(tt.needle, tt.haystack)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

package prompt

import (
	"encoding/json"
	"fmt"
	"strings"
)

// renderMapAsString converts a map to a formatted string.
func renderMapAsString(m map[string]any) (string, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Example represents a single example in few-shot learning.
type Example struct {
	Input  map[string]any
	Output any
}

// FewShotTemplate manages few-shot examples in prompt templates.
type FewShotTemplate struct {
	examples    []Example
	exampleTpl  *Template
	inputPrefix string
	sep         string
}

// NewFewShotTemplate creates a new few-shot template.
func NewFewShotTemplate() *FewShotTemplate {
	return &FewShotTemplate{
		examples:    make([]Example, 0),
		inputPrefix: "Input: ",
		sep:         "\n\n",
	}
}

// AddExample adds a static example.
func (fst *FewShotTemplate) AddExample(input map[string]any, output any) *FewShotTemplate {
	fst.examples = append(fst.examples, Example{Input: input, Output: output})
	return fst
}

// WithExampleTemplate sets a template for rendering examples.
// Variables like {{input.key}} and {{output}} are available.
func (fst *FewShotTemplate) WithExampleTemplate(tpl *Template) *FewShotTemplate {
	fst.exampleTpl = tpl
	return fst
}

// WithInputPrefix sets the prefix for input sections.
func (fst *FewShotTemplate) WithInputPrefix(prefix string) *FewShotTemplate {
	fst.inputPrefix = prefix
	return fst
}

// WithSeparator sets the separator between examples.
func (fst *FewShotTemplate) WithSeparator(sep string) *FewShotTemplate {
	fst.sep = sep
	return fst
}

// Render renders all examples as a string.
func (fst *FewShotTemplate) Render() (string, error) {
	if len(fst.examples) == 0 {
		return "", nil
	}

	var sb strings.Builder
	for i, ex := range fst.examples {
		if i > 0 {
			sb.WriteString(fst.sep)
		}

		if fst.exampleTpl != nil {
			// Use custom template
			data := map[string]any{
				"input":  ex.Input,
				"output": ex.Output,
				"@index": i,
				"@first": i == 0,
				"@last":  i == len(fst.examples)-1,
			}
			result, err := fst.exampleTpl.Render(data)
			if err != nil {
				return "", fmt.Errorf("render example %d: %w", i, err)
			}
			sb.WriteString(result)
		} else {
			// Use default rendering
			inputStr, _ := renderMapAsString(ex.Input)
			outputStr := fmt.Sprintf("%v", ex.Output)
			sb.WriteString(fst.inputPrefix)
			sb.WriteString(inputStr)
			sb.WriteString("\nOutput: ")
			sb.WriteString(outputStr)
		}
	}
	return sb.String(), nil
}

// AddExamplesFromData adds examples from a data source.
// Each item should have "input" and "output" keys.
func (fst *FewShotTemplate) AddExamplesFromData(data []map[string]any) *FewShotTemplate {
	for _, item := range data {
		input, _ := item["input"].(map[string]any)
		output := item["output"]
		fst.AddExample(input, output)
	}
	return fst
}

// ExampleCount returns the number of examples.
func (fst *FewShotTemplate) ExampleCount() int {
	return len(fst.examples)
}

// Merge combines another few-shot template.
func (fst *FewShotTemplate) Merge(other *FewShotTemplate) *FewShotTemplate {
	result := NewFewShotTemplate()
	result.examples = append(fst.examples, other.examples...)
	result.exampleTpl = fst.exampleTpl
	result.inputPrefix = fst.inputPrefix
	result.sep = fst.sep
	return result
}

// DynamicExampleProvider allows dynamic example selection.
type DynamicExampleProvider interface {
	GetExamples(ctx map[string]any) ([]Example, error)
}

// ExampleSelector selects examples based on input.
type ExampleSelector struct {
	providers []DynamicExampleProvider
	examples  []Example
	limit     int
}

// NewExampleSelector creates a new example selector.
func NewExampleSelector() *ExampleSelector {
	return &ExampleSelector{
		examples: make([]Example, 0),
		limit:    5,
	}
}

// AddExamples adds static examples.
func (es *ExampleSelector) AddExamples(examples ...Example) *ExampleSelector {
	es.examples = append(es.examples, examples...)
	return es
}

// AddProvider adds a dynamic example provider.
func (es *ExampleSelector) AddProvider(provider DynamicExampleProvider) *ExampleSelector {
	es.providers = append(es.providers, provider)
	return es
}

// WithLimit sets the maximum number of examples to select.
func (es *ExampleSelector) WithLimit(limit int) *ExampleSelector {
	es.limit = limit
	return es
}

// Select returns selected examples for the given context.
func (es *ExampleSelector) Select(ctx map[string]any) ([]Example, error) {
	var allExamples []Example

	// Get examples from static pool
	allExamples = append(allExamples, es.examples...)

	// Get examples from providers
	for _, provider := range es.providers {
		examples, err := provider.GetExamples(ctx)
		if err != nil {
			return nil, err
		}
		allExamples = append(allExamples, examples...)
	}

	// Apply limit
	if len(allExamples) > es.limit {
		allExamples = allExamples[:es.limit]
	}

	return allExamples, nil
}

// SemanticSimilaritySelector selects examples based on semantic similarity.
// This requires an embedding model.
type SemanticSimilaritySelector struct {
	selector *ExampleSelector
	embedder func(string) ([]float32, error)
}

// NewSemanticSimilaritySelector creates a new semantic similarity selector.
func NewSemanticSimilaritySelector(embedder func(string) ([]float32, error)) *SemanticSimilaritySelector {
	return &SemanticSimilaritySelector{
		selector: NewExampleSelector(),
		embedder: embedder,
	}
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (float32(float64(normA) * float64(normB)))
}

// SelectBySimilarity selects top-k most similar examples.
func (ss *SemanticSimilaritySelector) SelectBySimilarity(query string, k int) ([]Example, error) {
	if ss.embedder == nil {
		return nil, fmt.Errorf("embedder not set")
	}

	queryVec, err := ss.embedder(query)
	if err != nil {
		return nil, err
	}

	examples, err := ss.selector.Select(nil)
	if err != nil {
		return nil, err
	}

	// Score each example
	type scoredExample struct {
		example    Example
		similarity float32
	}
	var scored []scoredExample
	for _, ex := range examples {
		// Simple concatenation of input values for embedding
		inputStr, _ := renderMapAsString(ex.Input)
		exVec, err := ss.embedder(inputStr)
		if err != nil {
			continue
		}
		sim := cosineSimilarity(queryVec, exVec)
		scored = append(scored, scoredExample{ex, sim})
	}

	// Sort by similarity
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].similarity > scored[i].similarity {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Return top-k
	result := make([]Example, 0, k)
	for i := 0; i < k && i < len(scored); i++ {
		result = append(result, scored[i].example)
	}
	return result, nil
}

// ExampleFormatter formats examples for output.
type ExampleFormatter struct {
	inputFormat  string
	outputFormat string
	sep          string
}

// NewExampleFormatter creates a new example formatter.
func NewExampleFormatter() *ExampleFormatter {
	return &ExampleFormatter{
		inputFormat:  "Input: {{input}}\n",
		outputFormat: "Output: {{output}}",
		sep:          "\n\n",
	}
}

// WithInputFormat sets the input format template.
func (ef *ExampleFormatter) WithInputFormat(format string) *ExampleFormatter {
	ef.inputFormat = format
	return ef
}

// WithOutputFormat sets the output format template.
func (ef *ExampleFormatter) WithOutputFormat(format string) *ExampleFormatter {
	ef.outputFormat = format
	return ef
}

// WithSeparator sets the separator between examples.
func (ef *ExampleFormatter) WithSeparator(sep string) *ExampleFormatter {
	ef.sep = sep
	return ef
}

// Format formats a single example.
func (ef *ExampleFormatter) Format(example Example) (string, error) {
	inputJSON, _ := json.Marshal(example.Input)
	outputStr := fmt.Sprintf("%v", example.Output)

	inputTpl, _ := NewTemplate("input", ef.inputFormat)
	outputTpl, _ := NewTemplate("output", ef.outputFormat)

	inputResult, _ := inputTpl.Render(map[string]any{"input": string(inputJSON)})
	outputResult, _ := outputTpl.Render(map[string]any{"output": outputStr})

	return inputResult + outputResult, nil
}

// FormatAll formats multiple examples.
func (ef *ExampleFormatter) FormatAll(examples []Example) (string, error) {
	var sb strings.Builder
	for i, ex := range examples {
		if i > 0 {
			sb.WriteString(ef.sep)
		}
		formatted, err := ef.Format(ex)
		if err != nil {
			return "", err
		}
		sb.WriteString(formatted)
	}
	return sb.String(), nil
}

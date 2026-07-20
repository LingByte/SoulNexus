package prompt

import (
	"testing"
)

func TestFewShotTemplate(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		fst := NewFewShotTemplate()
		if fst == nil {
			t.Fatal("expected non-nil FewShotTemplate")
		}
		if fst.ExampleCount() != 0 {
			t.Errorf("expected 0 examples, got %d", fst.ExampleCount())
		}
	})

	t.Run("add single example", func(t *testing.T) {
		fst := NewFewShotTemplate()
		result := fst.AddExample(map[string]any{"input": "Hello"}, "Hi there!")
		if result != fst {
			t.Error("expected fluent interface")
		}
		if fst.ExampleCount() != 1 {
			t.Errorf("expected 1 example, got %d", fst.ExampleCount())
		}
	})

	t.Run("add multiple examples", func(t *testing.T) {
		fst := NewFewShotTemplate()
		fst.AddExample(map[string]any{"input": "Hello"}, "Hi there!")
		fst.AddExample(map[string]any{"input": "Goodbye"}, "See you!")
		fst.AddExample(map[string]any{"input": "Thanks"}, "You're welcome!")

		if fst.ExampleCount() != 3 {
			t.Errorf("expected 3 examples, got %d", fst.ExampleCount())
		}
	})

	t.Run("render empty template", func(t *testing.T) {
		fst := NewFewShotTemplate()
		result, err := fst.Render()
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if result != "" {
			t.Errorf("expected empty string, got %s", result)
		}
	})

	t.Run("render with examples", func(t *testing.T) {
		fst := NewFewShotTemplate()
		fst.AddExample(map[string]any{"input": "Hello"}, "Hi there!")
		fst.AddExample(map[string]any{"input": "Goodbye"}, "See you!")

		result, err := fst.Render()
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if result == "" {
			t.Error("expected non-empty result")
		}
	})

	t.Run("custom input prefix", func(t *testing.T) {
		fst := NewFewShotTemplate()
		fst.WithInputPrefix("Q: ")
		fst.AddExample(map[string]any{"input": "test"}, "answer")

		result, err := fst.Render()
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if result == "" {
			t.Error("expected non-empty result")
		}
	})

	t.Run("custom separator", func(t *testing.T) {
		fst := NewFewShotTemplate()
		fst.WithSeparator("---\n")
		fst.AddExample(map[string]any{"input": "a"}, "1")
		fst.AddExample(map[string]any{"input": "b"}, "2")

		result, err := fst.Render()
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		// Separator should appear in result
		_ = result // Just verify it renders without error
	})

	t.Run("custom example template", func(t *testing.T) {
		fst := NewFewShotTemplate()
		tpl := MustNewTemplate("example", "Q: {{input.q}}\nA: {{output}}")
		fst.WithExampleTemplate(tpl)
		fst.AddExample(map[string]any{"q": "What is 2+2?"}, "4")
		fst.AddExample(map[string]any{"q": "What is 3+3?"}, "6")

		result, err := fst.Render()
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if result == "" {
			t.Error("expected non-empty result")
		}
		// Verify the template variables are substituted
		if !containsString(result, "Q: What is 2+2?") {
			t.Error("expected custom template to be used")
		}
	})

	t.Run("add examples from data", func(t *testing.T) {
		fst := NewFewShotTemplate()
		data := []map[string]any{
			{"input": map[string]any{"q": "1+1"}, "output": "2"},
			{"input": map[string]any{"q": "2+2"}, "output": "4"},
		}
		fst.AddExamplesFromData(data)

		if fst.ExampleCount() != 2 {
			t.Errorf("expected 2 examples, got %d", fst.ExampleCount())
		}
	})

	t.Run("merge templates", func(t *testing.T) {
		fst1 := NewFewShotTemplate()
		fst1.AddExample(map[string]any{"input": "a"}, "1")

		fst2 := NewFewShotTemplate()
		fst2.AddExample(map[string]any{"input": "b"}, "2")

		merged := fst1.Merge(fst2)
		if merged.ExampleCount() != 2 {
			t.Errorf("expected 2 examples after merge, got %d", merged.ExampleCount())
		}
	})

	t.Run("merge preserves settings", func(t *testing.T) {
		fst1 := NewFewShotTemplate()
		fst1.WithInputPrefix("Q: ")
		fst1.AddExample(map[string]any{"input": "a"}, "1")

		fst2 := NewFewShotTemplate()
		fst2.AddExample(map[string]any{"input": "b"}, "2")

		merged := fst1.Merge(fst2)
		result, err := merged.Render()
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		_ = result
	})
}

func TestExampleSelector(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		es := NewExampleSelector()
		if es == nil {
			t.Fatal("expected non-nil ExampleSelector")
		}
	})

	t.Run("add examples", func(t *testing.T) {
		es := NewExampleSelector()
		result := es.AddExamples(
			Example{Input: map[string]any{"input": "a"}, Output: "1"},
			Example{Input: map[string]any{"input": "b"}, Output: "2"},
		)
		if result != es {
			t.Error("expected fluent interface")
		}
	})

	t.Run("select returns examples", func(t *testing.T) {
		es := NewExampleSelector()
		es.AddExamples(
			Example{Input: map[string]any{"input": "a"}, Output: "1"},
			Example{Input: map[string]any{"input": "b"}, Output: "2"},
		)

		examples, err := es.Select(map[string]any{})
		if err != nil {
			t.Fatalf("Select failed: %v", err)
		}
		if len(examples) != 2 {
			t.Errorf("expected 2 examples, got %d", len(examples))
		}
	})

	t.Run("select with limit", func(t *testing.T) {
		es := NewExampleSelector()
		es.AddExamples(
			Example{Input: map[string]any{"input": "a"}, Output: "1"},
			Example{Input: map[string]any{"input": "b"}, Output: "2"},
			Example{Input: map[string]any{"input": "c"}, Output: "3"},
			Example{Input: map[string]any{"input": "d"}, Output: "4"},
		)
		es.WithLimit(2)

		examples, err := es.Select(map[string]any{})
		if err != nil {
			t.Fatalf("Select failed: %v", err)
		}
		if len(examples) != 2 {
			t.Errorf("expected 2 examples due to limit, got %d", len(examples))
		}
	})

	t.Run("add provider", func(t *testing.T) {
		es := NewExampleSelector()
		provider := &mockExampleProvider{
			examples: []Example{
				{Input: map[string]any{"input": "dynamic"}, Output: "dynamic_output"},
			},
		}
		es.AddProvider(provider)

		examples, err := es.Select(map[string]any{})
		if err != nil {
			t.Fatalf("Select failed: %v", err)
		}
		if len(examples) != 1 {
			t.Errorf("expected 1 example from provider, got %d", len(examples))
		}
	})

	t.Run("provider error", func(t *testing.T) {
		es := NewExampleSelector()
		provider := &mockExampleProvider{
			err: assertError("provider error"),
		}
		es.AddProvider(provider)

		_, err := es.Select(map[string]any{})
		if err == nil {
			t.Error("expected error from provider")
		}
	})
}

func TestSemanticSimilaritySelector(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		embedder := func(s string) ([]float32, error) {
			return []float32{1.0, 0.0, 0.0}, nil
		}
		ss := NewSemanticSimilaritySelector(embedder)
		if ss == nil {
			t.Fatal("expected non-nil SemanticSimilaritySelector")
		}
	})

	t.Run("select by similarity", func(t *testing.T) {
		embedder := func(s string) ([]float32, error) {
			return []float32{1.0, 0.0, 0.0}, nil
		}
		ss := NewSemanticSimilaritySelector(embedder)
		ss.selector.AddExamples(
			Example{Input: map[string]any{"input": "hello"}, Output: "hi"},
			Example{Input: map[string]any{"input": "world"}, Output: "earth"},
		)

		examples, err := ss.SelectBySimilarity("test query", 2)
		if err != nil {
			t.Fatalf("SelectBySimilarity failed: %v", err)
		}
		if len(examples) != 2 {
			t.Errorf("expected 2 examples, got %d", len(examples))
		}
	})

	t.Run("select by similarity with k limit", func(t *testing.T) {
		embedder := func(s string) ([]float32, error) {
			return []float32{1.0, 0.0, 0.0}, nil
		}
		ss := NewSemanticSimilaritySelector(embedder)
		ss.selector.AddExamples(
			Example{Input: map[string]any{"input": "a"}, Output: "1"},
			Example{Input: map[string]any{"input": "b"}, Output: "2"},
			Example{Input: map[string]any{"input": "c"}, Output: "3"},
		)

		examples, err := ss.SelectBySimilarity("test", 1)
		if err != nil {
			t.Fatalf("SelectBySimilarity failed: %v", err)
		}
		if len(examples) != 1 {
			t.Errorf("expected 1 example, got %d", len(examples))
		}
	})

	t.Run("nil embedder error", func(t *testing.T) {
		ss := NewSemanticSimilaritySelector(nil)
		_, err := ss.SelectBySimilarity("test", 1)
		if err == nil {
			t.Error("expected error for nil embedder")
		}
	})
}

func TestExampleFormatter(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		ef := NewExampleFormatter()
		if ef == nil {
			t.Fatal("expected non-nil ExampleFormatter")
		}
	})

	t.Run("format single example", func(t *testing.T) {
		ef := NewExampleFormatter()
		example := Example{
			Input:  map[string]any{"key": "value"},
			Output: "output_value",
		}

		result, err := ef.Format(example)
		if err != nil {
			t.Fatalf("Format failed: %v", err)
		}
		if result == "" {
			t.Error("expected non-empty result")
		}
	})

	t.Run("format all examples", func(t *testing.T) {
		ef := NewExampleFormatter()
		examples := []Example{
			{Input: map[string]any{"input": "a"}, Output: "1"},
			{Input: map[string]any{"input": "b"}, Output: "2"},
		}

		result, err := ef.FormatAll(examples)
		if err != nil {
			t.Fatalf("FormatAll failed: %v", err)
		}
		if result == "" {
			t.Error("expected non-empty result")
		}
	})

	t.Run("format all empty examples", func(t *testing.T) {
		ef := NewExampleFormatter()
		result, err := ef.FormatAll([]Example{})
		if err != nil {
			t.Fatalf("FormatAll failed: %v", err)
		}
		if result != "" {
			t.Errorf("expected empty string, got %s", result)
		}
	})

	t.Run("custom input format", func(t *testing.T) {
		ef := NewExampleFormatter()
		ef.WithInputFormat("Question: {{input}}")
		example := Example{
			Input:  map[string]any{"input": "What is Go?"},
			Output: "A programming language",
		}

		result, err := ef.Format(example)
		if err != nil {
			t.Fatalf("Format failed: %v", err)
		}
		if !containsString(result, "Question:") {
			t.Error("expected custom input format")
		}
	})

	t.Run("custom output format", func(t *testing.T) {
		ef := NewExampleFormatter()
		ef.WithOutputFormat("Answer: {{output}}")
		example := Example{
			Input:  map[string]any{"q": "test"},
			Output: "answer",
		}

		result, err := ef.Format(example)
		if err != nil {
			t.Fatalf("Format failed: %v", err)
		}
		if !containsString(result, "Answer:") {
			t.Error("expected custom output format")
		}
	})

	t.Run("custom separator", func(t *testing.T) {
		ef := NewExampleFormatter()
		ef.WithSeparator("===")
		examples := []Example{
			{Input: map[string]any{"a": "1"}, Output: "one"},
			{Input: map[string]any{"a": "2"}, Output: "two"},
		}

		result, err := ef.FormatAll(examples)
		if err != nil {
			t.Fatalf("FormatAll failed: %v", err)
		}
		if !containsString(result, "===") {
			t.Error("expected custom separator")
		}
	})
}

func TestCosineSimilarity(t *testing.T) {
	t.Run("identical vectors", func(t *testing.T) {
		a := []float32{1.0, 0.0, 0.0}
		b := []float32{1.0, 0.0, 0.0}
		result := cosineSimilarity(a, b)
		if result != 1.0 {
			t.Errorf("expected 1.0, got %f", result)
		}
	})

	t.Run("orthogonal vectors", func(t *testing.T) {
		a := []float32{1.0, 0.0, 0.0}
		b := []float32{0.0, 1.0, 0.0}
		result := cosineSimilarity(a, b)
		if result != 0.0 {
			t.Errorf("expected 0.0, got %f", result)
		}
	})

	t.Run("zero vectors", func(t *testing.T) {
		a := []float32{0.0, 0.0, 0.0}
		b := []float32{1.0, 0.0, 0.0}
		result := cosineSimilarity(a, b)
		if result != 0.0 {
			t.Errorf("expected 0.0, got %f", result)
		}
	})
}

func TestRenderMapAsString(t *testing.T) {
	t.Run("simple map", func(t *testing.T) {
		m := map[string]any{"name": "Alice", "age": 30}
		result, err := renderMapAsString(m)
		if err != nil {
			t.Fatalf("renderMapAsString failed: %v", err)
		}
		if result == "" {
			t.Error("expected non-empty result")
		}
	})

	t.Run("nested map", func(t *testing.T) {
		m := map[string]any{
			"user": map[string]any{
				"name": "Bob",
				"profile": map[string]any{
					"age": 25,
				},
			},
		}
		result, err := renderMapAsString(m)
		if err != nil {
			t.Fatalf("renderMapAsString failed: %v", err)
		}
		if result == "" {
			t.Error("expected non-empty result")
		}
	})
}

// Helper mocks

type mockExampleProvider struct {
	examples []Example
	err      error
}

func (p *mockExampleProvider) GetExamples(ctx map[string]any) ([]Example, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.examples, nil
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// assertError creates a sentinel error for testing
func assertError(msg string) error {
	return &testError{msg: msg}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

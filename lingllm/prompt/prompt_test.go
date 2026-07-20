package prompt

import (
	"testing"
)

func TestNewTemplate(t *testing.T) {
	t.Run("simple variable", func(t *testing.T) {
		tpl, err := NewTemplate("test", "Hello, {{name}}!")
		if err != nil {
			t.Fatalf("NewTemplate failed: %v", err)
		}
		if tpl.Name() != "test" {
			t.Errorf("unexpected name: %s", tpl.Name())
		}
	})

	t.Run("empty template", func(t *testing.T) {
		_, err := NewTemplate("empty", "")
		if err != nil {
			t.Fatalf("NewTemplate with empty source should not error")
		}
	})
}

func TestTemplateRender(t *testing.T) {
	t.Run("simple variable", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Hello, {{name}}!")
		result, err := tpl.Render(map[string]any{"name": "World"})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if result != "Hello, World!" {
			t.Errorf("unexpected result: %s", result)
		}
	})

	t.Run("multiple variables", func(t *testing.T) {
		tpl := MustNewTemplate("test", "{{greeting}}, {{name}}!")
		result, err := tpl.Render(map[string]any{"greeting": "Hello", "name": "Alice"})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if result != "Hello, Alice!" {
			t.Errorf("unexpected result: %s", result)
		}
	})

	t.Run("nested variable", func(t *testing.T) {
		tpl := MustNewTemplate("test", "User: {{user.name}}")
		result, err := tpl.Render(map[string]any{
			"user": map[string]any{"name": "Bob"},
		})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if result != "User: Bob" {
			t.Errorf("unexpected result: %s", result)
		}
	})

	t.Run("missing variable", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Hello, {{name}}!")
		result, err := tpl.Render(map[string]any{})
		if err != nil {
			t.Fatalf("Render should not error for missing variable")
		}
		if result != "Hello, !" {
			t.Errorf("unexpected result: %s", result)
		}
	})

	t.Run("default value", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Hello, {{name|Guest}}!")
		result, err := tpl.Render(map[string]any{})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if result != "Hello, Guest!" {
			t.Errorf("unexpected result: %s", result)
		}
	})

	t.Run("array access", func(t *testing.T) {
		tpl := MustNewTemplate("test", "First: {{first}}")
		result, err := tpl.Render(map[string]any{
			"first": "apple",
		})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if result != "First: apple" {
			t.Errorf("unexpected result: %s", result)
		}
	})

	t.Run("plain text", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Just plain text")
		result, err := tpl.Render(map[string]any{})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if result != "Just plain text" {
			t.Errorf("unexpected result: %s", result)
		}
	})
}

func TestTemplateValidate(t *testing.T) {
	t.Run("valid template", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Hello, {{name}}!")
		errors := tpl.Validate()
		if len(errors) > 0 {
			t.Errorf("unexpected validation errors: %v", errors)
		}
	})

	t.Run("empty variable path", func(t *testing.T) {
		tpl := MustNewTemplate("test", "Hello, {{}}!")
		errors := tpl.Validate()
		if len(errors) == 0 {
			t.Error("expected validation error for empty variable path")
		}
	})
}

func TestTemplateSet(t *testing.T) {
	t.Run("register and get", func(t *testing.T) {
		ts := NewTemplateSet()
		tpl := MustNewTemplate("test", "Hello, {{name}}!")
		ts.Register("greeting", tpl)

		got, ok := ts.Get("greeting")
		if !ok {
			t.Fatal("template not found")
		}
		if got.Name() != "test" {
			t.Errorf("unexpected name: %s", got.Name())
		}
	})

	t.Run("render", func(t *testing.T) {
		ts := NewTemplateSet()
		err := ts.ParseAndRegister("greeting", "Hello, {{name}}!")
		if err != nil {
			t.Fatalf("ParseAndRegister failed: %v", err)
		}

		result, err := ts.Render("greeting", map[string]any{"name": "Bob"})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if result != "Hello, Bob!" {
			t.Errorf("unexpected result: %s", result)
		}
	})

	t.Run("must get panic", func(t *testing.T) {
		ts := NewTemplateSet()
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for missing template")
			}
		}()
		ts.MustGet("nonexistent")
	})

	t.Run("list templates", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.ParseAndRegister("t1", "{{a}}")
		ts.ParseAndRegister("t2", "{{b}}")

		names := ts.List()
		if len(names) != 2 {
			t.Errorf("unexpected count: %d", len(names))
		}
	})
}

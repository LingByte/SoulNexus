package prompt

import (
	"strings"
	"sync"
	"testing"
)

func TestTemplateSetBasics(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		ts := NewTemplateSet()
		if ts == nil {
			t.Fatal("expected non-nil TemplateSet")
		}
		if ts.templates == nil {
			t.Error("expected templates map to be initialized")
		}
		if ts.partials == nil {
			t.Error("expected partials map to be initialized")
		}
	})

	t.Run("register and get", func(t *testing.T) {
		ts := NewTemplateSet()
		tpl := MustNewTemplate("test", "Hello, {{name}}!")
		ts.Register("greeting", tpl)

		got, ok := ts.Get("greeting")
		if !ok {
			t.Fatal("expected to find registered template")
		}
		if got.Name() != "test" {
			t.Errorf("expected name 'test', got '%s'", got.Name())
		}
	})

	t.Run("get nonexistent", func(t *testing.T) {
		ts := NewTemplateSet()
		_, ok := ts.Get("nonexistent")
		if ok {
			t.Error("expected not to find nonexistent template")
		}
	})

	t.Run("must get success", func(t *testing.T) {
		ts := NewTemplateSet()
		tpl := MustNewTemplate("test", "Hello")
		ts.Register("greeting", tpl)

		got := ts.MustGet("greeting")
		if got.Name() != "test" {
			t.Errorf("expected name 'test', got '%s'", got.Name())
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

	t.Run("register partial", func(t *testing.T) {
		ts := NewTemplateSet()
		tpl := MustNewTemplate("partial", "Partial content")
		ts.RegisterPartial("header", tpl)

		ts.mu.RLock()
		defer ts.mu.RUnlock()
		if _, ok := ts.partials["header"]; !ok {
			t.Error("expected partial to be registered")
		}
	})
}

func TestTemplateSetFuncs(t *testing.T) {
	t.Run("register custom func", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.RegisterFunc("double", func(n int) int { return n * 2 })

		ts.mu.RLock()
		defer ts.mu.RUnlock()
		if _, ok := ts.funcs["double"]; !ok {
			t.Error("expected function to be registered")
		}
	})

	t.Run("add default funcs", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.AddDefaultFuncs()

		ts.mu.RLock()
		defer ts.mu.RUnlock()
		if _, ok := ts.funcs["upper"]; !ok {
			t.Error("expected 'upper' function")
		}
		if _, ok := ts.funcs["lower"]; !ok {
			t.Error("expected 'lower' function")
		}
		if _, ok := ts.funcs["trim"]; !ok {
			t.Error("expected 'trim' function")
		}
		if _, ok := ts.funcs["len"]; !ok {
			t.Error("expected 'len' function")
		}
		if _, ok := ts.funcs["default"]; !ok {
			t.Error("expected 'default' function")
		}
		if _, ok := ts.funcs["ternary"]; !ok {
			t.Error("expected 'ternary' function")
		}
	})
}

func TestTemplateSetRender(t *testing.T) {
	t.Run("parse and register", func(t *testing.T) {
		ts := NewTemplateSet()
		err := ts.ParseAndRegister("greeting", "Hello, {{name}}!")
		if err != nil {
			t.Fatalf("ParseAndRegister failed: %v", err)
		}

		got, ok := ts.Get("greeting")
		if !ok {
			t.Fatal("expected to find template")
		}
		if got.Name() != "greeting" {
			t.Errorf("expected name 'greeting', got '%s'", got.Name())
		}
	})

	t.Run("render named template", func(t *testing.T) {
		ts := NewTemplateSet()
		err := ts.ParseAndRegister("greeting", "Hello, {{name}}!")
		if err != nil {
			t.Fatalf("ParseAndRegister failed: %v", err)
		}

		result, err := ts.Render("greeting", map[string]any{"name": "Alice"})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if result != "Hello, Alice!" {
			t.Errorf("expected 'Hello, Alice!', got '%s'", result)
		}
	})

	t.Run("render nonexistent", func(t *testing.T) {
		ts := NewTemplateSet()
		_, err := ts.Render("nonexistent", map[string]any{})
		if err == nil {
			t.Error("expected error for nonexistent template")
		}
	})

	t.Run("must render", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.ParseAndRegister("test", "Hello, {{name}}!")

		result := ts.MustRender("test", map[string]any{"name": "Bob"})
		if result != "Hello, Bob!" {
			t.Errorf("expected 'Hello, Bob!', got '%s'", result)
		}
	})

	t.Run("must render panic", func(t *testing.T) {
		ts := NewTemplateSet()
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for missing template")
			}
		}()
		ts.MustRender("nonexistent", map[string]any{})
	})
}

func TestTemplateSetExtend(t *testing.T) {
	t.Run("extend parent template", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.ParseAndRegister("base", "Hello, {{name}}! {{$super}}")

		err := ts.Extend("child", "base", "Welcome to the app.")
		if err != nil {
			t.Fatalf("Extend failed: %v", err)
		}

		child, ok := ts.Get("child")
		if !ok {
			t.Fatal("expected child template")
		}
		if !containsString(child.Source(), "Welcome") {
			t.Error("expected child to include new content")
		}
	})

	t.Run("extend nonexistent parent", func(t *testing.T) {
		ts := NewTemplateSet()
		err := ts.Extend("child", "nonexistent", "content")
		if err == nil {
			t.Error("expected error for nonexistent parent")
		}
	})
}

func TestTemplateSetList(t *testing.T) {
	t.Run("list empty set", func(t *testing.T) {
		ts := NewTemplateSet()
		names := ts.List()
		if len(names) != 0 {
			t.Errorf("expected empty list, got %d", len(names))
		}
	})

	t.Run("list with templates", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.ParseAndRegister("a", "{{a}}")
		ts.ParseAndRegister("b", "{{b}}")
		ts.ParseAndRegister("c", "{{c}}")

		names := ts.List()
		if len(names) != 3 {
			t.Errorf("expected 3 templates, got %d", len(names))
		}
	})
}

func TestTemplateSetValidateAll(t *testing.T) {
	t.Run("validate all valid templates", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.ParseAndRegister("t1", "Hello, {{name}}!")
		ts.ParseAndRegister("t2", "Goodbye, {{name}}!")

		results := ts.ValidateAll()
		if len(results) != 0 {
			t.Errorf("expected no errors, got %d", len(results))
		}
	})

	t.Run("validate with invalid template", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.ParseAndRegister("valid", "Hello, {{name}}!")
		ts.ParseAndRegister("invalid", "Hello, {{}}!")

		results := ts.ValidateAll()
		if len(results) != 1 {
			t.Errorf("expected 1 template with errors, got %d", len(results))
		}
	})
}

func TestGlobalRegistry(t *testing.T) {
	t.Run("register global template", func(t *testing.T) {
		// Save original registry
		original := TemplateRegistry
		defer func() { TemplateRegistry = original }()

		TemplateRegistry = NewTemplateSet()
		tpl := MustNewTemplate("test", "Hello")
		RegisterGlobal("greeting", tpl)

		got, ok := TemplateRegistry.Get("greeting")
		if !ok {
			t.Fatal("expected to find global template")
		}
		if got.Name() != "test" {
			t.Errorf("expected name 'test', got '%s'", got.Name())
		}
	})

	t.Run("parse and register global", func(t *testing.T) {
		original := TemplateRegistry
		defer func() { TemplateRegistry = original }()

		TemplateRegistry = NewTemplateSet()
		err := ParseAndRegisterGlobal("hello", "Hello, {{name}}!")
		if err != nil {
			t.Fatalf("ParseAndRegisterGlobal failed: %v", err)
		}

		result, err := RenderGlobal("hello", map[string]any{"name": "World"})
		if err != nil {
			t.Fatalf("RenderGlobal failed: %v", err)
		}
		if result != "Hello, World!" {
			t.Errorf("expected 'Hello, World!', got '%s'", result)
		}
	})

	t.Run("render global nonexistent", func(t *testing.T) {
		original := TemplateRegistry
		defer func() { TemplateRegistry = original }()

		TemplateRegistry = NewTemplateSet()
		_, err := RenderGlobal("nonexistent", map[string]any{})
		if err == nil {
			t.Error("expected error for nonexistent template")
		}
	})
}

func TestMapFuncs(t *testing.T) {
	t.Run("default function behavior", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.AddDefaultFuncs()

		ts.mu.RLock()
		upperFn := ts.funcs["upper"].(func(string) string)
		ts.mu.RUnlock()

		result := upperFn("hello")
		if result != "HELLO" {
			t.Errorf("expected 'HELLO', got '%s'", result)
		}
	})

	t.Run("ternary function true", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.AddDefaultFuncs()

		ts.mu.RLock()
		ternaryFn := ts.funcs["ternary"].(func(bool, any, any) any)
		ts.mu.RUnlock()

		result := ternaryFn(true, "yes", "no")
		if result != "yes" {
			t.Errorf("expected 'yes', got '%v'", result)
		}
	})

	t.Run("ternary function false", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.AddDefaultFuncs()

		ts.mu.RLock()
		ternaryFn := ts.funcs["ternary"].(func(bool, any, any) any)
		ts.mu.RUnlock()

		result := ternaryFn(false, "yes", "no")
		if result != "no" {
			t.Errorf("expected 'no', got '%v'", result)
		}
	})

	t.Run("default function with value", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.AddDefaultFuncs()

		ts.mu.RLock()
		defaultFn := ts.funcs["default"].(func(any, any) any)
		ts.mu.RUnlock()

		result := defaultFn("value", "default")
		if result != "value" {
			t.Errorf("expected 'value', got '%v'", result)
		}
	})

	t.Run("default function with nil", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.AddDefaultFuncs()

		ts.mu.RLock()
		defaultFn := ts.funcs["default"].(func(any, any) any)
		ts.mu.RUnlock()

		result := defaultFn(nil, "default")
		if result != "default" {
			t.Errorf("expected 'default', got '%v'", result)
		}
	})

	t.Run("default function with empty string", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.AddDefaultFuncs()

		ts.mu.RLock()
		defaultFn := ts.funcs["default"].(func(any, any) any)
		ts.mu.RUnlock()

		result := defaultFn("", "default")
		if result != "default" {
			t.Errorf("expected 'default', got '%v'", result)
		}
	})
}

func TestTemplateSetConcurrency(t *testing.T) {
	t.Run("concurrent reads and writes", func(t *testing.T) {
		ts := NewTemplateSet()
		ts.ParseAndRegister("initial", "Hello")

		var wg sync.WaitGroup
		wg.Add(4)

		// Concurrent writes
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				ts.ParseAndRegister("template1", strings.Repeat("a", i))
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				ts.ParseAndRegister("template2", strings.Repeat("b", i))
			}
		}()

		// Concurrent reads
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				ts.Get("template1")
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				ts.List()
			}
		}()

		wg.Wait()
	})
}

package prompt

import (
	"fmt"
	"strings"
	"sync"
)

// TemplateSet manages a collection of templates with shared partials.
type TemplateSet struct {
	templates map[string]*Template
	partials  map[string]*Template
	funcs     MapFuncs
	mu        sync.RWMutex
}

// MapFuncs contains custom functions available in templates.
type MapFuncs map[string]any

// NewTemplateSet creates a new template set.
func NewTemplateSet() *TemplateSet {
	return &TemplateSet{
		templates: make(map[string]*Template),
		partials:  make(map[string]*Template),
		funcs:     make(MapFuncs),
	}
}

// Register registers a template in the set.
func (ts *TemplateSet) Register(name string, tpl *Template) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.templates[name] = tpl
}

// RegisterPartial registers a partial template that can be included.
func (ts *TemplateSet) RegisterPartial(name string, tpl *Template) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.partials[name] = tpl
}

// Get retrieves a template by name.
func (ts *TemplateSet) Get(name string) (*Template, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	tpl, ok := ts.templates[name]
	return tpl, ok
}

// MustGet retrieves a template or panics.
func (ts *TemplateSet) MustGet(name string) *Template {
	tpl, ok := ts.Get(name)
	if !ok {
		panic(fmt.Sprintf("template %q not found", name))
	}
	return tpl
}

// RegisterFunc registers a custom function for use in templates.
func (ts *TemplateSet) RegisterFunc(name string, fn any) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.funcs[name] = fn
}

// AddDefaultFuncs adds default utility functions.
func (ts *TemplateSet) AddDefaultFuncs() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.funcs["upper"] = strings.ToUpper
	ts.funcs["lower"] = strings.ToLower
	ts.funcs["trim"] = strings.TrimSpace
	ts.funcs["len"] = func(s string) int { return len(s) }
	ts.funcs["default"] = func(v, def any) any {
		if v == nil || v == "" {
			return def
		}
		return v
	}
	ts.funcs["ternary"] = func(cond bool, a, b any) any {
		if cond {
			return a
		}
		return b
	}
}

// ParseAndRegister parses a source string and registers it as a template.
func (ts *TemplateSet) ParseAndRegister(name, source string) error {
	tpl, err := NewTemplate(name, source)
	if err != nil {
		return err
	}
	ts.Register(name, tpl)
	return nil
}

// Render renders a named template with data.
func (ts *TemplateSet) Render(name string, data map[string]any) (string, error) {
	tpl, ok := ts.Get(name)
	if !ok {
		return "", fmt.Errorf("template %q not found", name)
	}
	return tpl.Render(data)
}

// MustRender renders a named template and panics on error.
func (ts *TemplateSet) MustRender(name string, data map[string]any) string {
	result, err := ts.Render(name, data)
	if err != nil {
		panic(err)
	}
	return result
}

// Extend creates a new template by extending a parent.
// The parent block can be referenced as {{$super}}.
func (ts *TemplateSet) Extend(name, parentName, childSource string) error {
	parent, ok := ts.Get(parentName)
	if !ok {
		return fmt.Errorf("parent template %q not found", parentName)
	}

	// Create child template with parent's source
	childSource = strings.Replace(childSource, "{{$super}}", parent.source, -1)
	return ts.ParseAndRegister(name, childSource)
}

// List returns all registered template names.
func (ts *TemplateSet) List() []string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	names := make([]string, 0, len(ts.templates))
	for name := range ts.templates {
		names = append(names, name)
	}
	return names
}

// ValidateAll validates all templates in the set.
func (ts *TemplateSet) ValidateAll() map[string][]ValidationError {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	results := make(map[string][]ValidationError)
	for name, tpl := range ts.templates {
		if errors := tpl.Validate(); len(errors) > 0 {
			results[name] = errors
		}
	}
	return results
}

// TemplateRegistry is a global template registry.
// Use RegisterGlobal/ParseAndRegisterGlobal for convenience.
var TemplateRegistry = NewTemplateSet()

// RegisterGlobal registers a template in the global registry.
func RegisterGlobal(name string, tpl *Template) {
	TemplateRegistry.Register(name, tpl)
}

// ParseAndRegisterGlobal parses and registers in global registry.
func ParseAndRegisterGlobal(name, source string) error {
	return TemplateRegistry.ParseAndRegister(name, source)
}

// RenderGlobal renders a template from the global registry.
func RenderGlobal(name string, data map[string]any) (string, error) {
	return TemplateRegistry.Render(name, data)
}

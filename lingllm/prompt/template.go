package prompt

import (
	"fmt"
	"strings"
	"unicode"
)

// Template represents a compiled prompt template.
type Template struct {
	name       string
	source     string
	blocks     []Block
	strictMode bool
}

// Block represents a parsed block in the template.
type Block interface {
	String() string
}

// TextBlock represents static text content.
type TextBlock struct {
	Content string
}

func (b *TextBlock) String() string { return b.Content }

// VariableBlock represents a variable substitution.
type VariableBlock struct {
	Path    string
	Default string
}

// Interpolate evaluates the block and returns the result.
func (b *VariableBlock) Interpolate(data map[string]any) string {
	val := getNestedValue(data, b.Path)
	if val == nil {
		if b.Default != "" {
			return b.Default
		}
		return ""
	}
	return fmt.Sprintf("%v", val)
}

func (b *VariableBlock) String() string {
	if b.Default != "" {
		return "{{" + b.Path + "|" + b.Default + "}}"
	}
	return "{{" + b.Path + "}}"
}

// IfBlock represents conditional logic.
type IfBlock struct {
	Condition  string
	ThenBlocks []Block
	ElseBlocks []Block
	Negate     bool
}

func (b *IfBlock) String() string {
	prefix := "{{"
	if b.Negate {
		prefix += "^"
	}
	then := blockListToString(b.ThenBlocks)
	elsePart := ""
	if len(b.ElseBlocks) > 0 {
		elsePart = "{{else}}" + blockListToString(b.ElseBlocks)
	}
	return prefix + "#if " + b.Condition + "}}" + then + elsePart + "{{/if}}"
}

// EachBlock represents iteration over a collection.
type EachBlock struct {
	Path        string
	ItemName    string
	KeyName     string
	InnerBlocks []Block
}

func (b *EachBlock) String() string {
	return "{{#each " + b.Path + "}}" + blockListToString(b.InnerBlocks) + "{{/each}}"
}

// IncludeBlock represents template composition (include another template).
type IncludeBlock struct {
	TemplateName string
	Params       map[string]any
}

func (b *IncludeBlock) String() string {
	return "{{>" + b.TemplateName + "}}"
}

// SwitchBlock represents switch/case logic.
type SwitchBlock struct {
	Variable string
	Cases    []CaseBlock
	Default  []Block
}

// CaseBlock represents a single case in switch.
type CaseBlock struct {
	Value  string
	Blocks []Block
}

// Helper to convert block list to string.
func blockListToString(blocks []Block) string {
	var sb strings.Builder
	for _, b := range blocks {
		sb.WriteString(b.String())
	}
	return sb.String()
}

// getNestedValue retrieves nested values from a map using dot notation.
// Supports: "user.name", "items[0]", "data[0].name"
func getNestedValue(data map[string]any, path string) any {
	// Handle direct array access like "[0]"
	if len(path) > 2 && path[0] == '[' {
		if idx := parseArrayIndex(path); idx >= 0 {
			if arr, ok := data[""].([]any); ok && idx < len(arr) {
				return arr[idx]
			}
		}
	}

	// Split path by '.' but also handle array access within parts
	parts := splitWithArrayAccess(path)
	current := any(data)

	for _, part := range parts {
		// Check if this part has array access
		if idx := strings.Index(part, "["); idx >= 0 && strings.HasSuffix(part, "]") {
			// It's an array access like "items[0]"
			key := part[:idx]
			arrIdx := parseArrayIndex(part[idx:])
			if m, ok := current.(map[string]any); ok {
				val := m[key]
				if arr, ok := val.([]any); ok && arrIdx >= 0 && arrIdx < len(arr) {
					current = arr[arrIdx]
				} else {
					return nil
				}
			} else {
				return nil
			}
		} else {
			// Regular key access
			if m, ok := current.(map[string]any); ok {
				current = m[part]
			} else {
				return nil
			}
		}
	}
	return current
}

// splitWithArrayAccess splits by '.' while preserving array access notation.
func splitWithArrayAccess(path string) []string {
	var parts []string
	var current strings.Builder
	for i := 0; i < len(path); i++ {
		if path[i] == '.' && (i == 0 || path[i-1] != '[') {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(path[i])
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// parseArrayIndex extracts numeric index from "name[index]" format or just "[index]".
func parseArrayIndex(s string) int {
	// Handle "[index]" format
	if s[0] == '[' && s[len(s)-1] == ']' {
		n := 0
		for _, c := range s[1 : len(s)-1] {
			if c < '0' || c > '9' {
				return -1
			}
			n = n*10 + int(c-'0')
		}
		return n
	}

	// Handle "name[index]" format
	for i := 0; i < len(s); i++ {
		if s[i] == '[' && s[len(s)-1] == ']' {
			n := 0
			for _, c := range s[i+1 : len(s)-1] {
				if c < '0' || c > '9' {
					return -1
				}
				n = n*10 + int(c-'0')
			}
			return n
		}
	}
	return -1
}

// NewTemplate creates a new template from a source string.
func NewTemplate(name, source string) (*Template, error) {
	tpl := &Template{
		name:   name,
		source: source,
	}
	if err := tpl.parse(); err != nil {
		return nil, fmt.Errorf("template parse error: %w", err)
	}
	return tpl, nil
}

// MustNewTemplate creates a new template and panics on error.
func MustNewTemplate(name, source string) *Template {
	tpl, err := NewTemplate(name, source)
	if err != nil {
		panic(err)
	}
	return tpl
}

// Name returns the template name.
func (t *Template) Name() string { return t.name }

// Source returns the original source string.
func (t *Template) Source() string { return t.source }

// parse parses the template source into blocks.
func (t *Template) parse() error {
	blocks, err := parseTemplate(t.source)
	if err != nil {
		return err
	}
	t.blocks = blocks
	return nil
}

// parseTemplate is a simple template parser that handles {{variable}} and {{variable|default}}.
func parseTemplate(source string) ([]Block, error) {
	var blocks []Block
	i := 0

	for i < len(source) {
		// Find the next template expression
		start := strings.Index(source[i:], "{{")
		if start == -1 {
			// No more expressions, rest is text
			if i < len(source) {
				blocks = append(blocks, &TextBlock{Content: source[i:]})
			}
			break
		}

		// Add text before the expression
		if start > 0 || i > 0 {
			text := source[i : i+start]
			if text != "" {
				blocks = append(blocks, &TextBlock{Content: text})
			}
		}

		// Find the closing }}
		remaining := source[i+start+2:]
		endIdx := findMatchingBraceSimple(remaining, "{{", "}}")
		if endIdx == -1 {
			// No closing brace, treat rest as text
			blocks = append(blocks, &TextBlock{Content: source[i:]})
			break
		}

		// Extract expression content
		content := remaining[:endIdx]

		// Parse the expression
		if block, err := parseExpressionBlock(content); err == nil && block != nil {
			blocks = append(blocks, block)
		}

		// Move past the expression
		i = i + start + 2 + endIdx + 2
	}

	return blocks, nil
}

// findMatchingBraceSimple finds the matching closing brace.
func findMatchingBraceSimple(s, open, close string) int {
	depth := 1
	for i := 0; i < len(s) && depth > 0; i++ {
		if i+1 < len(s) && s[i] == open[0] && s[i+1] == open[1] {
			depth++
			i++
			continue
		}
		if i+1 < len(s) && s[i] == close[0] && s[i+1] == close[1] {
			depth--
			if depth == 0 {
				return i
			}
			i++
			continue
		}
	}
	return -1
}

// parseExpressionBlock parses a template expression.
func parseExpressionBlock(content string) (Block, error) {
	content = strings.TrimSpace(content)

	// Skip control structures for now (if, each, etc.)
	// Focus on variable substitution
	if strings.HasPrefix(content, "#") || strings.HasPrefix(content, "^") || strings.HasPrefix(content, ">") {
		return nil, nil // Skip unsupported block types for now
	}

	// Handle variable with default: {{name|default}}
	if idx := strings.Index(content, "|"); idx >= 0 {
		path := strings.TrimSpace(content[:idx])
		defaultVal := strings.TrimSpace(content[idx+1:])
		return &VariableBlock{Path: path, Default: defaultVal}, nil
	}

	// Simple variable: {{name}}
	return &VariableBlock{Path: content}, nil
}

// Render renders the template with the given data.
func (t *Template) Render(data map[string]any) (string, error) {
	var sb strings.Builder
	for _, block := range t.blocks {
		if err := t.renderBlock(&sb, block, data); err != nil {
			return "", err
		}
	}
	return sb.String(), nil
}

// renderBlock renders a single block.
func (t *Template) renderBlock(sb *strings.Builder, block Block, data map[string]any) error {
	switch b := block.(type) {
	case *TextBlock:
		sb.WriteString(b.Content)

	case *VariableBlock:
		sb.WriteString(b.Interpolate(data))

	case *IfBlock:
		condition := evaluateCondition(data, b.Condition, b.Negate)
		var blocksToRender []Block
		if condition {
			blocksToRender = b.ThenBlocks
		} else {
			blocksToRender = b.ElseBlocks
		}
		for _, inner := range blocksToRender {
			if err := t.renderBlock(sb, inner, data); err != nil {
				return err
			}
		}

	case *EachBlock:
		val := getNestedValue(data, b.Path)
		if val == nil {
			return nil
		}

		switch arr := val.(type) {
		case []any:
			for i, item := range arr {
				itemData := map[string]any{
					b.ItemName: item,
					"@index":   i,
					"@first":   i == 0,
					"@last":    i == len(arr)-1,
				}
				if b.KeyName != "" {
					itemData[b.KeyName] = i
				}
				for _, inner := range b.InnerBlocks {
					if err := t.renderBlock(sb, inner, itemData); err != nil {
						return err
					}
				}
			}
		case map[string]any:
			keys := make([]string, 0, len(arr))
			for k := range arr {
				keys = append(keys, k)
			}
			for i, key := range keys {
				itemData := map[string]any{
					b.ItemName: arr[key],
					b.KeyName:  key,
					"@index":   i,
					"@first":   i == 0,
					"@last":    i == len(keys)-1,
				}
				for _, inner := range b.InnerBlocks {
					if err := t.renderBlock(sb, inner, itemData); err != nil {
						return err
					}
				}
			}
		}

	case *IncludeBlock:
		// Handled by TemplateSet
		sb.WriteString("{{>" + b.TemplateName + "}}")
	}
	return nil
}

// evaluateCondition evaluates a condition expression.
func evaluateCondition(data map[string]any, condition string, negate bool) bool {
	condition = strings.TrimSpace(condition)

	// Check for equality: key == value
	if strings.Contains(condition, "==") {
		parts := strings.Split(condition, "==")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(strings.Trim(parts[1], "'\""))
			val := fmt.Sprintf("%v", getNestedValue(data, key))
			result := val == value
			return negate != result
		}
	}

	// Check for existence: key (true if not nil/empty)
	val := getNestedValue(data, condition)
	exists := val != nil && val != "" && val != 0
	if s, ok := val.(string); ok {
		exists = s != ""
	}
	return negate != exists
}

// Validate validates the template for common issues.
func (t *Template) Validate() []ValidationError {
	var errors []ValidationError
	validateBlocks(t.blocks, t.name, &errors)
	return errors
}

// ValidationError represents a template validation error.
type ValidationError struct {
	Path    string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

func validateBlocks(blocks []Block, path string, errors *[]ValidationError) {
	for i, block := range blocks {
		currentPath := fmt.Sprintf("%s[%d]", path, i)
		switch b := block.(type) {
		case *VariableBlock:
			if strings.TrimSpace(b.Path) == "" {
				*errors = append(*errors, ValidationError{
					Path:    currentPath,
					Message: "empty variable path",
				})
			}
		case *IfBlock:
			if strings.TrimSpace(b.Condition) == "" {
				*errors = append(*errors, ValidationError{
					Path:    currentPath,
					Message: "empty condition in #if block",
				})
			}
			validateBlocks(b.ThenBlocks, currentPath+".then", errors)
			validateBlocks(b.ElseBlocks, currentPath+".else", errors)
		case *EachBlock:
			if strings.TrimSpace(b.Path) == "" {
				*errors = append(*errors, ValidationError{
					Path:    currentPath,
					Message: "empty path in #each block",
				})
			}
			if strings.TrimSpace(b.ItemName) == "" {
				*errors = append(*errors, ValidationError{
					Path:    currentPath,
					Message: "empty item name in #each block",
				})
			}
			validateBlocks(b.InnerBlocks, currentPath+".inner", errors)
		}
	}
}

// skipWhitespace skips whitespace in a string.
func skipWhitespace(s string) string {
	for i, c := range s {
		if !unicode.IsSpace(c) {
			return s[i:]
		}
	}
	return ""
}

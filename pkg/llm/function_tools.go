package llm

import (
	"encoding/json"
	"sort"
	"sync"
)

// FunctionToolCallback defines a function tool callback.
type FunctionToolCallback func(args map[string]interface{}, llmService interface{}) (string, error)

// FunctionToolDefinition defines a function tool.
type FunctionToolDefinition struct {
	Name        string
	Description string
	Parameters  interface{}
	Callback    FunctionToolCallback
}

// FunctionToolManager stores registered function tools.
type FunctionToolManager struct {
	mu    sync.RWMutex
	tools map[string]*FunctionToolDefinition
}

func NewFunctionToolManager() *FunctionToolManager {
	return &FunctionToolManager{
		tools: make(map[string]*FunctionToolDefinition),
	}
}

func (m *FunctionToolManager) RegisterTool(name, description string, parameters interface{}, callback FunctionToolCallback) {
	if name == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[name] = &FunctionToolDefinition{
		Name:        name,
		Description: description,
		Parameters:  normalizeToolParameters(parameters),
		Callback:    callback,
	}
}

func (m *FunctionToolManager) RegisterToolDefinition(def *FunctionToolDefinition) {
	if def == nil || def.Name == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *def
	copied.Parameters = normalizeToolParameters(def.Parameters)
	m.tools[def.Name] = &copied
}

func (m *FunctionToolManager) GetTools() []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]interface{}, 0, len(m.tools))
	for _, def := range m.tools {
		out = append(out, def)
	}
	return out
}

func (m *FunctionToolManager) ListTools() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.tools))
	for name := range m.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func normalizeToolParameters(parameters interface{}) interface{} {
	if parameters == nil {
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}
	switch p := parameters.(type) {
	case json.RawMessage:
		var obj interface{}
		if err := json.Unmarshal(p, &obj); err == nil {
			return obj
		}
		return string(p)
	case []byte:
		var obj interface{}
		if err := json.Unmarshal(p, &obj); err == nil {
			return obj
		}
		return string(p)
	case string:
		var obj interface{}
		if err := json.Unmarshal([]byte(p), &obj); err == nil {
			return obj
		}
		return p
	default:
		return parameters
	}
}

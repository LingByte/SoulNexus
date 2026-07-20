// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package humax

import (
	"reflect"
	"strings"
	"sync"

	"github.com/danielgtaylor/huma/v2"
)

var (
	bodyMu       sync.RWMutex
	bodyByRoute  = map[string]*huma.Schema{} // key: "METHOD /openapi/path"
	bodyExamples = map[string]any{}
)

func bodyKey(method, oapiPath string) string {
	return strings.ToUpper(method) + " " + GinPathToOpenAPI(oapiPath)
}

// RegisterJSONBody associates a Go request struct with a route so Try-It shows real fields.
// path may use Gin (:id) or OpenAPI ({id}) style.
func RegisterJSONBody(method, path string, sample any) {
	if sample == nil {
		return
	}
	schema, example := SchemaFromStruct(sample)
	if schema == nil {
		return
	}
	key := bodyKey(method, path)
	bodyMu.Lock()
	bodyByRoute[key] = schema
	bodyExamples[key] = example
	bodyMu.Unlock()
}

// RegisterJSONBodyBoth registers the same body for multiple methods on one path.
func RegisterJSONBodyBoth(path string, sample any, methods ...string) {
	if len(methods) == 0 {
		methods = []string{"POST", "PUT", "PATCH"}
	}
	for _, m := range methods {
		RegisterJSONBody(m, path, sample)
	}
}

func lookupBody(method, oapiPath string) (*huma.Schema, any) {
	key := bodyKey(method, oapiPath)
	bodyMu.RLock()
	defer bodyMu.RUnlock()
	return bodyByRoute[key], bodyExamples[key]
}

// SchemaFromStruct builds an OpenAPI object schema from json/binding struct tags.
func SchemaFromStruct(sample any) (*huma.Schema, any) {
	t := reflect.TypeOf(sample)
	if t == nil {
		return &huma.Schema{Type: "object", AdditionalProperties: true}, map[string]any{}
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return &huma.Schema{Type: "object", AdditionalProperties: true}, map[string]any{}
	}
	return schemaFromType(t, 0)
}

func schemaFromType(t reflect.Type, depth int) (*huma.Schema, any) {
	if depth > 6 {
		return &huma.Schema{Type: "object", AdditionalProperties: true}, map[string]any{}
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		if t.PkgPath() == "time" && t.Name() == "Time" {
			return &huma.Schema{Type: "string", Format: "date-time"}, "2026-01-01T00:00:00Z"
		}
		props := map[string]*huma.Schema{}
		required := []string{}
		example := map[string]any{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			if f.Anonymous {
				sub, subEx := schemaFromType(f.Type, depth+1)
				if sub != nil && sub.Properties != nil {
					for k, v := range sub.Properties {
						props[k] = v
					}
					required = append(required, sub.Required...)
				}
				if m, ok := subEx.(map[string]any); ok {
					for k, v := range m {
						example[k] = v
					}
				}
				continue
			}
			name, skip := jsonFieldName(f)
			if skip || name == "" {
				continue
			}
			fs, fe := schemaFromType(f.Type, depth+1)
			if fs == nil {
				continue
			}
			fs.Description = fieldDoc(f)
			if enum := bindingEnum(f); len(enum) > 0 {
				fs.Enum = enum
			}
			if isRequiredBinding(f) {
				required = append(required, name)
			}
			props[name] = fs
			example[name] = enrichExample(name, fe)
		}
		return &huma.Schema{
			Type:       "object",
			Properties: props,
			Required:   required,
		}, example
	case reflect.Slice, reflect.Array:
		item, itemEx := schemaFromType(t.Elem(), depth+1)
		return &huma.Schema{Type: "array", Items: item}, []any{itemEx}
	case reflect.Map:
		return &huma.Schema{Type: "object", AdditionalProperties: true}, map[string]any{}
	case reflect.String:
		return &huma.Schema{Type: "string"}, ""
	case reflect.Bool:
		return &huma.Schema{Type: "boolean"}, true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &huma.Schema{Type: "integer"}, 0
	case reflect.Float32, reflect.Float64:
		return &huma.Schema{Type: "number"}, 0
	case reflect.Interface:
		return &huma.Schema{Description: "any JSON value"}, nil
	default:
		return &huma.Schema{Type: "object", AdditionalProperties: true}, map[string]any{}
	}
}

func jsonFieldName(f reflect.StructField) (name string, skip bool) {
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", true
	}
	if tag == "" {
		return f.Name, false
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	if name == "" {
		name = f.Name
	}
	return name, false
}

func isRequiredBinding(f reflect.StructField) bool {
	for _, p := range strings.Split(f.Tag.Get("binding"), ",") {
		if p == "required" {
			return true
		}
	}
	return false
}

func bindingEnum(f reflect.StructField) []any {
	for _, p := range strings.Split(f.Tag.Get("binding"), ",") {
		if strings.HasPrefix(p, "oneof=") {
			vals := strings.Fields(strings.TrimPrefix(p, "oneof="))
			out := make([]any, 0, len(vals))
			for _, v := range vals {
				out = append(out, v)
			}
			return out
		}
	}
	return nil
}

func fieldDoc(f reflect.StructField) string {
	if d := f.Tag.Get("doc"); d != "" {
		return d
	}
	if b := f.Tag.Get("binding"); b != "" {
		return "validation: " + b
	}
	return ""
}

func enrichExample(name string, fallback any) any {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "email"):
		return "user@example.com"
	case n == "password":
		return "********"
	case n == "name":
		return "example"
	case strings.Contains(n, "channeltype"):
		return "email"
	case n == "driver":
		return "smtp"
	case strings.Contains(n, "host"):
		return "smtp.example.com"
	case strings.Contains(n, "port"):
		return 465
	default:
		return fallback
	}
}

// DebugBodyCount returns how many route bodies are registered.
func DebugBodyCount() int {
	bodyMu.RLock()
	defer bodyMu.RUnlock()
	return len(bodyByRoute)
}

// LookupBodyForTest exposes registry lookup for tests.
func LookupBodyForTest(method, path string) *huma.Schema {
	s, _ := lookupBody(method, path)
	return s
}

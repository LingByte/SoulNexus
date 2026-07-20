// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package humax

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/danielgtaylor/huma/v2"
)

var pathParamRe = regexp.MustCompile(`\{([^}/]+)\}`)

func document(api huma.API, method, ginPath string) {
	if api == nil {
		return
	}
	oapiPath := GinPathToOpenAPI(ginPath)
	params := pathParams(oapiPath)
	if isListGET(method, oapiPath) {
		params = append(params, listQueryParams()...)
	}

	tag := HumanTag(oapiPath)
	EnsureTag(api, tag)
	op := &huma.Operation{
		OperationID:   OperationID(method, oapiPath),
		Method:        method,
		Path:          oapiPath,
		Summary:       HumanSummary(method, oapiPath),
		Description:   humanDescription(method, oapiPath),
		Tags:          []string{tag},
		Parameters:    params,
		DefaultStatus: http.StatusOK,
		Security:      []map[string][]string{{"BearerAuth": {}}},
		Responses: map[string]*huma.Response{
			"200": {Description: "成功（具体 JSON 信封因接口而异）"},
			"201": {Description: "已创建"},
			"204": {Description: "无内容"},
			"400": {Description: "请求参数错误"},
			"401": {Description: "未登录 — 请设置 Authorization: Bearer <jwt>"},
			"403": {Description: "无权限"},
			"404": {Description: "资源不存在"},
			"429": {Description: "请求过于频繁"},
			"500": {Description: "服务内部错误"},
		},
	}

	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		schema, example := lookupBody(method, oapiPath)
		if schema == nil {
			schema = &huma.Schema{
				Type:                 "object",
				AdditionalProperties: true,
				Description:          "No typed body registered for this route yet. Fill JSON manually or add RegisterJSONBody.",
			}
			example = map[string]any{}
		}
		op.RequestBody = &huma.RequestBody{
			Description: "JSON request body (fields from Go request struct when registered).",
			Required:    method != http.MethodPatch,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Schema:  schema,
					Example: example,
				},
			},
		}
	}

	api.OpenAPI().AddOperation(op)
}

func pathParams(oapiPath string) []*huma.Param {
	matches := pathParamRe.FindAllStringSubmatch(oapiPath, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]*huma.Param, 0, len(matches))
	seen := map[string]bool{}
	for _, m := range matches {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, &huma.Param{
			Name:        name,
			In:          "path",
			Required:    true,
			Description: fmt.Sprintf("Path parameter `%s`", name),
			Schema: &huma.Schema{
				Type:     inferParamType(name),
				Examples: []any{exampleForParam(name)},
			},
			Example: exampleForParam(name),
		})
	}
	return out
}

func listQueryParams() []*huma.Param {
	return []*huma.Param{
		{
			Name:        "page",
			In:          "query",
			Required:    false,
			Description: "Page number (1-based). Used by many list endpoints.",
			Schema:      &huma.Schema{Type: "integer", Default: 1, Examples: []any{1}},
			Example:     1,
		},
		{
			Name:        "size",
			In:          "query",
			Required:    false,
			Description: "Page size / pageSize. Alias varies by handler (`size` or `pageSize`).",
			Schema:      &huma.Schema{Type: "integer", Default: 20, Examples: []any{20}},
			Example:     20,
		},
		{
			Name:        "pageSize",
			In:          "query",
			Required:    false,
			Description: "Alternate page size query key used by some handlers.",
			Schema:      &huma.Schema{Type: "integer", Examples: []any{20}},
		},
		{
			Name:        "q",
			In:          "query",
			Required:    false,
			Description: "Optional search / keyword filter when supported.",
			Schema:      &huma.Schema{Type: "string"},
		},
	}
}

func isListGET(method, oapiPath string) bool {
	if method != http.MethodGet {
		return false
	}
	// Ends with a concrete resource segment (not a path param) → likely a collection list.
	parts := strings.Split(strings.Trim(oapiPath, "/"), "/")
	if len(parts) == 0 {
		return false
	}
	last := parts[len(parts)-1]
	return !strings.HasPrefix(last, "{")
}

func inferParamType(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.HasSuffix(n, "id"), n == "id", strings.Contains(n, "id"):
		// Many IDs are numeric uints; some are UUIDs/strings — use string for Try-It flexibility.
		if n == "id" || strings.HasSuffix(n, "_id") || strings.HasSuffix(n, "Id") || strings.HasSuffix(n, "ID") {
			return "string"
		}
		return "string"
	case strings.Contains(n, "page"), strings.Contains(n, "size"), strings.Contains(n, "limit"), strings.Contains(n, "offset"):
		return "integer"
	default:
		return "string"
	}
}

func exampleForParam(name string) any {
	n := strings.ToLower(name)
	switch {
	case n == "id" || strings.HasSuffix(n, "id"):
		return "1"
	case strings.Contains(n, "version"):
		return "1"
	default:
		return "example"
	}
}

func titleCaseWords(slug string) string {
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		// Keep known acronyms
		upper := strings.ToUpper(p)
		switch upper {
		case "API", "JWT", "NLU", "MCP", "SMS", "AI", "JS", "KB", "QA":
			parts[i] = upper
			continue
		}
		runes := []rune(p)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

// EnsureSecuritySchemes registers Bearer JWT so Scalar "Try it" can send Authorization.
func EnsureSecuritySchemes(api huma.API) {
	if api == nil {
		return
	}
	oapi := api.OpenAPI()
	if oapi.Components == nil {
		oapi.Components = &huma.Components{}
	}
	if oapi.Components.SecuritySchemes == nil {
		oapi.Components.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	oapi.Components.SecuritySchemes["BearerAuth"] = &huma.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "租户或平台登录后的 JWT。请求头：`Authorization: Bearer <token>`。",
	}
	oapi.Components.SecuritySchemes["ApiKeyAuth"] = &huma.SecurityScheme{
		Type:        "apiKey",
		In:          "header",
		Name:        "X-Access-Key",
		Description: "可选 AKSK 访问密钥（配合产品使用的签名头）。",
	}
}

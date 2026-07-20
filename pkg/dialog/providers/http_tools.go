package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"go.uber.org/zap"
)

const (
	CatalogToolKindHTTP     = "http"
	CatalogToolKindMCPStdio = "mcp_stdio"
	CatalogToolKindMCPSSE   = "mcp_sse"
)

// CatalogToolRow is a runtime catalog entry loaded at voice attach time.
type CatalogToolRow struct {
	ID             uint
	Name           string
	DisplayName    string
	Description    string
	Kind           string
	Enabled        bool
	Method         string
	URL            string
	HeadersJSON    []byte
	BodyTemplate   string
	TimeoutMS      int
	ParametersJSON []byte
	McpCommand     string
	McpArgsJSON    []byte
	McpEnvsJSON    []byte
	McpSSEURL      string
}

// HTTPToolConfig is a runtime HTTP function tool definition.
type HTTPToolConfig struct {
	Name         string
	DisplayName  string
	Description  string
	Method       string
	URL          string
	Headers      map[string]string
	BodyTemplate string
	TimeoutMS    int
	Parameters   json.RawMessage
}

// HTTPToolFromCatalogRow converts a catalog row into a runtime HTTP tool config.
func HTTPToolFromCatalogRow(row CatalogToolRow) (HTTPToolConfig, error) {
	if row.Kind != CatalogToolKindHTTP {
		return HTTPToolConfig{}, fmt.Errorf("%w: not an http tool", utils.ErrInvalidParams)
	}
	headers, err := parseToolHeadersJSON(row.HeadersJSON)
	if err != nil {
		return HTTPToolConfig{}, fmt.Errorf("%w: headers: %v", utils.ErrInvalidParams, err)
	}
	cfg := HTTPToolConfig{
		Name:         sanitizeToolName(row.Name),
		DisplayName:  strings.TrimSpace(row.DisplayName),
		Description:  strings.TrimSpace(row.Description),
		Method:       strings.ToUpper(strings.TrimSpace(row.Method)),
		URL:          strings.TrimSpace(row.URL),
		Headers:      headers,
		BodyTemplate: row.BodyTemplate,
		TimeoutMS:    row.TimeoutMS,
	}
	if cfg.Method == "" {
		cfg.Method = http.MethodPost
	}
	if cfg.TimeoutMS <= 0 {
		cfg.TimeoutMS = 15000
	}
	if len(row.ParametersJSON) > 0 {
		cfg.Parameters = append(json.RawMessage(nil), row.ParametersJSON...)
	}
	if cfg.Name == "" {
		return HTTPToolConfig{}, utils.ErrEmptyToolName
	}
	if cfg.URL == "" {
		return HTTPToolConfig{}, fmt.Errorf("%w: empty url", utils.ErrInvalidParams)
	}
	return cfg, nil
}

// ParseHTTPTools converts enabled catalog rows of kind=http into runtime configs.
func ParseHTTPTools(rows []CatalogToolRow) []HTTPToolConfig {
	out := make([]HTTPToolConfig, 0, len(rows))
	for _, row := range rows {
		if !row.Enabled || row.Kind != CatalogToolKindHTTP {
			continue
		}
		cfg, err := HTTPToolFromCatalogRow(row)
		if err != nil {
			continue
		}
		out = append(out, cfg)
	}
	return out
}

// RegisterHTTPTools registers one LLM function tool per HTTP catalog entry.
func RegisterHTTPTools(provider ChatLLM, tools []HTTPToolConfig, lg *zap.Logger) {
	if provider == nil || len(tools) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		tool := tool
		toolName := sanitizeToolName(tool.Name)
		if toolName == "" {
			if lg != nil {
				lg.Warn("voice: skip http tool with empty name")
			}
			continue
		}
		if _, ok := seen[toolName]; ok {
			if lg != nil {
				lg.Warn("voice: duplicate HTTP tool", zap.String("tool", toolName), zap.Error(utils.ErrToolAlreadyRegistered))
			}
			continue
		}
		seen[toolName] = struct{}{}
		desc := tool.Description
		if desc == "" {
			if tool.DisplayName != "" {
				desc = tool.DisplayName
			} else {
				desc = fmt.Sprintf("Call HTTP tool %q", tool.Name)
			}
		}
		params := tool.Parameters
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","additionalProperties":true}`)
		}
		provider.RegisterFunctionTool(toolName, desc, params, func(args map[string]interface{}, _ interface{}) (string, error) {
			return invokeHTTPTool(context.Background(), tool, args, lg)
		})
		if lg != nil {
			lg.Info("voice: HTTP tool registered", zap.String("tool", toolName), zap.String("method", tool.Method), zap.String("url", tool.URL))
		}
	}
}

func invokeHTTPTool(ctx context.Context, tool HTTPToolConfig, args map[string]interface{}, lg *zap.Logger) (string, error) {
	timeout := time.Duration(tool.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := renderToolTemplate(tool.URL, args)
	if err := common.ValidateTenantConfiguredURL(url); err != nil {
		return "", fmt.Errorf("%w: %v", utils.ErrInvalidParams, err)
	}
	body, hasBody, err := buildHTTPToolBody(tool, args)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, tool.Method, url, body)
	if err != nil {
		return "", fmt.Errorf("%w: %v", utils.ErrToolExecutionFailed, err)
	}
	for k, v := range tool.Headers {
		req.Header.Set(k, renderToolTemplate(v, args))
	}
	if hasBody && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	client := common.NewTenantToolHTTPClient(timeout)
	resp, err := client.Do(req)
	if err != nil {
		if lg != nil {
			lg.Warn("http tool invoke failed", zap.String("tool", tool.Name), zap.Error(err))
		}
		return "", fmt.Errorf("%w: %s: %v", utils.ErrToolExecutionFailed, tool.Name, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("%w: read body: %v", utils.ErrToolExecutionFailed, err)
	}
	out := strings.TrimSpace(string(raw))
	if resp.StatusCode >= 400 {
		if out == "" {
			out = resp.Status
		}
		return "", fmt.Errorf("%w: %s: HTTP %d: %s", utils.ErrToolExecutionFailed, tool.Name, resp.StatusCode, out)
	}
	if out == "" {
		return "ok", nil
	}
	return out, nil
}

func buildHTTPToolBody(tool HTTPToolConfig, args map[string]interface{}) (io.Reader, bool, error) {
	switch strings.ToUpper(strings.TrimSpace(tool.Method)) {
	case http.MethodGet, http.MethodDelete:
		return nil, false, nil
	}
	tpl := strings.TrimSpace(tool.BodyTemplate)
	if tpl == "" {
		if len(args) == 0 {
			return nil, false, nil
		}
		b, err := json.Marshal(args)
		if err != nil {
			return nil, false, fmt.Errorf("%w: marshal args: %v", utils.ErrParseJSONRPC, err)
		}
		return bytes.NewReader(b), true, nil
	}
	rendered := renderToolTemplate(tpl, args)
	return strings.NewReader(rendered), true, nil
}

func renderToolTemplate(tpl string, args map[string]interface{}) string {
	if tpl == "" || len(args) == 0 {
		return tpl
	}
	out := tpl
	for k, v := range args {
		placeholder := "{{" + k + "}}"
		val := fmt.Sprint(v)
		out = strings.ReplaceAll(out, placeholder, val)
	}
	return out
}

func parseToolHeadersJSON(raw []byte) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func parseToolStringSliceJSON(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

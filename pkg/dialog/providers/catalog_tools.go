package providers

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"go.uber.org/zap"
)

// CatalogToolLoader loads enabled tenant catalog tools at call attach time.
type CatalogToolLoader func(ctx context.Context, tenantID uint) ([]CatalogToolRow, error)

var (
	catalogLoaderMu sync.RWMutex
	catalogLoader   CatalogToolLoader
)

// SetCatalogToolLoader installs the DB-backed tenant tool catalog loader.
func SetCatalogToolLoader(fn CatalogToolLoader) {
	catalogLoaderMu.Lock()
	catalogLoader = fn
	catalogLoaderMu.Unlock()
}

func catalogToolLoader() CatalogToolLoader {
	catalogLoaderMu.RLock()
	fn := catalogLoader
	catalogLoaderMu.RUnlock()
	return fn
}

// catalogToolSelection describes assistant binding from agentConfig.customToolIds.
// Keys may be:
//   - "123" → whole catalog row (HTTP tool or all tools on an MCP SSE server)
//   - "123:order_lookup" → one MCP tool on catalog row 123
type catalogToolSelection struct {
	Restricted bool
	CatalogIDs map[uint]struct{}
	MCPTools   map[uint]map[string]struct{}
}

// RegisterCatalogTools registers tenant catalog tools (HTTP + mcp_sse + mcp_stdio) on the LLM session.
// Assistant-level mcpServers stdio entries are registered separately and are not replaced.
func RegisterCatalogTools(ctx context.Context, provider ChatLLM, env tenantcfg.VoiceEnv, callID string, lg *zap.Logger) {
	if provider == nil || env.TenantID == 0 {
		return
	}
	loader := catalogToolLoader()
	if loader == nil {
		return
	}
	rows, err := loader(ctx, env.TenantID)
	if err != nil {
		if lg != nil {
			lg.Warn("voice: catalog tool load failed", zap.Uint("tenant_id", env.TenantID), zap.Error(err))
		}
		return
	}
	sel := parseCatalogToolSelection(env.AgentConfigRaw)
	rows = filterCatalogToolsBySelection(rows, sel)
	httpTools := ParseHTTPTools(rows)
	RegisterHTTPTools(provider, httpTools, lg)
	RegisterMCPSSETools(ctx, provider, catalogMCPSSEEndpoints(rows, sel), callID, lg)
	mcpServers := catalogMcpStdioServers(rows)
	if len(mcpServers) > 0 {
		RegisterMCPTools(provider, mcpServers, lg)
	}
}

func filterCatalogToolsByAgentConfig(rows []CatalogToolRow, agent map[string]any) []CatalogToolRow {
	return filterCatalogToolsBySelection(rows, parseCatalogToolSelection(agent))
}

func filterCatalogToolsBySelection(rows []CatalogToolRow, sel catalogToolSelection) []CatalogToolRow {
	if !sel.Restricted {
		return rows
	}
	if len(sel.CatalogIDs) == 0 && len(sel.MCPTools) == 0 {
		return nil
	}
	out := make([]CatalogToolRow, 0, len(rows))
	for _, row := range rows {
		if _, ok := sel.CatalogIDs[row.ID]; ok {
			out = append(out, row)
			continue
		}
		if _, ok := sel.MCPTools[row.ID]; ok {
			out = append(out, row)
		}
	}
	return out
}

func parseCatalogToolSelection(agent map[string]any) catalogToolSelection {
	ids, restricted := agentCustomToolIDsRaw(agent)
	if !restricted {
		return catalogToolSelection{Restricted: false}
	}
	sel := catalogToolSelection{
		Restricted: true,
		CatalogIDs: make(map[uint]struct{}),
		MCPTools:   make(map[uint]map[string]struct{}),
	}
	for _, raw := range ids {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if catID, toolName, ok := splitCatalogMCPBindKey(raw); ok {
			if sel.MCPTools[catID] == nil {
				sel.MCPTools[catID] = make(map[string]struct{})
			}
			sel.MCPTools[catID][toolName] = struct{}{}
			continue
		}
		if i, err := strconv.ParseUint(raw, 10, 64); err == nil && i > 0 {
			sel.CatalogIDs[uint(i)] = struct{}{}
		}
	}
	return sel
}

// FormatCatalogMCPBindKey builds "catalogId:toolName" for assistant customToolIds.
func FormatCatalogMCPBindKey(catalogID uint, toolName string) string {
	return strconv.FormatUint(uint64(catalogID), 10) + ":" + strings.TrimSpace(toolName)
}

func splitCatalogMCPBindKey(raw string) (catalogID uint, toolName string, ok bool) {
	i := strings.IndexByte(raw, ':')
	if i <= 0 || i >= len(raw)-1 {
		return 0, "", false
	}
	idPart := strings.TrimSpace(raw[:i])
	name := strings.TrimSpace(raw[i+1:])
	if name == "" {
		return 0, "", false
	}
	n, err := strconv.ParseUint(idPart, 10, 64)
	if err != nil || n == 0 {
		return 0, "", false
	}
	return uint(n), name, true
}

// agentCustomToolIDs returns selected catalog IDs and whether the assistant
// explicitly configured the key. Missing key → unrestricted (legacy: all enabled tools).
// Present key (including empty array) → only those IDs.
// Deprecated shape helpers keep numeric-only parsing for older tests.
func agentCustomToolIDs(agent map[string]any) (ids []uint, restricted bool) {
	raw, restricted := agentCustomToolIDsRaw(agent)
	if !restricted {
		return nil, false
	}
	out := make([]uint, 0, len(raw))
	for _, s := range raw {
		if i, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64); err == nil && i > 0 {
			out = append(out, uint(i))
		}
	}
	return out, true
}

func agentCustomToolIDsRaw(agent map[string]any) (ids []string, restricted bool) {
	if len(agent) == 0 {
		return nil, false
	}
	raw, ok := agent["customToolIds"]
	if !ok {
		raw, ok = agent["custom_tool_ids"]
	}
	if !ok {
		return nil, false
	}
	if raw == nil {
		return nil, true
	}
	switch v := raw.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			switch n := item.(type) {
			case float64:
				if n > 0 {
					out = append(out, strconv.FormatUint(uint64(n), 10))
				}
			case json.Number:
				out = append(out, n.String())
			case string:
				if s := strings.TrimSpace(n); s != "" {
					out = append(out, s)
				}
			case int:
				if n > 0 {
					out = append(out, strconv.Itoa(n))
				}
			case int64:
				if n > 0 {
					out = append(out, strconv.FormatInt(n, 10))
				}
			}
		}
		return out, true
	case []uint:
		out := make([]string, 0, len(v))
		for _, n := range v {
			if n > 0 {
				out = append(out, strconv.FormatUint(uint64(n), 10))
			}
		}
		return out, true
	case []string:
		out := make([]string, 0, len(v))
		for _, s := range v {
			if t := strings.TrimSpace(s); t != "" {
				out = append(out, t)
			}
		}
		return out, true
	default:
		return nil, true
	}
}

func catalogMcpStdioServers(rows []CatalogToolRow) []McpServerConfig {
	out := make([]McpServerConfig, 0)
	for _, row := range rows {
		if !row.Enabled || row.Kind != CatalogToolKindMCPStdio {
			continue
		}
		args, _ := parseToolStringSliceJSON(row.McpArgsJSON)
		envs, _ := parseToolHeadersJSON(row.McpEnvsJSON)
		out = append(out, McpServerConfig{
			Name:    row.Name,
			Type:    "stdio",
			Command: row.McpCommand,
			Args:    args,
			Envs:    envs,
		})
	}
	return out
}

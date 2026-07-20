package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"github.com/LingByte/lingllm/realtime"
	"go.uber.org/zap"
)

// RealtimeCatalogBundle is catalog HTTP + MCP SSE tools for Qwen-Omni-Realtime FC.
type RealtimeCatalogBundle struct {
	Tools  []realtime.Tool
	invoke map[string]func(map[string]interface{}) (string, error)
}

// Invoke runs a catalog tool by registered name. ok=false when name is not catalog-owned.
func (b RealtimeCatalogBundle) Invoke(name string, args map[string]any) (out string, ok bool) {
	if b.invoke == nil {
		return "", false
	}
	fn, ok := b.invoke[name]
	if !ok {
		return "", false
	}
	if args == nil {
		args = map[string]any{}
	}
	rawArgs := make(map[string]interface{}, len(args))
	for k, v := range args {
		rawArgs[k] = v
	}
	result, err := fn(rawArgs)
	if err != nil {
		return realtimeCatalogToolJSON(map[string]any{"ok": false, "error": err.Error()}), true
	}
	return result, true
}

// BuildRealtimeCatalogTools loads tenant catalog tools bound on the assistant (customToolIds)
// and prepares realtime FC definitions + invoke handlers (HTTP + MCP SSE).
// callID enables per-speaker credential injection via EnrichMCPArgs.
func BuildRealtimeCatalogTools(ctx context.Context, env tenantcfg.VoiceEnv, callID string, lg *zap.Logger) RealtimeCatalogBundle {
	out := RealtimeCatalogBundle{invoke: make(map[string]func(map[string]interface{}) (string, error))}
	if env.TenantID == 0 {
		return out
	}
	boundCallID := strings.TrimSpace(callID)
	loader := catalogToolLoader()
	if loader == nil {
		return out
	}
	rows, err := loader(ctx, env.TenantID)
	if err != nil {
		if lg != nil {
			lg.Warn("realtime: catalog tool load failed", zap.Uint("tenant_id", env.TenantID), zap.Error(err))
		}
		return out
	}
	sel := parseCatalogToolSelection(env.AgentConfigRaw)
	rows = filterCatalogToolsBySelection(rows, sel)

	seen := make(map[string]struct{})
	for _, tool := range ParseHTTPTools(rows) {
		regName := tool.Name
		if regName == "" {
			continue
		}
		if _, dup := seen[regName]; dup {
			continue
		}
		seen[regName] = struct{}{}
		desc := strings.TrimSpace(tool.Description)
		if desc == "" {
			desc = fmt.Sprintf("HTTP tool %s", tool.Name)
		}
		params := tool.Parameters
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","additionalProperties":true}`)
		}
		toolCopy := tool
		out.Tools = append(out.Tools, realtime.Tool{
			Name:        regName,
			Description: desc,
			Parameters:  params,
		})
		out.invoke[regName] = func(args map[string]interface{}) (string, error) {
			return invokeHTTPTool(context.Background(), toolCopy, args, lg)
		}
	}

	for _, ep := range catalogMCPSSEEndpoints(rows, sel) {
		if err := common.ValidateTenantConfiguredURL(ep.SSEURL); err != nil {
			if lg != nil {
				lg.Warn("realtime: skip mcp_sse url", zap.String("name", ep.Name), zap.Error(err))
			}
			continue
		}
		discovered, err := DiscoverMCPSSETools(ctx, ep.SSEURL, ep.Headers, ep.TimeoutMS)
		if err != nil {
			if lg != nil {
				lg.Warn("realtime: mcp_sse discover failed",
					zap.Uint("catalog_id", ep.CatalogID),
					zap.String("name", ep.Name),
					zap.Error(err),
				)
			}
			continue
		}
		for _, tool := range discovered {
			if !mcpToolAllowed(ep, tool.Name) {
				continue
			}
			regName := sanitizeToolName(tool.Name)
			if regName == "" {
				continue
			}
			if _, ok := seen[regName]; ok {
				prefixed := sanitizeToolName(ep.Name + "_" + tool.Name)
				if prefixed == "" || prefixed == regName {
					continue
				}
				regName = prefixed
			}
			if _, ok := seen[regName]; ok {
				continue
			}
			seen[regName] = struct{}{}

			desc := strings.TrimSpace(tool.Description)
			if desc == "" {
				desc = fmt.Sprintf("MCP tool %q via %s", tool.Name, ep.Name)
			}
			params := tool.InputSchema
			if len(params) == 0 {
				params = json.RawMessage(`{"type":"object","additionalProperties":true}`)
			}
			mcpToolName := tool.Name
			sseURL := ep.SSEURL
			headers := ep.Headers
			timeoutMS := ep.TimeoutMS
			out.Tools = append(out.Tools, realtime.Tool{
				Name:        regName,
				Description: desc,
				Parameters:  params,
			})
			out.invoke[regName] = func(args map[string]interface{}) (string, error) {
				args = callbinding.EnrichMCPArgs(boundCallID, mcpToolName, args)
				return invokeMCPSSETool(context.Background(), sseURL, headers, timeoutMS, mcpToolName, args, lg)
			}
			if lg != nil {
				lg.Info("realtime: catalog MCP tool registered",
					zap.String("tool", regName),
					zap.String("mcp_name", mcpToolName),
				)
			}
		}
	}
	return out
}

func realtimeCatalogToolJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"ok":false,"error":"marshal failed"}`
	}
	return string(b)
}

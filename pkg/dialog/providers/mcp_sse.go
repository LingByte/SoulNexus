package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

const (
	// Voice calls need fast tool turns; catalog timeoutMs is capped at invoke time.
	defaultMCPSSEInvokeTimeout = 8 * time.Second
	maxMCPSSEInvokeTimeout     = 10 * time.Second
	defaultMCPSSEDiscoverTimeout = 10 * time.Second
)

func mcpSSEInvokeTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return defaultMCPSSEInvokeTimeout
	}
	d := time.Duration(timeoutMS) * time.Millisecond
	if d > maxMCPSSEInvokeTimeout {
		return maxMCPSSEInvokeTimeout
	}
	if d < 2*time.Second {
		return 2 * time.Second
	}
	return d
}

func mcpSSEDiscoverTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return defaultMCPSSEDiscoverTimeout
	}
	d := time.Duration(timeoutMS) * time.Millisecond
	if d > 15*time.Second {
		return 15 * time.Second
	}
	return d
}

// DiscoveredMCPTool is one tool returned by MCP tools/list.
type DiscoveredMCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// MCPSSEEndpoint describes a tenant-catalog MCP SSE server.
type MCPSSEEndpoint struct {
	CatalogID   uint
	Name        string
	SSEURL      string
	Headers     map[string]string
	TimeoutMS   int
	AllowTools  map[string]struct{} // empty + WholeServer → all tools
	WholeServer bool
}

// DiscoverMCPSSETools connects to an MCP SSE endpoint, lists tools, then closes.
func DiscoverMCPSSETools(ctx context.Context, sseURL string, headers map[string]string, timeoutMS int) ([]DiscoveredMCPTool, error) {
	sseURL = strings.TrimSpace(sseURL)
	if sseURL == "" {
		return nil, fmt.Errorf("%w: empty mcp sse url", utils.ErrInvalidParams)
	}
	if err := common.ValidateTenantConfiguredURL(sseURL); err != nil {
		return nil, fmt.Errorf("%w: %v", utils.ErrInvalidParams, err)
	}
	timeout := mcpSSEDiscoverTimeout(timeoutMS)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cli, err := dialMCPSSEClient(ctx, sseURL, headers)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	result, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("%w: tools/list: %v", utils.ErrToolExecutionFailed, err)
	}
	out := make([]DiscoveredMCPTool, 0, len(result.Tools))
	for _, tool := range result.Tools {
		out = append(out, discoveredFromMCPTool(tool))
	}
	return out, nil
}

// RegisterMCPSSETools discovers tools from each SSE endpoint and registers them on the LLM session.
// Each invocation opens a short-lived MCP session (initialize → tools/call → close).
func RegisterMCPSSETools(ctx context.Context, provider ChatLLM, endpoints []MCPSSEEndpoint, callID string, lg *zap.Logger) {
	if provider == nil || len(endpoints) == 0 {
		return
	}
	seen := make(map[string]struct{})
	for _, ep := range endpoints {
		ep := ep
		if strings.TrimSpace(ep.SSEURL) == "" {
			continue
		}
		if err := common.ValidateTenantConfiguredURL(ep.SSEURL); err != nil {
			if lg != nil {
				lg.Warn("voice: skip mcp_sse url", zap.String("name", ep.Name), zap.Error(err))
			}
			continue
		}
		tools, err := DiscoverMCPSSETools(ctx, ep.SSEURL, ep.Headers, ep.TimeoutMS)
		if err != nil {
			if lg != nil {
				lg.Warn("voice: mcp_sse discover failed",
					zap.Uint("catalog_id", ep.CatalogID),
					zap.String("name", ep.Name),
					zap.Error(err),
				)
			}
			continue
		}
		for _, tool := range tools {
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
					if lg != nil {
						lg.Warn("voice: duplicate MCP SSE tool", zap.String("tool", regName))
					}
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
			boundCallID := callID
			provider.RegisterFunctionTool(regName, desc, params, func(args map[string]interface{}, _ interface{}) (string, error) {
				args = callbinding.EnrichMCPArgs(boundCallID, mcpToolName, args)
				return invokeMCPSSETool(context.Background(), sseURL, headers, timeoutMS, mcpToolName, args, lg)
			})
			if lg != nil {
				lg.Info("voice: MCP SSE tool registered",
					zap.String("tool", regName),
					zap.String("mcp_name", mcpToolName),
					zap.String("sse_url", sseURL),
				)
			}
		}
	}
}

func mcpToolAllowed(ep MCPSSEEndpoint, toolName string) bool {
	if ep.WholeServer || len(ep.AllowTools) == 0 {
		return true
	}
	_, ok := ep.AllowTools[toolName]
	return ok
}

func catalogMCPSSEEndpoints(rows []CatalogToolRow, sel catalogToolSelection) []MCPSSEEndpoint {
	out := make([]MCPSSEEndpoint, 0)
	for _, row := range rows {
		if !row.Enabled || row.Kind != CatalogToolKindMCPSSE {
			continue
		}
		headers, _ := parseToolHeadersJSON(row.HeadersJSON)
		ep := MCPSSEEndpoint{
			CatalogID:   row.ID,
			Name:        row.Name,
			SSEURL:      strings.TrimSpace(row.McpSSEURL),
			Headers:     headers,
			TimeoutMS:   row.TimeoutMS,
			AllowTools:  map[string]struct{}{},
			WholeServer: true,
		}
		if sel.Restricted {
			names, hasPartial := sel.MCPTools[row.ID]
			whole := false
			if _, ok := sel.CatalogIDs[row.ID]; ok {
				whole = true
			}
			if !whole && !hasPartial {
				continue
			}
			ep.WholeServer = whole
			if hasPartial && !whole {
				ep.AllowTools = names
				ep.WholeServer = false
			}
		}
		if ep.SSEURL == "" {
			continue
		}
		out = append(out, ep)
	}
	return out
}

func discoveredFromMCPTool(tool mcp.Tool) DiscoveredMCPTool {
	d := DiscoveredMCPTool{
		Name:        tool.Name,
		Description: tool.Description,
	}
	if schema, err := json.Marshal(tool.InputSchema); err == nil && len(schema) > 2 {
		d.InputSchema = schema
	} else if len(tool.RawInputSchema) > 0 {
		d.InputSchema = append(json.RawMessage(nil), tool.RawInputSchema...)
	}
	return d
}

func openMCPSSEClient(ctx context.Context, sseURL string, headers map[string]string, _ time.Duration) (*mcpclient.Client, error) {
	// SSE GET stays open; only per-request ctx deadlines should bound RPC calls.
	httpClient := common.NewTenantToolHTTPClient(0)
	cli, err := mcpclient.NewSSEMCPClient(sseURL,
		mcpclient.WithHTTPClient(httpClient),
		mcpclient.WithHeaders(headers),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: mcp sse client: %v", utils.ErrToolExecutionFailed, err)
	}
	if err := cli.Start(ctx); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("%w: mcp sse start: %v", utils.ErrToolExecutionFailed, err)
	}
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "SoulNexus",
		Version: "1.0.0",
	}
	if _, err := cli.Initialize(ctx, initReq); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("%w: mcp initialize: %v", utils.ErrToolExecutionFailed, err)
	}
	return cli, nil
}

func dialMCPSSEClient(ctx context.Context, sseURL string, headers map[string]string) (*mcpclient.Client, error) {
	cli, err := openMCPSSEClient(ctx, sseURL, headers, 0)
	if err == nil {
		return cli, nil
	}
	if alt, ok := loopbackAlternateSSEURL(sseURL); ok && (isMCPSSEEndpointWaitError(err) || isMCPSSETransportError(err)) {
		if cli2, err2 := openMCPSSEClient(ctx, alt, headers, 0); err2 == nil {
			return cli2, nil
		}
	}
	return nil, err
}

func invokeMCPSSETool(ctx context.Context, sseURL string, headers map[string]string, timeoutMS int, toolName string, args map[string]interface{}, lg *zap.Logger) (string, error) {
	opTimeout := mcpSSEInvokeTimeout(timeoutMS)
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	if args == nil {
		args = map[string]interface{}{}
	}
	req.Params.Arguments = args

	tryURLs := []string{strings.TrimSpace(sseURL)}
	if alt, ok := loopbackAlternateSSEURL(sseURL); ok {
		tryURLs = append(tryURLs, alt)
	}

	var lastErr error
	for _, url := range tryURLs {
		if url == "" {
			continue
		}
		cli, err := openMCPSSEClient(ctx, url, headers, 0)
		if err != nil {
			lastErr = err
			if lg != nil {
				lg.Warn("mcp sse dial failed", zap.String("tool", toolName), zap.String("sse_url", url), zap.Error(err))
			}
			continue
		}
		result, err := cli.CallTool(ctx, req)
		_ = cli.Close()
		if err == nil {
			if result == nil {
				return "ok", nil
			}
			if result.IsError {
				return "", fmt.Errorf("%w: %s: %s", utils.ErrToolExecutionFailed, toolName, flattenMCPContent(result.Content))
			}
			out := flattenMCPContent(result.Content)
			if out == "" {
				return "ok", nil
			}
			return out, nil
		}
		lastErr = err
		if lg != nil {
			lg.Warn("mcp sse tools/call failed", zap.String("tool", toolName), zap.String("sse_url", url), zap.Error(err))
		}
		if !isMCPSSETransportError(err) {
			break
		}
	}
	if lastErr == nil {
		lastErr = utils.ErrToolExecutionFailed
	}
	return "", fmt.Errorf("%w: %s: %v", utils.ErrToolExecutionFailed, toolName, lastErr)
}

func flattenMCPContent(content []mcp.Content) string {
	if len(content) == 0 {
		return ""
	}
	parts := make([]string, 0, len(content))
	for _, c := range content {
		switch v := c.(type) {
		case mcp.TextContent:
			if s := strings.TrimSpace(v.Text); s != "" {
				parts = append(parts, s)
			}
		default:
			if b, err := json.Marshal(v); err == nil {
				parts = append(parts, string(b))
			}
		}
	}
	return strings.Join(parts, "\n")
}

func isMCPSSETransportError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unexpected eof") ||
		strings.Contains(msg, "transport error") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "context canceled")
}

func isMCPSSEEndpointWaitError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "waiting for endpoint") ||
		strings.Contains(msg, "timeout waiting for endpoint")
}

// loopbackAlternateSSEURL swaps 127.0.0.1 ↔ localhost for MCP SSE URLs.
// LingMCP (older builds) advertises message endpoints with localhost while
// tenants often save 127.0.0.1 — mcp-go rejects the host mismatch.
func loopbackAlternateSSEURL(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return "", false
	}
	host := u.Hostname()
	switch host {
	case "127.0.0.1":
		u.Host = strings.Replace(u.Host, "127.0.0.1", "localhost", 1)
		return u.String(), true
	case "localhost":
		u.Host = strings.Replace(u.Host, "localhost", "127.0.0.1", 1)
		return u.String(), true
	default:
		return "", false
	}
}

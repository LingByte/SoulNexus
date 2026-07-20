package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	lingutils "github.com/LingByte/lingllm/utils"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
)

// McpServerConfig is one stdio MCP-style tool server entry on the assistant.
type McpServerConfig struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Envs    map[string]string `json:"envs"`
}

// ParseMcpServers decodes assistant mcpServers JSON array.
func ParseMcpServers(raw []byte) []McpServerConfig {
	servers, _ := ParseMcpServersStrict(raw)
	return servers
}

// ParseMcpServersStrict decodes assistant mcpServers JSON array and returns a decode error.
func ParseMcpServersStrict(raw []byte) ([]McpServerConfig, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var rows []McpServerConfig
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("%w: %v", utils.ErrInvalidJSONRPCFormat, err)
	}
	out := make([]McpServerConfig, 0, len(rows))
	for _, r := range rows {
		if strings.TrimSpace(r.Command) == "" {
			continue
		}
		if strings.TrimSpace(r.Name) == "" {
			r.Name = "mcp_" + strings.TrimSpace(r.Command)
		}
		out = append(out, r)
	}
	return out, nil
}

func validateMcpServerConfig(srv McpServerConfig) error {
	if strings.TrimSpace(srv.Name) == "" {
		return utils.ErrEmptyToolName
	}
	if err := lingutils.ValidateStdioConfig(srv.Command, srv.Args, srv.Envs); err != nil {
		return fmt.Errorf("%w: %v", utils.ErrInvalidParams, err)
	}
	return nil
}

// RegisterMCPTools registers one LLM function tool per configured MCP server.
// Tool call forwards JSON args to the server stdin and returns stdout.
func RegisterMCPTools(provider ChatLLM, servers []McpServerConfig, lg *zap.Logger) {
	if provider == nil || len(servers) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(servers))
	for _, srv := range servers {
		srv := srv
		if err := validateMcpServerConfig(srv); err != nil {
			if lg != nil {
				lg.Warn("voice: skip invalid MCP server", zap.String("name", srv.Name), zap.Error(err))
			}
			continue
		}
		toolName := sanitizeToolName(srv.Name)
		if _, ok := seen[toolName]; ok {
			if lg != nil {
				lg.Warn("voice: duplicate MCP tool", zap.String("tool", toolName), zap.Error(utils.ErrToolAlreadyRegistered))
			}
			continue
		}
		seen[toolName] = struct{}{}
		desc := fmt.Sprintf("Call MCP server %q", srv.Name)
		params := json.RawMessage(`{"type":"object","properties":{"input":{"type":"string"}},"additionalProperties":true}`)
		provider.RegisterFunctionTool(toolName, desc, params, func(args map[string]interface{}, _ interface{}) (string, error) {
			return invokeMCPServer(context.Background(), srv, args, lg)
		})
		if lg != nil {
			lg.Info("voice: MCP tool registered", zap.String("tool", toolName), zap.String("command", srv.Command))
		}
	}
}

func sanitizeToolName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	if name == "" {
		return "mcp_tool"
	}
	return name
}

func invokeMCPServer(ctx context.Context, srv McpServerConfig, args map[string]interface{}, lg *zap.Logger) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "invoke",
		"params": map[string]any{
			"server": srv.Name,
			"args":   args,
		},
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", utils.ErrParseJSONRPC, err)
	}
	cmd := exec.CommandContext(ctx, srv.Command, srv.Args...)
	cmd.Env = append(cmd.Environ(), envPairs(lingutils.FilterSafeEnvVars(srv.Envs))...)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if lg != nil {
			lg.Warn("mcp tool invoke failed",
				zap.String("command", srv.Command),
				zap.Error(err),
				zap.String("stderr", stderr.String()),
			)
		}
		return "", fmt.Errorf("%w: %s: %v", utils.ErrToolExecutionFailed, srv.Name, err)
	}
	out := strings.TrimSpace(stdout.String())
	if out != "" {
		var rpcResp map[string]json.RawMessage
		if err := json.Unmarshal([]byte(out), &rpcResp); err != nil {
			return "", fmt.Errorf("%w: %v", utils.ErrInvalidJSONRPCResponse, err)
		}
		if errMsg, ok := rpcResp["error"]; ok && len(errMsg) > 0 && !bytes.Equal(errMsg, []byte("null")) {
			return "", fmt.Errorf("%w: %s", utils.ErrInvalidJSONRPCResponse, string(errMsg))
		}
	}
	if out == "" {
		return "ok", nil
	}
	return out, nil
}

func envPairs(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}

// IsMcpConfigError reports whether err is one of the MCP/tool sentinel errors.
func IsMcpConfigError(err error) bool {
	return errors.Is(err, utils.ErrEmptyToolName) ||
		errors.Is(err, utils.ErrInvalidParams) ||
		errors.Is(err, utils.ErrInvalidJSONRPCFormat)
}

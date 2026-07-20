package bootstrap

import (
	"context"

	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	"github.com/LingByte/SoulNexus/pkg/dialog/voiceattach"
	"go.uber.org/zap"
)

// registerVoiceLLMTools wires LLM function tools for voice dialog:
// knowledge, legacy mcpServers JSON, and tenant catalog (HTTP + MCP SSE).
func registerVoiceLLMTools(ctx context.Context, provider providers.ChatLLM, env voiceattach.VoiceEnv, callID string, lg *zap.Logger) {
	providers.RegisterLLMTools(provider, callID, lg)
	providers.RegisterMCPTools(provider, providers.ParseMcpServers(env.McpServersRaw), lg)
	providers.RegisterCatalogTools(ctx, provider, env, callID, lg)
	if lg == nil || provider == nil {
		return
	}
	tools := provider.ListFunctionTools()
	lg.Info("voice LLM tools registered",
		zap.String("call_id", callID),
		zap.Uint("tenant_id", env.TenantID),
		zap.Uint("assistant_id", env.AssistantID),
		zap.Strings("tools", tools),
	)
}

package providers

import (
	"context"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestDiscoverMCPSSETools_WithTestServer(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0.0", server.WithToolCapabilities(true))
	mcpServer.AddTool(
		mcp.NewTool("ping",
			mcp.WithDescription("ping tool"),
			mcp.WithString("msg", mcp.Description("message")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("pong"), nil
		},
	)
	ts := server.NewTestServer(mcpServer)
	defer ts.Close()

	sseURL := ts.URL + "/sse"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tools, err := DiscoverMCPSSETools(ctx, sseURL, nil, 10000)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "ping" {
		t.Fatalf("tools=%+v", tools)
	}

	out, err := invokeMCPSSETool(ctx, sseURL, nil, 10000, "ping", map[string]interface{}{"msg": "hi"}, nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if out != "pong" {
		t.Fatalf("out=%q", out)
	}
}

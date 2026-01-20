package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"strings"
)

func main() {
	var serverURL string
	var toolName string
	var argsJSON string

	flag.StringVar(&serverURL, "url", "http://localhost:3001/sse", "MCP server URL (should include /sse path)")
	flag.StringVar(&toolName, "tool", "", "Tool name to call")
	flag.StringVar(&argsJSON, "args", "{}", "Tool arguments as JSON string")
	flag.Parse()

	if toolName == "" {
		fmt.Println("Usage: go run cmd/mcp-client/main.go -tool <tool_name> [-url <server_url>] [-args <json_args>]")
		fmt.Println("\nExample:")
		fmt.Println("  # List all tools")
		fmt.Println("  go run cmd/mcp-client/main.go -tool list")
		fmt.Println("\n  # Call echo tool")
		fmt.Println("  go run cmd/mcp-client/main.go -tool echo -args '{\"text\":\"Hello World\"}'")
		fmt.Println("\n  # Call calculator tool")
		fmt.Println("  go run cmd/mcp-client/main.go -tool calculator -args '{\"expression\":\"2+2\"}'")
		os.Exit(1)
	}

	// Ensure URL includes /sse path
	if !strings.HasSuffix(serverURL, "/sse") {
		if !strings.HasSuffix(serverURL, "/") {
			serverURL += "/"
		}
		serverURL += "sse"
	}

	// Create client
	httpTransport, err := transport.NewSSE(serverURL)
	if err != nil {
		fmt.Printf("Failed to create transport: %v\n", err)
		os.Exit(1)
	}

	mcpClient := client.NewClient(httpTransport)
	if err = mcpClient.Start(context.Background()); err != nil {
		fmt.Printf("Failed to start client: %v\n", err)
		os.Exit(1)
	}
	defer mcpClient.Close()

	// Initialize
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	serverInfo, err := mcpClient.Initialize(context.Background(), initRequest)
	if err != nil {
		fmt.Printf("Failed to initialize: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Connected to MCP server: %s v%s\n\n", serverInfo.ServerInfo.Name, serverInfo.ServerInfo.Version)

	// If it's a list command, list all tools
	if toolName == "list" {
		listTools(serverURL)
		return
	}

	// Parse arguments
	var arguments map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &arguments); err != nil {
		fmt.Printf("Failed to parse arguments JSON: %v\n", err)
		os.Exit(1)
	}

	// Call tool
	request := mcp.CallToolRequest{}
	request.Params.Name = toolName
	request.Params.Arguments = arguments

	fmt.Printf("Calling tool: %s\n", toolName)
	fmt.Printf("Arguments: %s\n\n", argsJSON)

	result, err := mcpClient.CallTool(context.Background(), request)
	if err != nil {
		fmt.Printf("Error calling tool: %v\n", err)
		os.Exit(1)
	}

	if result.IsError {
		fmt.Printf("Tool returned error\n")
	}

	// Print result
	fmt.Println("Result:")
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			fmt.Println(textContent.Text)
		} else {
			fmt.Printf("%+v\n", content)
		}
	}
}

func listTools(serverURL string) {
	// Ensure URL includes /sse path
	if !strings.HasSuffix(serverURL, "/sse") {
		if !strings.HasSuffix(serverURL, "/") {
			serverURL += "/"
		}
		serverURL += "sse"
	}

	httpTransport, err := transport.NewSSE(serverURL)
	if err != nil {
		fmt.Printf("Failed to create transport: %v\n", err)
		os.Exit(1)
	}

	mcpClient := client.NewClient(httpTransport)
	if err = mcpClient.Start(context.Background()); err != nil {
		fmt.Printf("Failed to start client: %v\n", err)
		os.Exit(1)
	}
	defer mcpClient.Close()

	// Initialize
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	_, err = mcpClient.Initialize(context.Background(), initRequest)
	if err != nil {
		fmt.Printf("Failed to initialize: %v\n", err)
		os.Exit(1)
	}

	// List tools
	toolsRequest := mcp.ListToolsRequest{}
	toolsResult, err := mcpClient.ListTools(context.Background(), toolsRequest)
	if err != nil {
		fmt.Printf("Failed to list tools: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Available tools (%d):\n\n", len(toolsResult.Tools))
	for i, tool := range toolsResult.Tools {
		fmt.Printf("%d. %s\n", i+1, tool.Name)
		fmt.Printf("   Description: %s\n", tool.Description)
		if tool.InputSchema.Properties != nil {
			schemaBytes, _ := json.MarshalIndent(tool.InputSchema, "   ", "  ")
			fmt.Printf("   Schema: %s\n", string(schemaBytes))
		}
		fmt.Println()
	}
}

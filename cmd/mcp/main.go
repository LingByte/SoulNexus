package main

import (
	"flag"
	"os"

	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	lingechoMCP "github.com/LingByte/SoulNexus/pkg/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

func main() {
	// 1. Parse command line arguments
	var transport string
	var port string
	var mode string

	flag.StringVar(&transport, "transport", "sse", "Transport type (stdio or sse)")
	flag.StringVar(&port, "port", "3001", "Port to run the MCP server on (only for SSE transport)")
	flag.StringVar(&mode, "mode", "", "Running environment (development, test, production)")
	flag.Parse()

	// 2. Set environment variables
	if mode != "" {
		os.Setenv("APP_ENV", mode)
	}

	// 3. Load configuration
	if err := config.Load(); err != nil {
		panic("config load failed: " + err.Error())
	}

	// 4. Initialize logging
	err := logger.Init(&config.GlobalConfig.Log, config.GlobalConfig.Mode)
	if err != nil {
		panic(err)
	}

	log := zap.L()

	logger.Info("Starting MCP server",
		zap.String("transport", transport),
		zap.String("port", port),
		zap.String("mode", config.GlobalConfig.Mode),
	)

	// 5. Create MCP server
	mcpServer := lingechoMCP.NewMCPServer(&lingechoMCP.Config{
		Name:                   "LingEcho/mcp",
		Version:                "1.0.0",
		Logger:                 log,
		EnableLogging:          true,
		EnableToolCapabilities: true,
	})

	// 6. Register default tools
	lingechoMCP.RegisterDefaultTools(mcpServer)

	// 7. Start server
	if transport == "sse" {
		sseServer := server.NewSSEServer(mcpServer.GetServer())
		log.Info("SSE server listening", zap.String("port", port))

		if err := sseServer.Start(":" + port); err != nil {
			logger.Fatal("Server error", zap.Error(err))
		}
	} else {
		logger.Info("Starting stdio server")
		if err := server.ServeStdio(mcpServer.GetServer()); err != nil {
			logger.Fatal("Server error", zap.Error(err))
		}
	}
}

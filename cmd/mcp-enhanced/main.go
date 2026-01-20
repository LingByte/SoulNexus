package main

import (
	"context"
	"flag"
	"os"

	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	"github.com/LingByte/SoulNexus/pkg/agent"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/graph"
	"github.com/LingByte/SoulNexus/pkg/knowledge"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/logger"
	lingechoMCP "github.com/LingByte/SoulNexus/pkg/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
	"gorm.io/gorm"
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

	logger.Info("Starting Enhanced MCP server with Multi-Agent System",
		zap.String("transport", transport),
		zap.String("port", port),
		zap.String("mode", config.GlobalConfig.Mode),
	)

	// 5. Initialize database (if needed for agents)
	// Note: Database is optional for MCP server, only needed if RAG agent is used
	var db *gorm.DB
	if config.GlobalConfig.DBDriver != "" && config.GlobalConfig.DSN != "" {
		db, err = bootstrap.SetupDatabase(os.Stdout, &bootstrap.Options{
			AutoMigrate: false, // Don't auto-migrate for MCP server
			SeedNonProd: false, // Don't seed for MCP server
		})
		if err != nil {
			logger.Warn("Database initialization failed, RAG agent will not work", zap.Error(err))
		} else {
			logger.Info("Database initialized successfully")
		}
	} else {
		logger.Info("Database not configured, RAG agent will not work")
	}

	// 6. Initialize knowledge base manager
	// Note: Knowledge base providers are auto-registered via init() functions
	// in their respective packages (aliyun.go, milvus.go, qdrant.go, etc.)
	kbManager := knowledge.GetManager()
	logger.Info("Knowledge base manager initialized",
		zap.Strings("providers", kbManager.ListProviders()),
	)

	// 7. Initialize graph store (if enabled)
	var graphStore graph.Store
	if config.GlobalConfig.Neo4jEnabled {
		graphStore, err = graph.NewNeo4jStore(
			config.GlobalConfig.Neo4jURI,
			config.GlobalConfig.Neo4jUsername,
			config.GlobalConfig.Neo4jPassword,
			config.GlobalConfig.Neo4jDatabase,
		)
		if err != nil {
			logger.Warn("Neo4j initialization failed, graph memory agent will not work", zap.Error(err))
		} else {
			logger.Info("Neo4j graph store initialized")
		}
	}

	// 8. Initialize LLM provider
	// Note: LLM provider requires API key and configuration
	// For MCP server, we create a default provider if configuration is available
	var llmProvider llm.LLMProvider
	ctx := context.Background()

	// Try to get LLM configuration from environment or config
	llmApiKey := os.Getenv("OPENAI_API_KEY")
	if llmApiKey == "" {
		llmApiKey = os.Getenv("LLM_API_KEY")
	}
	if llmApiKey == "" {
		llmApiKey = config.GlobalConfig.LLMApiKey
	}

	llmBaseURL := os.Getenv("OPENAI_BASE_URL")
	if llmBaseURL == "" {
		llmBaseURL = os.Getenv("LLM_BASE_URL")
	}
	if llmBaseURL == "" {
		llmBaseURL = config.GlobalConfig.LLMBaseURL
	}
	if llmBaseURL == "" {
		llmBaseURL = "https://api.openai.com/v1" // Default OpenAI endpoint
	}

	llmProviderType := os.Getenv("LLM_PROVIDER")
	if llmProviderType == "" {
		llmProviderType = "openai" // Default to OpenAI
	}

	if llmApiKey != "" {
		llmProvider, err = llm.NewLLMProviderFromConfig(ctx, llmProviderType, llmApiKey, llmBaseURL, "", nil)
		if err != nil {
			logger.Warn("LLM provider initialization failed, LLM agent will not work", zap.Error(err))
		} else {
			logger.Info("LLM provider initialized successfully",
				zap.String("provider", llmProviderType),
				zap.String("baseURL", llmBaseURL),
			)
		}
	} else {
		logger.Info("LLM API key not configured, LLM agent will not work. Set OPENAI_API_KEY environment variable to enable.")
	}

	// 9. Create MCP server
	mcpServer := lingechoMCP.NewMCPServer(&lingechoMCP.Config{
		Name:                   "LingEcho/mcp-enhanced",
		Version:                "2.0.0",
		Logger:                 log,
		EnableLogging:          true,
		EnableToolCapabilities: true,
	})

	// 10. Register default tools
	lingechoMCP.RegisterDefaultTools(mcpServer)

	// 11. Initialize Agent Manager
	agentConfig := &agent.Config{
		DB:          db,
		GraphStore:  graphStore,
		KBManager:   kbManager,
		LLMProvider: llmProvider,
		MCPServer:   mcpServer,
		Logger:      log,
	}

	agentManager, err := agent.NewManager(agentConfig)
	if err != nil {
		logger.Error("Failed to initialize agent manager", zap.Error(err))
		// Continue without agent system
	} else {
		logger.Info("Agent manager initialized successfully",
			zap.Int("agentsCount", len(agentManager.ListAgents())),
		)

		// 12. Register agent tools in MCP server
		agent.RegisterAgentTools(mcpServer, agentManager, log)
	}

	// 13. Start server
	if transport == "sse" {
		sseServer := server.NewSSEServer(mcpServer.GetServer())
		log.Info("Enhanced SSE server listening", zap.String("port", port))

		if err := sseServer.Start(":" + port); err != nil {
			logger.Fatal("Server error", zap.Error(err))
		}
	} else {
		logger.Info("Starting enhanced stdio server")
		if err := server.ServeStdio(mcpServer.GetServer()); err != nil {
			logger.Fatal("Server error", zap.Error(err))
		}
	}
}

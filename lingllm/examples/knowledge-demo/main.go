package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/LingByte/lingllm/knowledge"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║        LingLLM Knowledge Base Query Demo                    ║")
	fmt.Print("╚════════════════════════════════════════════════════════════╝\n")

	reader := bufio.NewReader(os.Stdin)

	// Select provider
	fmt.Println("Select knowledge base provider:")
	fmt.Println("1. Qdrant")
	fmt.Println("2. Milvus")
	fmt.Println("3. RAGFlow")
	fmt.Println("4. Alibaba Bailian")
	fmt.Print("\nEnter choice (1-4): ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var handler knowledge.KnowledgeHandler
	var namespace string
	var err error

	switch choice {
	case "1":
		handler, err = setupQdrant(reader)
	case "2":
		handler, err = setupMilvus(reader)
	case "3":
		handler, err = setupRAGFlow(reader)
	case "4":
		handler, namespace, err = setupAliyun(reader)
	default:
		fmt.Println("Invalid choice")
		return
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Create knowledge base
	kb, err := knowledge.NewKnowledgeBase(knowledge.KnowledgeBaseConfig{
		Handler:   handler,
		Namespace: namespace,
	})
	if err != nil {
		fmt.Printf("Error creating knowledge base: %v\n", err)
		return
	}
	defer kb.Close()

	fmt.Print("✓ Knowledge base initialized\n\n")

	// Interactive query session
	interactiveSession(kb)
}

func setupQdrant(reader *bufio.Reader) (knowledge.KnowledgeHandler, error) {
	fmt.Print("Enter Qdrant BaseURL (e.g., http://localhost:6333): ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("BaseURL is required")
	}

	fmt.Print("Enter Qdrant API Key (optional, press Enter to skip): ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	fmt.Print("Enter Collection/Namespace name: ")
	namespace, _ := reader.ReadString('\n')
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("Namespace is required")
	}

	fmt.Println("\nConnecting to Qdrant...")

	handler, err := knowledge.NewKnowledgeHandler(knowledge.HandlerFactoryParams{
		Provider:  knowledge.ProviderQdrant,
		Namespace: namespace,
		QdrantConfig: &knowledge.QdrantConfig{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Timeout: 30 * time.Second,
		},
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = handler.Ping(ctx)
	cancel()
	if err != nil {
		return nil, fmt.Errorf("error connecting to Qdrant: %v", err)
	}

	fmt.Print("✓ Connected to Qdrant\n\n")
	return handler, nil
}

func setupMilvus(reader *bufio.Reader) (knowledge.KnowledgeHandler, error) {
	fmt.Print("Enter Milvus Address (e.g., localhost:19530): ")
	address, _ := reader.ReadString('\n')
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("Address is required")
	}

	fmt.Print("Enter Milvus Username (optional): ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Enter Milvus Password (optional): ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	fmt.Print("Enter Collection/Namespace name: ")
	namespace, _ := reader.ReadString('\n')
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("Namespace is required")
	}

	fmt.Println("\nConnecting to Milvus...")

	handler, err := knowledge.NewKnowledgeHandler(knowledge.HandlerFactoryParams{
		Provider:  knowledge.ProviderMilvus,
		Namespace: namespace,
		MilvusConfig: &knowledge.MilvusConfig{
			Address:  address,
			Username: username,
			Password: password,
			DBName:   "default",
		},
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = handler.Ping(ctx)
	cancel()
	if err != nil {
		return nil, fmt.Errorf("error connecting to Milvus: %v", err)
	}

	fmt.Print("✓ Connected to Milvus\n\n")
	return handler, nil
}

func setupRAGFlow(reader *bufio.Reader) (knowledge.KnowledgeHandler, error) {
	fmt.Print("Enter RAGFlow BaseURL (e.g., http://localhost:9380): ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("BaseURL is required")
	}

	fmt.Print("Enter RAGFlow API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("API Key is required")
	}

	fmt.Print("Enter Dataset/Namespace name: ")
	namespace, _ := reader.ReadString('\n')
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("Namespace is required")
	}

	fmt.Println("\nConnecting to RAGFlow...")

	handler, err := knowledge.NewKnowledgeHandler(knowledge.HandlerFactoryParams{
		Provider:  knowledge.ProviderRAGFlow,
		Namespace: namespace,
		RAGFlowConfig: &knowledge.RAGFlowConfig{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Timeout: 30 * time.Second,
		},
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = handler.Ping(ctx)
	cancel()
	if err != nil {
		return nil, fmt.Errorf("error connecting to RAGFlow: %v", err)
	}

	fmt.Print("✓ Connected to RAGFlow\n\n")
	return handler, nil
}

func setupAliyun(reader *bufio.Reader) (knowledge.KnowledgeHandler, string, error) {
	fmt.Print("Enter Alibaba Bailian Endpoint (e.g., bailian.cn-beijing.aliyuncs.com or https://bailian.cn-beijing.aliyuncs.com): ")
	endpoint, _ := reader.ReadString('\n')
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, "", fmt.Errorf("Endpoint is required")
	}

	// Note: The Alibaba Cloud SDK will handle the protocol prefix
	// You can provide either "bailian.cn-beijing.aliyuncs.com" or "https://bailian.cn-beijing.aliyuncs.com"

	fmt.Print("Enter Access Key ID: ")
	accessKeyID, _ := reader.ReadString('\n')
	accessKeyID = strings.TrimSpace(accessKeyID)
	if accessKeyID == "" {
		return nil, "", fmt.Errorf("Access Key ID is required")
	}

	fmt.Print("Enter Access Key Secret: ")
	accessKeySecret, _ := reader.ReadString('\n')
	accessKeySecret = strings.TrimSpace(accessKeySecret)
	if accessKeySecret == "" {
		return nil, "", fmt.Errorf("Access Key Secret is required")
	}

	fmt.Print("Enter Workspace ID: ")
	workspaceID, _ := reader.ReadString('\n')
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, "", fmt.Errorf("Workspace ID is required")
	}

	fmt.Print("Enter Index/Namespace name: ")
	namespace, _ := reader.ReadString('\n')
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, "", fmt.Errorf("Namespace is required")
	}

	fmt.Println("\nConnecting to Alibaba Bailian...")

	handler, err := knowledge.NewKnowledgeHandler(knowledge.HandlerFactoryParams{
		Provider:  knowledge.ProviderAliyun,
		Namespace: namespace,
		AliyunConfig: &knowledge.AliyunConfig{
			Endpoint:        endpoint,
			AccessKeyID:     accessKeyID,
			AccessKeySecret: accessKeySecret,
			WorkspaceID:     workspaceID,
			Timeout:         30 * time.Second,
		},
	})
	if err != nil {
		return nil, "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = handler.Ping(ctx)
	cancel()
	if err != nil {
		return nil, "", fmt.Errorf("error connecting to Alibaba Bailian: %v", err)
	}

	fmt.Print("✓ Connected to Alibaba Bailian\n\n")
	return handler, namespace, nil
}

func interactiveSession(kb *knowledge.KnowledgeBase) {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║           Enter your query (type 'exit' to quit)            ║")
	fmt.Print("╚════════════════════════════════════════════════════════════╝\n")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Query: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		if input == "exit" {
			fmt.Println("\nGoodbye!")
			return
		}

		// Query the knowledge base with timing
		startTime := time.Now()
		results, err := kb.Query(context.Background(), input, 2)
		duration := time.Since(startTime)

		if err != nil {
			fmt.Printf("Error: %v\n\n", err)
			continue
		}

		if len(results) == 0 {
			fmt.Printf("No results found (took %.2fms)\n\n", duration.Seconds()*1000)
			continue
		}

		fmt.Printf("\nFound %d results (took %.2fms):\n\n", len(results), duration.Seconds()*1000)
		for i, result := range results {
			fmt.Printf("%d. %s\n", i+1, result.Record.Title)
			fmt.Printf("   Content: %s\n", result.Record.Content)
			fmt.Printf("   Score: %.4f\n\n", result.Score)
		}
	}
}

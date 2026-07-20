package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/LingByte/lingllm/embedder"
	"github.com/LingByte/lingllm/knowledge"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║          LingLLM Qdrant RAG Workflow Demo                   ║")
	fmt.Print("╚════════════════════════════════════════════════════════════╝\n")

	reader := bufio.NewReader(os.Stdin)

	// Step 1: Configure inputs
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║              Step 1: Configuration                         ║")
	fmt.Print("╚════════════════════════════════════════════════════════════╝\n")

	fmt.Print("Enter Qdrant BaseURL (default: http://localhost:6333): ")
	qdrantURL, _ := reader.ReadString('\n')
	qdrantURL = strings.TrimSpace(qdrantURL)
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333"
	}

	fmt.Print("Enter Qdrant API Key (optional, press Enter to skip): ")
	qdrantAPIKey, _ := reader.ReadString('\n')
	qdrantAPIKey = strings.TrimSpace(qdrantAPIKey)

	fmt.Print("Enter Collection name (default: documents): ")
	collectionName, _ := reader.ReadString('\n')
	collectionName = strings.TrimSpace(collectionName)
	if collectionName == "" {
		collectionName = "documents"
	}

	fmt.Print("Enter OpenAI API Key: ")
	openaiKey, _ := reader.ReadString('\n')
	openaiKey = strings.TrimSpace(openaiKey)
	if openaiKey == "" {
		fmt.Println("Error: OpenAI API Key is required")
		return
	}

	fmt.Print("Enter OpenAI Model (default: text-embedding-3-small): ")
	openaiModel, _ := reader.ReadString('\n')
	openaiModel = strings.TrimSpace(openaiModel)
	if openaiModel == "" {
		openaiModel = "text-embedding-3-small"
	}

	fmt.Print("Enter OpenAI BaseURL (default: https://api.openai.com/v1): ")
	openaiBaseURL, _ := reader.ReadString('\n')
	openaiBaseURL = strings.TrimSpace(openaiBaseURL)
	if openaiBaseURL == "" {
		openaiBaseURL = "https://api.openai.com/v1"
	}

	// Step 2: Initialize handlers
	fmt.Print("\n")
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║           Step 2: Initialize Handlers                      ║")
	fmt.Print("╚════════════════════════════════════════════════════════════╝\n")

	fmt.Println("Creating OpenAI embedder...")
	openaiEmbedder := embedder.NewOpenAIEmbedder(&embedder.Config{
		APIKey:  openaiKey,
		Model:   openaiModel,
		BaseURL: openaiBaseURL,
		Timeout: 30,
	})
	defer openaiEmbedder.Close()

	fmt.Printf("✓ OpenAI embedder created (model: %s, dimension: %d)\n\n", openaiModel, openaiEmbedder.Dimension())

	fmt.Println("Creating Qdrant handler...")
	qdrantHandler, err := knowledge.NewKnowledgeHandler(knowledge.HandlerFactoryParams{
		Provider:  knowledge.ProviderQdrant,
		Namespace: collectionName,
		QdrantConfig: &knowledge.QdrantConfig{
			BaseURL: qdrantURL,
			APIKey:  qdrantAPIKey,
			Timeout: 30 * time.Second,
		},
	})
	if err != nil {
		fmt.Printf("Error creating Qdrant handler: %v\n", err)
		return
	}

	fmt.Println("Testing Qdrant connection...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = qdrantHandler.Ping(ctx)
	cancel()
	if err != nil {
		fmt.Printf("Error connecting to Qdrant: %v\n", err)
		return
	}
	fmt.Print("✓ Connected to Qdrant\n\n")

	// Step 3: Prepare sample documents
	documents := []struct {
		title   string
		content string
		tags    []string
	}{
		{
			title: "Go Programming Language",
			content: "Go is an open-source programming language that makes it easy to build simple, reliable, and efficient software. " +
				"It is designed for concurrent programming and has excellent support for networking and systems programming. " +
				"Go's simplicity and performance make it ideal for cloud infrastructure and microservices. " +
				"Go features include fast compilation, built-in concurrency with goroutines and channels, garbage collection, " +
				"static typing with type inference, and a comprehensive standard library. " +
				"Go is widely used for building web servers, APIs, command-line tools, and distributed systems. " +
				"Popular Go projects include Docker, Kubernetes, Prometheus, and Terraform. " +
				"Go's syntax is clean and minimalist, making it easy to learn for developers from other languages. " +
				"The Go community is active and supportive, with many open-source libraries and frameworks available.",
			tags: []string{"programming", "golang", "backend"},
		},
		{
			title: "Python for Data Science",
			content: "Python has become the de facto standard for data science and machine learning. " +
				"Libraries like NumPy, Pandas, and Scikit-learn provide powerful tools for data analysis and modeling. " +
				"Python's simplicity and extensive ecosystem make it perfect for data scientists and researchers. " +
				"NumPy provides efficient numerical computing with multi-dimensional arrays and mathematical functions. " +
				"Pandas offers data manipulation and analysis with DataFrames and Series. " +
				"Scikit-learn provides machine learning algorithms for classification, regression, and clustering. " +
				"TensorFlow and PyTorch are popular deep learning frameworks for neural networks. " +
				"Jupyter notebooks enable interactive data exploration and visualization. " +
				"Matplotlib and Seaborn provide powerful data visualization capabilities. " +
				"Python's readability and extensive documentation make it ideal for collaborative data science projects.",
			tags: []string{"python", "data-science", "machine-learning"},
		},
		{
			title: "Kubernetes Container Orchestration",
			content: "Kubernetes is an open-source container orchestration platform that automates deployment, scaling, and management of containerized applications. " +
				"It provides declarative configuration and powerful automation capabilities for managing complex distributed systems. " +
				"Kubernetes has become the industry standard for container orchestration. " +
				"Key features include automatic scaling, self-healing, rolling updates, and service discovery. " +
				"Kubernetes uses pods as the smallest deployable units, containing one or more containers. " +
				"Services provide stable network endpoints for pod groups, enabling load balancing. " +
				"ConfigMaps and Secrets manage application configuration and sensitive data. " +
				"Persistent volumes handle stateful data storage in containerized environments. " +
				"Kubernetes is widely adopted by enterprises for managing microservices architectures.",
			tags: []string{"kubernetes", "containers", "devops"},
		},
		{
			title: "Machine Learning Fundamentals",
			content: "Machine learning is a subset of artificial intelligence that enables systems to learn and improve from experience without being explicitly programmed. " +
				"Key concepts include supervised learning, unsupervised learning, and reinforcement learning. " +
				"Understanding these fundamentals is essential for building effective ML systems. " +
				"Supervised learning uses labeled data to train models for classification and regression tasks. " +
				"Unsupervised learning discovers patterns in unlabeled data through clustering and dimensionality reduction. " +
				"Reinforcement learning trains agents to make sequential decisions through reward signals. " +
				"Feature engineering transforms raw data into meaningful features for model training. " +
				"Model evaluation uses metrics like accuracy, precision, recall, and F1-score. " +
				"Cross-validation prevents overfitting and ensures model generalization.",
			tags: []string{"machine-learning", "ai", "algorithms"},
		},
		{
			title: "Cloud Computing Architecture",
			content: "Cloud computing provides on-demand access to computing resources over the internet. " +
				"Major cloud providers like AWS, Azure, and Google Cloud offer various services including compute, storage, and databases. " +
				"Cloud architecture patterns help design scalable and resilient systems. " +
				"Infrastructure as a Service (IaaS) provides virtualized computing resources over the internet. " +
				"Platform as a Service (PaaS) offers development platforms and tools for building applications. " +
				"Software as a Service (SaaS) delivers applications over the internet. " +
				"Auto-scaling adjusts resources based on demand, optimizing costs and performance. " +
				"Load balancing distributes traffic across multiple servers for high availability. " +
				"Cloud security includes encryption, identity management, and compliance controls.",
			tags: []string{"cloud", "aws", "architecture"},
		},
		{
			title: "REST API Design Best Practices",
			content: "REST (Representational State Transfer) is an architectural style for designing networked applications. " +
				"Best practices include using proper HTTP methods, meaningful URLs, and appropriate status codes. " +
				"Well-designed REST APIs are easy to understand, use, and maintain. " +
				"Use GET for retrieving resources, POST for creating, PUT for updating, and DELETE for removing. " +
				"Resource URLs should be nouns (e.g., /users, /products) not verbs. " +
				"Versioning APIs ensures backward compatibility when making breaking changes. " +
				"Pagination handles large result sets efficiently. " +
				"Proper error responses with meaningful messages help API consumers debug issues. " +
				"Authentication and authorization protect API resources from unauthorized access.",
			tags: []string{"api", "rest", "web-development"},
		},
		{
			title: "Database Optimization Techniques",
			content: "Database optimization is crucial for application performance. " +
				"Techniques include proper indexing, query optimization, and connection pooling. " +
				"Understanding database internals helps design efficient data access patterns. " +
				"Indexes speed up data retrieval but slow down write operations, requiring careful planning. " +
				"Query analysis tools identify slow queries and optimization opportunities. " +
				"Denormalization trades storage for query performance in specific scenarios. " +
				"Connection pooling reduces overhead of creating new database connections. " +
				"Caching frequently accessed data reduces database load. " +
				"Partitioning large tables improves query performance and maintenance.",
			tags: []string{"database", "optimization", "sql"},
		},
		{
			title: "DevOps and CI/CD Pipelines",
			content: "DevOps practices emphasize collaboration between development and operations teams. " +
				"CI/CD pipelines automate testing and deployment processes, enabling faster and more reliable releases. " +
				"Tools like Jenkins, GitLab CI, and GitHub Actions facilitate modern DevOps workflows. " +
				"Continuous Integration automatically builds and tests code changes on every commit. " +
				"Continuous Deployment automatically releases validated changes to production. " +
				"Infrastructure as Code manages infrastructure through version-controlled code. " +
				"Monitoring and logging provide visibility into application and system health. " +
				"Incident response procedures minimize downtime and impact. " +
				"DevOps culture promotes shared responsibility and continuous improvement.",
			tags: []string{"devops", "cicd", "automation"},
		},
	}

	fmt.Printf("Prepared %d sample documents:\n\n", len(documents))
	for i, doc := range documents {
		fmt.Printf("%d. %s\n", i+1, doc.title)
	}
	fmt.Println()

	// Step 4: Create records and upsert
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║          Step 4: Upsert Documents to Qdrant                ║")
	fmt.Print("╚════════════════════════════════════════════════════════════╝\n")

	records := make([]knowledge.Record, 0, len(documents))
	for i, doc := range documents {
		record := knowledge.Record{
			ID:        fmt.Sprintf("%d", i+1),
			Title:     doc.title,
			Content:   doc.content,
			Source:    "qdrant-demo",
			Tags:      doc.tags,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata: map[string]any{
				"doc_index": i + 1,
				"category":  "technical",
			},
		}
		records = append(records, record)
	}

	fmt.Printf("Upserting %d documents...\n", len(records))

	// Set embedder on handler for upsert operation
	// We need to do this because Qdrant handler uses embedder for vector generation
	if qh, ok := qdrantHandler.(*knowledge.QdrantHandler); ok {
		qh.Embedder = openaiEmbedder
	}

	startTime := time.Now()
	ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
	err = qdrantHandler.Upsert(ctx, records, &knowledge.UpsertOptions{
		Namespace: collectionName,
		BatchSize: 10,
	})
	cancel()
	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("Error upserting documents: %v\n", err)
		return
	}

	fmt.Printf("✓ Successfully upserted %d documents in %.2fs\n\n", len(records), duration.Seconds())

	// Step 5: Create knowledge base and interactive session
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║          Step 5: Interactive Query Session                 ║")
	fmt.Print("╚════════════════════════════════════════════════════════════╝\n")

	kb, err := knowledge.NewKnowledgeBase(knowledge.KnowledgeBaseConfig{
		Handler:   qdrantHandler,
		Embedder:  openaiEmbedder,
		Namespace: collectionName,
	})
	if err != nil {
		fmt.Printf("Error creating knowledge base: %v\n", err)
		return
	}
	defer kb.Close()

	fmt.Print("Knowledge base initialized. Enter your queries (type 'exit' to quit):\n\n")

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

		// Query with timing
		startTime := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		results, err := kb.Query(ctx, input, 5)
		cancel()
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
			fmt.Printf("   Content: %s\n", truncateString(result.Record.Content, 100))
			fmt.Printf("   Score: %.4f\n", result.Score)
			if len(result.Record.Tags) > 0 {
				fmt.Printf("   Tags: %s\n", strings.Join(result.Record.Tags, ", "))
			}
			fmt.Println()
		}
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

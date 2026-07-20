package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/LingByte/lingllm/embedder"
)

func main() {
	provider := flag.String("provider", "local", "Embedder provider: openai, ollama, nvidia, dashscope, local")
	model := flag.String("model", "", "Model name")
	apiKey := flag.String("key", "", "API key (for openai, nvidia, dashscope)")
	baseURL := flag.String("url", "", "Base URL (for ollama, custom endpoints)")
	dimension := flag.Int("dim", 0, "Vector dimension (0 = default)")
	flag.Parse()

	fmt.Print("=== LingLLM Embedder Demo ===\n\n")

	// Set default model based on provider
	if *model == "" {
		switch *provider {
		case "local":
			*model = "local"
		case "ollama":
			*model = "nomic-embed-text"
		case "openai":
			*model = "text-embedding-3-small"
		case "nvidia":
			*model = "nvidia/nv-embed-qa-4"
		case "dashscope":
			*model = "text-embedding-v4"
		}
	}

	// Create embedder based on provider
	cfg := &embedder.Config{
		Provider:  *provider,
		Model:     *model,
		APIKey:    *apiKey,
		BaseURL:   *baseURL,
		Dimension: *dimension,
		Timeout:   30,
	}

	ctx := context.Background()
	emb, err := embedder.Create(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}
	defer emb.Close()

	fmt.Printf("✓ Created %s embedder\n", emb.Provider())
	fmt.Printf("  Model: %s\n", *model)
	fmt.Printf("  Dimension: %d\n", emb.Dimension())
	if *baseURL != "" {
		fmt.Printf("  Base URL: %s\n", *baseURL)
	}
	fmt.Println()

	// Test texts
	texts := []string{
		"The quick brown fox jumps over the lazy dog",
		"Machine learning is a subset of artificial intelligence",
		"Natural language processing enables computers to understand human language",
		"Deep learning uses neural networks with multiple layers",
		"Vector embeddings represent text as numerical values",
	}

	// Demo 1: Single text embedding
	demoSingleEmbedding(ctx, emb, texts[0])

	// Demo 2: Batch embedding
	demoBatchEmbedding(ctx, emb, texts)

	// Demo 3: Semantic similarity
	demoSemanticSimilarity(ctx, emb, texts)

	// Demo 4: Performance metrics
	demoPerformanceMetrics(ctx, emb, texts)
}

// demoSingleEmbedding demonstrates single text embedding
func demoSingleEmbedding(ctx context.Context, emb embedder.Embedder, text string) {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║ Demo 1: Single Text Embedding")
	fmt.Print("╚════════════════════════════════════════════════════════════╝\n")

	start := time.Now()
	vector, err := emb.EmbedSingle(ctx, text)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Error embedding text: %v", err)
		return
	}

	fmt.Printf("Text: %s\n", text)
	fmt.Printf("Vector dimension: %d\n", len(vector))
	fmt.Printf("Vector norm: %.4f\n", calculateNorm(vector))
	fmt.Printf("First 5 values: [")
	for i := 0; i < 5 && i < len(vector); i++ {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("%.6f", vector[i])
	}
	fmt.Printf("]\n")
	fmt.Printf("Time: %v\n\n", duration)
}

// demoBatchEmbedding demonstrates batch embedding
func demoBatchEmbedding(ctx context.Context, emb embedder.Embedder, texts []string) {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║ Demo 2: Batch Embedding")
	fmt.Print("╚════════════════════════════════════════════════════════════╝\n")

	start := time.Now()
	vectors, err := emb.Embed(ctx, texts)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Error embedding texts: %v", err)
		return
	}

	fmt.Printf("✓ Embedded %d texts\n", len(vectors))
	fmt.Printf("Time: %v\n", duration)
	fmt.Printf("Average time per text: %v\n\n", duration/time.Duration(len(texts)))

	for i, text := range texts {
		if i < len(vectors) {
			norm := calculateNorm(vectors[i])
			fmt.Printf("%d. %s\n", i+1, truncate(text, 50))
			fmt.Printf("   Norm: %.4f\n", norm)
		}
	}
	fmt.Println()
}

// demoSemanticSimilarity demonstrates semantic similarity calculation
func demoSemanticSimilarity(ctx context.Context, emb embedder.Embedder, texts []string) {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║ Demo 3: Semantic Similarity (Cosine Distance)")
	fmt.Print("╚════════════════════════════════════════════════════════════╝\n")

	vectors, err := emb.Embed(ctx, texts)
	if err != nil {
		log.Printf("Error embedding texts: %v", err)
		return
	}

	// Calculate similarity matrix
	fmt.Printf("Similarity Matrix (%d x %d):\n\n", len(texts), len(texts))
	fmt.Print("     ")
	for i := 0; i < len(texts); i++ {
		fmt.Printf("  T%d  ", i+1)
	}
	fmt.Println()

	for i := 0; i < len(texts); i++ {
		fmt.Printf("T%d   ", i+1)
		for j := 0; j < len(texts); j++ {
			similarity := cosineSimilarity(vectors[i], vectors[j])
			fmt.Printf("%.3f ", similarity)
		}
		fmt.Println()
	}
	fmt.Println()

	// Find most similar pairs
	fmt.Println("Most Similar Pairs:")
	type pair struct {
		i, j       int
		similarity float32
	}
	var pairs []pair

	for i := 0; i < len(texts); i++ {
		for j := i + 1; j < len(texts); j++ {
			sim := cosineSimilarity(vectors[i], vectors[j])
			pairs = append(pairs, pair{i, j, sim})
		}
	}

	// Sort by similarity (simple bubble sort for demo)
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].similarity > pairs[i].similarity {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	for i := 0; i < 3 && i < len(pairs); i++ {
		p := pairs[i]
		fmt.Printf("%d. T%d ↔ T%d: %.4f\n", i+1, p.i+1, p.j+1, p.similarity)
		fmt.Printf("   %s\n", truncate(texts[p.i], 40))
		fmt.Printf("   %s\n", truncate(texts[p.j], 40))
	}
	fmt.Println()
}

// demoPerformanceMetrics demonstrates performance metrics
func demoPerformanceMetrics(ctx context.Context, emb embedder.Embedder, texts []string) {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║ Demo 4: Performance Metrics")
	fmt.Print("╚════════════════════════════════════════════════════════════╝\n")

	// Warm up
	emb.Embed(ctx, texts[:1])

	// Measure performance
	iterations := 3
	var totalDuration time.Duration

	fmt.Printf("Running %d iterations...\n\n", iterations)

	for iter := 0; iter < iterations; iter++ {
		start := time.Now()
		_, err := emb.Embed(ctx, texts)
		duration := time.Since(start)

		if err != nil {
			log.Printf("Error in iteration %d: %v", iter+1, err)
			return
		}

		totalDuration += duration
		fmt.Printf("Iteration %d: %v\n", iter+1, duration)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	fmt.Printf("\nAverage duration: %v\n", avgDuration)
	fmt.Printf("Texts per second: %.2f\n", float64(len(texts))*float64(time.Second)/float64(avgDuration))
	fmt.Println()
}

// Helper functions

func calculateNorm(vector []float32) float32 {
	var sum float32
	for _, v := range vector {
		sum += v * v
	}
	return float32(math.Sqrt(float64(sum)))
}

func cosineSimilarity(v1, v2 []float32) float32 {
	if len(v1) != len(v2) {
		return 0
	}

	var dotProduct float32
	var norm1, norm2 float32

	for i := range v1 {
		dotProduct += v1[i] * v2[i]
		norm1 += v1[i] * v1[i]
		norm2 += v2[i] * v2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0
	}

	return dotProduct / float32(math.Sqrt(float64(norm1*norm2)))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

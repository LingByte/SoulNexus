# Embedder Demo

Complete demonstration of LingLLM's embedder module with support for multiple providers.

## Features

### 4 Demo Scenarios

1. **Single Text Embedding**
   - Embed a single text
   - Display vector statistics
   - Show performance metrics

2. **Batch Embedding**
   - Embed multiple texts efficiently
   - Display per-text statistics
   - Measure throughput

3. **Semantic Similarity**
   - Calculate cosine similarity matrix
   - Find most similar text pairs
   - Visualize relationships

4. **Performance Metrics**
   - Measure embedding latency
   - Calculate throughput
   - Compare providers

## Supported Providers

### 1. Local (Default)
```bash
go run examples/embedder-demo/main.go -provider local
```
- No API key required
- Deterministic embeddings
- Fast for testing

### 2. Ollama (Local Deployment)
```bash
# Start Ollama first
ollama serve

# In another terminal
go run examples/embedder-demo/main.go \
  -provider ollama \
  -model nomic-embed-text \
  -url http://localhost:11434
```
- Requires Ollama running locally
- Supports various embedding models
- Good for offline use

### 3. Nvidia (Enterprise)
```bash
go run examples/embedder-demo/main.go \
  -provider nvidia \
  -model nvidia/nv-embed-qa-4 \
  -key nvapi-xxxxxxxxxxxxxxxxxxxxxxxx
```
- Requires Nvidia API key
- High-quality embeddings
- Batch processing support

### 4. DashScope (Alibaba)
```bash
go run examples/embedder-demo/main.go \
  -provider dashscope \
  -model text-embedding-v4 \
  -key sk-xxxxxxxxxxxxxxxxxxxxxxxx \
  -dim 1024
```
- Requires DashScope API key
- Supports v3 and v4 models
- Flexible dimensions (64-2048)

### 5. OpenAI
```bash
go run examples/embedder-demo/main.go \
  -provider openai \
  -model text-embedding-3-small \
  -key sk-xxxxxxxxxxxxxxxxxxxxxxxx
```
- Requires OpenAI API key
- High-quality embeddings
- Supports multiple models

## Usage Examples

### Test Local Embedder
```bash
go run examples/embedder-demo/main.go
```

### Test Ollama with Custom Model
```bash
go run examples/embedder-demo/main.go \
  -provider ollama \
  -model all-minilm:22m \
  -url http://localhost:11434
```

### Test DashScope with Custom Dimension
```bash
go run examples/embedder-demo/main.go \
  -provider dashscope \
  -model text-embedding-v4 \
  -key $DASHSCOPE_API_KEY \
  -dim 1536
```

### Test Nvidia with Custom URL
```bash
go run examples/embedder-demo/main.go \
  -provider nvidia \
  -model nvidia/nv-embed-qa-4 \
  -key $NVIDIA_API_KEY \
  -url https://api.nvcf.nvidia.com/v2
```

## Command Line Options

```
-provider string
    Embedder provider: openai, ollama, nvidia, dashscope, local (default "local")

-model string
    Model name (required for most providers)

-key string
    API key (for openai, nvidia, dashscope)

-url string
    Base URL (for ollama, custom endpoints)

-dim int
    Vector dimension (0 = default)
```

## Output Example

```
=== LingLLM Embedder Demo ===

✓ Created dashscope embedder
  Model: text-embedding-v4
  Dimension: 1024

╔════════════════════════════════════════════════════════════╗
║ Demo 1: Single Text Embedding
╚════════════════════════════════════════════════════════════╝

Text: The quick brown fox jumps over the lazy dog
Vector dimension: 1024
Vector norm: 1.0000
First 5 values: [-0.006929, 0.030681, -0.069539, 0.015234, -0.042156]
Time: 234ms

╔════════════════════════════════════════════════════════════╗
║ Demo 2: Batch Embedding
╚════════════════════════════════════════════════════════════╝

✓ Embedded 5 texts
Time: 456ms
Average time per text: 91ms

1. The quick brown fox jumps over the lazy dog
   Norm: 1.0000
2. Machine learning is a subset of artificial intelligence
   Norm: 1.0000
...

╔════════════════════════════════════════════════════════════╗
║ Demo 3: Semantic Similarity (Cosine Distance)
╚════════════════════════════════════════════════════════════╝

Similarity Matrix (5 x 5):

       T1    T2    T3    T4    T5  
T1   1.000 0.234 0.156 0.189 0.412
T2   0.234 1.000 0.678 0.523 0.345
T3   0.156 0.678 1.000 0.612 0.278
T4   0.189 0.523 0.612 1.000 0.401
T5   0.412 0.345 0.278 0.401 1.000

Most Similar Pairs:
1. T2 ↔ T3: 0.6780
   Machine learning is a subset of artificial intelligence
   Natural language processing enables computers to understand...
2. T3 ↔ T4: 0.6120
   Natural language processing enables computers to understand...
   Deep learning uses neural networks with multiple layers
3. T2 ↔ T4: 0.5230
   Machine learning is a subset of artificial intelligence
   Deep learning uses neural networks with multiple layers

╔════════════════════════════════════════════════════════════╗
║ Demo 4: Performance Metrics
╚════════════════════════════════════════════════════════════╝

Running 3 iterations...

Iteration 1: 456ms
Iteration 2: 423ms
Iteration 3: 445ms

Average duration: 441ms
Texts per second: 11.34
```

## Performance Comparison

### Expected Performance (Approximate)

| Provider | Latency | Throughput | Quality | Cost |
|----------|---------|-----------|---------|------|
| Local | <1ms | 1000+/s | Low | Free |
| Ollama | 10-50ms | 20-100/s | Medium | Free |
| DashScope | 50-200ms | 5-20/s | High | ¥0.0005/1K tokens |
| Nvidia | 100-300ms | 3-10/s | High | Variable |
| OpenAI | 100-300ms | 3-10/s | Very High | $0.02/1M tokens |

## Use Cases

### 1. Semantic Search
```go
// Embed query and documents
queryVec, _ := emb.EmbedSingle(ctx, "machine learning")
docVecs, _ := emb.Embed(ctx, documents)

// Calculate similarity and rank
for i, docVec := range docVecs {
    sim := cosineSimilarity(queryVec, docVec)
    // Use similarity for ranking
}
```

### 2. Document Clustering
```go
// Embed all documents
vectors, _ := emb.Embed(ctx, documents)

// Use vectors with clustering algorithm
// (k-means, DBSCAN, etc.)
```

### 3. Recommendation System
```go
// Embed user preferences and items
userVec, _ := emb.EmbedSingle(ctx, userProfile)
itemVecs, _ := emb.Embed(ctx, items)

// Find similar items
```

### 4. Duplicate Detection
```go
// Embed documents
vectors, _ := emb.Embed(ctx, documents)

// Find pairs with high similarity
for i := 0; i < len(vectors); i++ {
    for j := i + 1; j < len(vectors); j++ {
        if cosineSimilarity(vectors[i], vectors[j]) > 0.95 {
            // Likely duplicates
        }
    }
}
```

## Testing Different Providers

### Comparison Test Script

```bash
#!/bin/bash

echo "Testing Local Embedder..."
go run examples/embedder-demo/main.go -provider local

echo -e "\n\nTesting Ollama Embedder..."
go run examples/embedder-demo/main.go \
  -provider ollama \
  -model nomic-embed-text

echo -e "\n\nTesting DashScope Embedder..."
go run examples/embedder-demo/main.go \
  -provider dashscope \
  -model text-embedding-v4 \
  -key $DASHSCOPE_API_KEY

echo -e "\n\nTesting Nvidia Embedder..."
go run examples/embedder-demo/main.go \
  -provider nvidia \
  -model nvidia/nv-embed-qa-4 \
  -key $NVIDIA_API_KEY
```

## Troubleshooting

### Ollama Connection Error
- Ensure Ollama is running: `ollama serve`
- Check URL is correct: `http://localhost:11434`
- Verify model is installed: `ollama list`

### API Key Error
- Check API key is valid
- Verify key is set in environment or command line
- Check API key has required permissions

### Dimension Mismatch
- DashScope: Supports 64, 128, 256, 512, 768, 1024, 1536, 2048
- Nvidia: Check model documentation
- OpenAI: Depends on model (usually 1536)

### Rate Limiting
- Reduce batch size
- Add delays between requests
- Use local embedder for testing

## Next Steps

1. Try different providers and compare quality
2. Benchmark performance with your data
3. Integrate into your application
4. Use embeddings for semantic search or clustering
5. Monitor costs and performance

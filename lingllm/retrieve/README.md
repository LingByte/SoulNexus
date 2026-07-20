# Retrieve Module

Strategy-based document retrieval with support for vector similarity, full-text search, and hybrid approaches.

## Features

### Three Retrieval Strategies

1. **Vector Strategy** (`StrategyVector`)
   - Dense vector similarity search
   - Ideal for semantic search
   - Requires: `VectorRetriever`

2. **Keyword Strategy** (`StrategyKeyword`)
   - Full-text search
   - Fast keyword matching
   - Requires: `SearchEngine`

3. **Hybrid Strategy** (`StrategyHybrid`)
   - Combines vector and keyword results
   - Configurable weighting
   - Requires: Both `VectorRetriever` and `SearchEngine`

### Optional Features

- **Reranking**: Re-score documents after initial retrieval
- **Min Score Filtering**: Filter results by minimum score threshold
- **Custom Keyword Fields**: Specify which fields to search
- **Vector Weight Control**: Adjust vector vs keyword importance in hybrid mode

## Usage

### Vector Retrieval

```go
retriever, err := retrieve.New(retrieve.Config{
    Strategy: retrieve.StrategyVector,
    Vector:   vectorStore,
    TopK:     10,
})
if err != nil {
    log.Fatal(err)
}

docs, err := retriever.Retrieve(ctx, "machine learning", 10)
```

### Keyword Retrieval

```go
retriever, err := retrieve.New(retrieve.Config{
    Strategy:      retrieve.StrategyKeyword,
    Search:        searchEngine,
    TopK:          10,
    KeywordFields: []string{"title", "content"},
})
if err != nil {
    log.Fatal(err)
}

docs, err := retriever.Retrieve(ctx, "neural networks", 10)
```

### Hybrid Retrieval

```go
retriever, err := retrieve.New(retrieve.Config{
    Strategy:     retrieve.StrategyHybrid,
    Vector:       vectorStore,
    Search:       searchEngine,
    TopK:         10,
    VectorWeight: 0.65,  // 65% vector, 35% keyword
})
if err != nil {
    log.Fatal(err)
}

docs, err := retriever.Retrieve(ctx, "deep learning", 10)
```

### With Reranking

```go
retriever, err := retrieve.New(retrieve.Config{
    Strategy:         retrieve.StrategyVector,
    Vector:           vectorStore,
    TopK:             5,
    Reranker:         reranker,
    RerankCandidates: 20,  // Fetch 20, rerank to 5
})
if err != nil {
    log.Fatal(err)
}

docs, err := retriever.Retrieve(ctx, "query", 5)
```

## Configuration

### Config Fields

```go
type Config struct {
    // Strategy: vector, keyword, or hybrid
    Strategy Strategy
    
    // VectorRetriever for vector-based search
    Vector VectorRetriever
    
    // SearchEngine for keyword-based search
    Search SearchEngine
    
    // TopK: number of results to return (default: 3)
    TopK int
    
    // MinScore: minimum score threshold (default: 0)
    MinScore float64
    
    // KeywordFields: fields to search in keyword mode
    KeywordFields []string
    
    // VectorWeight: weight for vector scores in hybrid mode (0-1, default: 0.65)
    VectorWeight float64
    
    // Reranker: optional reranking function
    Reranker Reranker
    
    // RerankCandidates: docs to fetch before reranking (default: TopK*3)
    RerankCandidates int
}
```

## Interfaces

### VectorRetriever

```go
type VectorRetriever interface {
    Retrieve(ctx context.Context, query string, topK int) ([]*Document, error)
}
```

### SearchEngine

```go
type SearchEngine interface {
    Search(ctx context.Context, query string, fields []string, size int) ([]SearchHit, error)
}
```

### Reranker

```go
type Reranker interface {
    Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)
}
```

## Document Structure

```go
type Document struct {
    ID       string                 // Document ID
    Content  string                 // Document content
    Score    float64                // Relevance score
    Metadata map[string]string      // Additional metadata
}
```

## Hybrid Search Details

In hybrid mode, the retriever:

1. Fetches results from both vector and keyword searches
2. Merges results by document ID
3. Combines scores using configurable weights:
   - Vector score: `score * VectorWeight`
   - Keyword score: `score * (1 - VectorWeight)`
4. Sorts by combined score
5. Returns top K results

### Example Weighting

```
VectorWeight = 0.65 (default)
KeywordWeight = 0.35

Combined Score = (VectorScore * 0.65) + (KeywordScore * 0.35)
```

## Testing

Run tests with:

```bash
go test ./retrieve -v
```

Test coverage includes:
- Vector retrieval
- Keyword retrieval
- Hybrid retrieval
- Reranking
- Configuration validation
- Default values
- Error handling

## Performance Considerations

### Vector Strategy
- **Pros**: Semantic understanding, fast
- **Cons**: Requires vector embeddings
- **Best for**: Semantic search, similarity matching

### Keyword Strategy
- **Pros**: Exact matching, no embeddings needed
- **Cons**: No semantic understanding
- **Best for**: Exact phrase search, known keywords

### Hybrid Strategy
- **Pros**: Combines strengths of both
- **Cons**: Slower than single strategy
- **Best for**: Balanced search quality

## Examples

### Semantic Search with Reranking

```go
retriever, _ := retrieve.New(retrieve.Config{
    Strategy:         retrieve.StrategyVector,
    Vector:           vectorStore,
    TopK:             5,
    Reranker:         crossEncoder,
    RerankCandidates: 20,
})

docs, _ := retriever.Retrieve(ctx, "best practices", 5)
// Fetches 20 similar docs, reranks to 5 best matches
```

### Keyword Search with Filtering

```go
retriever, _ := retrieve.New(retrieve.Config{
    Strategy:      retrieve.StrategyKeyword,
    Search:        searchEngine,
    TopK:          10,
    MinScore:      0.5,
    KeywordFields: []string{"title", "tags"},
})

docs, _ := retriever.Retrieve(ctx, "golang", 10)
// Only returns docs with score >= 0.5
```

### Balanced Hybrid Search

```go
retriever, _ := retrieve.New(retrieve.Config{
    Strategy:     retrieve.StrategyHybrid,
    Vector:       vectorStore,
    Search:       searchEngine,
    TopK:         10,
    VectorWeight: 0.5,  // Equal weighting
})

docs, _ := retriever.Retrieve(ctx, "query", 10)
// Balances semantic and keyword relevance equally
```

## Error Handling

```go
docs, err := retriever.Retrieve(ctx, query, topK)
if err != nil {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        // Handle timeout
    case strings.Contains(err.Error(), "rerank"):
        // Handle reranking error
    default:
        // Handle other errors
    }
}
```

## Future Enhancements

- [ ] Caching layer for frequent queries
- [ ] Query expansion and reformulation
- [ ] Multi-field vector search
- [ ] Faceted search support
- [ ] Query analytics and logging
- [ ] Distributed retrieval

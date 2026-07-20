package turn

// KnowledgeHitRecord is one retrieved chunk attached to a dialog turn.
type KnowledgeHitRecord struct {
	Title    string  `json:"title,omitempty"`
	Content  string  `json:"content,omitempty"`
	Source   string  `json:"source,omitempty"`
	Score    float64 `json:"score,omitempty"`
	RecordID string  `json:"recordId,omitempty"`
	ChunkID  uint    `json:"chunkId,omitempty"`
	Quoted   bool    `json:"quoted,omitempty"`
}

// KnowledgeRetrievalRecord captures one search_knowledge_base invocation for audit.
type KnowledgeRetrievalRecord struct {
	Query       string               `json:"query,omitempty"`
	SearchQuery string               `json:"searchQuery,omitempty"`
	Strategy    string               `json:"strategy,omitempty"`
	RecallMs    int64                `json:"recallMs,omitempty"`
	EmbedMs     int64                `json:"embedMs,omitempty"`
	QdrantMs    int64                `json:"qdrantMs,omitempty"`
	HitCount    int                  `json:"hitCount,omitempty"`
	Hits        []KnowledgeHitRecord `json:"hits,omitempty"`
}

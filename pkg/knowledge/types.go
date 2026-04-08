package knowledge

import (
	"context"
	"time"
)

const (
	// KnowledgeAliyun Aliyun Bailian Knowledge Base
	KnowledgeAliyun = "aliyun"
	// KnowledgeQdrant Qdrant Vector Database
	KnowledgeQdrant = "qdrant"
)

// KnowledgeProvider common provider type
type KnowledgeProvider string

// ToString toString for llm
func (kp KnowledgeProvider) ToString() string {
	return string(kp)
}

type Record struct {
	ID        string
	Source    string
	Title     string
	Content   string
	Tags      []string
	Metadata  map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UpsertOptions struct {
	Namespace string
	Overwrite bool
}

type QueryOptions struct {
	Namespace string
	TopK      int
	Filters   map[string]any
}

type QueryResult struct {
	Record Record
	Score  float64
}

type DeleteOptions struct {
	Namespace string
}

type GetOptions struct {
	Namespace string
}

type ListOptions struct {
	Namespace string
	Limit     int
	Offset    string
	Filters   map[string]any
}

type ListResult struct {
	Records    []Record
	NextOffset string
}

type KnowledgeHandler interface {
	Provider() string

	Upsert(ctx context.Context, records []Record, options *UpsertOptions) error

	Query(ctx context.Context, text string, options *QueryOptions) ([]QueryResult, error)

	Get(ctx context.Context, ids []string, options *GetOptions) ([]Record, error)

	List(ctx context.Context, options *ListOptions) (*ListResult, error)

	Delete(ctx context.Context, ids []string, options *DeleteOptions) error
}

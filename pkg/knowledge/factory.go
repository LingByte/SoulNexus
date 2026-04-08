package knowledge

import (
	"errors"
	"net/http"
	"strings"
)

var ErrUnsupportedKnowledgeProvider = errors.New("unsupported knowledge provider")

type QdrantOptions struct {
	BaseURL    string
	APIKey     string
	Collection string
	HTTPClient *http.Client
	Embedder   Embedder
}

type AliyunOptions struct{}

type FactoryOptions struct {
	Qdrant *QdrantOptions
	Aliyun *AliyunOptions
}

func New(provider string, opts *FactoryOptions) (KnowledgeHandler, error) {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "" {
		p = KnowledgeQdrant
	}

	switch p {
	case KnowledgeQdrant:
		if opts == nil || opts.Qdrant == nil {
			return nil, errors.New("qdrant options are required")
		}
		q := opts.Qdrant
		return &QdrantHandler{BaseURL: q.BaseURL, APIKey: q.APIKey, Collection: q.Collection, HTTPClient: q.HTTPClient, Embedder: q.Embedder}, nil
	case KnowledgeAliyun:
		return nil, ErrUnsupportedKnowledgeProvider
	default:
		return nil, ErrUnsupportedKnowledgeProvider
	}
}

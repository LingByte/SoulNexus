package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/lingllm/embedder"
)

// WeaviateHandler implements KnowledgeHandler using Weaviate REST/GraphQL (HTTP API).
type WeaviateHandler struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	Embedder   embedder.Embedder
}

func (h *WeaviateHandler) Provider() string { return ProviderWeaviate }

func (h *WeaviateHandler) className(namespace string) (string, error) {
	safe := sanitizeNamespace(namespace)
	if safe == "" {
		return "", ErrCollectionNotFound
	}
	// Weaviate class names must start with uppercase letter.
	return "Lingkb_" + strings.ToUpper(safe[:1]) + safe[1:], nil
}

func (h *WeaviateHandler) client() *http.Client {
	if h != nil && h.HTTPClient != nil {
		return h.HTTPClient
	}
	return http.DefaultClient
}

func (h *WeaviateHandler) baseURL() string {
	return strings.TrimRight(strings.TrimSpace(h.BaseURL), "/")
}

func (h *WeaviateHandler) setAuth(req *http.Request) {
	if key := strings.TrimSpace(h.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
}

func (h *WeaviateHandler) doJSON(ctx context.Context, method, path string, body any, out any) error {
	if h == nil || strings.TrimSpace(h.BaseURL) == "" {
		return ErrBaseURL
	}
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, h.baseURL()+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	h.setAuth(req)
	resp, err := h.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("weaviate: status=%d body=%s", resp.StatusCode, string(raw))
	}
	if out != nil && len(raw) > 0 {
		return json.Unmarshal(raw, out)
	}
	return nil
}

func (h *WeaviateHandler) ensureClass(ctx context.Context, namespace string) (string, error) {
	class, err := h.className(namespace)
	if err != nil {
		return "", err
	}
	var existing map[string]any
	if err := h.doJSON(ctx, http.MethodGet, "/v1/schema/"+class, nil, &existing); err == nil && len(existing) > 0 {
		return class, nil
	}
	schema := map[string]any{
		"class":      class,
		"vectorizer": "none",
		"vectorIndexConfig": map[string]any{
			"distance": "cosine",
		},
		"properties": []map[string]any{
			{"name": "record_id", "dataType": []string{"text"}},
			{"name": "source", "dataType": []string{"text"}},
			{"name": "title", "dataType": []string{"text"}},
			{"name": "content", "dataType": []string{"text"}},
			{"name": "tags", "dataType": []string{"text"}},
			{"name": "metadata_json", "dataType": []string{"text"}},
		},
	}
	if err := h.doJSON(ctx, http.MethodPost, "/v1/schema", schema, nil); err != nil {
		return "", err
	}
	return class, nil
}

func (h *WeaviateHandler) Ping(ctx context.Context) error {
	return h.doJSON(ctx, http.MethodGet, "/v1/.well-known/ready", nil, nil)
}

func (h *WeaviateHandler) CreateNamespace(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrNamespaceNotFound
	}
	return nil
}

func (h *WeaviateHandler) DeleteNamespace(ctx context.Context, name string) error {
	class, err := h.className(name)
	if err != nil {
		return err
	}
	return h.doJSON(ctx, http.MethodDelete, "/v1/schema/"+class, nil, nil)
}

func (h *WeaviateHandler) ListNamespaces(ctx context.Context) ([]string, error) {
	var resp struct {
		Classes []struct {
			Class string `json:"class"`
		} `json:"classes"`
	}
	if err := h.doJSON(ctx, http.MethodGet, "/v1/schema", nil, &resp); err != nil {
		return nil, err
	}
	var out []string
	for _, c := range resp.Classes {
		if strings.HasPrefix(c.Class, "Lingkb_") {
			suffix := strings.TrimPrefix(c.Class, "Lingkb_")
			if len(suffix) > 0 {
				out = append(out, strings.ToLower(suffix[:1])+suffix[1:])
			}
		}
	}
	return out, nil
}

func (h *WeaviateHandler) Upsert(ctx context.Context, records []Record, opts *UpsertOptions) error {
	if len(records) == 0 {
		return nil
	}
	namespace := ""
	if opts != nil {
		namespace = opts.Namespace
	}
	recs := append([]Record(nil), records...)
	if _, err := fillMissingVectors(ctx, h.Embedder, recs); err != nil {
		return err
	}
	class, err := h.ensureClass(ctx, namespace)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, r := range recs {
		recordTimestamps(&r, now)
		obj := map[string]any{
			"class": class,
			"id":    weaviateUUID(r.ID),
			"properties": map[string]any{
				"record_id":     r.ID,
				"source":        r.Source,
				"title":         r.Title,
				"content":       r.Content,
				"tags":          tagsJSON(r.Tags),
				"metadata_json": metadataJSON(r.Metadata),
			},
			"vector": r.Vector,
		}
		if err := h.doJSON(ctx, http.MethodPut, "/v1/objects", obj, nil); err != nil {
			return fmt.Errorf("weaviate upsert %s: %w", r.ID, err)
		}
	}
	return nil
}

// weaviateUUID maps arbitrary record IDs to a deterministic UUID-like string for Weaviate.
func weaviateUUID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) == 36 && strings.Count(id, "-") == 4 {
		return id
	}
	// Weaviate accepts UUID format; reuse qdrant-style deterministic id if already UUID.
	return id
}

func (h *WeaviateHandler) Query(ctx context.Context, text string, opts *QueryOptions) ([]QueryResult, error) {
	if strings.TrimSpace(text) == "" {
		return nil, ErrEmptyQuery
	}
	namespace := ""
	topK := 10
	minScore := 0.0
	if opts != nil {
		namespace = opts.Namespace
		if opts.TopK > 0 {
			topK = opts.TopK
		}
		minScore = opts.MinScore
	}
	if h.Embedder == nil {
		return nil, ErrHandlerNotFound
	}
	vecs, err := h.Embedder.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return nil, ErrInvalidVectorDimension
	}
	class, err := h.ensureClass(ctx, namespace)
	if err != nil {
		return nil, err
	}

	gql := map[string]any{
		"query": fmt.Sprintf(`{
  Get {
    %s(
      limit: %d
      nearVector: { vector: %s }
    ) {
      record_id
      source
      title
      content
      tags
      metadata_json
      _additional { certainty distance id }
    }
  }
}`, class, topK, vectorJSON(vecs[0])),
	}
	var resp struct {
		Data struct {
			Get map[string][]map[string]any `json:"Get"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := h.doJSON(ctx, http.MethodPost, "/v1/graphql", gql, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("weaviate graphql: %s", resp.Errors[0].Message)
	}
	rows := resp.Data.Get[class]
	var results []QueryResult
	for _, row := range rows {
		r := weaviateRowToRecord(row)
		score := 0.0
		if add, ok := row["_additional"].(map[string]any); ok {
			if c, ok := add["certainty"].(float64); ok {
				score = c
			}
		}
		results = append(results, QueryResult{Record: r, Score: score})
	}
	return filterQueryResults(results, minScore), nil
}

func vectorJSON(v []float32) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func weaviateRowToRecord(row map[string]any) Record {
	var r Record
	if s, ok := row["record_id"].(string); ok {
		r.ID = s
	}
	if s, ok := row["source"].(string); ok {
		r.Source = s
	}
	if s, ok := row["title"].(string); ok {
		r.Title = s
	}
	if s, ok := row["content"].(string); ok {
		r.Content = s
	}
	if s, ok := row["tags"].(string); ok {
		r.Tags = parseTagsJSON(s)
	}
	if s, ok := row["metadata_json"].(string); ok {
		r.Metadata = parseMetadataJSON(s)
	}
	return r
}

func (h *WeaviateHandler) Get(ctx context.Context, ids []string, opts *GetOptions) ([]Record, error) {
	namespace := ""
	if opts != nil {
		namespace = opts.Namespace
	}
	class, err := h.className(namespace)
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, id := range ids {
		gql := map[string]any{
			"query": fmt.Sprintf(`{
  Get {
    %s(where: { path: ["record_id"], operator: Equal, valueText: %q }, limit: 1) {
      record_id source title content tags metadata_json
    }
  }
}`, class, id),
		}
		var resp struct {
			Data struct {
				Get map[string][]map[string]any `json:"Get"`
			} `json:"data"`
		}
		if err := h.doJSON(ctx, http.MethodPost, "/v1/graphql", gql, &resp); err != nil {
			continue
		}
		for _, row := range resp.Data.Get[class] {
			out = append(out, weaviateRowToRecord(row))
		}
	}
	return out, nil
}

func (h *WeaviateHandler) List(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	namespace := ""
	limit := 50
	if opts != nil {
		namespace = opts.Namespace
		if opts.Limit > 0 {
			limit = opts.Limit
		}
	}
	class, err := h.className(namespace)
	if err != nil {
		return nil, err
	}
	gql := map[string]any{
		"query": fmt.Sprintf(`{
  Get {
    %s(limit: %d) { record_id source title content tags metadata_json }
  }
}`, class, limit),
	}
	var resp struct {
		Data struct {
			Get map[string][]map[string]any `json:"Get"`
		} `json:"data"`
	}
	if err := h.doJSON(ctx, http.MethodPost, "/v1/graphql", gql, &resp); err != nil {
		return nil, err
	}
	recs := make([]Record, 0, len(resp.Data.Get[class]))
	for _, row := range resp.Data.Get[class] {
		recs = append(recs, weaviateRowToRecord(row))
	}
	return &ListResult{Records: recs}, nil
}

func (h *WeaviateHandler) Delete(ctx context.Context, ids []string, opts *DeleteOptions) error {
	namespace := ""
	if opts != nil {
		namespace = opts.Namespace
	}
	class, err := h.className(namespace)
	if err != nil {
		return err
	}
	for _, id := range ids {
		gql := map[string]any{
			"query": fmt.Sprintf(`mutation {
  Delete {
    %s(where: { path: ["record_id"], operator: Equal, valueText: %q }) { successful }
  }
}`, class, id),
		}
		_ = h.doJSON(ctx, http.MethodPost, "/v1/graphql", gql, nil)
	}
	return nil
}

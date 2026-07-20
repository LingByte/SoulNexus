package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LingByte/lingllm/embedder"
)

// ElasticsearchHandler implements KnowledgeHandler using Elasticsearch 8+ kNN search (HTTP API).
type ElasticsearchHandler struct {
	BaseURL    string
	APIKey     string
	IndexPrefix string
	HTTPClient *http.Client
	Embedder   embedder.Embedder
}

func (h *ElasticsearchHandler) Provider() string { return ProviderElasticsearch }

func (h *ElasticsearchHandler) indexName(namespace string) (string, error) {
	safe := sanitizeNamespace(namespace)
	if safe == "" {
		return "", ErrCollectionNotFound
	}
	prefix := strings.TrimSpace(h.IndexPrefix)
	if prefix == "" {
		prefix = "lingkb"
	}
	return prefix + "_" + safe, nil
}

func (h *ElasticsearchHandler) client() *http.Client {
	if h != nil && h.HTTPClient != nil {
		return h.HTTPClient
	}
	return http.DefaultClient
}

func (h *ElasticsearchHandler) baseURL() string {
	return strings.TrimRight(strings.TrimSpace(h.BaseURL), "/")
}

func (h *ElasticsearchHandler) doJSON(ctx context.Context, method, path string, body any, out any) (int, error) {
	if h == nil || strings.TrimSpace(h.BaseURL) == "" {
		return 0, ErrBaseURL
	}
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, h.baseURL()+path, reader)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(h.APIKey); key != "" {
		req.Header.Set("Authorization", "ApiKey "+key)
	}
	resp, err := h.client().Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("elasticsearch: status=%d body=%s", resp.StatusCode, string(raw))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

func (h *ElasticsearchHandler) ensureIndex(ctx context.Context, namespace string, dim int) (string, error) {
	index, err := h.indexName(namespace)
	if err != nil {
		return "", err
	}
	if dim <= 0 {
		return "", ErrInvalidVectorDimension
	}
	headReq, err := http.NewRequestWithContext(ctx, http.MethodHead, h.baseURL()+"/"+url.PathEscape(index), nil)
	if err == nil {
		if key := strings.TrimSpace(h.APIKey); key != "" {
			headReq.Header.Set("Authorization", "ApiKey "+key)
		}
		if resp, err := h.client().Do(headReq); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return index, nil
			}
		}
	}
	mapping := map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"id":            map[string]string{"type": "keyword"},
				"source":        map[string]string{"type": "keyword"},
				"title":         map[string]string{"type": "text"},
				"content":       map[string]string{"type": "text"},
				"tags":          map[string]string{"type": "keyword"},
				"metadata_json": map[string]any{"type": "object", "enabled": false},
				"created_at":    map[string]string{"type": "date"},
				"updated_at":    map[string]string{"type": "date"},
				"vector": map[string]any{
					"type":       "dense_vector",
					"dims":       dim,
					"index":      true,
					"similarity": "cosine",
				},
			},
		},
	}
	_, err = h.doJSON(ctx, http.MethodPut, "/"+url.PathEscape(index), mapping, nil)
	return index, err
}

func (h *ElasticsearchHandler) Ping(ctx context.Context) error {
	_, err := h.doJSON(ctx, http.MethodGet, "/", nil, nil)
	return err
}

func (h *ElasticsearchHandler) CreateNamespace(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrNamespaceNotFound
	}
	return nil
}

func (h *ElasticsearchHandler) DeleteNamespace(ctx context.Context, name string) error {
	index, err := h.indexName(name)
	if err != nil {
		return err
	}
	_, err = h.doJSON(ctx, http.MethodDelete, "/"+url.PathEscape(index), nil, nil)
	return err
}

func (h *ElasticsearchHandler) ListNamespaces(ctx context.Context) ([]string, error) {
	var resp struct {
		Indices map[string]json.RawMessage `json:""`
	}
	// cat indices is simpler
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.baseURL()+"/_cat/indices/"+url.PathEscape("lingkb_*")+"?format=json", nil)
	if err != nil {
		return nil, err
	}
	if key := strings.TrimSpace(h.APIKey); key != "" {
		req.Header.Set("Authorization", "ApiKey "+key)
	}
	r, err := h.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	raw, _ := io.ReadAll(r.Body)
	if r.StatusCode < 200 || r.StatusCode >= 300 {
		return nil, fmt.Errorf("elasticsearch: list indices status=%d", r.StatusCode)
	}
	var rows []struct {
		Index string `json:"index"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	prefix := strings.TrimSpace(h.IndexPrefix)
	if prefix == "" {
		prefix = "lingkb"
	}
	prefix = prefix + "_"
	var out []string
	for _, row := range rows {
		if strings.HasPrefix(row.Index, prefix) {
			out = append(out, strings.TrimPrefix(row.Index, prefix))
		}
	}
	_ = resp
	return out, nil
}

func (h *ElasticsearchHandler) Upsert(ctx context.Context, records []Record, opts *UpsertOptions) error {
	if len(records) == 0 {
		return nil
	}
	namespace := ""
	if opts != nil {
		namespace = opts.Namespace
	}
	recs := append([]Record(nil), records...)
	vectorDim, err := fillMissingVectors(ctx, h.Embedder, recs)
	if err != nil {
		return err
	}
	index, err := h.ensureIndex(ctx, namespace, vectorDim)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	var bulk bytes.Buffer
	for _, r := range recs {
		recordTimestamps(&r, now)
		meta := map[string]any{"index": map[string]string{"_index": index, "_id": r.ID}}
		b, _ := json.Marshal(meta)
		bulk.Write(b)
		bulk.WriteByte('\n')
		doc := map[string]any{
			"id":            r.ID,
			"source":        r.Source,
			"title":         r.Title,
			"content":       r.Content,
			"tags":          r.Tags,
			"metadata_json": r.Metadata,
			"vector":        r.Vector,
			"created_at":    r.CreatedAt,
			"updated_at":    r.UpdatedAt,
		}
		b, _ = json.Marshal(doc)
		bulk.Write(b)
		bulk.WriteByte('\n')
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL()+"/_bulk?refresh=wait_for", &bulk)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	if key := strings.TrimSpace(h.APIKey); key != "" {
		req.Header.Set("Authorization", "ApiKey "+key)
	}
	resp, err := h.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("elasticsearch bulk: status=%d body=%s", resp.StatusCode, string(raw))
	}
	var bulkResp struct {
		Errors bool `json:"errors"`
	}
	_ = json.Unmarshal(raw, &bulkResp)
	if bulkResp.Errors {
		return fmt.Errorf("elasticsearch bulk: partial failure: %s", string(raw))
	}
	return nil
}

func (h *ElasticsearchHandler) Query(ctx context.Context, text string, opts *QueryOptions) ([]QueryResult, error) {
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
	index, err := h.ensureIndex(ctx, namespace, len(vecs[0]))
	if err != nil {
		return nil, err
	}
	numCandidates := topK * 10
	if numCandidates < 100 {
		numCandidates = 100
	}
	body := map[string]any{
		"knn": map[string]any{
			"field":          "vector",
			"query_vector":   vecs[0],
			"k":              topK,
			"num_candidates": numCandidates,
		},
		"_source": []string{"id", "source", "title", "content", "tags", "metadata_json"},
	}
	var resp struct {
		Hits struct {
			Hits []struct {
				Score  float64                `json:"_score"`
				Source map[string]json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	_, err = h.doJSON(ctx, http.MethodPost, "/"+url.PathEscape(index)+"/_search", body, &resp)
	if err != nil {
		return nil, err
	}
	var results []QueryResult
	for _, hit := range resp.Hits.Hits {
		r := esHitToRecord(hit.Source)
		results = append(results, QueryResult{Record: r, Score: hit.Score})
	}
	return filterQueryResults(results, minScore), nil
}

func esHitToRecord(src map[string]json.RawMessage) Record {
	var r Record
	if v, ok := src["id"]; ok {
		_ = json.Unmarshal(v, &r.ID)
	}
	if v, ok := src["source"]; ok {
		_ = json.Unmarshal(v, &r.Source)
	}
	if v, ok := src["title"]; ok {
		_ = json.Unmarshal(v, &r.Title)
	}
	if v, ok := src["content"]; ok {
		_ = json.Unmarshal(v, &r.Content)
	}
	if v, ok := src["tags"]; ok {
		_ = json.Unmarshal(v, &r.Tags)
	}
	if v, ok := src["metadata_json"]; ok {
		var meta map[string]any
		_ = json.Unmarshal(v, &meta)
		r.Metadata = meta
	}
	return r
}

func (h *ElasticsearchHandler) Get(ctx context.Context, ids []string, opts *GetOptions) ([]Record, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	namespace := ""
	if opts != nil {
		namespace = opts.Namespace
	}
	index, err := h.indexName(namespace)
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, id := range ids {
		var resp struct {
			Source map[string]json.RawMessage `json:"_source"`
			Found  bool                       `json:"found"`
		}
		_, err := h.doJSON(ctx, http.MethodGet, "/"+url.PathEscape(index)+"/_doc/"+url.PathEscape(id), nil, &resp)
		if err != nil {
			continue
		}
		if resp.Found {
			out = append(out, esHitToRecord(resp.Source))
		}
	}
	return out, nil
}

func (h *ElasticsearchHandler) List(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	namespace := ""
	limit := 50
	if opts != nil {
		namespace = opts.Namespace
		if opts.Limit > 0 {
			limit = opts.Limit
		}
	}
	index, err := h.indexName(namespace)
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"size":    limit,
		"query":   map[string]any{"match_all": map[string]any{}},
		"_source": []string{"id", "source", "title", "content", "tags", "metadata_json"},
	}
	var resp struct {
		Hits struct {
			Hits []struct {
				Source map[string]json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	_, err = h.doJSON(ctx, http.MethodPost, "/"+url.PathEscape(index)+"/_search", body, &resp)
	if err != nil {
		return nil, err
	}
	recs := make([]Record, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		recs = append(recs, esHitToRecord(hit.Source))
	}
	return &ListResult{Records: recs}, nil
}

func (h *ElasticsearchHandler) Delete(ctx context.Context, ids []string, opts *DeleteOptions) error {
	if len(ids) == 0 {
		return nil
	}
	namespace := ""
	if opts != nil {
		namespace = opts.Namespace
	}
	index, err := h.indexName(namespace)
	if err != nil {
		return err
	}
	for _, id := range ids {
		_, _ = h.doJSON(ctx, http.MethodDelete, "/"+url.PathEscape(index)+"/_doc/"+url.PathEscape(id), nil, nil)
	}
	return nil
}

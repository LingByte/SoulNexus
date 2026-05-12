package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

type QdrantHandler struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	Embedder   Embedder
}

func (qh *QdrantHandler) Provider() string { return ProviderQdrant }

func normalizeVec64InPlace(v []float64) {
	if len(v) == 0 {
		return
	}
	var sum float64
	for _, x := range v {
		sum += x * x
	}
	if sum <= 0 {
		return
	}
	n := math.Sqrt(sum)
	if n == 0 || math.IsNaN(n) || math.IsInf(n, 0) {
		return
	}
	for i := range v {
		v[i] = v[i] / n
	}
}

func (qh *QdrantHandler) collectionNameFromOptions(namespace string) (string, error) {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return "", ErrCollectionNotFound
	}
	return ns, nil
}

func (qh *QdrantHandler) ensureCollection(ctx context.Context, collection string, vectorDim int) error {
	// If already exists, Qdrant returns 200.
	existsURL := qh.baseURL(collection) + "/collections/" + url.PathEscape(collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, existsURL, nil)
	if err != nil {
		return err
	}
	if strings.TrimSpace(qh.APIKey) != "" {
		req.Header.Set("api-key", strings.TrimSpace(qh.APIKey))
	}
	resp, err := qh.HTTPClient.Do(req)
	if err == nil && resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
	}

	if vectorDim <= 0 {
		return ErrInvalidVectorDimension
	}

	// Create a collection with a single vector config.
	createURL := qh.baseURL(collection) + "/collections/" + url.PathEscape(collection)
	payload := map[string]any{
		"vectors": map[string]any{
			"size":     vectorDim,
			"distance": "Cosine",
		},
	}
	return qh.doJSON(ctx, http.MethodPut, createURL, payload, nil)
}

func (qh *QdrantHandler) baseURL(_ string) string {
	return strings.TrimRight(strings.TrimSpace(qh.BaseURL), "/")
}

func (qh *QdrantHandler) doJSON(ctx context.Context, method, fullURL string, reqBody any, out any) error {
	var body io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(qh.APIKey) != "" {
		req.Header.Set("api-key", strings.TrimSpace(qh.APIKey))
	}

	resp, err := qh.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if len(respBytes) == 0 {
			return fmt.Errorf("qdrant_http_%s: status=%d", method, resp.StatusCode)
		}
		return fmt.Errorf("qdrant_http_%s: status=%d, body=%s", method, resp.StatusCode, truncateForErr(respBytes))
	}
	if out == nil {
		return nil
	}
	if len(respBytes) == 0 {
		return nil
	}
	return json.Unmarshal(respBytes, out)
}

func truncateForErr(b []byte) string {
	const n = 2048
	s := strings.TrimSpace(string(b))
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func qdrantPointIDFromString(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Qdrant REST expects point id as either an unsigned integer or a UUID string.
	if u, err := strconv.ParseUint(s, 10, 64); err == nil {
		return u
	}
	return s
}

func (qh *QdrantHandler) pointsUpsertPath(collection string) string {
	return "/collections/" + url.PathEscape(collection) + "/points"
}

func (qh *QdrantHandler) Upsert(ctx context.Context, records []Record, opts *UpsertOptions) error {
	if qh == nil {
		return ErrHandlerNotFound
	}
	if strings.TrimSpace(qh.BaseURL) == "" {
		return ErrBaseURL
	}
	if len(records) == 0 {
		return nil
	}

	var namespace string
	if opts != nil {
		namespace = opts.Namespace
	}
	collection, err := qh.collectionNameFromOptions(namespace)
	if err != nil {
		return err
	}

	// Determine vector dimension from first available vector, otherwise from embedding.
	vectorDim := 0
	for i := range records {
		if len(records[i].Vector) > 0 {
			vectorDim = len(records[i].Vector)
			break
		}
	}
	if vectorDim <= 0 {
		if qh.Embedder == nil {
			return ErrEmbedderNotFound
		}
		if strings.TrimSpace(records[0].Content) == "" {
			return ErrEmptyQuery
		}
		vecs, err := qh.Embedder.Embed(ctx, []string{records[0].Content})
		if err != nil {
			return err
		}
		if len(vecs) == 0 || len(vecs[0]) == 0 {
			return ErrInvalidVectorDimension
		}
		vectorDim = len(vecs[0])
		tmp := make([]float32, vectorDim)
		for j := range tmp {
			tmp[j] = float32(vecs[0][j])
		}
		records[0].Vector = tmp
	}

	if err := qh.ensureCollection(ctx, collection, vectorDim); err != nil {
		return err
	}

	batchSize := 64
	if opts != nil && opts.BatchSize > 0 {
		batchSize = opts.BatchSize
	}
	if batchSize < 1 {
		batchSize = 64
	}

	// Qdrant upsert is idempotent; Overwrite currently does not implement conditional "insert-only" semantics.
	for start := 0; start < len(records); start += batchSize {
		end := start + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[start:end]

		// Fill missing vectors via embedder.
		var needIdx []int
		var inputs []string
		for i := range batch {
			if len(batch[i].Vector) > 0 {
				if len(batch[i].Vector) != vectorDim {
					return ErrInvalidVectorDimension
				}
				continue
			}
			needIdx = append(needIdx, i)
			inputs = append(inputs, batch[i].Content)
		}
		if len(needIdx) > 0 {
			if qh.Embedder == nil {
				return ErrEmbedderNotFound
			}
			vecs, err := qh.Embedder.Embed(ctx, inputs)
			if err != nil {
				return err
			}
			if len(vecs) != len(needIdx) {
				return fmt.Errorf("embedder_vector_mismatch: want=%d got=%d", len(needIdx), len(vecs))
			}
			for k := range needIdx {
				if len(vecs[k]) != vectorDim {
					return ErrInvalidVectorDimension
				}
				tmp := make([]float32, vectorDim)
				for j := range tmp {
					tmp[j] = float32(vecs[k][j])
				}
				batch[needIdx[k]].Vector = tmp
			}
		}

		points := make([]map[string]any, 0, len(batch))
		for i := range batch {
			r := batch[i]
			payload := map[string]any{
				"source":     r.Source,
				"title":      r.Title,
				"content":    r.Content,
				"tags":       r.Tags,
				"metadata":   r.Metadata,
				"created_at": r.CreatedAt,
				"updated_at": r.UpdatedAt,
			}
			points = append(points, map[string]any{
				"id":      qdrantPointIDFromString(r.ID),
				"vector":  r.Vector,
				"payload": payload,
			})
		}

		wait := true
		reqURL := qh.baseURL(collection) + qh.pointsUpsertPath(collection) + "?wait=" + strconv.FormatBool(wait)
		reqBody := map[string]any{"points": points}
		if err := qh.doJSON(ctx, http.MethodPut, reqURL, reqBody, nil); err != nil {
			return err
		}
	}
	return nil
}

func (qh *QdrantHandler) toFloat32s(v []float64) ([]float32, error) {
	if len(v) == 0 {
		return nil, ErrInvalidVectorDimension
	}
	out := make([]float32, len(v))
	for i := range out {
		out[i] = float32(v[i])
	}
	return out, nil
}

func buildQdrantFilter(filters []Filter) map[string]any {
	if len(filters) == 0 {
		return nil
	}
	must := make([]any, 0, len(filters))
	for _, f := range filters {
		key := strings.TrimSpace(f.Field)
		if key == "" {
			continue
		}
		switch f.Operator {
		case FilterOpEqual:
			if len(f.Value) == 0 {
				continue
			}
			must = append(must, map[string]any{"key": key, "match": map[string]any{"value": f.Value[0]}})
		case FilterOpNotEqual:
			if len(f.Value) == 0 {
				continue
			}
			must = append(must, map[string]any{"key": key, "match": map[string]any{"value": f.Value[0]}}) // Qdrant doesn't have direct $ne in the simple form.
		case FilterOpIn:
			must = append(must, map[string]any{"key": key, "match": map[string]any{"any": f.Value}})
		case FilterOpNotIn:
			// Not implemented precisely; fall back to equality semantics at caller.
			if len(f.Value) == 0 {
				continue
			}
			must = append(must, map[string]any{"key": key, "match": map[string]any{"value": f.Value[0]}})
		case FilterOpGt, FilterOpGte, FilterOpLt, FilterOpLte:
			if len(f.Value) == 0 {
				continue
			}
			r := map[string]any{}
			switch f.Operator {
			case FilterOpGt:
				r["gt"] = f.Value[0]
			case FilterOpGte:
				r["gte"] = f.Value[0]
			case FilterOpLt:
				r["lt"] = f.Value[0]
			case FilterOpLte:
				r["lte"] = f.Value[0]
			}
			must = append(must, map[string]any{"key": key, "range": r})
		case FilterOpContainsAll:
			must = append(must, map[string]any{"key": key, "match": map[string]any{"all": f.Value}})
		case FilterOpContainsAny:
			must = append(must, map[string]any{"key": key, "match": map[string]any{"any": f.Value}})
		default:
		}
	}
	if len(must) == 0 {
		return nil
	}
	return map[string]any{"must": must}
}

func payloadToRecord(id any, payload map[string]any) Record {
	idStr := fmt.Sprintf("%v", id)
	var r Record
	r.ID = idStr
	if payload == nil {
		return r
	}
	if v, ok := payload["source"].(string); ok {
		r.Source = v
	}
	if v, ok := payload["title"].(string); ok {
		r.Title = v
	}
	if v, ok := payload["content"].(string); ok {
		r.Content = v
	}
	if v, ok := payload["metadata"].(map[string]any); ok {
		r.Metadata = v
	}
	if v, ok := payload["tags"].([]any); ok {
		var tags []string
		for _, tv := range v {
			if ts, ok := tv.(string); ok {
				tags = append(tags, ts)
			}
		}
		r.Tags = tags
	} else if v, ok := payload["tags"].([]string); ok {
		r.Tags = v
	}
	// timestamps are optional; best-effort parse.
	if s, ok := payload["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			r.CreatedAt = t
		}
	}
	if s, ok := payload["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			r.UpdatedAt = t
		}
	}
	return r
}

func (qh *QdrantHandler) Query(ctx context.Context, text string, opts *QueryOptions) ([]QueryResult, error) {
	if qh == nil {
		return nil, ErrHandlerNotFound
	}
	if strings.TrimSpace(qh.BaseURL) == "" {
		return nil, ErrBaseURL
	}
	if strings.TrimSpace(text) == "" {
		return nil, ErrEmptyQuery
	}

	namespace := ""
	topK := 10
	minScore := 0.0
	var filters []Filter
	if opts != nil {
		namespace = opts.Namespace
		if opts.TopK > 0 {
			topK = opts.TopK
		}
		minScore = opts.MinScore
		filters = opts.Filters
	}
	collection, err := qh.collectionNameFromOptions(namespace)
	if err != nil {
		return nil, err
	}
	if qh.Embedder == nil {
		return nil, ErrEmbedderNotFound
	}
	vecs, err := qh.Embedder.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, ErrInvalidVectorDimension
	}
	normalizeVec64InPlace(vecs[0])
	qvec, err := qh.toFloat32s(vecs[0])
	if err != nil {
		return nil, err
	}

	filter := buildQdrantFilter(filters)
	reqURL := qh.baseURL(collection) + "/collections/" + url.PathEscape(collection) + "/points/search"
	reqBody := map[string]any{
		"vector":          qvec,
		"limit":           topK,
		"score_threshold": minScore,
		"with_payload":    true,
	}
	if filter != nil {
		reqBody["filter"] = filter
	}

	var resp struct {
		Result []struct {
			ID      any            `json:"id"`
			Payload map[string]any `json:"payload"`
			Score   float64        `json:"score"`
		} `json:"result"`
	}
	if err := qh.doJSON(ctx, http.MethodPost, reqURL, reqBody, &resp); err != nil {
		return nil, err
	}

	out := make([]QueryResult, 0, len(resp.Result))
	for _, it := range resp.Result {
		out = append(out, QueryResult{
			Record: payloadToRecord(it.ID, it.Payload),
			Score:  it.Score,
		})
	}
	return out, nil
}

func (qh *QdrantHandler) Get(ctx context.Context, ids []string, opts *GetOptions) ([]Record, error) {
	if qh == nil {
		return nil, ErrHandlerNotFound
	}
	if strings.TrimSpace(qh.BaseURL) == "" {
		return nil, ErrBaseURL
	}
	if len(ids) == 0 {
		return nil, ErrRecordNotFound
	}
	namespace := ""
	if opts != nil {
		namespace = opts.Namespace
	}
	collection, err := qh.collectionNameFromOptions(namespace)
	if err != nil {
		return nil, err
	}

	reqURL := qh.baseURL(collection) + "/collections/" + url.PathEscape(collection) + "/points/lookup"
	lookupIDs := make([]any, 0, len(ids))
	for _, id := range ids {
		lookupIDs = append(lookupIDs, qdrantPointIDFromString(id))
	}
	reqBody := map[string]any{
		"ids":          lookupIDs,
		"with_payload": true,
		"with_vector":  false,
	}
	var resp struct {
		Result []struct {
			ID      any            `json:"id"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}
	if err := qh.doJSON(ctx, http.MethodPost, reqURL, reqBody, &resp); err != nil {
		return nil, err
	}

	if len(resp.Result) == 0 {
		return nil, ErrRecordNotFound
	}

	out := make([]Record, 0, len(resp.Result))
	for _, it := range resp.Result {
		out = append(out, payloadToRecord(it.ID, it.Payload))
	}
	return out, nil
}

func (qh *QdrantHandler) List(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	if qh == nil {
		return nil, ErrHandlerNotFound
	}
	if strings.TrimSpace(qh.BaseURL) == "" {
		return nil, ErrBaseURL
	}
	if opts == nil {
		return &ListResult{Records: []Record{}}, nil
	}
	collection, err := qh.collectionNameFromOptions(opts.Namespace)
	if err != nil {
		return nil, err
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}

	reqURL := qh.baseURL(collection) + "/collections/" + url.PathEscape(collection) + "/points/scroll"
	reqBody := map[string]any{
		"limit":        limit,
		"with_payload": true,
		"with_vector":  false,
	}
	if opts.Offset != "" {
		reqBody["offset"] = map[string]any{"id": qdrantPointIDFromString(opts.Offset)}
	}
	if len(opts.Filters) > 0 {
		if f := buildQdrantFilter(opts.Filters); f != nil {
			reqBody["filter"] = f
		}
	}
	if strings.TrimSpace(opts.OrderBy) != "" {
		dir := strings.ToLower(strings.TrimSpace(opts.OrderDir))
		if dir != "asc" && dir != "desc" {
			dir = "asc"
		}
		reqBody["order_by"] = map[string]any{"key": opts.OrderBy, "direction": dir}
	}

	// Qdrant: POST /points/scroll => {"result":{"points":[...],"next_page_offset":...}, "status":"ok", ...}
	var resp struct {
		Result struct {
			Points []struct {
				ID      any            `json:"id"`
				Payload map[string]any `json:"payload"`
			} `json:"points"`
			NextPageOffset any `json:"next_page_offset"`
		} `json:"result"`
	}
	if err := qh.doJSON(ctx, http.MethodPost, reqURL, reqBody, &resp); err != nil {
		return nil, err
	}

	out := &ListResult{Records: make([]Record, 0, len(resp.Result.Points))}
	for _, it := range resp.Result.Points {
		out.Records = append(out.Records, payloadToRecord(it.ID, it.Payload))
	}
	if resp.Result.NextPageOffset != nil {
		out.NextOffset = fmt.Sprintf("%v", resp.Result.NextPageOffset)
	}
	return out, nil
}

func (qh *QdrantHandler) Delete(ctx context.Context, ids []string, opts *DeleteOptions) error {
	if qh == nil {
		return ErrHandlerNotFound
	}
	if strings.TrimSpace(qh.BaseURL) == "" {
		return ErrBaseURL
	}
	if len(ids) == 0 {
		return nil
	}
	namespace := ""
	if opts != nil {
		namespace = opts.Namespace
	}
	collection, err := qh.collectionNameFromOptions(namespace)
	if err != nil {
		return err
	}

	reqURL := qh.baseURL(collection) + "/collections/" + url.PathEscape(collection) + "/points/delete"
	pts := make([]any, 0, len(ids))
	for _, id := range ids {
		pts = append(pts, qdrantPointIDFromString(id))
	}
	reqBody := map[string]any{"points": pts}
	// Some Qdrant versions accept `{"points":[id1,id2]}`. Our tests focus on compile/runtime shape, not Qdrant strictness.
	if err := qh.doJSON(ctx, http.MethodPost, reqURL, reqBody, nil); err != nil {
		return err
	}
	return nil
}

func (qh *QdrantHandler) Ping(ctx context.Context) error {
	if qh == nil {
		return ErrHandlerNotFound
	}
	if strings.TrimSpace(qh.BaseURL) == "" {
		return ErrBaseURL
	}
	// Health check: list collections (works across Qdrant versions).
	reqURL := qh.baseURL("") + "/collections"
	return qh.doJSON(ctx, http.MethodGet, reqURL, nil, &struct{}{})
}

func (qh *QdrantHandler) CreateNamespace(ctx context.Context, name string) error {
	if qh == nil {
		return ErrHandlerNotFound
	}
	if strings.TrimSpace(name) == "" {
		return ErrNamespaceNotFound
	}
	if qh.Embedder == nil {
		return ErrEmbedderNotFound
	}

	// Best-effort dimension inference from embedder.
	vecs, err := qh.Embedder.Embed(ctx, []string{"dimension_probe"})
	if err != nil {
		return err
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return ErrInvalidVectorDimension
	}
	return qh.ensureCollection(ctx, name, len(vecs[0]))
}

func (qh *QdrantHandler) DeleteNamespace(ctx context.Context, name string) error {
	if qh == nil {
		return ErrHandlerNotFound
	}
	if strings.TrimSpace(name) == "" {
		return ErrNamespaceNotFound
	}
	reqURL := qh.baseURL("") + "/collections/" + url.PathEscape(strings.TrimSpace(name))
	return qh.doJSON(ctx, http.MethodDelete, reqURL, nil, nil)
}

func (qh *QdrantHandler) ListNamespaces(ctx context.Context) ([]string, error) {
	if qh == nil {
		return nil, ErrHandlerNotFound
	}
	if strings.TrimSpace(qh.BaseURL) == "" {
		return nil, ErrBaseURL
	}
	reqURL := qh.baseURL("") + "/collections"
	// Qdrant: GET /collections => {"result":{"collections":[{"name":"..."}]}, "status":"ok", ...}
	var resp struct {
		Result struct {
			Collections []struct {
				Name string `json:"name"`
			} `json:"collections"`
		} `json:"result"`
	}
	if err := qh.doJSON(ctx, http.MethodGet, reqURL, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(resp.Result.Collections))
	for i := range resp.Result.Collections {
		out = append(out, resp.Result.Collections[i].Name)
	}
	return out, nil
}

package knowledge

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Embedder interface {
	Embed(ctx context.Context, inputs []string) ([][]float64, error)
}

func recordFromPayload(p map[string]any) Record {
	rec := Record{}
	if p == nil {
		return rec
	}
	if v, ok := p["record_id"].(string); ok {
		rec.ID = v
	}
	if v, ok := p["source"].(string); ok {
		rec.Source = v
	}
	if v, ok := p["title"].(string); ok {
		rec.Title = v
	}
	if v, ok := p["content"].(string); ok {
		rec.Content = v
	}
	if v, ok := p["metadata"].(map[string]any); ok {
		rec.Metadata = v
	}
	if v, ok := p["tags"].([]any); ok {
		tags := make([]string, 0, len(v))
		for _, t := range v {
			if s, ok := t.(string); ok {
				tags = append(tags, s)
			}
		}
		rec.Tags = tags
	}
	return rec
}

type QdrantHandler struct {
	BaseURL    string
	APIKey     string
	Collection string
	HTTPClient *http.Client
	Embedder   Embedder
}

func (qh *QdrantHandler) Provider() string {
	return KnowledgeQdrant
}

func (qh *QdrantHandler) Upsert(ctx context.Context, records []Record, options *UpsertOptions) error {
	if qh == nil {
		return errors.New("handler is nil")
	}
	if qh.BaseURL == "" {
		return errors.New("BaseURL is required")
	}
	if qh.Collection == "" {
		return errors.New("Collection is required")
	}
	if qh.Embedder == nil {
		return errors.New("Embedder is required")
	}
	if len(records) == 0 {
		return nil
	}

	texts := make([]string, 0, len(records))
	for _, r := range records {
		content := strings.TrimSpace(r.Title + "\n" + r.Content)
		if content == "" {
			content = strings.TrimSpace(r.Content)
		}
		texts = append(texts, content)
	}
	vectors, err := qh.Embedder.Embed(ctx, texts)
	if err != nil {
		return err
	}
	if len(vectors) != len(records) {
		return fmt.Errorf("embedding count mismatch: got=%d want=%d", len(vectors), len(records))
	}

	// Ensure collection exists.
	if err := qh.ensureCollection(ctx, len(vectors[0])); err != nil {
		return err
	}

	type point struct {
		ID      any            `json:"id"`
		Vector  []float64      `json:"vector"`
		Payload map[string]any `json:"payload"`
	}
	points := make([]point, 0, len(records))
	for i, r := range records {
		originalID := strings.TrimSpace(r.ID)
		pointID := qdrantPointIDFromString(originalID)
		payload := map[string]any{
			"record_id":  originalID,
			"source":     r.Source,
			"title":      r.Title,
			"content":    r.Content,
			"tags":       r.Tags,
			"metadata":   r.Metadata,
			"created_at": r.CreatedAt.Unix(),
			"updated_at": r.UpdatedAt.Unix(),
		}
		points = append(points, point{ID: pointID, Vector: vectors[i], Payload: payload})
	}

	body := map[string]any{"points": points}
	_, err = qh.doJSON(ctx, http.MethodPut, qh.pointsUpsertPath()+"?wait=true", body, nil)
	return err
}

func (qh *QdrantHandler) Query(ctx context.Context, text string, options *QueryOptions) ([]QueryResult, error) {
	if qh == nil {
		return nil, errors.New("handler is nil")
	}
	if qh.BaseURL == "" {
		return nil, errors.New("BaseURL is required")
	}
	if qh.Collection == "" {
		return nil, errors.New("Collection is required")
	}
	if qh.Embedder == nil {
		return nil, errors.New("Embedder is required")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New("text is empty")
	}

	topK := 5
	if options != nil && options.TopK > 0 {
		topK = options.TopK
	}

	vecs, err := qh.Embedder.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, errors.New("no embedding returned")
	}

	searchBody := map[string]any{
		"vector":          vecs[0],
		"limit":           topK,
		"with_payload":    true,
		"with_vectors":    false,
		"score_threshold": nil,
	}
	// Basic filter passthrough (exact shape depends on qdrant); allow caller to provide already-formed filter.
	if options != nil && options.Filters != nil {
		searchBody["filter"] = options.Filters
	}

	var resp struct {
		Result []struct {
			ID      any            `json:"id"`
			Score   float64        `json:"score"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}
	_, err = qh.doJSON(ctx, http.MethodPost, qh.pointsSearchPath(), searchBody, &resp)
	if err != nil {
		return nil, err
	}

	out := make([]QueryResult, 0, len(resp.Result))
	for _, r := range resp.Result {
		rec := recordFromPayload(r.Payload)
		out = append(out, QueryResult{Record: rec, Score: r.Score})
	}
	return out, nil
}

func (qh *QdrantHandler) Get(ctx context.Context, ids []string, options *GetOptions) ([]Record, error) {
	if qh == nil {
		return nil, errors.New("handler is nil")
	}
	if qh.BaseURL == "" {
		return nil, errors.New("BaseURL is required")
	}
	if qh.Collection == "" {
		return nil, errors.New("Collection is required")
	}
	if len(ids) == 0 {
		return []Record{}, nil
	}
	_ = options

	pointIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		pointIDs = append(pointIDs, qdrantPointIDFromString(id))
	}

	body := map[string]any{
		"ids":          pointIDs,
		"with_payload": true,
		"with_vectors": false,
	}
	var resp struct {
		Result []struct {
			ID      any            `json:"id"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}
	_, err := qh.doJSON(ctx, http.MethodPost, qh.pointsRetrievePath(), body, &resp)
	if err != nil {
		return nil, err
	}

	out := make([]Record, 0, len(resp.Result))
	for _, r := range resp.Result {
		out = append(out, recordFromPayload(r.Payload))
	}
	return out, nil
}

func (qh *QdrantHandler) List(ctx context.Context, options *ListOptions) (*ListResult, error) {
	if qh == nil {
		return nil, errors.New("handler is nil")
	}
	if qh.BaseURL == "" {
		return nil, errors.New("BaseURL is required")
	}
	if qh.Collection == "" {
		return nil, errors.New("Collection is required")
	}

	limit := 50
	if options != nil && options.Limit > 0 {
		limit = options.Limit
	}

	body := map[string]any{
		"limit":        limit,
		"with_payload": true,
		"with_vectors": false,
	}
	if options != nil {
		if strings.TrimSpace(options.Offset) != "" {
			body["offset"] = strings.TrimSpace(options.Offset)
		}
		if options.Filters != nil {
			body["filter"] = options.Filters
		}
	}

	// Qdrant scroll: result is { "points": [...], "next_page_offset": ... }, not a bare array.
	var resp struct {
		Result struct {
			Points []struct {
				ID      any            `json:"id"`
				Payload map[string]any `json:"payload"`
			} `json:"points"`
			NextPageOffset any `json:"next_page_offset"`
		} `json:"result"`
	}
	_, err := qh.doJSON(ctx, http.MethodPost, qh.pointsScrollPath(), body, &resp)
	if err != nil {
		return nil, err
	}

	points := resp.Result.Points
	recs := make([]Record, 0, len(points))
	for _, r := range points {
		recs = append(recs, recordFromPayload(r.Payload))
	}

	next := ""
	if resp.Result.NextPageOffset != nil {
		next = strings.TrimSpace(fmt.Sprint(resp.Result.NextPageOffset))
	}
	return &ListResult{Records: recs, NextOffset: next}, nil
}

func (qh *QdrantHandler) Delete(ctx context.Context, ids []string, options *DeleteOptions) error {
	if qh == nil {
		return errors.New("handler is nil")
	}
	if qh.BaseURL == "" {
		return errors.New("BaseURL is required")
	}
	if qh.Collection == "" {
		return errors.New("Collection is required")
	}
	if len(ids) == 0 {
		return nil
	}

	pointIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		pointIDs = append(pointIDs, qdrantPointIDFromString(id))
	}
	body := map[string]any{
		"points": pointIDs,
	}
	_, err := qh.doJSON(ctx, http.MethodPost, qh.pointsDeletePath()+"?wait=true", body, nil)
	return err
}

var qdrantUUIDRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// qdrantPointIDFromString maps arbitrary IDs into a Qdrant-valid point ID.
// Qdrant accepts either uint or UUID; we use UUID for broad compatibility.
//
// - If s is already a UUID, keep it.
// - Otherwise, derive a stable UUID from SHA1(s).
// - Empty input generates a random-ish UUID from timestamp entropy.
func qdrantPointIDFromString(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return sha1ToUUIDString(sha1.Sum([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))))
	}
	if qdrantUUIDRe.MatchString(s) {
		return strings.ToLower(s)
	}
	sum := sha1.Sum([]byte(s))
	return sha1ToUUIDString(sum)
}

func sha1ToUUIDString(sum [sha1.Size]byte) string {
	// UUIDv5-like formatting with RFC4122 variant.
	b := make([]byte, 16)
	copy(b, sum[:16])
	// set version to 5
	b[6] = (b[6] & 0x0f) | 0x50
	// set variant to RFC4122
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
		uint16(b[4])<<8|uint16(b[5]),
		uint16(b[6])<<8|uint16(b[7]),
		uint16(b[8])<<8|uint16(b[9]),
		b[10:16],
	)
}

func (qh *QdrantHandler) ensureCollection(ctx context.Context, vectorSize int) error {
	// GET collection
	_, err := qh.doJSON(ctx, http.MethodGet, qh.collectionPath(), nil, nil)
	if err == nil {
		return nil
	}
	// Try to create (PUT) with cosine distance.
	createBody := map[string]any{
		"vectors": map[string]any{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}
	_, err2 := qh.doJSON(ctx, http.MethodPut, qh.collectionPath(), createBody, nil)
	return err2
}

func (qh *QdrantHandler) doJSON(ctx context.Context, method string, path string, reqBody any, out any) ([]byte, error) {
	cl := qh.HTTPClient
	if cl == nil {
		cl = &http.Client{Timeout: 30 * time.Second}
	}
	url := strings.TrimRight(qh.BaseURL, "/") + path

	var bodyReader io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if qh.APIKey != "" {
		req.Header.Set("api-key", qh.APIKey)
	}

	resp, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return respBytes, fmt.Errorf("qdrant request failed: method=%s path=%s status=%d body=%s", method, path, resp.StatusCode, string(respBytes))
	}
	if out != nil {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return respBytes, err
		}
	}
	return respBytes, nil
}

func (qh *QdrantHandler) collectionPath() string {
	return "/collections/" + qh.Collection
}

func (qh *QdrantHandler) pointsUpsertPath() string {
	return "/collections/" + qh.Collection + "/points"
}

func (qh *QdrantHandler) pointsSearchPath() string {
	return "/collections/" + qh.Collection + "/points/search"
}

func (qh *QdrantHandler) pointsDeletePath() string {
	return "/collections/" + qh.Collection + "/points/delete"
}

func (qh *QdrantHandler) pointsRetrievePath() string {
	return "/collections/" + qh.Collection + "/points/retrieve"
}

func (qh *QdrantHandler) pointsScrollPath() string {
	return "/collections/" + qh.Collection + "/points/scroll"
}

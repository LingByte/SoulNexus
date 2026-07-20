package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/lingllm/embedder"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

const pgTablePrefix = "lingkb_"

// PostgresHandler implements KnowledgeHandler using PostgreSQL + pgvector.
type PostgresHandler struct {
	DSN      string
	Embedder embedder.Embedder
	pool     *pgxpool.Pool
}

func (h *PostgresHandler) Provider() string { return ProviderPostgres }

func (h *PostgresHandler) tableName(namespace string) (string, error) {
	safe := sanitizeNamespace(namespace)
	if safe == "" {
		return "", ErrCollectionNotFound
	}
	return pgTablePrefix + safe, nil
}

func (h *PostgresHandler) ensurePool(ctx context.Context) (*pgxpool.Pool, error) {
	if h == nil {
		return nil, ErrHandlerNotFound
	}
	if h.pool != nil {
		return h.pool, nil
	}
	dsn := strings.TrimSpace(h.DSN)
	if dsn == "" {
		return nil, errors.New("postgres: DSN is required")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse dsn: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}
	h.pool = pool
	return h.pool, nil
}

func (h *PostgresHandler) ensureTable(ctx context.Context, namespace string, dim int) (string, error) {
	table, err := h.tableName(namespace)
	if err != nil {
		return "", err
	}
	if dim <= 0 {
		return "", ErrInvalidVectorDimension
	}
	pool, err := h.ensurePool(ctx)
	if err != nil {
		return "", err
	}
	if _, err := pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS vector`); err != nil {
		return "", fmt.Errorf("postgres: create extension vector: %w", err)
	}
	ddl := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  id TEXT PRIMARY KEY,
  source TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  content TEXT NOT NULL,
  tags JSONB NOT NULL DEFAULT '[]'::jsonb,
  metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  embedding vector(%d) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`, table, dim)
	if _, err := pool.Exec(ctx, ddl); err != nil {
		return "", fmt.Errorf("postgres: create table: %w", err)
	}
	idxName := table + "_embedding_hnsw"
	idxDDL := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON %s USING hnsw (embedding vector_cosine_ops)`,
		idxName, table,
	)
	_, _ = pool.Exec(ctx, idxDDL)
	return table, nil
}

func (h *PostgresHandler) Ping(ctx context.Context) error {
	pool, err := h.ensurePool(ctx)
	if err != nil {
		return err
	}
	return pool.Ping(ctx)
}

func (h *PostgresHandler) CreateNamespace(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNamespaceNotFound
	}
	// Dimension is unknown until first upsert; table is created on upsert.
	return nil
}

func (h *PostgresHandler) DeleteNamespace(ctx context.Context, name string) error {
	table, err := h.tableName(name)
	if err != nil {
		return err
	}
	pool, err := h.ensurePool(ctx)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %s`, table))
	return err
}

func (h *PostgresHandler) ListNamespaces(ctx context.Context) ([]string, error) {
	pool, err := h.ensurePool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
SELECT tablename FROM pg_tables
WHERE schemaname = 'public' AND tablename LIKE $1`, pgTablePrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		out = append(out, strings.TrimPrefix(table, pgTablePrefix))
	}
	return out, rows.Err()
}

func (h *PostgresHandler) Upsert(ctx context.Context, records []Record, opts *UpsertOptions) error {
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
	table, err := h.ensureTable(ctx, namespace, vectorDim)
	if err != nil {
		return err
	}
	pool, err := h.ensurePool(ctx)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, r := range recs {
		recordTimestamps(&r, now)
		vec := pgvector.NewVector(r.Vector)
		tagsRaw, _ := json.Marshal(r.Tags)
		metaRaw := metadataJSON(r.Metadata)
		_, err := pool.Exec(ctx, fmt.Sprintf(`
INSERT INTO %s (id, source, title, content, tags, metadata_json, embedding, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5::jsonb,$6::jsonb,$7,$8,$9)
ON CONFLICT (id) DO UPDATE SET
  source = EXCLUDED.source,
  title = EXCLUDED.title,
  content = EXCLUDED.content,
  tags = EXCLUDED.tags,
  metadata_json = EXCLUDED.metadata_json,
  embedding = EXCLUDED.embedding,
  updated_at = EXCLUDED.updated_at`, table),
			r.ID, r.Source, r.Title, r.Content, string(tagsRaw), metaRaw, vec, r.CreatedAt, r.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("postgres: upsert %s: %w", r.ID, err)
		}
	}
	return nil
}

func (h *PostgresHandler) Query(ctx context.Context, text string, opts *QueryOptions) ([]QueryResult, error) {
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
	table, err := h.ensureTable(ctx, namespace, len(vecs[0]))
	if err != nil {
		return nil, err
	}
	pool, err := h.ensurePool(ctx)
	if err != nil {
		return nil, err
	}

	qvec := pgvector.NewVector(vecs[0])
	rows, err := pool.Query(ctx, fmt.Sprintf(`
SELECT id, source, title, content, tags, metadata_json,
       1 - (embedding <=> $1) AS score
FROM %s
ORDER BY embedding <=> $1
LIMIT $2`, table), qvec, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []QueryResult
	for rows.Next() {
		var r Record
		var tagsRaw []byte
		var metaRaw []byte
		var score float64
		if err := rows.Scan(&r.ID, &r.Source, &r.Title, &r.Content, &tagsRaw, &metaRaw, &score); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(tagsRaw, &r.Tags)
		r.Metadata = parseMetadataJSON(string(metaRaw))
		results = append(results, QueryResult{Record: r, Score: score})
	}
	return filterQueryResults(results, minScore), rows.Err()
}

func (h *PostgresHandler) Get(ctx context.Context, ids []string, opts *GetOptions) ([]Record, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	namespace := ""
	if opts != nil {
		namespace = opts.Namespace
	}
	table, err := h.tableName(namespace)
	if err != nil {
		return nil, err
	}
	pool, err := h.ensurePool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, fmt.Sprintf(`
SELECT id, source, title, content, tags, metadata_json, created_at, updated_at
FROM %s WHERE id = ANY($1)`, table), ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPostgresRecords(rows)
}

func (h *PostgresHandler) List(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	namespace := ""
	limit := 50
	if opts != nil {
		namespace = opts.Namespace
		if opts.Limit > 0 {
			limit = opts.Limit
		}
	}
	table, err := h.tableName(namespace)
	if err != nil {
		return nil, err
	}
	pool, err := h.ensurePool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, fmt.Sprintf(`
SELECT id, source, title, content, tags, metadata_json, created_at, updated_at
FROM %s ORDER BY created_at DESC LIMIT $1`, table), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	recs, err := scanPostgresRecords(rows)
	if err != nil {
		return nil, err
	}
	return &ListResult{Records: recs}, nil
}

func (h *PostgresHandler) Delete(ctx context.Context, ids []string, opts *DeleteOptions) error {
	if len(ids) == 0 {
		return nil
	}
	namespace := ""
	if opts != nil {
		namespace = opts.Namespace
	}
	table, err := h.tableName(namespace)
	if err != nil {
		return err
	}
	pool, err := h.ensurePool(ctx)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = ANY($1)`, table), ids)
	return err
}

func scanPostgresRecords(rows pgx.Rows) ([]Record, error) {
	var out []Record
	for rows.Next() {
		var r Record
		var tagsRaw []byte
		var metaRaw []byte
		if err := rows.Scan(&r.ID, &r.Source, &r.Title, &r.Content, &tagsRaw, &metaRaw, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(tagsRaw, &r.Tags)
		r.Metadata = parseMetadataJSON(string(metaRaw))
		out = append(out, r)
	}
	return out, rows.Err()
}

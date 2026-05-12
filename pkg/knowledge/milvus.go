package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// milvusDB is the subset of the Milvus SDK client used by MilvusHandler (implemented by client.Client; mockable in tests).
type milvusDB interface {
	HasCollection(ctx context.Context, collName string) (bool, error)
	LoadCollection(ctx context.Context, collName string, async bool, opts ...client.LoadCollectionOption) error
	CreateCollection(ctx context.Context, schema *entity.Schema, shardsNum int32, opts ...client.CreateCollectionOption) error
	CreateIndex(ctx context.Context, collName string, fieldName string, idx entity.Index, async bool, opts ...client.IndexOption) error
	ListCollections(ctx context.Context, opts ...client.ListCollectionOption) ([]*entity.Collection, error)
	Upsert(ctx context.Context, collName string, partitionName string, columns ...entity.Column) (entity.Column, error)
	Flush(ctx context.Context, collName string, async bool, opts ...client.FlushOption) error
	Search(ctx context.Context, collName string, partitions []string,
		expr string, outputFields []string, vectors []entity.Vector, vectorField string, metricType entity.MetricType, topK int, sp entity.SearchParam, opts ...client.SearchQueryOptionFunc) ([]client.SearchResult, error)
	Query(ctx context.Context, collectionName string, partitionNames []string, expr string, outputFields []string, opts ...client.SearchQueryOptionFunc) (client.ResultSet, error)
	Delete(ctx context.Context, collName string, partitionName string, expr string) error
	DropCollection(ctx context.Context, collName string, opts ...client.DropCollectionOption) error
}

var _ milvusDB = (client.Client)(nil)

// MilvusHandler implements KnowledgeHandler using Milvus.
//
// - id (VarChar primary key)
// - vector (FloatVector)
// - content/title/source/tags/metadata_json (VarChar)
// - org_id/doc_id/file_hash (VarChar) for simple filtering compatibility
// - created_at/updated_at (Int64 unix seconds)
type MilvusHandler struct {
	Address  string
	Username string
	Password string
	Token    string
	DBName   string

	Embedder Embedder

	cli milvusDB
}

func (h *MilvusHandler) Provider() string { return ProviderMilvus }

func (h *MilvusHandler) ensureClient(ctx context.Context) (milvusDB, error) {
	if h == nil {
		return nil, ErrHandlerNotFound
	}
	if h.cli != nil {
		return h.cli, nil
	}
	addr := strings.TrimSpace(h.Address)
	if addr == "" {
		return nil, errors.New("milvus: address is required (MILVUS_ADDRESS)")
	}

	cfg := client.Config{
		Address:  addr,
		Username: strings.TrimSpace(h.Username),
		Password: strings.TrimSpace(h.Password),
		DBName:   strings.TrimSpace(h.DBName),
		APIKey:   strings.TrimSpace(h.Token),
	}
	c, err := client.NewClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	h.cli = c
	return h.cli, nil
}

func (h *MilvusHandler) collectionNameFromOptions(namespace string) (string, error) {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return "", ErrCollectionNotFound
	}
	return ns, nil
}

func (h *MilvusHandler) ensureCollection(ctx context.Context, collection string, dim int) error {
	cli, err := h.ensureClient(ctx)
	if err != nil {
		return err
	}
	has, err := cli.HasCollection(ctx, collection)
	if err == nil && has {
		// ensure loaded
		_ = cli.LoadCollection(ctx, collection, false)
		return nil
	}
	if dim <= 0 {
		return ErrInvalidVectorDimension
	}

	schema := entity.NewSchema().
		WithName(collection).
		WithDescription("LingVoice knowledge vectors")

	schema.WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeVarChar).WithIsPrimaryKey(true).WithIsAutoID(false).WithMaxLength(128))
	schema.WithField(entity.NewField().WithName("vector").WithDataType(entity.FieldTypeFloatVector).WithDim(int64(dim)))
	schema.WithField(entity.NewField().WithName("content").WithDataType(entity.FieldTypeVarChar).WithMaxLength(65535))
	schema.WithField(entity.NewField().WithName("title").WithDataType(entity.FieldTypeVarChar).WithMaxLength(2048))
	schema.WithField(entity.NewField().WithName("source").WithDataType(entity.FieldTypeVarChar).WithMaxLength(256))
	schema.WithField(entity.NewField().WithName("tags").WithDataType(entity.FieldTypeVarChar).WithMaxLength(4096))
	schema.WithField(entity.NewField().WithName("metadata_json").WithDataType(entity.FieldTypeVarChar).WithMaxLength(65535))
	schema.WithField(entity.NewField().WithName("org_id").WithDataType(entity.FieldTypeVarChar).WithMaxLength(64))
	schema.WithField(entity.NewField().WithName("doc_id").WithDataType(entity.FieldTypeVarChar).WithMaxLength(64))
	schema.WithField(entity.NewField().WithName("file_hash").WithDataType(entity.FieldTypeVarChar).WithMaxLength(128))
	schema.WithField(entity.NewField().WithName("created_at").WithDataType(entity.FieldTypeInt64))
	schema.WithField(entity.NewField().WithName("updated_at").WithDataType(entity.FieldTypeInt64))

	if err := cli.CreateCollection(ctx, schema, 2); err != nil {
		return err
	}

	// Create vector index.
	idx, err := entity.NewIndexHNSW(entity.COSINE, 16, 200)
	if err != nil {
		return err
	}
	if err := cli.CreateIndex(ctx, collection, "vector", idx, false); err != nil {
		return err
	}
	if err := cli.LoadCollection(ctx, collection, false); err != nil {
		return err
	}
	return nil
}

func (h *MilvusHandler) Ping(ctx context.Context) error {
	cli, err := h.ensureClient(ctx)
	if err != nil {
		return err
	}
	_, err = cli.ListCollections(ctx)
	return err
}

func (h *MilvusHandler) CreateNamespace(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNamespaceNotFound
	}
	cli, err := h.ensureClient(ctx)
	if err != nil {
		return err
	}
	has, err := cli.HasCollection(ctx, name)
	if err != nil {
		return err
	}
	if has {
		return nil
	}
	return errors.New("milvus: collection does not exist yet; create happens on first upsert with known vector dimension")
}

func (h *MilvusHandler) DeleteNamespace(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNamespaceNotFound
	}
	cli, err := h.ensureClient(ctx)
	if err != nil {
		return err
	}
	has, err := cli.HasCollection(ctx, name)
	if err == nil && !has {
		return nil
	}
	return cli.DropCollection(ctx, name)
}

func (h *MilvusHandler) ListNamespaces(ctx context.Context) ([]string, error) {
	cli, err := h.ensureClient(ctx)
	if err != nil {
		return nil, err
	}
	colls, err := cli.ListCollections(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(colls))
	for _, c := range colls {
		if c == nil {
			continue
		}
		if s := strings.TrimSpace(c.Name); s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}

func (h *MilvusHandler) Upsert(ctx context.Context, records []Record, opts *UpsertOptions) error {
	if h == nil {
		return ErrHandlerNotFound
	}
	if len(records) == 0 {
		return nil
	}

	var namespace string
	if opts != nil {
		namespace = opts.Namespace
	}
	collection, err := h.collectionNameFromOptions(namespace)
	if err != nil {
		return err
	}

	// Determine vector dim.
	dim := 0
	for i := range records {
		if len(records[i].Vector) > 0 {
			dim = len(records[i].Vector)
			break
		}
	}
	if dim <= 0 {
		if h.Embedder == nil {
			return ErrEmbedderNotFound
		}
		if strings.TrimSpace(records[0].Content) == "" {
			return ErrEmptyQuery
		}
		vecs, err := h.Embedder.Embed(ctx, []string{records[0].Content})
		if err != nil {
			return err
		}
		if len(vecs) == 0 || len(vecs[0]) == 0 {
			return ErrInvalidVectorDimension
		}
		dim = len(vecs[0])
		tmp := make([]float32, dim)
		for j := range tmp {
			tmp[j] = float32(vecs[0][j])
		}
		records[0].Vector = tmp
	}

	if err := h.ensureCollection(ctx, collection, dim); err != nil {
		return err
	}
	cli, err := h.ensureClient(ctx)
	if err != nil {
		return err
	}

	batchSize := 64
	if opts != nil && opts.BatchSize > 0 {
		batchSize = opts.BatchSize
	}
	if batchSize < 1 {
		batchSize = 64
	}

	for start := 0; start < len(records); start += batchSize {
		end := start + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[start:end]

		// Fill missing vectors.
		var needIdx []int
		var inputs []string
		for i := range batch {
			if len(batch[i].Vector) > 0 {
				if len(batch[i].Vector) != dim {
					return ErrInvalidVectorDimension
				}
				continue
			}
			needIdx = append(needIdx, i)
			inputs = append(inputs, batch[i].Content)
		}
		if len(needIdx) > 0 {
			if h.Embedder == nil {
				return ErrEmbedderNotFound
			}
			vecs, err := h.Embedder.Embed(ctx, inputs)
			if err != nil {
				return err
			}
			if len(vecs) != len(needIdx) {
				return fmt.Errorf("embedder_vector_mismatch: want=%d got=%d", len(needIdx), len(vecs))
			}
			for k := range needIdx {
				if len(vecs[k]) != dim {
					return ErrInvalidVectorDimension
				}
				tmp := make([]float32, dim)
				for j := range tmp {
					tmp[j] = float32(vecs[k][j])
				}
				batch[needIdx[k]].Vector = tmp
			}
		}

		ids := make([]string, 0, len(batch))
		vectors := make([][]float32, 0, len(batch))
		contents := make([]string, 0, len(batch))
		titles := make([]string, 0, len(batch))
		sources := make([]string, 0, len(batch))
		tags := make([]string, 0, len(batch))
		metas := make([]string, 0, len(batch))
		orgIDs := make([]string, 0, len(batch))
		docIDs := make([]string, 0, len(batch))
		fileHashes := make([]string, 0, len(batch))
		created := make([]int64, 0, len(batch))
		updated := make([]int64, 0, len(batch))

		now := time.Now()
		for _, r := range batch {
			id := strings.TrimSpace(r.ID)
			if id == "" {
				return errors.New("milvus: record id is required")
			}
			ids = append(ids, id)
			vectors = append(vectors, r.Vector)
			contents = append(contents, strings.TrimSpace(r.Content))
			titles = append(titles, strings.TrimSpace(r.Title))
			sources = append(sources, strings.TrimSpace(r.Source))
			tags = append(tags, strings.Join(r.Tags, ","))
			mb, _ := json.Marshal(r.Metadata)
			metas = append(metas, string(mb))
			orgIDs = append(orgIDs, metaString(r.Metadata, "org_id"))
			docIDs = append(docIDs, metaString(r.Metadata, "doc_id"))
			fileHashes = append(fileHashes, metaString(r.Metadata, "file_hash"))
			ca := r.CreatedAt
			if ca.IsZero() {
				ca = now
			}
			ua := r.UpdatedAt
			if ua.IsZero() {
				ua = now
			}
			created = append(created, ca.Unix())
			updated = append(updated, ua.Unix())
		}

		cols := []entity.Column{
			entity.NewColumnVarChar("id", ids),
			entity.NewColumnFloatVector("vector", dim, vectors),
			entity.NewColumnVarChar("content", contents),
			entity.NewColumnVarChar("title", titles),
			entity.NewColumnVarChar("source", sources),
			entity.NewColumnVarChar("tags", tags),
			entity.NewColumnVarChar("metadata_json", metas),
			entity.NewColumnVarChar("org_id", orgIDs),
			entity.NewColumnVarChar("doc_id", docIDs),
			entity.NewColumnVarChar("file_hash", fileHashes),
			entity.NewColumnInt64("created_at", created),
			entity.NewColumnInt64("updated_at", updated),
		}

		if _, err := cli.Upsert(ctx, collection, "", cols...); err != nil {
			return err
		}
	}
	_ = h.cli.Flush(ctx, collection, false)
	return nil
}

func metaString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case string:
			return strings.TrimSpace(t)
		case fmt.Stringer:
			return strings.TrimSpace(t.String())
		case float64:
			if t == float64(int64(t)) {
				return strconv.FormatInt(int64(t), 10)
			}
			return strconv.FormatFloat(t, 'f', -1, 64)
		case int:
			return strconv.Itoa(t)
		case int64:
			return strconv.FormatInt(t, 10)
		case uint:
			return strconv.FormatUint(uint64(t), 10)
		case uint64:
			return strconv.FormatUint(t, 10)
		}
	}
	return ""
}

func (h *MilvusHandler) Query(ctx context.Context, text string, opts *QueryOptions) ([]QueryResult, error) {
	if h == nil {
		return nil, ErrHandlerNotFound
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyQuery
	}
	if h.Embedder == nil {
		return nil, ErrEmbedderNotFound
	}

	var namespace string
	if opts != nil {
		namespace = opts.Namespace
	}
	collection, err := h.collectionNameFromOptions(namespace)
	if err != nil {
		return nil, err
	}
	cli, err := h.ensureClient(ctx)
	if err != nil {
		return nil, err
	}

	topK := 5
	minScore := 0.0
	var filters []Filter
	if opts != nil {
		if opts.TopK > 0 {
			topK = opts.TopK
		}
		if opts.MinScore > 0 {
			minScore = opts.MinScore
		}
		filters = opts.Filters
	}

	vecs, err := h.Embedder.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return nil, ErrInvalidVectorDimension
	}
	dim := len(vecs[0])
	qv := make([]float32, dim)
	for i := range qv {
		qv[i] = float32(vecs[0][i])
	}

	expr := milvusExprFromFilters(filters)
	sp, _ := entity.NewIndexHNSWSearchParam(64)
	res, err := cli.Search(
		ctx,
		collection,
		[]string{}, // partitions
		expr,
		[]string{"id", "content", "title", "source", "tags", "metadata_json", "created_at", "updated_at"},
		[]entity.Vector{entity.FloatVector(qv)},
		"vector",
		entity.COSINE,
		topK,
		sp,
	)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	out := make([]QueryResult, 0, topK)
	for i := 0; i < len(res[0].Scores); i++ {
		score := float64(res[0].Scores[i])
		if score < minScore {
			continue
		}
		id, _ := res[0].IDs.GetAsString(i)
		rec := Record{
			ID:       id,
			Content:  rsGetString(res[0].Fields, "content", i),
			Title:    rsGetString(res[0].Fields, "title", i),
			Source:   rsGetString(res[0].Fields, "source", i),
			Tags:     splitComma(rsGetString(res[0].Fields, "tags", i)),
			Metadata: parseJSONMap(rsGetString(res[0].Fields, "metadata_json", i)),
		}
		if ca := rsGetInt64(res[0].Fields, "created_at", i); ca > 0 {
			rec.CreatedAt = time.Unix(ca, 0)
		}
		if ua := rsGetInt64(res[0].Fields, "updated_at", i); ua > 0 {
			rec.UpdatedAt = time.Unix(ua, 0)
		}
		out = append(out, QueryResult{Record: rec, Score: score})
	}
	return out, nil
}

func (h *MilvusHandler) Get(ctx context.Context, ids []string, opts *GetOptions) ([]Record, error) {
	if h == nil {
		return nil, ErrHandlerNotFound
	}
	if len(ids) == 0 {
		return nil, nil
	}
	var namespace string
	if opts != nil {
		namespace = opts.Namespace
	}
	collection, err := h.collectionNameFromOptions(namespace)
	if err != nil {
		return nil, err
	}
	cli, err := h.ensureClient(ctx)
	if err != nil {
		return nil, err
	}
	expr := fmt.Sprintf(`id in [%s]`, milvusStringList(ids))
	rows, err := cli.Query(ctx, collection, []string{}, expr, []string{"id", "content", "title", "source", "tags", "metadata_json", "created_at", "updated_at"})
	if err != nil {
		return nil, err
	}
	return recordsFromMilvusColumns(rows), nil
}

func (h *MilvusHandler) List(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	if h == nil {
		return nil, ErrHandlerNotFound
	}
	var namespace string
	limit := 20
	var filters []Filter
	if opts != nil {
		namespace = opts.Namespace
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		filters = opts.Filters
	}
	collection, err := h.collectionNameFromOptions(namespace)
	if err != nil {
		return nil, err
	}
	cli, err := h.ensureClient(ctx)
	if err != nil {
		return nil, err
	}
	expr := milvusExprFromFilters(filters)
	rows, err := cli.Query(ctx, collection, []string{}, expr, []string{"id", "content", "title", "source", "tags", "metadata_json", "created_at", "updated_at"})
	if err != nil {
		return nil, err
	}
	recs := recordsFromMilvusColumns(rows)
	if len(recs) > limit {
		recs = recs[:limit]
	}
	return &ListResult{Records: recs}, nil
}

func (h *MilvusHandler) Delete(ctx context.Context, ids []string, opts *DeleteOptions) error {
	if h == nil {
		return ErrHandlerNotFound
	}
	if len(ids) == 0 {
		return nil
	}
	var namespace string
	if opts != nil {
		namespace = opts.Namespace
	}
	collection, err := h.collectionNameFromOptions(namespace)
	if err != nil {
		return err
	}
	cli, err := h.ensureClient(ctx)
	if err != nil {
		return err
	}
	expr := fmt.Sprintf(`id in [%s]`, milvusStringList(ids))
	return cli.Delete(ctx, collection, "", expr)
}

func milvusExprFromFilters(filters []Filter) string {
	if len(filters) == 0 {
		return ""
	}
	var parts []string
	for _, f := range filters {
		field := strings.TrimSpace(f.Field)
		if field == "" {
			continue
		}
		// Only support a minimal subset used by current handlers (doc_id equality).
		switch f.Operator {
		case FilterOpEqual:
			if len(f.Value) == 0 {
				continue
			}
			parts = append(parts, fmt.Sprintf(`%s == %q`, field, fmt.Sprint(f.Value[0])))
		case FilterOpIn:
			if len(f.Value) == 0 {
				continue
			}
			var ss []string
			for _, v := range f.Value {
				s := strings.TrimSpace(fmt.Sprint(v))
				if s != "" {
					ss = append(ss, s)
				}
			}
			if len(ss) > 0 {
				parts = append(parts, fmt.Sprintf(`%s in [%s]`, field, milvusStringList(ss)))
			}
		}
	}
	return strings.Join(parts, " && ")
}

func milvusStringList(ss []string) string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		out = append(out, strconv.Quote(strings.TrimSpace(s)))
	}
	return strings.Join(out, ",")
}

func rsGetString(rs client.ResultSet, name string, i int) string {
	col := rs.GetColumn(name)
	if col == nil {
		return ""
	}
	s, _ := col.GetAsString(i)
	return s
}

func rsGetInt64(rs client.ResultSet, name string, i int) int64 {
	col := rs.GetColumn(name)
	if col == nil {
		return 0
	}
	n, _ := col.GetAsInt64(i)
	return n
}

func splitComma(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if x := strings.TrimSpace(p); x != "" {
			out = append(out, x)
		}
	}
	return out
}

func parseJSONMap(s string) map[string]any {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		return nil
	}
	var m map[string]any
	_ = json.Unmarshal([]byte(s), &m)
	return m
}

func recordsFromMilvusColumns(cols []entity.Column) []Record {
	// Column based return (Query API). We'll map by index.
	n := -1
	colStr := func(name string) []string {
		for _, c := range cols {
			if c.Name() == name {
				if v, ok := c.(*entity.ColumnVarChar); ok {
					return v.Data()
				}
			}
		}
		return nil
	}
	colI64 := func(name string) []int64 {
		for _, c := range cols {
			if c.Name() == name {
				if v, ok := c.(*entity.ColumnInt64); ok {
					return v.Data()
				}
			}
		}
		return nil
	}
	ids := colStr("id")
	if ids == nil {
		return nil
	}
	n = len(ids)
	content := colStr("content")
	title := colStr("title")
	source := colStr("source")
	tags := colStr("tags")
	meta := colStr("metadata_json")
	created := colI64("created_at")
	updated := colI64("updated_at")

	out := make([]Record, 0, n)
	for i := 0; i < n; i++ {
		r := Record{ID: ids[i]}
		if i < len(content) {
			r.Content = content[i]
		}
		if i < len(title) {
			r.Title = title[i]
		}
		if i < len(source) {
			r.Source = source[i]
		}
		if i < len(tags) {
			r.Tags = splitComma(tags[i])
		}
		if i < len(meta) {
			r.Metadata = parseJSONMap(meta[i])
		}
		if i < len(created) && created[i] > 0 {
			r.CreatedAt = time.Unix(created[i], 0)
		}
		if i < len(updated) && updated[i] > 0 {
			r.UpdatedAt = time.Unix(updated[i], 0)
		}
		out = append(out, r)
	}
	return out
}

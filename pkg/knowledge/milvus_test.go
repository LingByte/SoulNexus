package knowledge

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// fakeMilvusDB is an in-memory milvusDB for tests (no real Milvus / gRPC).
type fakeMilvusDB struct {
	mu          sync.Mutex
	collections map[string]bool
	rows        map[string]map[string]Record
}

func newFakeMilvusDB() *fakeMilvusDB {
	return &fakeMilvusDB{
		collections: map[string]bool{},
		rows:        map[string]map[string]Record{},
	}
}

func (f *fakeMilvusDB) ensureRows(coll string) {
	if f.rows[coll] == nil {
		f.rows[coll] = map[string]Record{}
	}
}

func varcharColsData(cols []entity.Column) map[string][]string {
	out := map[string][]string{}
	for _, col := range cols {
		switch c := col.(type) {
		case *entity.ColumnVarChar:
			out[c.Name()] = c.Data()
		}
	}
	return out
}

func int64ColsData(cols []entity.Column) map[string][]int64 {
	out := map[string][]int64{}
	for _, col := range cols {
		switch c := col.(type) {
		case *entity.ColumnInt64:
			out[c.Name()] = c.Data()
		}
	}
	return out
}

func (f *fakeMilvusDB) HasCollection(ctx context.Context, collName string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.collections[collName], nil
}

func (f *fakeMilvusDB) LoadCollection(ctx context.Context, collName string, async bool, opts ...client.LoadCollectionOption) error {
	return nil
}

func (f *fakeMilvusDB) CreateCollection(ctx context.Context, schema *entity.Schema, shardsNum int32, opts ...client.CreateCollectionOption) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	name := schema.CollectionName
	f.collections[name] = true
	f.ensureRows(name)
	return nil
}

func (f *fakeMilvusDB) CreateIndex(ctx context.Context, collName string, fieldName string, idx entity.Index, async bool, opts ...client.IndexOption) error {
	return nil
}

func (f *fakeMilvusDB) ListCollections(ctx context.Context, opts ...client.ListCollectionOption) ([]*entity.Collection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*entity.Collection
	for name := range f.collections {
		out = append(out, &entity.Collection{Name: name})
	}
	return out, nil
}

func (f *fakeMilvusDB) Upsert(ctx context.Context, collName string, partitionName string, columns ...entity.Column) (entity.Column, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.collections[collName] = true
	f.ensureRows(collName)
	m := varcharColsData(columns)
	im := int64ColsData(columns)
	ids := m["id"]
	if len(ids) == 0 {
		return nil, nil
	}
	n := len(ids)
	get := func(key string, i int) string {
		v := m[key]
		if i < len(v) {
			return v[i]
		}
		return ""
	}
	getI64 := func(key string, i int) int64 {
		v := im[key]
		if i < len(v) {
			return v[i]
		}
		return 0
	}
	metaParse := func(s string) map[string]any {
		s = strings.TrimSpace(s)
		if s == "" || s == "null" {
			return nil
		}
		var mm map[string]any
		_ = json.Unmarshal([]byte(s), &mm)
		return mm
	}
	for i := 0; i < n; i++ {
		id := strings.TrimSpace(ids[i])
		if id == "" {
			continue
		}
		rec := Record{
			ID:       id,
			Content:  get("content", i),
			Title:    get("title", i),
			Source:   get("source", i),
			Tags:     splitComma(get("tags", i)),
			Metadata: metaParse(get("metadata_json", i)),
		}
		if ca := getI64("created_at", i); ca > 0 {
			rec.CreatedAt = time.Unix(ca, 0)
		}
		if ua := getI64("updated_at", i); ua > 0 {
			rec.UpdatedAt = time.Unix(ua, 0)
		}
		f.rows[collName][id] = rec
	}
	return nil, nil
}

func (f *fakeMilvusDB) Flush(ctx context.Context, collName string, async bool, opts ...client.FlushOption) error {
	return nil
}

func recordsMapToResultSet(recs map[string]Record) client.ResultSet {
	if len(recs) == 0 {
		return client.ResultSet{
			entity.NewColumnVarChar("id", []string{}),
			entity.NewColumnVarChar("content", []string{}),
			entity.NewColumnVarChar("title", []string{}),
			entity.NewColumnVarChar("source", []string{}),
			entity.NewColumnVarChar("tags", []string{}),
			entity.NewColumnVarChar("metadata_json", []string{}),
			entity.NewColumnInt64("created_at", []int64{}),
			entity.NewColumnInt64("updated_at", []int64{}),
		}
	}
	keys := make([]string, 0, len(recs))
	for id := range recs {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	var ids, contents, titles, sources, tags, metas []string
	var created, updated []int64
	for _, id := range keys {
		r := recs[id]
		ids = append(ids, r.ID)
		contents = append(contents, r.Content)
		titles = append(titles, r.Title)
		sources = append(sources, r.Source)
		tags = append(tags, strings.Join(r.Tags, ","))
		mb, _ := json.Marshal(r.Metadata)
		metas = append(metas, string(mb))
		created = append(created, r.CreatedAt.Unix())
		updated = append(updated, r.UpdatedAt.Unix())
	}
	return client.ResultSet{
		entity.NewColumnVarChar("id", ids),
		entity.NewColumnVarChar("content", contents),
		entity.NewColumnVarChar("title", titles),
		entity.NewColumnVarChar("source", sources),
		entity.NewColumnVarChar("tags", tags),
		entity.NewColumnVarChar("metadata_json", metas),
		entity.NewColumnInt64("created_at", created),
		entity.NewColumnInt64("updated_at", updated),
	}
}

func (f *fakeMilvusDB) rowsToResultSet(coll string) client.ResultSet {
	recs := f.rows[coll]
	if recs == nil {
		return recordsMapToResultSet(nil)
	}
	return recordsMapToResultSet(recs)
}

func (f *fakeMilvusDB) Search(ctx context.Context, collName string, partitions []string, expr string, outputFields []string, vectors []entity.Vector, vectorField string, metricType entity.MetricType, topK int, sp entity.SearchParam, opts ...client.SearchQueryOptionFunc) ([]client.SearchResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rs := f.rowsToResultSet(collName)
	if rs.Len() == 0 {
		return nil, nil
	}
	n := rs.Len()
	if topK > 0 && topK < n {
		n = topK
	}

	var ids, contents, titles, sources, tags, metas []string
	var cas, uas []int64
	scores := make([]float32, 0, n)
	for i := 0; i < n; i++ {
		ids = append(ids, rsGetString(rs, "id", i))
		contents = append(contents, rsGetString(rs, "content", i))
		titles = append(titles, rsGetString(rs, "title", i))
		sources = append(sources, rsGetString(rs, "source", i))
		tags = append(tags, rsGetString(rs, "tags", i))
		metas = append(metas, rsGetString(rs, "metadata_json", i))
		cas = append(cas, rsGetInt64(rs, "created_at", i))
		uas = append(uas, rsGetInt64(rs, "updated_at", i))
		scores = append(scores, 0.99-float32(i)*0.01)
	}
	fields := client.ResultSet{
		entity.NewColumnVarChar("id", ids),
		entity.NewColumnVarChar("content", contents),
		entity.NewColumnVarChar("title", titles),
		entity.NewColumnVarChar("source", sources),
		entity.NewColumnVarChar("tags", tags),
		entity.NewColumnVarChar("metadata_json", metas),
		entity.NewColumnInt64("created_at", cas),
		entity.NewColumnInt64("updated_at", uas),
	}

	idColumn := entity.NewColumnVarChar("id", ids)
	return []client.SearchResult{{
		ResultCount: n,
		IDs:         idColumn,
		Fields:      fields,
		Scores:      scores,
	}}, nil
}

func (f *fakeMilvusDB) Query(ctx context.Context, collectionName string, partitionNames []string, expr string, outputFields []string, opts ...client.SearchQueryOptionFunc) (client.ResultSet, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	recs := f.rows[collectionName]
	if recs == nil {
		return recordsMapToResultSet(nil), nil
	}
	if idsFilter := extractMilvusInList(expr); len(idsFilter) > 0 {
		sub := make(map[string]Record, len(idsFilter))
		for _, id := range idsFilter {
			if r, ok := recs[id]; ok {
				sub[id] = r
			}
		}
		return recordsMapToResultSet(sub), nil
	}
	return recordsMapToResultSet(recs), nil
}

func extractMilvusInList(expr string) []string {
	i := strings.Index(expr, "in [")
	if i < 0 {
		return nil
	}
	s := expr[i+len("in ["):]
	j := strings.Index(s, "]")
	if j < 0 {
		return nil
	}
	s = s[:j]
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"`)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (f *fakeMilvusDB) Delete(ctx context.Context, collName string, partitionName string, expr string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, id := range extractMilvusInList(expr) {
		delete(f.rows[collName], id)
	}
	return nil
}

func (f *fakeMilvusDB) DropCollection(ctx context.Context, collName string, opts ...client.DropCollectionOption) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.collections, collName)
	delete(f.rows, collName)
	return nil
}

func TestMetaString(t *testing.T) {
	m := map[string]any{
		"a": " x ",
		"b": float64(10),
		"c": int64(11),
		"d": uint64(12),
	}
	if got := metaString(m, "a"); got != "x" {
		t.Fatalf("want x, got %q", got)
	}
	if got := metaString(m, "b"); got != "10" {
		t.Fatalf("want 10, got %q", got)
	}
	if got := metaString(m, "c"); got != "11" {
		t.Fatalf("want 11, got %q", got)
	}
	if got := metaString(m, "d"); got != "12" {
		t.Fatalf("want 12, got %q", got)
	}
}

func TestMilvusExprFromFiltersAndStringList(t *testing.T) {
	if milvusExprFromFilters(nil) != "" {
		t.Fatalf("expected empty")
	}
	expr := milvusExprFromFilters([]Filter{
		{Field: "doc_id", Operator: FilterOpEqual, Value: []any{"a"}},
		{Field: "org_id", Operator: FilterOpIn, Value: []any{"o1", "o2"}},
	})
	if expr == "" {
		t.Fatalf("expected expr")
	}
	if milvusStringList([]string{"x", "y"}) == "" {
		t.Fatalf("expected quoted list")
	}
}

func TestSplitCommaAndParseJSONMap(t *testing.T) {
	out := splitComma("a, b,,c")
	if len(out) != 3 {
		t.Fatalf("want 3, got %d", len(out))
	}
	if parseJSONMap("") != nil {
		t.Fatalf("want nil")
	}
	m := parseJSONMap(`{"a":1}`)
	if m == nil || m["a"] == nil {
		t.Fatalf("expected parsed map")
	}
}

func TestRecordsFromMilvusColumns(t *testing.T) {
	cols := []entity.Column{
		entity.NewColumnVarChar("id", []string{"1"}),
		entity.NewColumnVarChar("content", []string{"c"}),
		entity.NewColumnVarChar("title", []string{"t"}),
		entity.NewColumnVarChar("source", []string{"s"}),
		entity.NewColumnVarChar("tags", []string{"a,b"}),
		entity.NewColumnVarChar("metadata_json", []string{`{"k":1}`}),
		entity.NewColumnInt64("created_at", []int64{100}),
		entity.NewColumnInt64("updated_at", []int64{200}),
	}
	recs := recordsFromMilvusColumns(cols)
	if len(recs) != 1 || recs[0].ID != "1" || len(recs[0].Tags) != 2 {
		t.Fatalf("unexpected: %#v", recs)
	}
	if recordsFromMilvusColumns(nil) != nil {
		t.Fatalf("want nil")
	}
}

func TestMilvusEnsureClientErrors(t *testing.T) {
	var h *MilvusHandler
	_, err := h.ensureClient(context.Background())
	if err == nil {
		t.Fatalf("expected err for nil handler")
	}

	h2 := &MilvusHandler{Address: "   "}
	_, err = h2.ensureClient(context.Background())
	if err == nil {
		t.Fatalf("expected err for empty address")
	}
}

func TestMilvusHandler_MockDB_Flow(t *testing.T) {
	db := newFakeMilvusDB()
	h := &MilvusHandler{
		cli:      db,
		Embedder: fakeEmbedder{dim: 4},
	}
	ctx := context.Background()
	ns := "test_coll"

	now := time.Now().UTC()
	recs := []Record{
		{ID: "1", Content: "hello longer", Title: "t1", Source: "s1", Tags: []string{"x"}, Metadata: map[string]any{"doc_id": "d1"}, CreatedAt: now, UpdatedAt: now},
		{ID: "2", Content: "world!!!!!", Title: "t2", Source: "s2", Tags: []string{"y"}, Metadata: map[string]any{"org_id": "o1"}, CreatedAt: now, UpdatedAt: now},
	}
	if err := h.Upsert(ctx, recs, &UpsertOptions{Namespace: ns, BatchSize: 1}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if err := h.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	names, err := h.ListNamespaces(ctx)
	if err != nil || len(names) != 1 {
		t.Fatalf("ListNamespaces: err=%v names=%v", err, names)
	}

	qout, err := h.Query(ctx, "hello", &QueryOptions{Namespace: ns, TopK: 5, MinScore: 0, Filters: []Filter{{Field: "doc_id", Operator: FilterOpEqual, Value: []any{"d1"}}}})
	if err != nil || len(qout) == 0 {
		t.Fatalf("Query: err=%v n=%d", err, len(qout))
	}

	got, err := h.Get(ctx, []string{"1"}, &GetOptions{Namespace: ns})
	if err != nil || len(got) != 1 {
		t.Fatalf("Get: err=%v len=%d", err, len(got))
	}

	ls, err := h.List(ctx, &ListOptions{Namespace: ns, Limit: 10})
	if err != nil || ls == nil || len(ls.Records) != 2 {
		t.Fatalf("List: err=%v records=%d", err, len(ls.Records))
	}

	if err := h.Delete(ctx, []string{"1"}, &DeleteOptions{Namespace: ns}); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if err := h.DeleteNamespace(ctx, ns); err != nil {
		t.Fatalf("DeleteNamespace: %v", err)
	}

	if err := h.CreateNamespace(ctx, "missing"); err == nil {
		t.Fatalf("expected CreateNamespace error when collection missing")
	}
	db.collections["missing"] = true
	db.ensureRows("missing")
	if err := h.CreateNamespace(ctx, "missing"); err != nil {
		t.Fatalf("CreateNamespace: %v", err)
	}
}

func TestMilvusHandler_UpsertErrors(t *testing.T) {
	ctx := context.Background()
	var h *MilvusHandler
	if err := h.Upsert(ctx, []Record{{ID: "1", Content: "x"}}, &UpsertOptions{Namespace: "n"}); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	h2 := &MilvusHandler{cli: newFakeMilvusDB()}
	if err := h2.Upsert(ctx, []Record{{ID: "", Content: "x"}}, &UpsertOptions{Namespace: "n"}); err == nil {
		t.Fatalf("expected empty id error")
	}

	h3 := &MilvusHandler{cli: newFakeMilvusDB()}
	if err := h3.Upsert(ctx, []Record{{ID: "1", Content: "hello"}}, &UpsertOptions{Namespace: "n"}); err != ErrEmbedderNotFound {
		t.Fatalf("want ErrEmbedderNotFound, got %v", err)
	}

	h4 := &MilvusHandler{cli: newFakeMilvusDB(), Embedder: fakeEmbedder{dim: 2}}
	// First row fixes dim=1; second row has a different vector length → ErrInvalidVectorDimension.
	if err := h4.Upsert(ctx, []Record{
		{ID: "1", Content: "a", Vector: []float32{1}},
		{ID: "2", Content: "b", Vector: []float32{1, 2}},
	}, &UpsertOptions{Namespace: "n"}); err != ErrInvalidVectorDimension {
		t.Fatalf("want dim mismatch, got %v", err)
	}
}

func TestMilvusHandler_QueryErrors(t *testing.T) {
	ctx := context.Background()
	var h *MilvusHandler
	if _, err := h.Query(ctx, "x", nil); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}
	h2 := &MilvusHandler{cli: newFakeMilvusDB()}
	if _, err := h2.Query(ctx, "", nil); err != ErrEmptyQuery {
		t.Fatalf("want ErrEmptyQuery, got %v", err)
	}
	h3 := &MilvusHandler{cli: newFakeMilvusDB()}
	if _, err := h3.Query(ctx, "x", nil); err != ErrEmbedderNotFound {
		t.Fatalf("want ErrEmbedderNotFound, got %v", err)
	}
}

func TestMilvusEnsureCollection_InvalidDim(t *testing.T) {
	h := &MilvusHandler{cli: newFakeMilvusDB()}
	if err := h.ensureCollection(context.Background(), "c", 0); err != ErrInvalidVectorDimension {
		t.Fatalf("want ErrInvalidVectorDimension, got %v", err)
	}
}

func TestMilvusEnsureCollection_Existing(t *testing.T) {
	db := newFakeMilvusDB()
	db.collections["c"] = true
	db.ensureRows("c")
	h := &MilvusHandler{cli: db}
	if err := h.ensureCollection(context.Background(), "c", 8); err != nil {
		t.Fatalf("ensureCollection: %v", err)
	}
}

package search

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2/mapping"

	"github.com/blevesearch/bleve/v2"
)

var ErrClosed = errors.New("search engine closed")

type Engine interface {
	Index(ctx context.Context, doc Doc) error
	IndexBatch(ctx context.Context, docs []Doc) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, req SearchRequest) (SearchResult, error)
	GetAutoCompleteSuggestions(ctx context.Context, keyword string) ([]string, error)
	GetSearchSuggestions(ctx context.Context, keyword string) ([]string, error)
	Close() error
}

type bleveEngine struct {
	cfg           Config
	index         bleve.Index
	defaultFields []string
	mu            sync.RWMutex
	closed        bool
}

func New(cfg Config, m mapping.IndexMapping) (Engine, error) { // mapping 引自 bleve
	be := &bleveEngine{cfg: cfg, defaultFields: cfg.DefaultSearchFields}

	var idx bleve.Index
	if _, err := os.Stat(cfg.IndexPath); err == nil {
		i, e := bleve.Open(cfg.IndexPath)
		if e != nil {
			return nil, e
		}
		idx = i
	} else if os.IsNotExist(err) {
		i, e := bleve.New(cfg.IndexPath, m)
		if e != nil {
			return nil, e
		}
		idx = i
	} else {
		return nil, err
	}
	be.index = idx
	return be, nil
}

func (e *bleveEngine) guard() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.closed {
		return ErrClosed
	}
	return nil
}

func (e *bleveEngine) withDeadline(ctx context.Context, d time.Duration, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if d <= 0 {
		return fn(ctx)
	}
	c, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	ch := make(chan error, 1)
	go func() {
		ch <- fn(c)
	}()
	select {
	case <-c.Done():
		return c.Err()
	case err := <-ch:
		return err
	}
}

func (e *bleveEngine) Index(ctx context.Context, doc Doc) error {
	if err := e.guard(); err != nil {
		return err
	}
	return e.withDeadline(ctx, e.cfg.QueryTimeout, func(ctx context.Context) error {
		data := make(map[string]any, len(doc.Fields)+1)
		for k, v := range doc.Fields {
			data[k] = v
		}
		if doc.Type != "" {
			data["type"] = doc.Type
		}
		return e.index.Index(doc.ID, data)
	})
}

func (e *bleveEngine) IndexBatch(ctx context.Context, docs []Doc) error {
	if err := e.guard(); err != nil {
		return err
	}
	bs := e.cfg.BatchSize
	if bs <= 0 {
		bs = 200
	}
	return e.withDeadline(ctx, 0, func(ctx context.Context) error {
		for i := 0; i < len(docs); i += bs {
			end := i + bs
			if end > len(docs) {
				end = len(docs)
			}
			b := e.index.NewBatch()
			for _, d := range docs[i:end] {
				data := make(map[string]any, len(d.Fields)+1)
				for k, v := range d.Fields {
					data[k] = v
				}
				if d.Type != "" {
					data["type"] = d.Type
				}
				if err := b.Index(d.ID, data); err != nil {
					return err
				}
			}
			if err := e.index.Batch(b); err != nil {
				return err
			}
		}
		return nil
	})
}

func (e *bleveEngine) Delete(ctx context.Context, id string) error {
	if err := e.guard(); err != nil {
		return err
	}
	return e.withDeadline(ctx, e.cfg.QueryTimeout, func(ctx context.Context) error {
		return e.index.Delete(id)
	})
}

func (e *bleveEngine) Search(ctx context.Context, req SearchRequest) (SearchResult, error) {
	if err := e.guard(); err != nil {
		return SearchResult{}, err
	}

	q := buildQuery(req, e.defaultFields)
	sr := bleve.NewSearchRequest(q)

	// 分页
	if req.Size <= 0 {
		req.Size = 10
	}
	if req.From < 0 {
		req.From = 0
	}
	sr.Size = req.Size
	sr.From = req.From

	// 排序
	if len(req.SortBy) > 0 {
		sr.SortBy(req.SortBy)
	}

	// 字段
	if len(req.IncludeFields) == 0 {
		sr.Fields = []string{"*"}
	} else {
		sr.Fields = req.IncludeFields
	}

	// 高亮
	if req.Highlight {
		hl := bleve.NewHighlightWithStyle("html")
		// 如果你想限定高亮字段（可选）
		// 注意：v2 没有 SetFragmentSize/SetMaxFragments
		// 有些版本没有 AddField 方法；若没有，就直接用默认（所有可高亮字段）
		for _, f := range req.HighlightFields {
			// 如果你的 bleve 版本有 AddField:
			// hl.AddField(f)
			// 否则可以忽略字段选择，使用默认行为
			_ = f
		}
		sr.Highlight = hl
	}

	// Facets
	if len(req.Facets) > 0 {
		sr.Facets = make(map[string]*bleve.FacetRequest, len(req.Facets))
		for _, f := range req.Facets {
			size := f.Size
			if size <= 0 {
				size = 10
			}
			sr.Facets[f.Name] = bleve.NewFacetRequest(f.Field, size)
		}
	}

	var res *bleve.SearchResult
	err := e.withDeadline(ctx, e.cfg.QueryTimeout, func(ctx context.Context) error {
		r, e2 := e.index.Search(sr)
		if e2 != nil {
			return e2
		}
		res = r
		return nil
	})
	if err != nil {
		return SearchResult{}, err
	}

	out := SearchResult{
		Total:  res.Total,
		Took:   res.Took,
		Hits:   make([]Hit, 0, len(res.Hits)),
		Facets: map[string]FacetResult{},
	}
	for _, h := range res.Hits {
		out.Hits = append(out.Hits, Hit{
			ID:        h.ID,
			Score:     h.Score,
			Fields:    h.Fields,
			Fragments: h.Fragments,
		})
	}
	// Facets
	if res.Facets != nil {
		for name, fr := range res.Facets {
			ft := FacetResult{Total: fr.Total}
			if fr.Terms != nil {
				for _, t := range fr.Terms.Terms() {
					ft.Terms = append(ft.Terms, FacetTerm{Term: t.Term, Count: t.Count})
				}
			}
			out.Facets[name] = ft
		}
	}
	return out, nil
}

func (e *bleveEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil
	}
	e.closed = true
	return e.index.Close()
}

func (e *bleveEngine) GetAutoCompleteSuggestions(ctx context.Context, keyword string) ([]string, error) {
	if err := e.guard(); err != nil {
		return nil, err
	}
	if keyword == "" {
		return []string{}, nil
	}

	var suggestions []string
	err := e.withDeadline(ctx, e.cfg.QueryTimeout, func(ctx context.Context) error {
		// 这里假设你用前缀查询实现自动补全
		query := bleve.NewPrefixQuery(keyword)
		sr := bleve.NewSearchRequest(query)
		sr.Size = 5 // 限制返回最多5个建议

		searchResult, err := e.index.Search(sr)
		if err != nil {
			return err
		}

		suggestions = make([]string, 0, len(searchResult.Hits))
		for _, hit := range searchResult.Hits {
			// 根据需要，可以提取 `hit.Fields` 来作为补全建议
			suggestions = append(suggestions, hit.ID)
		}
		return nil
	})
	return suggestions, err
}

func (e *bleveEngine) GetSearchSuggestions(ctx context.Context, keyword string) ([]string, error) {
	if err := e.guard(); err != nil {
		return nil, err
	}
	if keyword == "" {
		return []string{}, nil
	}

	var suggestions []string
	err := e.withDeadline(ctx, e.cfg.QueryTimeout, func(ctx context.Context) error {
		// 这里可以通过索引中的某些字段获取搜索建议
		// 例如，你可以查询所有标题或者文章内容来生成相关建议
		query := bleve.NewMatchQuery(keyword)
		sr := bleve.NewSearchRequest(query)
		sr.Size = 5 // 限制返回最多5个建议

		searchResult, err := e.index.Search(sr)
		if err != nil {
			return err
		}

		suggestions = make([]string, 0, len(searchResult.Hits))
		for _, hit := range searchResult.Hits {
			// 假设我们通过 ID 来推荐建议，也可以根据需要提取其他字段
			suggestions = append(suggestions, hit.ID)
		}
		return nil
	})
	return suggestions, err
}

package search

import (
	"fmt"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	q "github.com/blevesearch/bleve/v2/search/query"
)

func buildQuery(req SearchRequest, defaultFields []string) q.Query {
	var must, should, mustNot []q.Query

	// 0) 兼容旧 Keyword（按字段 OR）
	keyword := strings.TrimSpace(req.Keyword)
	if keyword != "" {
		fields := req.SearchFields
		if len(fields) == 0 {
			fields = defaultFields
		}
		var qs string
		if len(fields) == 0 {
			qs = keyword
		} else {
			parts := make([]string, 0, len(fields))
			for _, f := range fields {
				if f != "" {
					parts = append(parts, fmt.Sprintf("%s:(%s)", f, keyword))
				}
			}
			if len(parts) > 0 {
				qs = strings.Join(parts, " OR ")
			} else {
				qs = keyword
			}
		}
		if qs != "" {
			must = append(must, bleve.NewQueryStringQuery(qs))
		}
	}

	// 1) QueryString 子句
	if req.QueryString != nil {
		qs := req.QueryString.Query
		if len(req.QueryString.Fields) > 0 {
			parts := make([]string, 0, len(req.QueryString.Fields))
			for _, f := range req.QueryString.Fields {
				parts = append(parts, fmt.Sprintf("%s:(%s)", f, qs))
			}
			qs = strings.Join(parts, " OR ")
		}
		qq := bleve.NewQueryStringQuery(qs)
		if req.QueryString.Boost != nil {
			qq.SetBoost(*req.QueryString.Boost)
		}
		should = append(should, qq) // 放入 should，利于相关性提升
	}

	// 2) Term 等值过滤
	for f, vs := range req.MustTerms {
		if len(vs) == 1 {
			tq := bleve.NewTermQuery(vs[0])
			tq.SetField(f)
			must = append(must, tq)
		} else if len(vs) > 1 {
			qs := make([]q.Query, 0, len(vs))
			for _, v := range vs {
				tq := bleve.NewTermQuery(v)
				tq.SetField(f)
				qs = append(qs, tq)
			}
			must = append(must, bleve.NewDisjunctionQuery(qs...))
		}
	}
	for f, vs := range req.MustNotTerms {
		for _, v := range vs {
			tq := bleve.NewTermQuery(v)
			tq.SetField(f)
			mustNot = append(mustNot, tq)
		}
	}
	for f, vs := range req.ShouldTerms {
		for _, v := range vs {
			tq := bleve.NewTermQuery(v)
			tq.SetField(f)
			should = append(should, tq)
		}
	}

	// 3) 高级子句
	for _, m := range req.Matches {
		mq := bleve.NewMatchQuery(m.Query)
		if m.Field != "" {
			mq.SetField(m.Field)
		}
		if m.Boost != nil {
			mq.SetBoost(*m.Boost)
		}
		if strings.ToLower(m.Operator) == "and" {
			mq.SetOperator(q.MatchQueryOperatorAnd)
		}
		should = append(should, mq)
	}
	for _, p := range req.Phrases {
		pq := bleve.NewMatchPhraseQuery(p.Phrase)
		if p.Field != "" {
			pq.SetField(p.Field)
		}
		if p.Boost != nil {
			pq.SetBoost(*p.Boost)
		}
		// 不再设置 Slop
		should = append(should, pq)
	}
	for _, pr := range req.Prefixes {
		pq := bleve.NewPrefixQuery(pr.Prefix)
		if pr.Field != "" {
			pq.SetField(pr.Field)
		}
		if pr.Boost != nil {
			pq.SetBoost(*pr.Boost)
		}
		should = append(should, pq)
	}
	for _, w := range req.Wildcards {
		wq := bleve.NewWildcardQuery(w.Pattern)
		if w.Field != "" {
			wq.SetField(w.Field)
		}
		if w.Boost != nil {
			wq.SetBoost(*w.Boost)
		}
		should = append(should, wq)
	}
	for _, r := range req.Regexps {
		rq := bleve.NewRegexpQuery(r.Pattern)
		if r.Field != "" {
			rq.SetField(r.Field)
		}
		if r.Boost != nil {
			rq.SetBoost(*r.Boost)
		}
		should = append(should, rq)
	}
	for _, fz := range req.Fuzzies {
		fq := bleve.NewFuzzyQuery(fz.Term)
		if fz.Field != "" {
			fq.SetField(fz.Field)
		}
		if fz.Fuzziness > 0 {
			fq.SetFuzziness(fz.Fuzziness)
		}
		if fz.Prefix > 0 {
			fq.SetPrefix(fz.Prefix)
		}
		if fz.Boost != nil {
			fq.SetBoost(*fz.Boost)
		}
		should = append(should, fq)
	}

	// 4) 数值范围
	for _, r := range req.NumericRanges {
		rq := bleve.NewNumericRangeQuery(rMin(r), rMax(r))
		rq.SetField(r.Field)
		must = append(must, rq)
	}

	// 5) 时间范围
	for _, r := range req.TimeRanges {
		var start, end time.Time
		if r.From != nil {
			start = *r.From
		}
		if r.To != nil {
			end = *r.To
		}
		drq := bleve.NewDateRangeInclusiveQuery(start, end, boolPtr(r.IncFrom), boolPtr(r.IncTo))
		drq.SetField(r.Field)
		must = append(must, drq)
	}

	// 6) 组装 Boolean
	boolQ := bleve.NewBooleanQuery()
	if len(must) > 0 {
		boolQ.AddMust(must...)
	}
	if len(mustNot) > 0 {
		boolQ.AddMustNot(mustNot...)
	}

	if len(should) > 0 {
		if req.MinShould > 0 {
			// 把所有 should 子句放进一个 DisjunctionQuery，并设置最小匹配数
			disj := bleve.NewDisjunctionQuery(should...)
			disj.SetMin(float64(req.MinShould))
			// 把这个“至少命中 N 个 should”的条件当作 MUST 条件加入
			boolQ.AddMust(disj)
		} else {
			// 普通 should（没有最小匹配数要求）
			boolQ.AddShould(should...)
		}
	}
	return boolQ
}

func rMin(n NumericRangeFilter) *float64 {
	if n.GT != nil {
		return n.GT
	}
	if n.GTE != nil {
		return n.GTE
	}
	return nil
}
func rMax(n NumericRangeFilter) *float64 {
	if n.LT != nil {
		return n.LT
	}
	if n.LTE != nil {
		return n.LTE
	}
	return nil
}
func boolPtr(b bool) *bool { return &b }

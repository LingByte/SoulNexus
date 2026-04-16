package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// SelfQueryOptions configures a single SelfQuery extraction call.
type SelfQueryOptions struct {
	Model string

	// AllowedFields restricts which filter keys the model may use (prompt hint).
	AllowedFields []string

	// MaxJSONChars caps input length before JSON extraction (0 = default 16000).
	MaxJSONChars int

	// UsePlainQuery uses Query instead of QueryWithOptions with JSON object mode.
	// Set true for backends that handle JSON poorly in structured-output mode.
	UsePlainQuery bool
}

// SelfQueryFilterSpec is the structured filter subset produced by the model.
type SelfQueryFilterSpec struct {
	Namespace string   `json:"namespace,omitempty"`
	Source    string   `json:"source,omitempty"`
	DocType   string   `json:"doc_type,omitempty"`
	Location  string   `json:"location,omitempty"`
	Years     []string `json:"years,omitempty"`
	Dates     []string `json:"dates,omitempty"`
	TagsAny   []string `json:"tags_any,omitempty"`
}

// SelfQueryResult is the parsed self-query output plus a Qdrant-oriented filter map.
// It is an intermediate retrieval plan (rewritten query + metadata constraints), not
// retrieved documents and not the final user-facing answer; wire it to your search/vector
// layer, then run generation over returned chunks.
type SelfQueryResult struct {
	Query   string
	Filters map[string]any

	Spec SelfQueryFilterSpec
	Raw  string
}

type selfQueryLLMOutput struct {
	Query   string              `json:"query"`
	Filters SelfQueryFilterSpec `json:"filters"`
}

// SelfQueryExtractor turns a natural-language question into a search query + structured filters
// for the retrieval step of RAG. It does not execute search or produce grounded answers by itself.
// Use NewSelfQueryExtractor and call Extract only when you need that decomposition (on demand).
type SelfQueryExtractor struct {
	LLM LLMHandler

	AllowedFields []string
}

// NewSelfQueryExtractor returns an extractor. allowedFields may be empty (no restriction hint).
func NewSelfQueryExtractor(h LLMHandler, allowedFields []string) *SelfQueryExtractor {
	return &SelfQueryExtractor{LLM: h, AllowedFields: allowedFields}
}

// Extract runs the self-query prompt and parses JSON from the model output.
func (e *SelfQueryExtractor) Extract(ctx context.Context, question string, opt *SelfQueryOptions) (*SelfQueryResult, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	if e == nil || e.LLM == nil {
		return nil, errors.New("llm handler is nil")
	}
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, errors.New("question is empty")
	}

	model := ""
	maxJSON := 16_000
	allowed := e.AllowedFields
	usePlain := false
	if opt != nil {
		model = strings.TrimSpace(opt.Model)
		if opt.MaxJSONChars > 0 {
			maxJSON = opt.MaxJSONChars
		}
		if len(opt.AllowedFields) > 0 {
			allowed = opt.AllowedFields
		}
		usePlain = opt.UsePlainQuery
	}

	prompt := buildSelfQueryPrompt(question, allowed)

	var out string
	var err error
	if usePlain {
		out, err = e.LLM.Query(prompt, model)
	} else {
		var resp *QueryResponse
		resp, err = e.LLM.QueryWithOptions(prompt, &QueryOptions{
			Model:                     model,
			EnableSelfQueryJSONOutput: true,
			MaxTokens:                 1024,
			Temperature:               0,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || len(resp.Choices) == 0 {
			return nil, errors.New("empty selfquery response")
		}
		out = resp.Choices[0].Content
	}
	if err != nil {
		return nil, err
	}

	out = strings.TrimSpace(out)
	jsonText := ExtractJSONFromLLMOutput(out, maxJSON)
	if jsonText == "" {
		jsonText = out
	}

	var parsed selfQueryLLMOutput
	if err := json.Unmarshal([]byte(jsonText), &parsed); err != nil {
		return &SelfQueryResult{Raw: out}, fmt.Errorf("parse selfquery json failed: %w", err)
	}
	parsed.Query = strings.TrimSpace(parsed.Query)
	if parsed.Query == "" {
		parsed.Query = question
	}

	filters := selfQuerySpecToQdrantFilter(parsed.Filters)

	return &SelfQueryResult{
		Query:   parsed.Query,
		Filters: filters,
		Spec:    parsed.Filters,
		Raw:     out,
	}, nil
}

// SelfQueryExtract is a convenience wrapper around NewSelfQueryExtractor(...).Extract(...).
func SelfQueryExtract(ctx context.Context, h LLMHandler, question string, allowedFields []string, opt *SelfQueryOptions) (*SelfQueryResult, error) {
	return NewSelfQueryExtractor(h, allowedFields).Extract(ctx, question, opt)
}

// ExtractJSONFromLLMOutput pulls a JSON object/array substring from common LLM wrappers (fences, prose).
func ExtractJSONFromLLMOutput(s string, maxChars int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if maxChars > 0 && len(s) > maxChars {
		s = s[:maxChars]
	}
	if i := strings.Index(s, "```json"); i >= 0 {
		s2 := s[i+len("```json"):]
		if j := strings.Index(s2, "```"); j >= 0 {
			return strings.TrimSpace(s2[:j])
		}
	}
	l := strings.Index(s, "{")
	r := strings.LastIndex(s, "}")
	if l >= 0 && r > l {
		return strings.TrimSpace(s[l : r+1])
	}
	return ""
}

func buildSelfQueryPrompt(question string, allowedFields []string) string {
	fields := make([]string, 0, len(allowedFields))
	for _, f := range allowedFields {
		f = strings.TrimSpace(f)
		if f != "" {
			fields = append(fields, f)
		}
	}
	sort.Strings(fields)

	fieldsLine := ""
	if len(fields) > 0 {
		fieldsLine = "允许字段（只允许从这里选）：" + strings.Join(fields, ", ") + "\n"
	}

	return "你是一个检索自查询(Self-Query)抽取器。\n" +
		"目标：把用户问题抽取为：核心检索 query + 结构化过滤条件 filters。\n" +
		"输出必须是严格 JSON，不要解释，不要 markdown。\n" +
		"JSON schema：{" +
		"\"query\": string, " +
		"\"filters\": {" +
		"\"namespace\"?: string, " +
		"\"source\"?: string, " +
		"\"doc_type\"?: string, " +
		"\"location\"?: string, " +
		"\"years\"?: string[], " +
		"\"dates\"?: string[], " +
		"\"tags_any\"?: string[]" +
		"}" +
		"}\n" +
		fieldsLine +
		"用户问题：" + question + "\n"
}

func selfQuerySpecToQdrantFilter(s SelfQueryFilterSpec) map[string]any {
	must := make([]any, 0)
	should := make([]any, 0)

	addShould := func(key string, match any) {
		should = append(should, map[string]any{"key": key, "match": match})
	}
	addMust := func(key string, match any) {
		must = append(must, map[string]any{"key": key, "match": match})
	}

	if strings.TrimSpace(s.Namespace) != "" {
		addMust("namespace", map[string]any{"value": strings.TrimSpace(s.Namespace)})
	}
	if strings.TrimSpace(s.Source) != "" {
		addMust("source", map[string]any{"value": strings.TrimSpace(s.Source)})
	}
	if strings.TrimSpace(s.DocType) != "" {
		addMust("metadata.doc_type", map[string]any{"value": strings.TrimSpace(s.DocType)})
		addShould("tags", map[string]any{"any": []string{"type:" + strings.TrimSpace(s.DocType)}})
	}
	if strings.TrimSpace(s.Location) != "" {
		addMust("metadata.location", map[string]any{"value": strings.TrimSpace(s.Location)})
		addShould("tags", map[string]any{"any": []string{"loc:" + strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s.Location), " ", "_"))}})
	}
	if len(s.Years) > 0 {
		years := selfQueryDedupStrings(s.Years)
		addShould("metadata.years", map[string]any{"any": years})
		tags := make([]string, 0, len(years))
		for _, y := range years {
			tags = append(tags, "year:"+y)
		}
		addShould("tags", map[string]any{"any": tags})
	}
	if len(s.Dates) > 0 {
		dates := selfQueryDedupStrings(s.Dates)
		addShould("metadata.dates", map[string]any{"any": dates})
		tags := make([]string, 0, len(dates))
		for _, d := range dates {
			tags = append(tags, "date:"+d)
		}
		addShould("tags", map[string]any{"any": tags})
	}
	if len(s.TagsAny) > 0 {
		addShould("tags", map[string]any{"any": selfQueryDedupStrings(s.TagsAny)})
	}

	filter := map[string]any{}
	if len(must) > 0 {
		filter["must"] = must
	}
	if len(should) > 0 {
		if len(must) == 0 {
			filter["must"] = []any{}
		}
		filterMust := filter["must"].([]any)
		filterMust = append(filterMust, map[string]any{"should": should})
		filter["must"] = filterMust
	}

	if len(filter) == 0 {
		return nil
	}
	return filter
}

func selfQueryDedupStrings(in []string) []string {
	set := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if set[s] {
			continue
		}
		set[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

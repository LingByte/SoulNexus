package search

import (
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/v2/mapping"
)

func BuildIndexMapping(defaultAnalyzer string) *mapping.IndexMappingImpl {
	if defaultAnalyzer == "" {
		defaultAnalyzer = standard.Name
	}
	idx := mapping.NewIndexMapping()
	if idx == nil {
		panic("failed to create index mapping")
	}
	idx.DefaultAnalyzer = defaultAnalyzer
	idx.TypeField = "type"

	// 文本
	text := mapping.NewTextFieldMapping()
	text.Store = true
	text.Index = true
	text.Analyzer = defaultAnalyzer
	text.IncludeInAll = true
	text.IncludeTermVectors = true // 高亮更精准

	// 关键词
	kw := mapping.NewTextFieldMapping()
	kw.Store = true
	kw.Index = true
	kw.Analyzer = keyword.Name

	// 数值/时间
	num := mapping.NewNumericFieldMapping()
	num.Store = true
	num.Index = true
	dt := mapping.NewDateTimeFieldMapping()
	dt.Store = true
	dt.Index = true

	article := mapping.NewDocumentMapping()
	article.Dynamic = false
	article.AddFieldMappingsAt("title", text)
	article.AddFieldMappingsAt("body", text)
	article.AddFieldMappingsAt("tags", kw)
	article.AddFieldMappingsAt("author", kw)
	article.AddFieldMappingsAt("createdAt", dt)
	article.AddFieldMappingsAt("views", num)
	idx.AddDocumentMapping("article", article)

	def := mapping.NewDocumentMapping()
	def.Dynamic = true // 允许动态字段，支持用户自定义字段
	// 添加通用字段映射
	def.AddFieldMappingsAt("userId", kw)        // 用户ID，用于用户区分
	def.AddFieldMappingsAt("type", kw)          // 文档类型
	def.AddFieldMappingsAt("title", text)       // 标题
	def.AddFieldMappingsAt("description", text) // 描述
	def.AddFieldMappingsAt("content", text)     // 内容
	def.AddFieldMappingsAt("url", kw)           // URL
	def.AddFieldMappingsAt("icon", kw)          // 图标
	def.AddFieldMappingsAt("category", kw)      // 分类
	idx.DefaultMapping = def
	return idx
}

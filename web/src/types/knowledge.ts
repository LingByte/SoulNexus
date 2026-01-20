// 知识库类型定义
export interface KnowledgeItem {
  id: string
  title: string
  content: string
  category: string
  tags: string[]
  createdAt: string
  updatedAt: string
  source?: string
  priority: number
  isActive: boolean
}

export interface KnowledgeCategory {
  id: string
  name: string
  description: string
  icon?: string
  color?: string
  parentId?: string
  children?: KnowledgeCategory[]
}

export interface KnowledgeSearchQuery {
  query: string
  category?: string
  tags?: string[]
  limit?: number
  offset?: number
}

export interface KnowledgeSearchResult {
  item: KnowledgeItem
  relevance: number
  matchedFields: string[]
  highlights: string[]
}

export interface KnowledgeBase {
  id: string
  name: string
  description: string
  categories: KnowledgeCategory[]
  items: KnowledgeItem[]
  settings: KnowledgeBaseSettings
}

export interface KnowledgeBaseSettings {
  autoIndex: boolean
  searchThreshold: number
  maxResults: number
  enableSemanticSearch: boolean
  enableFullTextSearch: boolean
}

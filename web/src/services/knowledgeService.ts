import { 
  KnowledgeItem, 
  KnowledgeCategory, 
  KnowledgeSearchQuery, 
  KnowledgeSearchResult,
  KnowledgeBase,
  KnowledgeBaseSettings
} from '../types/knowledge'

// 默认知识库设置
const DEFAULT_SETTINGS: KnowledgeBaseSettings = {
  autoIndex: true,
  searchThreshold: 0.7,
  maxResults: 10,
  enableSemanticSearch: true,
  enableFullTextSearch: true
}

// 系统知识库数据
const SYSTEM_KNOWLEDGE: KnowledgeItem[] = [
  {
    id: 'react-basics',
    title: 'React基础概念',
    content: `React是一个用于构建用户界面的JavaScript库。

## 核心概念
- **组件**: React应用的基本构建块
- **JSX**: JavaScript的语法扩展
- **状态**: 组件的数据
- **Props**: 组件间传递数据的方式

## 常用Hook
- useState: 管理组件状态
- useEffect: 处理副作用
- useContext: 使用Context
- useCallback: 优化函数性能`,
    category: 'react',
    tags: ['react', '基础', '组件', 'hook'],
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    source: 'system',
    priority: 10,
    isActive: true
  },
  {
    id: 'component-library',
    title: '组件库使用指南',
    content: `本系统提供了丰富的UI组件库。

## 可用组件
- **Button**: 按钮组件，支持多种样式
- **Input**: 输入框组件
- **Card**: 卡片容器组件
- **Modal**: 模态框组件
- **Tabs**: 标签页组件

## 使用方法
\`\`\`tsx
import { Button } from '@/components/UI/Button'

function MyComponent() {
  return <Button onClick={() => console.log('clicked')}>点击我</Button>
}
\`\`\``,
    category: 'components',
    tags: ['组件', 'UI', '使用指南'],
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    source: 'system',
    priority: 9,
    isActive: true
  },
  {
    id: 'button-component',
    title: 'Button组件详细说明',
    content: `Button组件是系统中最常用的UI组件之一。

## 基本用法
\`\`\`tsx
import Button from '@/components/UI/Button'

// 基础按钮
<Button>点击我</Button>

// 主要按钮
<Button variant="primary">主要按钮</Button>

// 次要按钮
<Button variant="secondary">次要按钮</Button>

// 轮廓按钮
<Button variant="outline">轮廓按钮</Button>

// 幽灵按钮
<Button variant="ghost">幽灵按钮</Button>
\`\`\`

## 尺寸选项
- **sm**: 小尺寸按钮
- **md**: 中等尺寸按钮（默认）
- **lg**: 大尺寸按钮

\`\`\`tsx
<Button size="sm">小按钮</Button>
<Button size="md">中等按钮</Button>
<Button size="lg">大按钮</Button>
\`\`\`

## 状态控制
\`\`\`tsx
// 禁用状态
<Button disabled>禁用按钮</Button>

// 点击事件
<Button onClick={() => alert('按钮被点击了！')}>
  点击我
</Button>
\`\`\`

## 完整示例
\`\`\`tsx
import React, { useState } from 'react'
import Button from '@/components/UI/Button'

function ButtonExample() {
  const [count, setCount] = useState(0)

  return (
    <div className="space-x-2">
      <Button 
        variant="primary" 
        onClick={() => setCount(count + 1)}
      >
        计数: {count}
      </Button>
      
      <Button 
        variant="outline" 
        size="sm"
        onClick={() => setCount(0)}
      >
        重置
      </Button>
    </div>
  )
}
\`\`\``,
    category: 'components',
    tags: ['Button', '按钮', '组件', 'UI', '示例'],
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    source: 'system',
    priority: 8,
    isActive: true
  },
  {
    id: 'ai-features',
    title: 'AI功能说明',
    content: `系统集成了强大的AI功能。

## 主要功能
- **智能推荐**: 基于用户行为提供个性化推荐
- **智能搜索**: 语义搜索和意图理解
- **聊天机器人**: 智能对话助手
- **AI洞察**: 数据分析和洞察

## 配置说明
AI功能可以通过配置页面进行设置，包括模型选择、API密钥配置等。`,
    category: 'ai',
    tags: ['AI', '智能', '推荐', '搜索'],
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    source: 'system',
    priority: 8,
    isActive: true
  },
  {
    id: 'cache-system',
    title: '缓存系统',
    content: `系统提供了多层次的缓存解决方案。

## 缓存策略
- **内存缓存**: 快速访问的临时数据
- **本地存储**: 持久化的用户数据
- **会话存储**: 会话期间的数据
- **IndexedDB**: 大量结构化数据

## 使用场景
- 用户偏好设置
- 搜索结果缓存
- 组件状态缓存
- 离线数据存储`,
    category: 'system',
    tags: ['缓存', '性能', '存储'],
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    source: 'system',
    priority: 7,
    isActive: true
  }
]

const SYSTEM_CATEGORIES: KnowledgeCategory[] = [
  {
    id: 'react',
    name: 'React开发',
    description: 'React相关知识和最佳实践',
    icon: '⚛️',
    color: '#61DAFB'
  },
  {
    id: 'components',
    name: '组件库',
    description: 'UI组件使用指南',
    icon: '🧩',
    color: '#3B82F6'
  },
  {
    id: 'ai',
    name: 'AI功能',
    description: '人工智能功能说明',
    icon: '🤖',
    color: '#10B981'
  },
  {
    id: 'system',
    name: '系统功能',
    description: '系统架构和功能说明',
    icon: '⚙️',
    color: '#6B7280'
  }
]

class KnowledgeService {
  private knowledgeBase: KnowledgeBase
  private searchIndex: Map<string, string[]> = new Map()

  constructor() {
    this.knowledgeBase = {
      id: 'system',
      name: '系统知识库',
      description: 'SoulNexus 灵枢系统知识库',
      categories: SYSTEM_CATEGORIES,
      items: SYSTEM_KNOWLEDGE,
      settings: DEFAULT_SETTINGS
    }
    this.buildSearchIndex()
  }

  // 构建搜索索引
  private buildSearchIndex() {
    this.knowledgeBase.items.forEach(item => {
      const keywords = [
        item.title,
        item.content,
        ...item.tags,
        item.category
      ].join(' ').toLowerCase()
      
      this.searchIndex.set(item.id, keywords.split(/\s+/))
    })
  }

  // 搜索知识库
  async searchKnowledge(query: KnowledgeSearchQuery): Promise<KnowledgeSearchResult[]> {
    const { query: searchQuery, category, tags, limit = 10 } = query
    const results: KnowledgeSearchResult[] = []

    // 过滤条件
    let filteredItems = this.knowledgeBase.items.filter(item => item.isActive)
    
    if (category) {
      filteredItems = filteredItems.filter(item => item.category === category)
    }
    
    if (tags && tags.length > 0) {
      filteredItems = filteredItems.filter(item => 
        tags.some(tag => item.tags.includes(tag))
      )
    }

    // 搜索匹配
    const searchTerms = searchQuery.toLowerCase().split(/\s+/)
    
    filteredItems.forEach(item => {
      const keywords = this.searchIndex.get(item.id) || []
      let relevance = 0
      const matchedFields: string[] = []
      const highlights: string[] = []

      // 计算相关性
      searchTerms.forEach(term => {
        // 标题匹配权重最高
        if (item.title.toLowerCase().includes(term)) {
          relevance += 3
          matchedFields.push('title')
          highlights.push(`标题: ${item.title}`)
        }
        
        // 内容匹配
        if (item.content.toLowerCase().includes(term)) {
          relevance += 1
          matchedFields.push('content')
        }
        
        // 标签匹配
        if (item.tags.some(tag => tag.toLowerCase().includes(term))) {
          relevance += 2
          matchedFields.push('tags')
        }
        
        // 关键词匹配
        if (keywords.some(keyword => keyword.includes(term))) {
          relevance += 1
        }
      })

      if (relevance > 0) {
        results.push({
          item,
          relevance,
          matchedFields: [...new Set(matchedFields)],
          highlights
        })
      }
    })

    // 按相关性排序
    results.sort((a, b) => b.relevance - a.relevance)
    
    return results.slice(0, limit)
  }

  // 获取知识项
  getKnowledgeItem(id: string): KnowledgeItem | null {
    return this.knowledgeBase.items.find(item => item.id === id) || null
  }

  // 获取分类
  getCategories(): KnowledgeCategory[] {
    return this.knowledgeBase.categories
  }

  // 获取分类下的知识项
  getItemsByCategory(categoryId: string): KnowledgeItem[] {
    return this.knowledgeBase.items.filter(item => 
      item.category === categoryId && item.isActive
    )
  }

  // 添加知识项
  addKnowledgeItem(item: Omit<KnowledgeItem, 'id' | 'createdAt' | 'updatedAt'>): KnowledgeItem {
    const newItem: KnowledgeItem = {
      ...item,
      id: this.generateId(),
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    }
    
    this.knowledgeBase.items.push(newItem)
    this.buildSearchIndex()
    
    return newItem
  }

  // 更新知识项
  updateKnowledgeItem(id: string, updates: Partial<KnowledgeItem>): KnowledgeItem | null {
    const index = this.knowledgeBase.items.findIndex(item => item.id === id)
    if (index === -1) return null
    
    const updatedItem = {
      ...this.knowledgeBase.items[index],
      ...updates,
      updatedAt: new Date().toISOString()
    }
    
    this.knowledgeBase.items[index] = updatedItem
    this.buildSearchIndex()
    
    return updatedItem
  }

  // 删除知识项
  deleteKnowledgeItem(id: string): boolean {
    const index = this.knowledgeBase.items.findIndex(item => item.id === id)
    if (index === -1) return false
    
    this.knowledgeBase.items.splice(index, 1)
    this.searchIndex.delete(id)
    
    return true
  }

  // 生成唯一ID
  private generateId(): string {
    return Math.random().toString(36).substr(2, 9)
  }
}

export const knowledgeService = new KnowledgeService()

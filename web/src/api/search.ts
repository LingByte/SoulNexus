import { post, ApiResponse } from '@/utils/request'

// 后端搜索请求接口
export interface SearchRequest {
  keyword?: string
  searchFields?: string[]
  mustTerms?: Record<string, string[]>
  mustNotTerms?: Record<string, string[]>
  shouldTerms?: Record<string, string[]>
  numericRanges?: Array<{
    field: string
    gte?: number
    gt?: number
    lte?: number
    lt?: number
  }>
  timeRanges?: Array<{
    field: string
    from?: string
    to?: string
    incFrom?: boolean
    incTo?: boolean
  }>
  queryString?: {
    query: string
    fields?: string[]
    boost?: number
  }
  matches?: Array<{
    field: string
    query: string
    boost?: number
    operator?: string
  }>
  phrases?: Array<{
    field: string
    phrase: string
    slop?: number
    boost?: number
  }>
  prefixes?: Array<{
    field: string
    prefix: string
    boost?: number
  }>
  wildcards?: Array<{
    field: string
    pattern: string
    boost?: number
  }>
  regexps?: Array<{
    field: string
    pattern: string
    boost?: number
  }>
  fuzzies?: Array<{
    field: string
    term: string
    fuzziness?: number
    prefix?: number
    boost?: number
  }>
  minShould?: number
  facets?: Array<{
    name: string
    field: string
    size?: number
  }>
  sortBy?: string[]
  from?: number
  size?: number
  includeFields?: string[]
  highlight?: boolean
  highlightFields?: string[]
  fragmentSize?: number
  maxFragments?: number
}

// 后端搜索结果接口
export interface BackendSearchHit {
  id?: string
  ID?: string // 兼容旧版本大写字段
  score?: number
  Score?: number // 兼容旧版本大写字段
  fields?: Record<string, any>
  Fields?: Record<string, any> // 兼容旧版本大写字段
  fragments?: Record<string, string[]>
  Fragments?: Record<string, string[]> // 兼容旧版本大写字段
}

export interface BackendSearchResult {
  total?: number
  Total?: number // 兼容旧版本大写字段
  took?: number
  Took?: number // 兼容旧版本大写字段
  hits?: BackendSearchHit[]
  Hits?: BackendSearchHit[] // 兼容旧版本大写字段
  facets?: Record<string, {
    total: number
    terms: Array<{
      term: string
      count: number
    }>
  }>
}

// 执行搜索
export const search = async (request: SearchRequest): Promise<ApiResponse<BackendSearchResult>> => {
  return post('/search', request)
}

// 自动补全
export const autoComplete = async (keyword: string): Promise<ApiResponse<string[]>> => {
  return post('/search/auto-complete', { keyword })
}

// 搜索建议
export const getSuggestions = async (keyword: string): Promise<ApiResponse<string[]>> => {
  return post('/search/suggest', { keyword })
}


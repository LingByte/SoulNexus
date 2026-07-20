import { get, post, put, del, type ApiResponse } from '@/utils/request'
import type { AssistantToolRow } from '@/api/assistantTools'

export type McpMarketCategory = 'crm' | 'order' | 'utility' | 'custom' | string
export type McpMarketStatus = 'draft' | 'published' | 'archived'

export interface McpMarketItem {
  id: string
  slug: string
  name: string
  displayName?: string
  description?: string
  category: McpMarketCategory
  icon?: string
  logoUrl?: string
  tags?: string
  version?: string
  status: McpMarketStatus
  author?: string
  authorTenantId?: string
  sourceToolId?: string
  kind?: string
  mcpSseUrl?: string
  headers?: Record<string, string> | null
  timeoutMs?: number
  installCount?: number
  activated?: boolean
  createdAt?: string
  updatedAt?: string
}

export async function listMcpMarket(params?: {
  category?: string
  keyword?: string
}): Promise<ApiResponse<McpMarketItem[]>> {
  const q = new URLSearchParams()
  if (params?.category) q.set('category', params.category)
  if (params?.keyword) q.set('keyword', params.keyword)
  const suffix = q.toString() ? `?${q}` : ''
  return get(`/mcp-market${suffix}`)
}

export async function getMcpMarketItem(id: string): Promise<ApiResponse<McpMarketItem>> {
  return get(`/mcp-market/${id}`)
}

export async function activateMcpMarketItem(id: string): Promise<ApiResponse<AssistantToolRow>> {
  return post(`/mcp-market/${id}/activate`, {})
}

export async function publishCustomMcpToMarket(body: {
  toolId: string
  slug?: string
  displayName?: string
  description?: string
  category?: string
  version?: string
  logoUrl?: string
  tags?: string
  publish?: boolean
}): Promise<ApiResponse<McpMarketItem>> {
  return post('/mcp-market/publish', body)
}

export async function delistCustomMcpFromMarket(toolId: string): Promise<ApiResponse<McpMarketItem>> {
  return post('/mcp-market/delist', { toolId })
}

export async function uploadMcpMarketLogo(file: File): Promise<ApiResponse<{ logoUrl: string }>> {
  const form = new FormData()
  form.append('file', file)
  return post('/mcp-market/logo', form)
}

export async function listPlatformMcpMarket(params?: {
  status?: string
  page?: number
  size?: number
}): Promise<ApiResponse<{ list: McpMarketItem[]; total: number; page: number; size: number }>> {
  const q = new URLSearchParams()
  if (params?.status) q.set('status', params.status)
  if (params?.page) q.set('page', String(params.page))
  if (params?.size) q.set('size', String(params.size))
  const suffix = q.toString() ? `?${q}` : ''
  return get(`/platform/mcp-market${suffix}`)
}

export async function createPlatformMcpMarketItem(body: Partial<McpMarketItem> & {
  slug: string
  mcpSseUrl: string
}): Promise<ApiResponse<McpMarketItem>> {
  return post('/platform/mcp-market', body)
}

export async function updatePlatformMcpMarketItem(
  id: string,
  body: Partial<McpMarketItem>,
): Promise<ApiResponse<McpMarketItem>> {
  return put(`/platform/mcp-market/${id}`, body)
}

export async function deletePlatformMcpMarketItem(id: string): Promise<ApiResponse<{ id: string }>> {
  return del(`/platform/mcp-market/${id}`)
}

export async function uploadPlatformMcpMarketLogo(file: File): Promise<ApiResponse<{ logoUrl: string }>> {
  const form = new FormData()
  form.append('file', file)
  return post('/platform/mcp-market/logo', form)
}

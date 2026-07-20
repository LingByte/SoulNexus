import { get, post, put, del, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'

export interface TenantIMChannelRow {
  id: number
  tenantId: number
  provider: 'wecom' | 'feishu' | string
  code: string
  name: string
  enabled: boolean
  remark?: string
  config?: Record<string, unknown>
}

export async function listIMChannels(page = 1, size = 20): Promise<ApiResponse<Paginated<TenantIMChannelRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  return get(`/im-channels?${q.toString()}`)
}

export async function listIMProviders(): Promise<ApiResponse<{ providers: string[] }>> {
  return get('/im-channels/providers')
}

export async function createIMChannel(body: {
  provider: string
  code?: string
  name: string
  remark?: string
  enabled?: boolean
  config: Record<string, unknown>
}): Promise<ApiResponse<TenantIMChannelRow>> {
  return post('/im-channels', body)
}

export async function updateIMChannel(
  id: number,
  body: {
    name?: string
    remark?: string
    enabled?: boolean
    config?: Record<string, unknown>
  },
): Promise<ApiResponse<TenantIMChannelRow>> {
  return put(`/im-channels/${id}`, body)
}

export async function deleteIMChannel(id: number): Promise<ApiResponse<unknown>> {
  return del(`/im-channels/${id}`)
}

export async function testIMChannel(id: number): Promise<ApiResponse<unknown>> {
  return post(`/im-channels/${id}/test`, {})
}

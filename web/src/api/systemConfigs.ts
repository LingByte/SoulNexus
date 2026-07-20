import { del, get, post, put, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'

export type ConfigFormat = 'text' | 'json' | 'yaml' | 'int' | 'float' | 'bool'

export interface SystemConfigRow {
  id: number | string
  key: string
  desc?: string
  value?: string
  format?: ConfigFormat | string
  autoload?: boolean
  public?: boolean
  createdAt?: string
  updatedAt?: string
}

export async function listSystemConfigs(
  page = 1,
  size = 20,
  search?: string,
): Promise<ApiResponse<Paginated<SystemConfigRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  if (search?.trim()) q.set('search', search.trim())
  return get(`/system-configs?${q.toString()}`)
}

export async function getSystemConfig(id: number | string): Promise<ApiResponse<SystemConfigRow>> {
  return get(`/system-configs/${id}`)
}

export async function createSystemConfig(body: {
  key: string
  desc?: string
  value?: string
  format?: ConfigFormat | string
  autoload?: boolean
  public?: boolean
}): Promise<ApiResponse<SystemConfigRow>> {
  return post('/system-configs', body)
}

export async function updateSystemConfig(
  id: number | string,
  body: {
    desc?: string
    value?: string
    format?: ConfigFormat | string
    autoload?: boolean
    public?: boolean
  },
): Promise<ApiResponse<SystemConfigRow>> {
  return put(`/system-configs/${id}`, body)
}

export async function deleteSystemConfig(id: number | string): Promise<ApiResponse<{ id: number | string }>> {
  return del(`/system-configs/${id}`)
}

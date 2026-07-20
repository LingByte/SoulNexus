import { get, post, put, del, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'

/** Snowflake IDs (>2^53) must stay strings in JavaScript. */
export type SnowflakeId = string

export interface TenantUserRow {
  id: SnowflakeId
  email?: string
  phone?: string
  username?: string
  displayName?: string
  status?: string
  tenantGroups?: { id: SnowflakeId; name: string; isDefault?: boolean }[]
  tenantGroup?: { id: SnowflakeId; name: string; isDefault?: boolean }
}

export async function listTenantUsers(
  page = 1,
  size = 100,
  opts?: { status?: string; search?: string },
): Promise<ApiResponse<Paginated<TenantUserRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  if (opts?.status) q.set('status', opts.status)
  if (opts?.search) q.set('search', opts.search)
  return get(`/tenant-users?${q.toString()}`)
}

export async function createTenantUser(body: {
  email: string
  password?: string
  phone?: string
  username?: string
  displayName?: string
  status?: string
}): Promise<ApiResponse<TenantUserRow>> {
  return post('/tenant-users', body)
}

export async function updateTenantUser(
  id: SnowflakeId,
  body: {
    email?: string
    phone?: string
    username?: string
    displayName?: string
    status?: string
  },
): Promise<ApiResponse<{ id: SnowflakeId }>> {
  return put(`/tenant-users/${id}`, body)
}

export async function updateTenantUserStatus(
  id: SnowflakeId,
  status: string,
): Promise<ApiResponse<{ id: SnowflakeId; status: string }>> {
  return put(`/tenant-users/${id}/status`, { status })
}

export async function deleteTenantUser(id: SnowflakeId): Promise<ApiResponse<{ id: SnowflakeId }>> {
  return del(`/tenant-users/${id}`)
}

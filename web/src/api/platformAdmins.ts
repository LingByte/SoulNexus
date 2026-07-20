import { del, get, post, put, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'

export interface PlatformAdminRow {
  id: number | string
  email: string
  displayName?: string
  status: string
  createdAt?: string
  updatedAt?: string
}

export async function listPlatformAdmins(
  page = 1,
  size = 20,
  search?: string,
): Promise<ApiResponse<Paginated<PlatformAdminRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  if (search?.trim()) q.set('search', search.trim())
  return get(`/platform-admins?${q.toString()}`)
}

export async function getPlatformAdmin(id: number | string): Promise<ApiResponse<PlatformAdminRow>> {
  return get(`/platform-admins/${id}`)
}

export async function createPlatformAdmin(body: {
  email: string
  password: string
  displayName?: string
  status?: string
}): Promise<ApiResponse<PlatformAdminRow>> {
  return post('/platform-admins', body)
}

export async function updatePlatformAdmin(
  id: number | string,
  body: { email?: string; displayName?: string },
): Promise<ApiResponse<PlatformAdminRow>> {
  return put(`/platform-admins/${id}`, body)
}

export async function updatePlatformAdminStatus(
  id: number | string,
  status: 'active' | 'disabled',
): Promise<ApiResponse<{ id: number | string; status: string }>> {
  return put(`/platform-admins/${id}/status`, { status })
}

export async function resetPlatformAdminPassword(
  id: number | string,
  password: string,
): Promise<ApiResponse<{ id: number | string }>> {
  return put(`/platform-admins/${id}/password`, { password })
}

export async function deletePlatformAdmin(id: number | string): Promise<ApiResponse<{ id: number | string }>> {
  return del(`/platform-admins/${id}`)
}

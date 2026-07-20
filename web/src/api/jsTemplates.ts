import { get, post, put, del, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'

export interface JSTemplateRow {
  id: string
  tenantId?: string
  jsSourceId: string
  name: string
  avatarUrl?: string
  content: string
  usage?: string
  status?: string
  createdAt?: string
  updatedAt?: string
}

/** 有效模版主键：非空、非 0、非 new、纯数字 */
export function isValidJSTemplateId(id: unknown): id is string {
  const s = String(id ?? '').trim()
  if (!s || s === '0' || s === 'new') return false
  if (!/^\d+$/.test(s)) return false
  try {
    return BigInt(s) > 0n
  } catch {
    return false
  }
}

export async function listJSTemplates(
  page = 1,
  size = 50,
  opts?: { name?: string },
): Promise<ApiResponse<Paginated<JSTemplateRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  if (opts?.name) q.set('name', opts.name)
  return get(`/js-templates?${q.toString()}`)
}

export async function getJSTemplate(id: string): Promise<ApiResponse<JSTemplateRow>> {
  if (!isValidJSTemplateId(id)) {
    return Promise.reject(new Error('invalid js template id'))
  }
  return get(`/js-templates/${id}`)
}

export async function createJSTemplate(body: {
  name: string
  content: string
  usage?: string
  status?: string
  avatarUrl?: string
}): Promise<ApiResponse<JSTemplateRow>> {
  return post('/js-templates', body)
}

export async function updateJSTemplate(
  id: string,
  body: { name?: string; content?: string; usage?: string; status?: string; avatarUrl?: string },
): Promise<ApiResponse<JSTemplateRow>> {
  if (!isValidJSTemplateId(id)) {
    return Promise.reject(new Error('invalid js template id'))
  }
  return put(`/js-templates/${id}`, body)
}

export async function uploadJSTemplateAvatar(
  id: string,
  file: File,
): Promise<ApiResponse<{ avatarUrl?: string; template?: JSTemplateRow }>> {
  if (!isValidJSTemplateId(id)) {
    return Promise.reject(new Error('invalid js template id'))
  }
  const form = new FormData()
  form.append('file', file)
  return post(`/js-templates/${id}/avatar`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

export async function deleteJSTemplate(id: string): Promise<ApiResponse<{ id: string }>> {
  if (!isValidJSTemplateId(id)) {
    return Promise.reject(new Error('invalid js template id'))
  }
  return del(`/js-templates/${id}`)
}

export interface JSTemplateUsageRow {
  id: string
  tenantId?: string
  jsSourceId: string
  event: string
  sessionId?: string
  credentialId?: string
  userId?: string
  createdAt?: string
}

export async function listJSTemplateUsage(
  page = 1,
  size = 20,
  opts?: { jsSourceId?: string },
): Promise<ApiResponse<Paginated<JSTemplateUsageRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  if (opts?.jsSourceId) q.set('jsSourceId', opts.jsSourceId)
  return get(`/js-templates/usage?${q.toString()}`)
}

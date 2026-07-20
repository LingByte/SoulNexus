import { get, post, put, del, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'

export interface TenantWebhookRow {
  id: number
  tenantId: number
  name: string
  url: string
  secret?: string
  events: string[]
  enabled: boolean
  createdAt?: string
  updatedAt?: string
}

export interface TenantWebhookDeliveryRow {
  id: number
  webhookId: number
  event: string
  callId?: string
  status: string
  httpCode?: number
  error?: string
  createdAt?: string
}

export async function listWebhooks(page = 1, size = 20): Promise<ApiResponse<Paginated<TenantWebhookRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  return get(`/webhooks?${q.toString()}`)
}

export async function listWebhookEvents(): Promise<ApiResponse<{ events: string[] }>> {
  return get('/webhooks/events')
}

export async function createWebhook(body: {
  name: string
  url: string
  secret?: string
  events?: string[]
  enabled?: boolean
}): Promise<ApiResponse<TenantWebhookRow>> {
  return post('/webhooks', body)
}

export async function updateWebhook(
  id: number,
  body: Partial<{ name: string; url: string; secret: string; events: string[]; enabled: boolean }>,
): Promise<ApiResponse<TenantWebhookRow>> {
  return put(`/webhooks/${id}`, body)
}

export async function deleteWebhook(id: number): Promise<ApiResponse<{ id: number }>> {
  return del(`/webhooks/${id}`)
}

export async function testWebhook(id: number): Promise<ApiResponse<{ sent: boolean }>> {
  return post(`/webhooks/${id}/test`, {})
}

export async function listWebhookDeliveries(
  id: number,
  page = 1,
  size = 20,
): Promise<ApiResponse<Paginated<TenantWebhookDeliveryRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  return get(`/webhooks/${id}/deliveries?${q.toString()}`)
}

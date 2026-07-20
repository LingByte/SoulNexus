import { get, post, put, del, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'

export interface BillingPlanRow {
  id: number
  name: string
  description?: string
  currency: string
  callRatePerMinute: number
  llmRatePer1kTokens: number
  status: string
  createdAt?: string
  updatedAt?: string
}

export async function listBillingPlans(page = 1, size = 20, name?: string): Promise<ApiResponse<Paginated<BillingPlanRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  if (name?.trim()) q.set('name', name.trim())
  return get(`/platform/billing-plans?${q.toString()}`)
}

export async function createBillingPlan(body: {
  name: string
  description?: string
  currency?: string
  callRatePerMinute?: number
  llmRatePer1kTokens?: number
  status?: string
}): Promise<ApiResponse<BillingPlanRow>> {
  return post('/platform/billing-plans', body)
}

export async function updateBillingPlan(id: number, body: {
  name: string
  description?: string
  currency?: string
  callRatePerMinute?: number
  llmRatePer1kTokens?: number
  status?: string
}): Promise<ApiResponse<BillingPlanRow>> {
  return put(`/platform/billing-plans/${id}`, body)
}

export async function deleteBillingPlan(id: number): Promise<ApiResponse<{ id: number }>> {
  return del(`/platform/billing-plans/${id}`)
}

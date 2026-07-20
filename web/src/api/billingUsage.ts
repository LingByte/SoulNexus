import { get, post, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'
import type { TenantBillingAccount } from '@/api/tenantBills'

export interface TenantQuotaSnapshot {
  maxConcurrentCalls?: number
  dailyMinuteLimit?: number
  monthlyMinuteLimit?: number
  dailyMinutesUsed?: number
  monthlyMinutesUsed?: number
  heldMinutes?: number
  activeReservations?: number
  quotaSuspended?: boolean
  licenseExpiresAt?: string
  licenseValid?: boolean
}

export interface BillingUsageSummary {
  account: TenantBillingAccount
  quotas: TenantQuotaSnapshot
}

export interface TenantUsageEventRow {
  id: number
  tenantId?: number
  callId?: string
  durationSec?: number
  billedMinutes?: number
  minutesDeducted?: number
  amountCharged?: number
  createdAt?: string
}

export interface BusinessMetrics {
  rangeStart?: string
  rangeEnd?: string
  totalCalls?: number
  connectedCalls?: number
  connectRate?: number
  billedMinutes?: number
  avgDurationSec?: number
  aiToHumanCount?: number
  transferRate?: number
  failureReasons?: { reason: string; callCount: number }[]
}

export function getBillingUsageSummary(tenantId?: string): Promise<ApiResponse<BillingUsageSummary>> {
  const q = tenantId ? `?tenantId=${encodeURIComponent(tenantId)}` : ''
  return get<BillingUsageSummary>(`/billing/usage/summary${q}`)
}

export function listBillingUsageEvents(
  page = 1,
  size = 20,
  params?: { tenantId?: string; startAt?: string; endAt?: string },
): Promise<ApiResponse<Paginated<TenantUsageEventRow>>> {
  const sp = new URLSearchParams({ page: String(page), size: String(size) })
  if (params?.tenantId) sp.set('tenantId', params.tenantId)
  if (params?.startAt) sp.set('startAt', params.startAt)
  if (params?.endAt) sp.set('endAt', params.endAt)
  return get<Paginated<TenantUsageEventRow>>(`/billing/usage/events?${sp}`)
}

export function getBillingBusinessMetrics(params?: {
  days?: number
  from?: string
  to?: string
  tenantId?: string
}): Promise<ApiResponse<BusinessMetrics>> {
  const sp = new URLSearchParams()
  if (params?.days) sp.set('days', String(params.days))
  if (params?.from) sp.set('from', params.from)
  if (params?.to) sp.set('to', params.to)
  if (params?.tenantId) sp.set('tenantId', params.tenantId)
  const q = sp.toString()
  return get<BusinessMetrics>(`/billing/metrics${q ? `?${q}` : ''}`)
}

export function markTenantBillPaid(id: string): Promise<ApiResponse<unknown>> {
  return post(`/billing/bills/${id}/mark-paid`, {})
}

import { get, post, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'
import { getApiBaseURL } from '@/config/apiConfig'
import { readAuthToken } from '@/utils/authToken'

export type TenantBillingMode = 'prepaid' | 'postpaid'

export interface TenantBillingAccount {
  tenantId?: number
  billingMode?: TenantBillingMode
  billingUnlimited?: boolean
  prepaidMinutesRemaining?: number
  remainingMinutesDisplay?: string
  billingRatePerMinute?: number
  billingCurrency?: string
  meteredBilledMinutes?: number
  meteredCallCount?: number
}

export interface TenantBillDirectionUsage {
  direction?: string
  callCount?: number
  billedMinutes?: number
}

export interface TenantBillDailyUsage {
  day?: string
  callCount?: number
  billedMinutes?: number
}

export interface TenantBillUsageDetail {
  direction?: TenantBillDirectionUsage[]
  daily?: TenantBillDailyUsage[]
}

export interface TenantBillRow {
  id: string
  tenantId?: number
  billNo?: string
  periodStart?: string
  periodEnd?: string
  status?: 'draft' | 'finalized' | 'paid'
  currency?: string
  totalAmount?: number
  callCount?: number
  connectedCallCount?: number
  billedMinutes?: number
  inboundCallCount?: number
  outboundCallCount?: number
  aiToHumanCount?: number
  analysisCount?: number
  usageDetail?: TenantBillUsageDetail
  finalizedAt?: string
  paidAt?: string
  generatedBy?: string
  createdAt?: string
  updatedAt?: string
}

export type TenantBillListOptions = {
  period?: string
  status?: string
  tenantId?: string
  startAt?: string
  endAt?: string
}

export async function listTenantBills(
  page = 1,
  size = 20,
  opts: TenantBillListOptions = {},
): Promise<ApiResponse<Paginated<TenantBillRow>>> {
  const params = new URLSearchParams({ page: String(page), size: String(size) })
  if (opts.period) params.set('period', opts.period)
  if (opts.status) params.set('status', opts.status)
  if (opts.tenantId) params.set('tenantId', opts.tenantId)
  if (opts.startAt) params.set('startAt', opts.startAt)
  if (opts.endAt) params.set('endAt', opts.endAt)
  return get<Paginated<TenantBillRow>>(`/billing/bills?${params.toString()}`)
}

export async function getTenantBill(id: string): Promise<ApiResponse<TenantBillRow>> {
  return get<TenantBillRow>(`/billing/bills/${id}`)
}

export async function finalizeTenantBill(id: string): Promise<ApiResponse<TenantBillRow>> {
  return post<TenantBillRow>(`/billing/bills/${id}/finalize`, {})
}

export async function getTenantBillingAccount(tenantId?: string): Promise<ApiResponse<TenantBillingAccount>> {
  const q = tenantId ? `?tenantId=${encodeURIComponent(tenantId)}` : ''
  return get<TenantBillingAccount>(`/billing/account${q}`)
}

function parseExportFilename(contentDisposition: string | null, fallback: string): string {
  if (!contentDisposition) return fallback
  const m = /filename="([^"]+)"/i.exec(contentDisposition)
  return m?.[1]?.trim() || fallback
}

export async function downloadTenantBillExport(id: string, format: 'csv' | 'xlsx'): Promise<void> {
  const base = getApiBaseURL().replace(/\/$/, '')
  const token = readAuthToken()
  const url = `${base}/billing/bills/${id}/export?format=${format}&_t=${Date.now()}`
  const res = await fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) {
    throw new Error(`export failed (${res.status})`)
  }
  const blob = await res.blob()
  const ext = format === 'xlsx' ? 'xlsx' : 'csv'
  const filename = parseExportFilename(res.headers.get('Content-Disposition'), `bill-${id}.${ext}`)
  const objectUrl = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = objectUrl
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(objectUrl)
}

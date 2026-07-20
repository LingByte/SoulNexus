import { get, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'

export interface OperationLogRow {
  id: number
  createdAt?: string
  tenantId?: number
  operator?: string
  operatorKind?: string
  operatorId?: number
  action?: string
  resource?: string
  resourceId?: number
  resourceName?: string
  summary?: string
  detailJson?: string
  httpMethod?: string
  httpPath?: string
  requestId?: string
  clientIP?: string
  success?: boolean
  errorMsg?: string
}

export type OperationLogListOptions = {
  operator?: string
  action?: string
  resource?: string
  resourceId?: number
  success?: boolean
  from?: string
  to?: string
  tenantId?: number
}

function buildQuery(page: number, size: number, opts?: OperationLogListOptions) {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  if (opts?.operator) q.set('operator', opts.operator)
  if (opts?.action) q.set('action', opts.action)
  if (opts?.resource) q.set('resource', opts.resource)
  if (opts?.resourceId != null && opts.resourceId > 0) q.set('resourceId', String(opts.resourceId))
  if (opts?.success != null) q.set('success', opts.success ? 'true' : 'false')
  if (opts?.from) q.set('from', opts.from)
  if (opts?.to) q.set('to', opts.to)
  if (opts?.tenantId != null && opts.tenantId > 0) q.set('tenantId', String(opts.tenantId))
  return q
}

export async function listMyOperationLogs(
  page = 1,
  size = 20,
  opts?: OperationLogListOptions,
): Promise<ApiResponse<Paginated<OperationLogRow>>> {
  return get(`/operation-logs/mine?${buildQuery(page, size, opts).toString()}`)
}

export async function listTenantOperationLogs(
  page = 1,
  size = 20,
  opts?: OperationLogListOptions,
): Promise<ApiResponse<Paginated<OperationLogRow>>> {
  return get(`/operation-logs/tenant?${buildQuery(page, size, opts).toString()}`)
}

export async function listOperationLogsPlatform(
  page = 1,
  size = 20,
  opts?: OperationLogListOptions,
): Promise<ApiResponse<Paginated<OperationLogRow>>> {
  return get(`/operation-logs?${buildQuery(page, size, opts).toString()}`)
}

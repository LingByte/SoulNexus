import { get, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'

export interface LoginHistoryRow {
  id: string
  principalType: string
  principalId: string
  tenantId?: string
  email?: string
  clientIp?: string
  city?: string
  location?: string
  userAgent?: string
  loginMethod?: string
  success: boolean
  failureReason?: string
  deviceKey?: string
  createdAt?: string
}

export async function fetchMyLoginHistory(page = 1, size = 20): Promise<ApiResponse<Paginated<LoginHistoryRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  return get(`/me/login-history?${q.toString()}`)
}

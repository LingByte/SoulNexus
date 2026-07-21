import { del, get, post, type ApiResponse } from '@/utils/request'
import type { AIProviderPoolRow } from '@/api/platformAiPools'

export interface TenantAIPoolGrantRow {
  id: number
  tenantId: number
  poolId: number
  quotaLimit: number
  quotaUsed: number
  enabled: boolean
  pool?: AIProviderPoolRow
}

export async function listTenantAIPoolGrants(
  tenantId: number,
): Promise<ApiResponse<{ list: TenantAIPoolGrantRow[] }>> {
  return get(`/tenants/${tenantId}/ai-pool-grants`)
}

export async function upsertTenantAIPoolGrant(
  tenantId: number,
  body: { poolId: number; quotaLimit: number; enabled?: boolean },
): Promise<ApiResponse<TenantAIPoolGrantRow>> {
  return post(`/tenants/${tenantId}/ai-pool-grants`, body)
}

export async function deleteTenantAIPoolGrant(
  tenantId: number,
  grantId: number,
): Promise<ApiResponse<{ id: number }>> {
  return del(`/tenants/${tenantId}/ai-pool-grants/${grantId}`)
}

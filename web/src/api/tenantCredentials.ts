import { get, post, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'
import type { CredentialCreateResult, CredentialRow } from '@/api/credentials'

export async function listTenantCredentialsPlatform(
  tenantId: string,
  page = 1,
  size = 20,
): Promise<ApiResponse<Paginated<CredentialRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  return get(`/tenants/${tenantId}/credentials?${q.toString()}`)
}

export async function createTenantCredentialPlatform(
  tenantId: string,
  body: { name?: string; expiresAt?: string | null; voiceMode?: string },
): Promise<ApiResponse<CredentialCreateResult>> {
  return post(`/tenants/${tenantId}/credentials`, {
    kind: 'platform_bundle',
    ...body,
  })
}

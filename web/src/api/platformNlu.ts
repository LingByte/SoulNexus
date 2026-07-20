import { get, put, post, del } from '@/utils/request'
import type { NluParseResult, TenantNluModel, TenantNluSpec } from '@/api/nluModels'

export type PlatformNluModel = TenantNluModel & {
  tenantName?: string
}

export async function fetchPlatformNluConfig() {
  const res = await get<{ deployEnabled: boolean; platformReady: boolean; mode?: string; maxIntents?: number }>(
    '/platform/nlu-models/config'
  )
  return res.data!
}

export async function listPlatformNluModels(opts?: { page?: number; size?: number; tenantId?: string }) {
  const q = new URLSearchParams()
  if (opts?.page) q.set('page', String(opts.page))
  if (opts?.size) q.set('size', String(opts.size))
  if (opts?.tenantId) q.set('tenantId', opts.tenantId)
  const res = await get<{ list: PlatformNluModel[]; total: number; page: number; size: number }>(
    `/platform/nlu-models?${q.toString()}`
  )
  return res.data ?? { list: [], total: 0, page: 1, size: 20 }
}

export async function updatePlatformNluModel(
  id: string,
  body: { name?: string; description?: string; minConfidence?: number; spec?: TenantNluSpec }
) {
  const res = await put<PlatformNluModel>(`/platform/nlu-models/${id}`, body)
  return res.data!
}

export async function deletePlatformNluModel(id: string) {
  await del(`/platform/nlu-models/${id}`)
}

export async function trainPlatformNluModel(id: string) {
  const res = await post<{ message?: string; model?: PlatformNluModel }>(`/platform/nlu-models/${id}/train`)
  return res.data!
}

export async function parsePlatformNluModel(id: string, text: string) {
  const res = await post<NluParseResult>(`/platform/nlu-models/${id}/parse`, { text })
  return res.data!
}

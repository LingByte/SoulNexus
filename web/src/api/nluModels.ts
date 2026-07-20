import { get, post, put, del } from '@/utils/request'

export type TenantNluIntentDef = {
  name: string
  reply: string
  replyVariants?: string[]
  keywords?: string[]
  keywordBonus?: number
  samples?: string[]
}

export type TenantNluSpec = {
  minSoftmaxProb?: number
  keywordLogitBonus?: number
  minTopMargin?: number
  defaultReply?: string
  intents: TenantNluIntentDef[]
}

function normalizeNluModel(row: TenantNluModel): TenantNluModel {
  return {
    ...row,
    id: String(row.id),
    tenantId: String(row.tenantId),
  }
}

export type TenantNluModel = {
  id: string
  tenantId: string
  name: string
  description?: string
  status: 'draft' | 'training' | 'ready' | 'failed'
  numClasses: number
  minConfidence: number
  spec: TenantNluSpec
  trainError?: string
  modelPath?: string
  boundAssistants?: number
  createdAt?: string
  updatedAt?: string
}

export type NluParseResult = {
  channel: 'intent' | 'llm' | 'unknown'
  reply?: string
  prediction: {
    intent_name: string
    confidence: number
    keyword_bias_applied?: boolean
  }
  /** Server-measured inference latency in milliseconds. */
  latencyMs?: number
}

export async function fetchNluConfig() {
  const res = await get<{ deployEnabled: boolean; platformReady: boolean; mode?: string; maxIntents?: number }>(
    '/nlu-models/config'
  )
  return res.data!
}

export async function listNluModels(): Promise<TenantNluModel[]> {
  const res = await get<TenantNluModel[]>('/nlu-models')
  return (res.data ?? []).map(normalizeNluModel)
}

export async function getNluModel(id: string): Promise<TenantNluModel> {
  const sid = String(id).trim()
  try {
    const res = await get<TenantNluModel>(`/nlu-models/${sid}`)
    if (res.data) return normalizeNluModel(res.data)
  } catch (e) {
    const rows = await listNluModels()
    const hit = rows.find((m) => m.id === sid)
    if (hit) return hit
    throw e
  }
  throw new Error('not found')
}

export async function createNluModel(body: {
  name: string
  description?: string
  spec?: TenantNluSpec
}) {
  const res = await post<TenantNluModel>('/nlu-models', body)
  return normalizeNluModel(res.data!)
}

export async function updateNluModel(
  id: string,
  body: { name?: string; description?: string; minConfidence?: number; spec?: TenantNluSpec }
) {
  const res = await put<TenantNluModel>(`/nlu-models/${String(id)}`, body)
  return normalizeNluModel(res.data!)
}

export async function deleteNluModel(id: string) {
  await del(`/nlu-models/${id}`)
}

export async function trainNluModel(id: string) {
  const res = await post<{ message?: string; model?: TenantNluModel }>(`/nlu-models/${id}/train`)
  return res.data!
}

export async function parseNluModel(id: string, text: string) {
  const res = await post<NluParseResult>(`/nlu-models/${id}/parse`, { text })
  return res.data!
}

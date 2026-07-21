import { del, get, post, put, type ApiResponse } from '@/utils/request'

export interface AIProviderPoolRow {
  id: number
  name: string
  modality: 'asr' | 'tts' | 'llm' | 'realtime'
  provider: string
  config: Record<string, unknown>
  voiceIds?: string[]
  priority?: number
  enabled?: boolean
  quotaLimit?: number
  quotaUsed?: number
  description?: string
}

export async function listAIProviderPools(modality?: string): Promise<ApiResponse<{ list: AIProviderPoolRow[] }>> {
  const q = modality ? `?modality=${encodeURIComponent(modality)}` : ''
  return get(`/platform/ai-pools${q}`)
}

export async function createAIProviderPool(body: {
  name?: string
  modality: string
  provider: string
  config: Record<string, unknown>
  voiceIds?: string[]
  priority?: number
  enabled?: boolean
  quotaLimit?: number
  description?: string
}): Promise<ApiResponse<AIProviderPoolRow>> {
  return post('/platform/ai-pools', body)
}

export async function updateAIProviderPool(
  id: number,
  body: Partial<{
    name: string
    modality: string
    provider: string
    config: Record<string, unknown>
    voiceIds: string[]
    priority: number
    enabled: boolean
    quotaLimit: number
    description: string
  }>,
): Promise<ApiResponse<AIProviderPoolRow>> {
  return put(`/platform/ai-pools/${id}`, body)
}

export async function deleteAIProviderPool(id: number): Promise<ApiResponse<{ id: number }>> {
  return del(`/platform/ai-pools/${id}`)
}

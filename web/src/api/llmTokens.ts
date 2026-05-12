import { get, post, put, del, ApiResponse } from '@/utils/request'

export interface LLMToken {
  id: number
  user_id: number
  name: string
  api_key: string
  type: 'llm' | 'asr' | 'tts'
  status: 'active' | 'disabled' | 'expired'
  group: string
  model_whitelist: string
  unlimited_quota: boolean
  token_quota: number
  token_used: number
  quota_used: number
  request_quota: number
  request_used: number
  expires_at?: string | null
  last_used_at?: string | null
  created_at: string
  updated_at: string
}

export interface ListLLMTokensResp {
  tokens: LLMToken[]
  total: number
  page: number
  pageSize: number
}

export interface MeLLMTokenWriteReq {
  name?: string
  type?: 'llm' | 'asr' | 'tts'
  group?: string
  model_whitelist?: string
}

export const listMyLLMTokens = (params?: { page?: number; pageSize?: number; status?: string }) =>
  get<ListLLMTokensResp>('/me/llm-tokens', { params })

export const createMyLLMToken = (body: MeLLMTokenWriteReq) =>
  post<{ token: LLMToken; raw_api_key: string }>('/me/llm-tokens', body)

export const updateMyLLMToken = (id: number, body: MeLLMTokenWriteReq) =>
  put<{ token: LLMToken }>(`/me/llm-tokens/${id}`, body)

export const regenerateMyLLMToken = (id: number) =>
  post<{ token: LLMToken; raw_api_key: string }>(`/me/llm-tokens/${id}/regenerate`)

export const deleteMyLLMToken = (id: number) => del<null>(`/me/llm-tokens/${id}`)

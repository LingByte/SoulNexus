import { get, ApiResponse } from '@/utils/request'

export interface LLMUsageRecord {
  id: string
  request_id: string
  session_id?: string
  user_id: string
  provider: string
  model: string
  base_url?: string
  request_type: string
  input_tokens: number
  output_tokens: number
  total_tokens: number
  quota_delta: number
  latency_ms?: number
  ttft_ms?: number
  tps?: number
  success: boolean
  status_code?: number
  error_code?: string
  error_message?: string
  channel_id?: number
  requested_at: string
  completed_at?: string
}

export interface LLMUsageListResp {
  items: LLMUsageRecord[]
  total: number
  page: number
  pageSize: number
}

export interface LLMUsageSummary {
  total_requests: number
  success_requests: number
  input_tokens: number
  output_tokens: number
  total_tokens: number
  quota_delta: number
}

export interface LLMUsageByModel {
  model: string
  request_count: number
  total_tokens: number
}

export interface LLMUsageSummaryResp {
  summary: LLMUsageSummary
  by_model: LLMUsageByModel[]
}

export const listMyLLMUsage = (params?: {
  page?: number
  pageSize?: number
  model?: string
  success?: string
  from?: string
  to?: string
  search?: string
}): Promise<ApiResponse<LLMUsageListResp>> =>
  get('/me/llm-usage', { params })

export const getMyLLMUsageSummary = (params?: {
  from?: string
  to?: string
}): Promise<ApiResponse<LLMUsageSummaryResp>> =>
  get('/me/llm-usage/summary', { params })

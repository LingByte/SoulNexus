import { del, get, post, put, type ApiResponse } from '@/utils/request'

import type { Paginated } from '@/api/types'

export interface TenantRow {
  // id 是后端 Snowflake (uint64, >2^53)，必须以字符串透传，否则 JavaScript
  // Number 会把末几位抹平（典型表现：…3258368 → …3258000），随后 GET/PUT
  // /tenants/:id 全部 404。后端 `tenantPublic` 已经把 ID 序列化为字符串。
  id: string
  name: string
  slug: string
  description?: string
  status?: string
  contactEmail?: string
  maxUserCount?: number
  createdAt?: string
}

/** 平台管理员 GET / PUT 租户详情时携带的 JSON（列表接口不返回） */
export interface TenantDetail extends TenantRow {
  asrConfig?: Record<string, unknown> | null
  ttsConfig?: Record<string, unknown> | null
  llmConfig?: Record<string, unknown> | null
  /** 'pipeline'（默认，三层 ASR→LLM→TTS）或 'realtime'（单条 WS 多模态） */
  voiceMode?: 'pipeline' | 'realtime' | null
  realtimeConfig?: Record<string, unknown> | null
  automationConfig?: Record<string, never>
}

export async function listTenants(
  page = 1,
  size = 100,
  opts?: { search?: string },
): Promise<ApiResponse<Paginated<TenantRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  if (opts?.search) q.set('search', opts.search)
  return get(`/tenants?${q.toString()}`)
}

export async function getTenant(id: string | number): Promise<ApiResponse<{ tenant: TenantDetail }>> {
  return get(`/tenants/${id}`)
}

export async function createTenantPlatform(body: {
  companyName: string
  adminEmail: string
  adminPassword: string
  adminDisplayName?: string
  tenantDescription?: string
  maxUserCount?: number
}): Promise<ApiResponse<{ tenant: TenantDetail; adminUser: Record<string, unknown>; roleId: number }>> {
  return post('/tenants', body)
}

export async function updateTenantPlatform(
  id: string | number,
  body: {
    name?: string
    description?: string
    status?: string
    contactEmail?: string
    maxUserCount?: number
    asrConfig?: Record<string, unknown> | null
    ttsConfig?: Record<string, unknown> | null
    llmConfig?: Record<string, unknown> | null
    voiceMode?: 'pipeline' | 'realtime' | null
    realtimeConfig?: Record<string, unknown> | null
    automationConfig?: Record<string, never>
    /** @deprecated use automationConfig */
    automationConfig?: Record<string, never>
  },
): Promise<ApiResponse<{ tenant: TenantDetail }>> {
  return put(`/tenants/${id}`, body)
}

export async function deleteTenantPlatform(id: string | number): Promise<ApiResponse<{ id: string }>> {
  return del(`/tenants/${id}`)
}

export interface TenantLLMTestResult {
  provider?: string
  model?: string
  prompt?: string
  reply?: string
  firstTokenMs?: number
  wallMs?: number
  ok?: boolean
}

/** Platform admin: streaming LLM probe; returns first-token latency. */
export async function testTenantLLMStream(
  id: string | number,
  body?: { prompt?: string; llmConfig?: Record<string, unknown> },
): Promise<ApiResponse<TenantLLMTestResult>> {
  return post(`/tenants/${id}/llm-test`, body ?? {})
}


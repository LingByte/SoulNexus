import { get, post, put, del, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'

export type CredentialStatus = 'active' | 'disabled'
export type CredentialId = string

export type CredentialKind = 'platform_bundle' | 'user_bundle'

export interface CredentialRow {
  id: CredentialId
  tenantId: string
  name: string
  kind?: CredentialKind
  /** True for platform_bundle (transit / 号池) keys. */
  usesTenantAi?: boolean
  /** Lookup prefix shown in lists (full key is never returned after create). */
  apiKeyPrefix: string
  /** @deprecated alias of apiKeyPrefix */
  accessKey?: string
  /** True for old AK/SK rows that no longer authenticate. */
  legacyHmac?: boolean
  status: CredentialStatus
  voiceMode?: string
  asrConfig?: Record<string, unknown> | null
  ttsConfig?: Record<string, unknown> | null
  llmConfig?: Record<string, unknown> | null
  realtimeConfig?: Record<string, unknown> | null
  hasAiConfig?: boolean
  expiresAt?: string | null
  lastUsedAt?: string | null
  requestCount?: number
  createdAt?: string
  updatedAt?: string
  createBy?: string
}

/** Full apiKey appears only on create / regenerate. */
export interface CredentialCreateResult extends CredentialRow {
  apiKey: string
  notice?: string
}

export interface CredentialListQuery {
  page?: number
  size?: number
  status?: CredentialStatus
  name?: string
}

export interface CredentialCreateBody {
  name?: string
  kind?: CredentialKind
  expiresAt?: string | null
  voiceMode?: string
  asrConfig?: Record<string, unknown>
  ttsConfig?: Record<string, unknown>
  llmConfig?: Record<string, unknown>
  realtimeConfig?: Record<string, unknown>
}

export interface CredentialUpdateBody {
  name?: string
  expiresAt?: string | null
  voiceMode?: string
  asrConfig?: Record<string, unknown>
  ttsConfig?: Record<string, unknown>
  llmConfig?: Record<string, unknown>
  realtimeConfig?: Record<string, unknown>
}

export async function listCredentials(query: CredentialListQuery = {}): Promise<ApiResponse<Paginated<CredentialRow>>> {
  const params = new URLSearchParams({
    page: String(query.page ?? 1),
    size: String(query.size ?? 20),
  })
  if (query.status) params.set('status', query.status)
  if (query.name) params.set('name', query.name)
  return get(`/credentials?${params.toString()}`)
}

export async function createCredential(body: CredentialCreateBody): Promise<ApiResponse<CredentialCreateResult>> {
  return post('/credentials', body)
}

function idPath(id: CredentialId | number): string {
  return String(id)
}

export async function updateCredential(id: CredentialId | number, body: CredentialUpdateBody): Promise<ApiResponse<{ id: CredentialId }>> {
  return put(`/credentials/${idPath(id)}`, body)
}

export async function regenerateCredential(id: CredentialId | number): Promise<ApiResponse<CredentialCreateResult>> {
  return post(`/credentials/${idPath(id)}/regenerate`, {})
}

export async function disableCredential(id: CredentialId | number): Promise<ApiResponse<{ id: CredentialId; status: CredentialStatus }>> {
  return post(`/credentials/${idPath(id)}/disable`, {})
}

export async function enableCredential(id: CredentialId | number): Promise<ApiResponse<{ id: CredentialId; status: CredentialStatus }>> {
  return post(`/credentials/${idPath(id)}/enable`, {})
}

export async function deleteCredential(id: CredentialId | number): Promise<ApiResponse<{ id: CredentialId }>> {
  return del(`/credentials/${idPath(id)}`)
}

export interface CredentialLLMTestResult {
  provider: string
  model: string
  prompt: string
  reply: string
  firstTokenMs?: number
  wallMs?: number
  ok?: boolean
}

export async function testCredentialLLMStream(body: {
  prompt?: string
  credentialId?: string
  llmConfig?: Record<string, unknown>
}): Promise<ApiResponse<CredentialLLMTestResult>> {
  return post('/credentials/llm-test', body)
}

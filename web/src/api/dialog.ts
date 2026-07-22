import { get, post, put, del, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'
import { useAuthStore } from '@/stores/authStore'
import { getApiBaseURL } from '@/config/apiConfig'

const PREFIX = '/lingecho/dialog/v1'

export interface DialogConversation {
  id: string
  assistantId: string
  channel: string
  status: string
  welcomeText?: string
}

export interface DialogMessage {
  id: string
  role: string
  content: string
  latencyMs?: number
  createdAt?: string
  toolsJson?: string
  confidence?: number
  confidenceJson?: string
}

export interface TenantDialogChannelRow {
  id: number
  tenantId: number
  provider: string
  code: string
  name: string
  assistantId: string
  enabled: boolean
  remark?: string
  config?: Record<string, unknown>
}

export interface DialogTurnResult {
  reply: string
  latencyMs: number
  confidence?: number
  toolsJson?: string
}

export async function createDialogConversation(body: {
  assistantId: string
  channel?: string
  externalUserId?: string
  credentialId?: string
}): Promise<ApiResponse<DialogConversation>> {
  return post(`${PREFIX}/conversations`, body)
}

export async function postDialogMessage(
  conversationId: string,
  text: string,
): Promise<ApiResponse<DialogTurnResult>> {
  return post(`${PREFIX}/conversations/${conversationId}/messages`, { text })
}

/** POST messages with SSE (?stream=1). Yields stage/delta events and a final completed payload. */
export async function postDialogMessageStream(
  conversationId: string,
  text: string,
  handlers: {
    onStage?: (stage: string) => void
    onDelta?: (delta: string) => void
    onCompleted?: (result: DialogTurnResult) => void
    onError?: (message: string) => void
    signal?: AbortSignal
  } = {},
): Promise<DialogTurnResult> {
  const token = useAuthStore.getState().token
  const base = getApiBaseURL().replace(/\/$/, '')
  const res = await fetch(`${base}${PREFIX}/conversations/${conversationId}/messages?stream=1`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({ text, stream: true }),
    signal: handlers.signal,
  })
  if (!res.ok || !res.body) {
    const msg = `stream failed: ${res.status}`
    handlers.onError?.(msg)
    throw new Error(msg)
  }
  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let completed: DialogTurnResult | null = null
  let eventName = ''

  const flushBlock = (block: string) => {
    const lines = block.split('\n')
    let data = ''
    for (const line of lines) {
      if (line.startsWith('event:')) {
        eventName = line.slice(6).trim()
      } else if (line.startsWith('data:')) {
        data += line.slice(5).trim()
      }
    }
    if (!data) return
    try {
      const parsed = JSON.parse(data) as Record<string, unknown>
      if (eventName === 'message.stage' && typeof parsed.stage === 'string') {
        handlers.onStage?.(parsed.stage)
      } else if (eventName === 'message.delta' && typeof parsed.delta === 'string') {
        handlers.onDelta?.(parsed.delta)
      } else if (eventName === 'message.completed') {
        completed = {
          reply: String(parsed.reply ?? ''),
          latencyMs: Number(parsed.latencyMs ?? 0),
          confidence: typeof parsed.confidence === 'number' ? parsed.confidence : undefined,
          toolsJson: typeof parsed.toolsJson === 'string' ? parsed.toolsJson : undefined,
        }
        handlers.onCompleted?.(completed)
      } else if (eventName === 'error') {
        handlers.onError?.(String(parsed.message ?? 'stream error'))
      }
    } catch {
      /* ignore malformed SSE chunk */
    }
    eventName = ''
  }

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const parts = buffer.split('\n\n')
    buffer = parts.pop() ?? ''
    for (const part of parts) {
      flushBlock(part)
    }
  }
  if (buffer.trim()) {
    flushBlock(buffer)
  }
  if (!completed) {
    throw new Error('stream ended without message.completed')
  }
  return completed
}

export async function listDialogMessages(
  conversationId: string,
): Promise<ApiResponse<{ messages: DialogMessage[] }>> {
  return get(`${PREFIX}/conversations/${conversationId}/messages`)
}

export async function endDialogConversation(conversationId: string): Promise<ApiResponse<{ ok: boolean }>> {
  return post(`${PREFIX}/conversations/${conversationId}/end`, {})
}

export async function listDialogChannels(page = 1, size = 20): Promise<ApiResponse<Paginated<TenantDialogChannelRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  return get(`${PREFIX}/channels?${q.toString()}`)
}

export async function listDialogChannelProviders(): Promise<ApiResponse<{ providers: string[] }>> {
  return get(`${PREFIX}/channels/providers`)
}

export async function createDialogChannel(body: {
  provider: string
  code?: string
  name: string
  assistantId: string
  remark?: string
  enabled?: boolean
  config: Record<string, unknown>
}): Promise<ApiResponse<TenantDialogChannelRow>> {
  return post(`${PREFIX}/channels`, body)
}

export async function updateDialogChannel(
  id: number,
  body: {
    name?: string
    assistantId?: string
    remark?: string
    enabled?: boolean
    config?: Record<string, unknown>
  },
): Promise<ApiResponse<TenantDialogChannelRow>> {
  return put(`${PREFIX}/channels/${id}`, body)
}

export async function deleteDialogChannel(id: number): Promise<ApiResponse<{ ok: boolean }>> {
  return del(`${PREFIX}/channels/${id}`)
}

export interface TenantDialogSkillRow {
  id: number
  tenantId: number
  code: string
  name: string
  description?: string
  kind?: 'prompt' | 'python' | 'node' | string
  body: string
  scriptContent?: string
  entryFile?: string
  hasAssets?: boolean
  enabled: boolean
  toolName?: string
  createdAt?: string
  updatedAt?: string
}

export async function listDialogSkills(opts?: {
  enabled?: boolean
}): Promise<ApiResponse<TenantDialogSkillRow[]>> {
  const q = new URLSearchParams()
  if (opts?.enabled) q.set('enabled', 'true')
  const suffix = q.toString() ? `?${q.toString()}` : ''
  return get(`${PREFIX}/skills${suffix}`)
}

export async function listDialogSkillsPage(
  page = 1,
  size = 20,
): Promise<ApiResponse<Paginated<TenantDialogSkillRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  return get(`${PREFIX}/skills?${q.toString()}`)
}

export async function createDialogSkill(body: {
  code?: string
  name: string
  description?: string
  kind?: string
  body?: string
  scriptContent?: string
  entryFile?: string
  enabled?: boolean
}): Promise<ApiResponse<TenantDialogSkillRow>> {
  return post(`${PREFIX}/skills`, body)
}

export async function updateDialogSkill(
  id: number,
  body: {
    code?: string
    name?: string
    description?: string
    kind?: string
    body?: string
    scriptContent?: string
    entryFile?: string
    enabled?: boolean
  },
): Promise<ApiResponse<TenantDialogSkillRow>> {
  return put(`${PREFIX}/skills/${id}`, body)
}

export async function deleteDialogSkill(id: number): Promise<ApiResponse<{ ok: boolean }>> {
  return del(`${PREFIX}/skills/${id}`)
}

export async function uploadDialogSkillAssets(
  id: number,
  file: File,
): Promise<ApiResponse<TenantDialogSkillRow>> {
  const form = new FormData()
  form.append('file', file)
  return post(`${PREFIX}/skills/${id}/assets`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

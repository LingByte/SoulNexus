import { get, post, put, del, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'

export interface AssistantRow {
  id: string
  tenantId?: string
  name: string
  avatar?: string
  avatarUrl?: string
  scene?: string
  version?: string
  description?: string
  enabled: boolean
  welcome?: string
  prompt?: string
  knowledgeNamespace?: string
  nluModelId?: string
  asrConfig?: unknown
  ttsConfig?: unknown
  llmConfig?: unknown
  realtimeConfig?: unknown
  vadConfig?: unknown
  agentConfig?: unknown
  hotWords?: unknown
  interruptionConfig?: unknown
  audioTrackConfig?: unknown
  audioProcessConfig?: unknown
  queryRewriter?: unknown
  mcpServers?: unknown
  ttsVoice?: string
  realtimeVoice?: string
  voiceDialogWsUrl?: string
  boundJsTemplateSourceId?: string
  collect?: unknown
  publishedVersionId?: string
  createdAt?: string
  updatedAt?: string
}

export interface AssistantVersionRow {
  id: string
  assistantId?: string
  scene?: string
  version?: string
  versionNo?: number
  description?: string
  welcome?: string
  prompt?: string
  knowledgeNamespace?: string
  asrConfig?: unknown
  ttsConfig?: unknown
  llmConfig?: unknown
  realtimeConfig?: unknown
  vadConfig?: unknown
  agentConfig?: unknown
  hotWords?: unknown
  interruptionConfig?: unknown
  audioTrackConfig?: unknown
  audioProcessConfig?: unknown
  queryRewriter?: unknown
  mcpServers?: unknown
  ttsVoice?: string
  realtimeVoice?: string
  voiceDialogWsUrl?: string
  collect?: unknown
  publishedAt?: string
  publishedBy?: string
}

export async function listAssistants(
  page = 1,
  size = 20,
  opts?: { scene?: string; name?: string; tenantId?: string },
): Promise<ApiResponse<Paginated<AssistantRow>>> {
  const q = new URLSearchParams({ page: String(page), size: String(size) })
  if (opts?.scene) q.set('scene', opts.scene)
  if (opts?.name) q.set('name', opts.name)
  if (opts?.tenantId && opts.tenantId !== '0') q.set('tenantId', opts.tenantId)
  return get(`/assistants?${q.toString()}`)
}

export async function getAssistant(id: string): Promise<ApiResponse<AssistantRow>> {
  return get(`/assistants/${id}`)
}

export async function createAssistant(body: Record<string, unknown>): Promise<ApiResponse<AssistantRow>> {
  return post('/assistants', body)
}

export async function updateAssistant(id: string, body: Record<string, unknown>): Promise<ApiResponse<AssistantRow>> {
  return put(`/assistants/${id}`, body)
}

export async function deleteAssistant(id: string): Promise<ApiResponse<{ id: string }>> {
  return del(`/assistants/${id}`)
}

export async function publishAssistant(id: string): Promise<ApiResponse<{ version: AssistantVersionRow }>> {
  return post(`/assistants/${id}/publish`, {})
}

export async function rollbackAssistant(id: string, versionId: string): Promise<ApiResponse<AssistantRow>> {
  return post(`/assistants/${id}/rollback`, { versionId })
}

export async function listAssistantVersions(id: string): Promise<ApiResponse<AssistantVersionRow[]>> {
  return get(`/assistants/${id}/versions`)
}

export type AssistantDiffResult = {
  changedKeys: string[]
  from?: unknown
  to?: unknown
  fromSnapshot?: unknown
  toSnapshot?: unknown
}

/** Omit from/to → draft vs published. Pass version ids for version↔version. */
export async function diffAssistantVersions(
  id: string,
  opts?: { from?: string; to?: string },
): Promise<ApiResponse<AssistantDiffResult>> {
  const q = new URLSearchParams()
  if (opts?.from) q.set('from', opts.from)
  if (opts?.to) q.set('to', opts.to)
  const qs = q.toString()
  return get(`/assistants/${id}/diff${qs ? `?${qs}` : ''}`)
}

export async function importAssistantFromTenant(body: {
  name?: string
  tenantId?: string
}): Promise<ApiResponse<AssistantRow>> {
  return post('/assistants/import-from-tenant', body)
}

export type AssistantMemberRow = {
  userId: string
  role?: string
  email?: string
  name?: string
}

export async function listAssistantMembers(id: string): Promise<ApiResponse<AssistantMemberRow[]>> {
  return get(`/assistants/${id}/members`)
}

export async function updateAssistantMembers(id: string, userIds: string[]): Promise<ApiResponse<{ count: number }>> {
  return put(`/assistants/${id}/members`, { userIds })
}

export async function addAssistantMembers(
  id: string,
  userIds: string[],
): Promise<ApiResponse<{ count: number }>> {
  return post(`/assistants/${id}/members`, { userIds })
}

export async function removeAssistantMember(id: string, userId: string): Promise<ApiResponse<null>> {
  return del(`/assistants/${id}/members/${userId}`)
}

export async function patchAssistantSettings(
  id: string,
  body: { name?: string; description?: string },
): Promise<ApiResponse<AssistantRow>> {
  return put(`/assistants/${id}/settings`, body)
}

export async function uploadAssistantAvatar(
  id: string,
  file: File,
): Promise<ApiResponse<{ avatarUrl?: string; assistant?: AssistantRow }>> {
  const form = new FormData()
  form.append('file', file)
  return post(`/assistants/${id}/avatar`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

export const ASSISTANT_SCENES = [
  { value: 'general', label: '通用' },
  { value: 'inbound_knowledge', label: '对话 · 知识库问答' },
  { value: 'outbound_collect', label: '通知 · 信息收集' },
  { value: 'outbound_notify', label: '通知 · 消息推送' },
] as const

export const ASSISTANT_TEMPLATE_LABELS: Record<string, string> = {
  blank: '空白模板',
  customer_cleaning: '客户清洗',
  waking_up: '沉睡客户唤醒',
  inbound_knowledge: '知识库问答',
  outbound_notify: '消息通知',
  outbound_collect: '问卷调查',
}

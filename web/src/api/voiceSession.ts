import { del, post, type ApiResponse } from '@/utils/request'
import { buildWebSocketURL, getApiMountPath } from '@/config/apiConfig'

export type VoiceSessionTransport = 'websocket' | 'webrtc'

export interface VoiceSessionInfo {
  sessionId: string
  tenantId: number
  assistantId?: number
  transport: VoiceSessionTransport
  sampleRateHz?: number
  webSocketPath?: string
  webRtcOfferPath?: string
  dialogMode?: string
  voiceDialogWsUrl?: string
  createdAt?: string
}

export async function createVoiceSession(body: {
  transport: VoiceSessionTransport
  assistantId?: string
  credentialId?: string
  sampleRateHz?: number
  dialogMode?: 'engine' | 'gateway'
  temperature?: number
}): Promise<ApiResponse<VoiceSessionInfo>> {
  const payload: Record<string, unknown> = {
    transport: body.transport,
    sampleRateHz: body.sampleRateHz ?? 16000,
  }
  if (body.assistantId?.trim()) {
    payload.assistantId = body.assistantId.trim()
  }
  if (body.credentialId?.trim()) {
    payload.credentialId = body.credentialId.trim()
  }
  if (body.dialogMode) {
    payload.dialogMode = body.dialogMode
  }
  if (body.temperature != null && body.temperature > 0) {
    payload.temperature = body.temperature
  }
  return post('/lingecho/voice-session/v1/sessions', payload)
}

export async function endVoiceSession(sessionId: string): Promise<ApiResponse<null>> {
  return del(`/lingecho/voice-session/v1/sessions/${encodeURIComponent(sessionId)}`)
}

export function buildVoiceSessionWebSocketURL(sessionId: string, token?: string): string {
  const mount = getApiMountPath()
  const path = `${mount}/lingecho/voice-session/v1/ws`.replace(/\/+/g, '/')
  let url = buildWebSocketURL(path)
  const u = new URL(url)
  u.searchParams.set('session_id', sessionId)
  if (token) u.searchParams.set('token', token)
  return u.toString()
}

export async function postVoiceSessionWebRTCOffer(body: {
  sessionId: string
  sdp: string
  type: 'offer'
}): Promise<{ sessionId: string; sdp: string; type: string }> {
  const res = await post<{ sessionId: string; sdp: string; type: string }>(
    '/lingecho/voice-session/v1/webrtc/offer',
    body,
  )
  if (res.code !== 0 && res.code !== 200) {
    throw new Error(res.msg || 'WebRTC offer failed')
  }
  const data = res.data
  if (!data?.sdp) {
    throw new Error(res.msg || 'WebRTC offer failed')
  }
  return data
}

export interface VoiceSessionKnowledgeHit {
  title?: string
  content?: string
  source?: string
  score?: number
  quoted?: boolean
}

export interface VoiceSessionKnowledgeRetrieval {
  query?: string
  searchQuery?: string
  strategy?: string
  recallMs?: number
  embedMs?: number
  qdrantMs?: number
  hitCount?: number
  hits?: VoiceSessionKnowledgeHit[]
}

export interface VoiceSessionTurnMetrics {
  type?: string
  v?: number
  sessionId?: string
  turnId?: string
  userText?: string
  assistantText?: string
  llmFirstMs?: number
  llmWallMs?: number
  ttsFirstMs?: number
  pipelineMs?: number
  e2eFirstMs?: number
  mode?: string
  transport?: string
  completedAt?: string
  knowledgeRetrievals?: VoiceSessionKnowledgeRetrieval[]
}

export type VoiceSessionWireFrame = {
  type: string
  v?: number
  sessionId?: string
  turnId?: string
  sampleRateHz?: number
  data?: string
  message?: string
  text?: string
  final?: boolean
  state?: string
  llmFirstMs?: number
  llmWallMs?: number
  ttsFirstMs?: number
  pipelineMs?: number
  e2eFirstMs?: number
  mode?: string
  transport?: string
  completedAt?: string
  knowledgeRetrievals?: VoiceSessionKnowledgeRetrieval[]
}

export function encodePCMFrame(sessionId: string, sampleRateHz: number, pcm: Uint8Array): string {
  let binary = ''
  for (let i = 0; i < pcm.length; i++) binary += String.fromCharCode(pcm[i])
  const data = btoa(binary)
  return JSON.stringify({
    type: 'pcm',
    v: 1,
    sessionId,
    sampleRateHz,
    data,
  })
}

/** Raw PCM16LE mono for WebSocket binary uplink (preferred). */
export function pcm16ToBytes(pcm: Int16Array): Uint8Array {
  const out = new Uint8Array(pcm.length * 2)
  const view = new DataView(out.buffer)
  for (let i = 0; i < pcm.length; i++) {
    view.setInt16(i * 2, pcm[i], true)
  }
  return out
}

/** Decode server binary PCM downlink into Int16Array. */
export function decodeBinaryPCM(data: ArrayBuffer): Int16Array {
  const bytes = new Uint8Array(data)
  const aligned = bytes.byteOffset % 2 === 0 ? bytes : bytes.slice()
  return new Int16Array(aligned.buffer, aligned.byteOffset, aligned.byteLength / 2)
}

export function decodeBase64PCM(data: string): Uint8Array {
  const binary = atob(data)
  const out = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) out[i] = binary.charCodeAt(i)
  return out
}

export async function getAuthToken(): Promise<string | undefined> {
  try {
    const { useAuthStore } = await import('@/stores/authStore')
    return useAuthStore.getState().token || undefined
  } catch {
    return undefined
  }
}

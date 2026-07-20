import { del, get, post, put, type ApiResponse } from '@/utils/request'

export interface VoiceprintConfig {
  provider: string
  label?: string
  enabled: boolean
  supportsEnroll?: boolean
  supportsIdentify?: boolean
  supportsVerify?: boolean
  groupScoped?: boolean
  similarityThreshold?: number
  maxCandidates?: number
}

export interface VoiceprintProfile {
  /** Snowflake id — always treat as string to avoid JS Number precision loss. */
  id: string | number
  tenantId?: string | number
  assistantId?: string | number | null
  subjectId?: string | number | null
  scene?: string
  name: string
  provider: string
  featureId: string
  status: string
  description?: string
  createdAt?: string
  updatedAt?: string
}

/** Normalize API snowflake fields to decimal string (never Number()). */
export function snowflakeStr(v: string | number | null | undefined): string {
  if (v == null) return ''
  const s = String(v).trim()
  if (!s || s === '0' || s === 'null' || s === 'undefined') return ''
  return s
}

export function isSnowflakeSet(v: string | number | null | undefined): boolean {
  return snowflakeStr(v) !== ''
}

export interface SpeakerAttribute {
  key: string
  value: string
  visibility: 'llm' | 'internal' | 'tool' | string
}

export interface SpeakerCredentialView {
  provider: string
  scopes?: string
  hasSecret: boolean
}

export interface VoiceprintSpeakerBundle {
  profileId: string | number
  featureId: string
  name: string
  subjectId?: string | number | null
  subject?: { id: string | number; displayName: string; notes?: string; status?: string } | null
  attributes: SpeakerAttribute[]
  credentials: SpeakerCredentialView[]
}

export interface VoiceprintIdentifyResult {
  featureId: string
  score: number
  threshold: number
  isMatch: boolean
  confidence?: string
}

export interface VoiceprintSelfTestCheck {
  name: string
  ok: boolean
  detail?: string
}

export interface VoiceprintSelfTestReport {
  enabled: boolean
  provider?: string
  label?: string
  ok: boolean
  checks: VoiceprintSelfTestCheck[]
}

export async function getVoiceprintConfig(): Promise<ApiResponse<VoiceprintConfig>> {
  return get('/voiceprints/config')
}

export async function runVoiceprintSelfTest(probe = false): Promise<ApiResponse<VoiceprintSelfTestReport>> {
  const q = probe ? '?probe=true' : ''
  return get(`/voiceprints/self-test${q}`)
}

export async function listVoiceprints(): Promise<ApiResponse<VoiceprintProfile[]>> {
  return get('/voiceprints')
}

export async function createVoiceprint(
  name: string,
  audio: File,
  opts?: { description?: string; featureId?: string },
): Promise<ApiResponse<VoiceprintProfile>> {
  const fd = new FormData()
  fd.append('name', name)
  fd.append('audio', audio)
  if (opts?.description?.trim()) fd.append('description', opts.description.trim())
  if (opts?.featureId?.trim()) fd.append('featureId', opts.featureId.trim())
  return post('/voiceprints', fd)
}

export async function identifyVoiceprint(
  audio: File,
  opts?: { threshold?: number; featureIds?: string[] },
): Promise<ApiResponse<{ result: VoiceprintIdentifyResult; profile: VoiceprintProfile | null }>> {
  const fd = new FormData()
  fd.append('audio', audio)
  if (opts?.threshold != null && Number.isFinite(opts.threshold)) {
    fd.append('threshold', String(opts.threshold))
  }
  if (opts?.featureIds?.length) {
    fd.append('featureIds', opts.featureIds.join(','))
  }
  return post('/voiceprints/identify', fd)
}

export async function deleteVoiceprint(id: string | number): Promise<ApiResponse<{ deleted: boolean }>> {
  return del(`/voiceprints/${encodeURIComponent(String(id))}`)
}

export async function bindVoiceprintAssistant(
  id: string | number,
  assistantId: string | number | null,
): Promise<ApiResponse<VoiceprintProfile>> {
  return put(`/voiceprints/${encodeURIComponent(String(id))}/assistant`, {
    assistantId: assistantId == null || assistantId === '' ? null : String(assistantId),
  })
}

export async function getVoiceprintSpeaker(id: string | number): Promise<ApiResponse<VoiceprintSpeakerBundle>> {
  return get(`/voiceprints/${encodeURIComponent(String(id))}/speaker`)
}

export async function upsertVoiceprintSpeaker(
  id: string | number,
  body: {
    displayName?: string
    notes?: string
    attributes?: SpeakerAttribute[]
    credentials?: { provider: string; secretRef?: string; scopes?: string; clear?: boolean }[]
  },
): Promise<ApiResponse<VoiceprintSpeakerBundle>> {
  return put(`/voiceprints/${encodeURIComponent(String(id))}/speaker`, body)
}

export const VoiceprintSceneBusiness = 'business'
export const VoiceprintSceneAccount = 'account'

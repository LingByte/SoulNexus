import { del, get, post, type ApiResponse } from '@/utils/request'

export type VoiceCloneStatus = 'pending' | 'training' | 'success' | 'failed'

export interface VoiceCloneConfig {
  provider: string
  label: string
  supportsCreateTask?: boolean
  supportsTrainingTexts?: boolean
  speakerField?: string
  ttsProvider?: string
  hint?: string
}

export interface VoiceCloneProfile {
  id: number
  tenantId: number
  name: string
  provider: string
  status: VoiceCloneStatus
  taskId?: string
  assetId?: string
  speakerId?: string
  sex?: number
  language?: string
  trainText?: string
  textId?: number
  textSegId?: number
  failedReason?: string
  progress?: number
  createdAt?: string
  updatedAt?: string
}

export interface TrainingTextSegment {
  seg_id: number | string
  seg_text: string
}

export interface TrainingTextResponse {
  text_id: number
  text_name: string
  segments: TrainingTextSegment[]
}

export async function getVoiceCloneConfig(): Promise<ApiResponse<VoiceCloneConfig>> {
  return get('/voice-clones/config')
}

export async function listVoiceClones(status?: string): Promise<ApiResponse<VoiceCloneProfile[]>> {
  const q = status?.trim() ? `?status=${encodeURIComponent(status.trim())}` : ''
  return get(`/voice-clones${q}`)
}

export async function getVoiceCloneTrainingTexts(textId = 1): Promise<ApiResponse<TrainingTextResponse>> {
  return get(`/voice-clones/training-texts?textId=${textId}`)
}

export async function createVoiceClone(body: {
  name: string
  speakerId?: string
  sex?: number
  ageGroup?: number
  language?: string
  trainText?: string
  textId?: number
  textSegId?: number
}): Promise<ApiResponse<VoiceCloneProfile>> {
  return post('/voice-clones', body)
}

export async function submitVoiceCloneAudio(id: number, file: File): Promise<ApiResponse<VoiceCloneProfile>> {
  const fd = new FormData()
  fd.append('file', file)
  return post(`/voice-clones/${id}/audio`, fd, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

export async function syncVoiceCloneStatus(id: number): Promise<ApiResponse<VoiceCloneProfile>> {
  return post(`/voice-clones/${id}/sync`, {})
}

export async function previewVoiceClone(
  id: number,
  text?: string,
): Promise<
  ApiResponse<{
    pcmBase64?: string
    audioUrl?: string
    sampleRate: number
    format: string
    assetId: string
    cached?: boolean
  }>
> {
  return post(`/voice-clones/${id}/preview`, { text })
}

export async function deleteVoiceClone(id: number): Promise<ApiResponse<null>> {
  return del(`/voice-clones/${id}`)
}

export interface VoiceSynthesisHistory {
  id: string
  tenantId: number
  profileId: number
  provider: string
  assetId: string
  voiceName: string
  text: string
  sampleRate?: number
  audioUrl?: string
  status: 'success' | 'failed'
  errorMessage?: string
  createdAt?: string
}

export async function synthesizeVoiceClone(
  id: number,
  text: string,
): Promise<ApiResponse<{ pcmBase64: string; sampleRate: number; audioUrl?: string; text: string }>> {
  return post(`/voice-clones/${id}/synthesize`, { text })
}

export async function listVoiceSynthesisHistory(
  profileId?: number,
): Promise<ApiResponse<VoiceSynthesisHistory[]>> {
  const q = profileId ? `?profileId=${profileId}` : ''
  return get(`/voice-synthesis-history${q}`)
}

export async function deleteVoiceSynthesisHistory(id: string): Promise<ApiResponse<null>> {
  return del(`/voice-synthesis-history/${id}`)
}

import { get, post, type ApiResponse } from '@/utils/request'

export interface VoiceOption {
  id: string
  name?: string
  label: string
  gender?: string
  locale?: string
  category?: string
  previewUrl?: string
}

export interface VoiceCatalogResponse {
  provider: string
  mode: string
  voiceField?: string
  docUrl?: string
  voices: VoiceOption[]
  cached?: boolean
}

type catalogCacheEntry = {
  at: number
  res: ApiResponse<VoiceCatalogResponse>
}

const CATALOG_TTL_MS = 5 * 60 * 1000
const catalogCache = new Map<string, catalogCacheEntry>()
const catalogInflight = new Map<string, Promise<ApiResponse<VoiceCatalogResponse>>>()

export async function listVoices(
  provider: string,
  mode: 'tts' | 'realtime' = 'tts',
): Promise<ApiResponse<VoiceCatalogResponse>> {
  const key = `${provider.trim().toLowerCase()}|${mode}`
  const hit = catalogCache.get(key)
  if (hit && Date.now() - hit.at < CATALOG_TTL_MS) {
    return hit.res
  }
  const pending = catalogInflight.get(key)
  if (pending) return pending

  const q = new URLSearchParams({ provider, mode })
  const req = get(`/voices?${q.toString()}`)
    .then((res) => {
      if (res.code === 200 && res.data) {
        catalogCache.set(key, { at: Date.now(), res })
      }
      catalogInflight.delete(key)
      return res
    })
    .catch((err) => {
      catalogInflight.delete(key)
      throw err
    })
  catalogInflight.set(key, req)
  return req
}

/** Drop client-side catalog cache (e.g. after switching tenant TTS provider). */
export function clearVoiceCatalogCache() {
  catalogCache.clear()
  catalogInflight.clear()
}

export interface TenantVoiceProviders {
  voiceMode: 'pipeline' | 'realtime'
  ttsProvider: string
  realtimeProvider: string
  source?: 'tenant' | 'pool'
  ttsVoiceIds?: string[]
  realtimeVoiceIds?: string[]
  hasPoolGrant?: boolean
}

export async function getTenantVoiceProviders(
  tenantId?: string,
): Promise<ApiResponse<TenantVoiceProviders>> {
  const q = tenantId?.trim() ? `?tenantId=${encodeURIComponent(tenantId.trim())}` : ''
  return get(`/tenant-voice-providers${q}`)
}

export interface VoicePreviewResponse {
  pcmBase64?: string
  audioUrl?: string
  sampleRate: number
  format: string
  cached?: boolean
}

export async function previewVoice(body: {
  provider: string
  mode: 'tts' | 'realtime'
  voiceId: string
  text?: string
  tenantId?: string
  credentialId?: string
}): Promise<ApiResponse<VoicePreviewResponse>> {
  return post('/voices/preview', body)
}

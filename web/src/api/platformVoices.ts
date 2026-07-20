import { get, del, type ApiResponse } from '@/utils/request'
import type { VoiceCloneProfile, VoiceSynthesisHistory } from '@/api/voiceClone'
import type { VoiceCatalogResponse } from '@/api/voices'
import type { VoiceprintProfile, VoiceprintSelfTestReport } from '@/api/voiceprint'

export async function listPlatformVoiceCatalog(
  provider: string,
  mode = 'tts',
): Promise<ApiResponse<VoiceCatalogResponse>> {
  return get(`/platform/voices/catalog?provider=${encodeURIComponent(provider)}&mode=${encodeURIComponent(mode)}`)
}

export async function listPlatformVoiceCloneProfiles(params?: {
  page?: number
  pageSize?: number
  tenantId?: string
}): Promise<
  ApiResponse<{
    list: VoiceCloneProfile[]
    total: number
    page: number
    pageSize: number
    cloneEnabled: boolean
    cloneProvider: string
    cloneProviderLabel: string
  }>
> {
  const q = new URLSearchParams()
  if (params?.page) q.set('page', String(params.page))
  if (params?.pageSize) q.set('pageSize', String(params.pageSize))
  if (params?.tenantId) q.set('tenantId', params.tenantId)
  const suffix = q.toString() ? `?${q}` : ''
  return get(`/platform/voices/clone-profiles${suffix}`)
}

export async function listPlatformVoiceSynthesisHistory(params?: {
  page?: number
  pageSize?: number
  tenantId?: string
}): Promise<ApiResponse<{ list: VoiceSynthesisHistory[]; total: number; page: number; pageSize: number }>> {
  const q = new URLSearchParams()
  if (params?.page) q.set('page', String(params.page))
  if (params?.pageSize) q.set('pageSize', String(params.pageSize))
  if (params?.tenantId) q.set('tenantId', params.tenantId)
  const suffix = q.toString() ? `?${q}` : ''
  return get(`/platform/voices/synthesis-history${suffix}`)
}

export async function deletePlatformVoiceSynthesisHistory(id: string): Promise<ApiResponse<null>> {
  return del(`/platform/voices/synthesis-history/${id}`)
}

export async function listPlatformVoiceprintProfiles(params?: {
  page?: number
  pageSize?: number
  tenantId?: string
}): Promise<
  ApiResponse<{
    list: VoiceprintProfile[]
    total: number
    page: number
    pageSize: number
    voiceprintEnabled: boolean
    voiceprintProvider: string
    voiceprintProviderLabel: string
  }>
> {
  const q = new URLSearchParams()
  if (params?.page) q.set('page', String(params.page))
  if (params?.pageSize) q.set('pageSize', String(params.pageSize))
  if (params?.tenantId) q.set('tenantId', params.tenantId)
  const suffix = q.toString() ? `?${q}` : ''
  return get(`/platform/voices/voiceprint-profiles${suffix}`)
}

export async function runPlatformVoiceprintSelfTest(
  probe = false,
): Promise<ApiResponse<VoiceprintSelfTestReport>> {
  const q = probe ? '?probe=true' : ''
  return get(`/platform/voices/voiceprint/self-test${q}`)
}

export async function deletePlatformVoiceprintProfile(id: number): Promise<ApiResponse<{ deleted: boolean }>> {
  return del(`/platform/voices/voiceprint-profiles/${id}`)
}

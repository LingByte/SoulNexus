import { post, ApiResponse } from '@/utils/request'
import { getApiBaseURL } from '@/config/apiConfig'

function authToken(): string {
  return localStorage.getItem('auth_token') || localStorage.getItem('token') || ''
}

export interface PetPackageValidationIssue {
  field: string
  message: string
  level: 'error' | 'warn'
}

export interface PetPackageValidateResult {
  kind: string
  valid: boolean
  issues: PetPackageValidationIssue[]
  entry?: string
  files?: number
}

export interface PetAiManifestResult {
  manifest: string
  explanation?: string
}

export interface PetPackageImportResult {
  kind: string
  jsSourceId?: string
  template: {
    id: string
    jsSourceId: string
    name: string
  }
  storage: string
  prefix?: string
  entry: string
}

export const petPackageService = {
  async validate(files: Record<string, string>, entry = 'pet.js'): Promise<ApiResponse<PetPackageValidateResult>> {
    return post('/pet-packages/validate', { files, entry })
  },

  async aiAssistManifest(body: {
    instruction: string
    manifest: string
    kind?: string
    assetFiles?: string[]
  }): Promise<ApiResponse<PetAiManifestResult>> {
    return post('/pet-packages/ai/manifest', body)
  },

  async importZip(file: File, name?: string): Promise<ApiResponse<PetPackageImportResult>> {
    const form = new FormData()
    form.append('package', file)
    if (name?.trim()) form.append('name', name.trim())
    const token = authToken()
    const res = await fetch(`${getApiBaseURL()}/pet-packages/import`, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: form,
      credentials: 'include',
    })
    return res.json()
  },

  exportZipUrl(templateId: string): string {
    return `${getApiBaseURL()}/js-templates/${encodeURIComponent(templateId)}/export.zip`
  },

  async downloadExport(templateId: string, filename: string): Promise<void> {
    return this.downloadExportByUrl(this.exportZipUrl(templateId), filename)
  },

  async downloadExportByUrl(url: string, filename: string): Promise<void> {
    const token = authToken()
    const res = await fetch(url, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      credentials: 'include',
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const blob = await res.blob()
    const blobUrl = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = blobUrl
    a.download = filename.endsWith('.zip') ? filename : `${filename}.soulpet.zip`
    a.click()
    URL.revokeObjectURL(blobUrl)
  },
}

export default petPackageService

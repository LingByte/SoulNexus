import { get, post, ApiResponse } from '@/utils/request'

export interface PetPreviewSessionResponse {
  token: string
  baseUrl: string
}

export const petProjectService = {
  async registerPreviewSession(files: Record<string, string>): Promise<ApiResponse<PetPreviewSessionResponse>> {
    return post('/pet/studio-preview', { files })
  },

  projectFileUrl(templateId: string, path: string): string {
    const clean = path.replace(/^\/+/, '')
    return `/js-templates/${templateId}/project/file/${clean}`
  },
}

export default petProjectService

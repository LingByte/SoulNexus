import { get, post, ApiResponse } from '@/utils/request'
import { getApiBaseURL } from '@/config/apiConfig'
import { petPackageService } from './petPackage'

export interface PetMarketListing {
  id: string
  marketId: string
  name: string
  description?: string
  kind: string
  authorId: number
  tags?: string
  previewEmoji?: string
  visibility: string
  downloadCount: number
  forkCount: number
  rating?: number
  ratingCount?: number
  createdAt: string
  updatedAt: string
}

export interface PetMarketListResponse {
  data: PetMarketListing[]
  page: number
  limit: number
  total: number
}

export interface ForkMarketResult {
  template: { id: string; jsSourceId: string; name: string }
  marketId: string
}

export const petMarketListingService = {
  list(params?: { page?: number; limit?: number; keyword?: string }): Promise<ApiResponse<PetMarketListResponse>> {
    return get('/pet-market/listings', { params })
  },

  get(marketId: string): Promise<ApiResponse<PetMarketListing>> {
    return get(`/pet-market/listings/${encodeURIComponent(marketId)}`)
  },

  async publishZip(file: File, opts?: { name?: string; description?: string; tags?: string }): Promise<ApiResponse<{ marketId: string; listing: PetMarketListing }>> {
    const form = new FormData()
    form.append('package', file)
    if (opts?.name) form.append('name', opts.name)
    if (opts?.description) form.append('description', opts.description)
    if (opts?.tags) form.append('tags', opts.tags)
    const token = localStorage.getItem('auth_token') || localStorage.getItem('token') || ''
    const res = await fetch(`${getApiBaseURL()}/pet-market/listings`, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: form,
      credentials: 'include',
    })
    return res.json()
  },

  fork(marketId: string, name?: string): Promise<ApiResponse<ForkMarketResult>> {
    return post(`/pet-market/listings/${encodeURIComponent(marketId)}/fork`, name ? { name } : {})
  },

  rate(marketId: string, score: number): Promise<ApiResponse<{ rating: number; ratingCount: number; userRating: number }>> {
    return post(`/pet-market/listings/${encodeURIComponent(marketId)}/rate`, { score })
  },

  previewLoaderUrl(marketId: string): string {
    return `${getApiBaseURL().replace(/\/api\/?$/, '')}/api/pet-market/${encodeURIComponent(marketId)}/preview/loader.js`
  },

  downloadZip(marketId: string, filename: string): Promise<void> {
    const url = `${getApiBaseURL()}/pet-market/listings/${encodeURIComponent(marketId)}/download.zip`
    return petPackageService.downloadExportByUrl(url, filename)
  },
}

export default petMarketListingService

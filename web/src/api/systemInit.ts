import { get, type ApiResponse } from '@/utils/request'
import type { SiteConfig } from '@/contexts/siteConfig'

export async function fetchSystemInit(): Promise<ApiResponse<SiteConfig>> {
  return get('/system/init')
}

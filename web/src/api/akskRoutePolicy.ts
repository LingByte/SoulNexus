import { get, put, type ApiResponse } from '@/utils/request'

export interface AkskRouteCatalogEntry {
  id: string
  group: string
  groupLabel: string
  label: string
  description?: string
  method: string
  path: string
  permission?: string
}

export interface AkskRouteCatalogGroup {
  id: string
  label: string
  entries: AkskRouteCatalogEntry[]
}

export interface AkskRoutePolicy {
  enabled: boolean
  routeIds: string[]
}

export interface AkskRouteCatalogResponse {
  groups: AkskRouteCatalogGroup[]
  total: number
}

export async function getAkskRouteCatalog(): Promise<ApiResponse<AkskRouteCatalogResponse>> {
  return get('/system-configs/route-policy/catalog')
}

export async function getAkskRoutePolicy(): Promise<ApiResponse<AkskRoutePolicy>> {
  return get('/system-configs/route-policy')
}

export async function updateAkskRoutePolicy(body: AkskRoutePolicy): Promise<ApiResponse<AkskRoutePolicy>> {
  return put('/system-configs/route-policy', body)
}

export async function getCredentialAkskRouteCatalog(): Promise<ApiResponse<AkskRouteCatalogResponse>> {
  return get('/credentials/aksk-route-catalog')
}

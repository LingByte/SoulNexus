import { get, put, del, type ApiResponse } from '@/utils/request'
import type { Paginated } from '@/api/types'

export interface WorkflowMarketStats {
  pluginTotal: number
  pluginPublished: number
  pluginDraft: number
  pluginArchived: number
  installTotal: number
  downloadTotal: number
  workflowTotal: number
  workflowActive: number
  nodePluginTotal: number
  nodePluginPublished: number
  byCategory?: Record<string, number>
  topDownloaded?: Array<{
    id: number
    name: string
    displayName: string
    category: string
    status: string
    downloadCount: number
    starCount?: number
    rating?: number
  }>
  recentPublished?: Array<{
    id: number
    name: string
    displayName: string
    category: string
    updatedAt?: string
  }>
}

export interface AdminWorkflowPlugin {
  id: number
  name: string
  slug: string
  displayName: string
  description?: string
  category: string
  status: string
  version?: string
  author?: string
  downloadCount?: number
  starCount?: number
  rating?: number
  createdBy?: number
  groupId?: number
  createdAt?: string
  updatedAt?: string
}

export async function getWorkflowMarketStats(): Promise<ApiResponse<WorkflowMarketStats>> {
  return get('/admin/workflow-market/stats')
}

export async function listAdminWorkflowPlugins(params?: {
  page?: number
  size?: number
  status?: string
  category?: string
  search?: string
}): Promise<ApiResponse<Paginated<AdminWorkflowPlugin>>> {
  return get('/admin/workflow-market/plugins', { params })
}

export async function getAdminWorkflowPlugin(id: number): Promise<ApiResponse<AdminWorkflowPlugin>> {
  return get(`/admin/workflow-market/plugins/${id}`)
}

export async function updateAdminWorkflowPluginStatus(
  id: number,
  status: string,
): Promise<ApiResponse<{ id: number; status: string }>> {
  return put(`/admin/workflow-market/plugins/${id}/status`, { status })
}

export async function deleteAdminWorkflowPlugin(id: number): Promise<ApiResponse<{ id: number }>> {
  return del(`/admin/workflow-market/plugins/${id}`)
}

export async function listAdminWorkflows(params?: {
  page?: number
  size?: number
  status?: string
  search?: string
}): Promise<ApiResponse<Paginated<Record<string, unknown>>>> {
  return get('/admin/workflow-market/workflows', { params })
}

export async function deleteAdminWorkflow(id: number): Promise<ApiResponse<{ id: number }>> {
  return del(`/admin/workflow-market/workflows/${id}`)
}

export async function listAdminNodePlugins(params?: {
  page?: number
  size?: number
  status?: string
  search?: string
}): Promise<ApiResponse<Paginated<Record<string, unknown>>>> {
  return get('/admin/workflow-market/node-plugins', { params })
}

export async function deleteAdminNodePlugin(id: number): Promise<ApiResponse<{ id: number }>> {
  return del(`/admin/workflow-market/node-plugins/${id}`)
}

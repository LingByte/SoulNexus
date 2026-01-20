import { get, post, put, del, ApiResponse } from '@/utils/request'
import type { OverviewConfig } from '@/types/overview'

// 获取组织的概览配置
export const getOverviewConfig = async (organizationId: number): Promise<ApiResponse<OverviewConfig | null>> => {
  return get(`/group/${organizationId}/overview/config`)
}

// 保存概览配置
export const saveOverviewConfig = async (config: OverviewConfig): Promise<ApiResponse<OverviewConfig>> => {
  // 将配置转换为后端期望的格式
  const payload = {
    name: config.name,
    description: config.description || '',
    layout: config.layout,
    widgets: config.widgets,
    theme: config.theme || {},
    header: config.header,
    footer: config.footer
  }
  return post(`/group/${config.organizationId}/overview/config`, payload)
}

// 更新概览配置
export const updateOverviewConfig = async (config: OverviewConfig): Promise<ApiResponse<OverviewConfig>> => {
  // 将配置转换为后端期望的格式
  const payload = {
    name: config.name,
    description: config.description || '',
    layout: config.layout,
    widgets: config.widgets,
    theme: config.theme || {}
  }
  return put(`/group/${config.organizationId}/overview/config`, payload)
}

// 删除概览配置
export const deleteOverviewConfig = async (organizationId: number): Promise<ApiResponse<null>> => {
  return del(`/group/${organizationId}/overview/config`)
}

// 获取组织统计数据（用于Widget）
export const getOrganizationStats = async (organizationId: number): Promise<ApiResponse<any>> => {
  return get(`/group/${organizationId}/statistics`)
}


import { get, post, put, del } from '@/utils/request'

// 工作流插件分类
export type WorkflowPluginCategory = 
  | 'data_processing'
  | 'api_integration' 
  | 'ai_service'
  | 'notification'
  | 'utility'
  | 'business'
  | 'custom'

// 工作流插件状态
export type WorkflowPluginStatus = 'draft' | 'published' | 'archived'

// 参数定义
export interface WorkflowPluginParameter {
  name: string
  type: string
  required: boolean
  default?: any
  description?: string
  example?: any
}

// 输入输出参数定义
export interface WorkflowPluginIOSchema {
  parameters: WorkflowPluginParameter[]
}

// 工作流插件
export interface WorkflowPlugin {
  id: number
  userId: number
  groupId?: number
  workflowId: number
  name: string
  slug: string
  displayName: string
  description?: string
  category: WorkflowPluginCategory
  version?: string
  status: WorkflowPluginStatus
  icon?: string
  color?: string
  tags?: string[]
  inputSchema?: WorkflowPluginIOSchema
  outputSchema?: WorkflowPluginIOSchema
  workflowSnapshot?: any
  downloadCount?: number
  starCount?: number
  rating?: number
  author?: string
  homepage?: string
  repository?: string
  license?: string
  createdAt: string
  updatedAt: string
}

// 工作流插件安装记录
export interface WorkflowPluginInstallation {
  id: number
  userId: number
  pluginId: number
  version: string
  status: string
  config: Record<string, any>
  createdAt: string
  updatedAt: string
  plugin?: WorkflowPlugin
}

// 发布工作流为插件的请求
export interface PublishWorkflowAsPluginRequest {
  name: string
  displayName: string
  description: string
  category: WorkflowPluginCategory
  icon?: string
  color?: string
  tags?: string[]
  inputSchema: WorkflowPluginIOSchema
  outputSchema: WorkflowPluginIOSchema
  author?: string
  homepage?: string
  repository?: string
  license?: string
}

// 更新工作流插件的请求
export interface UpdateWorkflowPluginRequest {
  displayName?: string
  description?: string
  category?: WorkflowPluginCategory
  icon?: string
  color?: string
  tags?: string[]
  inputSchema?: WorkflowPluginIOSchema
  outputSchema?: WorkflowPluginIOSchema
  version?: string
  changeLog?: string
  author?: string
  homepage?: string
  repository?: string
  license?: string
}

// 安装工作流插件的请求
export interface InstallWorkflowPluginRequest {
  version?: string
  config?: Record<string, any>
}

// 查询工作流插件列表的参数
export interface ListWorkflowPluginsParams {
  category?: WorkflowPluginCategory
  status?: WorkflowPluginStatus
  keyword?: string
  userId?: number
  page?: number
  pageSize?: number
}

// API响应类型 - 使用统一的响应格式
import { ApiResponse } from '@/utils/request'

// 工作流插件服务
export const workflowPluginService = {
  // 发布工作流为插件
  publishWorkflowAsPlugin: (workflowId: number, data: PublishWorkflowAsPluginRequest): Promise<ApiResponse<WorkflowPlugin>> => {
    return post(`/workflow-plugins/publish/${workflowId}`, data)
  },

  // 获取工作流插件列表
  listWorkflowPlugins: (params?: ListWorkflowPluginsParams): Promise<ApiResponse<{
    plugins: WorkflowPlugin[]
    total: number
    page: number
    pageSize: number
  }>> => {
    return get('/workflow-plugins', { params })
  },

  // 获取工作流插件详情
  getWorkflowPlugin: (id: number): Promise<ApiResponse<WorkflowPlugin>> => {
    return get(`/workflow-plugins/${id}`)
  },

  // 更新工作流插件
  updateWorkflowPlugin: (id: number, data: UpdateWorkflowPluginRequest): Promise<ApiResponse<{ message: string }>> => {
    return put(`/workflow-plugins/${id}`, data)
  },

  // 删除工作流插件
  deleteWorkflowPlugin: (id: number): Promise<ApiResponse<{ message: string }>> => {
    return del(`/workflow-plugins/${id}`)
  },

  // 发布工作流插件
  publishWorkflowPlugin: (id: number): Promise<ApiResponse<{ message: string }>> => {
    return post(`/workflow-plugins/${id}/publish`)
  },

  // 安装工作流插件
  installWorkflowPlugin: (id: number, data?: InstallWorkflowPluginRequest): Promise<ApiResponse<{ message: string }>> => {
    return post(`/workflow-plugins/${id}/install`, data || {})
  },

  // 获取已安装的工作流插件
  listInstalledWorkflowPlugins: (): Promise<ApiResponse<WorkflowPluginInstallation[]>> => {
    return get('/workflow-plugins/installed')
  },

  // 获取用户创建的工作流插件
  getUserWorkflowPlugins: (): Promise<ApiResponse<WorkflowPlugin[]>> => {
    return get('/workflow-plugins/my-plugins')
  },

  // 获取工作流已发布的插件信息
  getWorkflowPublishedPlugin: (workflowId: number): Promise<ApiResponse<WorkflowPlugin | null>> => {
    return get(`/workflow-plugins/workflow/${workflowId}/published`)
  }
}
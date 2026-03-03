import { get, post, put, del, ApiResponse } from '@/utils/request'

// 插件状态类型
export type NodePluginStatus = 'draft' | 'published' | 'archived' | 'banned'

// 插件分类类型
export type NodePluginCategory = 'api' | 'data' | 'ai' | 'notification' | 'utility' | 'custom'

// 端口定义
export interface NodePluginPort {
  name: string
  type: string
  required: boolean
  description?: string
  default?: any
}

// 运行时配置
export interface NodePluginRuntime {
  type: string // script, http, builtin
  config: Record<string, any>
  timeout?: number
  retry?: number
}

// 表单字段选项
export interface FormFieldOption {
  label: string
  value: any
}

// 表单字段验证
export interface FormFieldValidation {
  min?: number
  max?: number
  pattern?: string
  message?: string
}

// 表单字段
export interface NodePluginFormField {
  name: string
  type: string // text, number, select, textarea, etc.
  label: string
  description?: string
  required: boolean
  default?: any
  options?: FormFieldOption[]
  validation?: FormFieldValidation
}

// UI配置
export interface NodePluginUI {
  configForm: NodePluginFormField[]
  preview?: string
  help?: string
}

// 插件定义
export interface NodePluginDefinition {
  type: string
  inputs: NodePluginPort[]
  outputs: NodePluginPort[]
  runtime: NodePluginRuntime
  ui: NodePluginUI
  dependencies?: string[]
}

// 配置属性
export interface SchemaProperty {
  type: string
  description?: string
  default?: any
  enum?: string[]
}

// 配置模式
export interface NodePluginSchema {
  properties: Record<string, SchemaProperty>
  required?: string[]
}

// 插件版本
export interface NodePluginVersion {
  id: number
  pluginId: number
  version: string
  definition: NodePluginDefinition
  schema: NodePluginSchema
  changeLog: string
  createdAt: string
}

// 插件评价
export interface NodePluginReview {
  id: number
  pluginId: number
  userId: number
  rating: number
  comment: string
  createdAt: string
  updatedAt: string
  user?: {
    id: number
    name: string
    avatar?: string
  }
}

// 插件安装记录
export interface NodePluginInstallation {
  id: number
  userId: number
  pluginId: number
  version: string
  status: string
  config: Record<string, any>
  installedAt: string
  updatedAt: string
  plugin?: NodePlugin
}

// 节点插件
export interface NodePlugin {
  id: number
  userId: number
  groupId?: number
  name: string
  slug: string
  displayName: string
  description: string
  category: NodePluginCategory
  version: string
  status: NodePluginStatus
  icon: string
  color: string
  tags: string[]
  definition: NodePluginDefinition
  schema: NodePluginSchema
  downloadCount: number
  starCount: number
  rating: number
  author: string
  homepage: string
  repository: string
  license: string
  createdAt: string
  updatedAt: string
  versions?: NodePluginVersion[]
  reviews?: NodePluginReview[]
}

// 创建插件请求
export interface CreateNodePluginRequest {
  name: string
  displayName: string
  description: string
  category: NodePluginCategory
  icon?: string
  color?: string
  tags?: string[]
  definition: NodePluginDefinition
  schema?: NodePluginSchema
  author?: string
  homepage?: string
  repository?: string
  license?: string
}

// 更新插件请求
export interface UpdateNodePluginRequest {
  displayName?: string
  description?: string
  category?: NodePluginCategory
  icon?: string
  color?: string
  tags?: string[]
  definition?: NodePluginDefinition
  schema?: NodePluginSchema
  version?: string
  changeLog?: string
  author?: string
  homepage?: string
  repository?: string
  license?: string
}

// 插件列表查询参数
export interface ListNodePluginsParams {
  category?: NodePluginCategory
  status?: NodePluginStatus
  keyword?: string
  userId?: number
  page?: number
  pageSize?: number
}

// 安装插件请求
export interface InstallPluginRequest {
  version?: string
  config?: Record<string, any>
}

// 节点插件API服务
export const nodePluginService = {
  /**
   * 创建插件
   */
  async createPlugin(data: CreateNodePluginRequest): Promise<ApiResponse<NodePlugin>> {
    return post<NodePlugin>('/node-plugins', data)
  },

  /**
   * 获取插件列表
   */
  async listPlugins(params?: ListNodePluginsParams): Promise<ApiResponse<{
    plugins: NodePlugin[]
    total: number
    page: number
    pageSize: number
  }>> {
    return get<{
      plugins: NodePlugin[]
      total: number
      page: number
      pageSize: number
    }>('/node-plugins', { params })
  },

  /**
   * 获取插件详情
   */
  async getPlugin(id: number): Promise<ApiResponse<NodePlugin>> {
    return get<NodePlugin>(`/node-plugins/${id}`)
  },

  /**
   * 更新插件
   */
  async updatePlugin(id: number, data: UpdateNodePluginRequest): Promise<ApiResponse<{ message: string }>> {
    return put<{ message: string }>(`/node-plugins/${id}`, data)
  },

  /**
   * 删除插件
   */
  async deletePlugin(id: number): Promise<ApiResponse<{ message: string }>> {
    return del<{ message: string }>(`/node-plugins/${id}`)
  },

  /**
   * 发布插件
   */
  async publishPlugin(id: number): Promise<ApiResponse<{ message: string }>> {
    return post<{ message: string }>(`/node-plugins/${id}/publish`)
  },

  /**
   * 安装插件
   */
  async installPlugin(id: number, data?: InstallPluginRequest): Promise<ApiResponse<{ message: string }>> {
    return post<{ message: string }>(`/node-plugins/${id}/install`, data || {})
  },

  /**
   * 卸载插件
   */
  async uninstallPlugin(id: number): Promise<ApiResponse<{ message: string }>> {
    return del<{ message: string }>(`/node-plugins/${id}/install`)
  },

  /**
   * 获取已安装插件列表
   */
  async listInstalledPlugins(): Promise<ApiResponse<NodePluginInstallation[]>> {
    return get<NodePluginInstallation[]>('/node-plugins/installed')
  },

  /**
   * 获取我的插件列表
   */
  async listMyPlugins(): Promise<ApiResponse<NodePlugin[]>> {
    return get<NodePlugin[]>('/node-plugins/my')
  },

  /**
   * 收藏插件
   */
  async starPlugin(id: number): Promise<ApiResponse<{ message: string }>> {
    return post<{ message: string }>(`/node-plugins/${id}/star`)
  },

  /**
   * 取消收藏插件
   */
  async unstarPlugin(id: number): Promise<ApiResponse<{ message: string }>> {
    return del<{ message: string }>(`/node-plugins/${id}/star`)
  },

  /**
   * 评价插件
   */
  async reviewPlugin(id: number, data: {
    rating: number
    comment: string
  }): Promise<ApiResponse<NodePluginReview>> {
    return post<NodePluginReview>(`/node-plugins/${id}/reviews`, data)
  },

  /**
   * 获取插件评价列表
   */
  async listPluginReviews(id: number): Promise<ApiResponse<NodePluginReview[]>> {
    return get<NodePluginReview[]>(`/node-plugins/${id}/reviews`)
  },

  /**
   * 获取插件版本列表
   */
  async listPluginVersions(id: number): Promise<ApiResponse<NodePluginVersion[]>> {
    return get<NodePluginVersion[]>(`/node-plugins/${id}/versions`)
  },

  /**
   * 获取插件特定版本
   */
  async getPluginVersion(id: number, versionId: number): Promise<ApiResponse<NodePluginVersion>> {
    return get<NodePluginVersion>(`/node-plugins/${id}/versions/${versionId}`)
  }
}

export default nodePluginService
import { get, post, put, del, ApiResponse } from '@/utils/request'

// JS模板接口定义
export interface JSTemplate {
  id: string
  jsSourceId: string
  name: string
  type: 'default' | 'custom'
  content: string
  usage?: string
  user_id?: number
  created_at: string
  updated_at: string
}

// 创建JS模板表单
export interface CreateJSTemplateForm {
  name: string
  type: 'default' | 'custom'
  content: string
  usage?: string
}

// 更新JS模板表单
export interface UpdateJSTemplateForm {
  name?: string
  content?: string
  usage?: string
}

export interface PetProjectPayload {
  name?: string
  usage?: string
  entry: string
  files: Record<string, string>
  changeNote?: string
  bumpVersion?: boolean
}

export interface PetProjectResponse {
  storage: 'object' | 'inline' | 'pending'
  prefix?: string
  entry: string
  files: Record<string, string>
  name?: string
  usage?: string
}

export interface PetProjectSaveResult {
  storage: string
  prefix?: string
  entry: string
  violations?: string[]
}

export interface CreateStudioProjectResult {
  template: JSTemplate
  storage: string
  prefix?: string
  entry: string
}

// JS模板列表响应
export interface JSTemplateListResponse {
  data: JSTemplate[]
  page: number
  limit: number
  total: number
}

// JS模板API服务
export const jsTemplateService = {
  // 创建JS模板
  async createTemplate(data: CreateJSTemplateForm): Promise<ApiResponse<JSTemplate>> {
    return post('/js-templates', data)
  },

  // 获取JS模板列表
  async getTemplates(params?: {
    page?: number
    limit?: number
  }): Promise<ApiResponse<JSTemplateListResponse>> {
    return get('/js-templates', { params })
  },

  // 获取单个JS模板
  async getTemplate(id: string): Promise<ApiResponse<JSTemplate>> {
    return get(`/js-templates/${id}`)
  },

  // 根据名称获取JS模板
  async getTemplatesByName(name: string): Promise<ApiResponse<JSTemplate[]>> {
    return get(`/js-templates/name/${name}`)
  },

  // 更新JS模板
  async updateTemplate(id: string, data: UpdateJSTemplateForm): Promise<ApiResponse<JSTemplate>> {
    return put(`/js-templates/${id}`, data)
  },

  // 删除JS模板
  async deleteTemplate(id: string): Promise<ApiResponse<{ message: string }>> {
    return del(`/js-templates/${id}`)
  },

  // 获取默认模板列表
  async getDefaultTemplates(params?: {
    page?: number
    limit?: number
  }): Promise<ApiResponse<JSTemplateListResponse>> {
    return get('/js-templates/default', { params })
  },

  // 获取自定义模板列表
  async getCustomTemplates(params?: {
    page?: number
    limit?: number
  }): Promise<ApiResponse<JSTemplateListResponse>> {
    return get('/js-templates/custom', { params })
  },

  // 搜索JS模板
  async searchTemplates(params: {
    keyword: string
    page?: number
    limit?: number
  }): Promise<ApiResponse<JSTemplateListResponse>> {
    return get('/js-templates/search', { params })
  },

  /** Atomically create template metadata + upload project files to object storage. */
  async createStudioProject(data: PetProjectPayload & { name: string }): Promise<ApiResponse<CreateStudioProjectResult>> {
    return post('/js-templates/studio-project', data)
  },

  /** Load project files from object storage (or legacy inline) into browser memory. */
  async getProject(id: string): Promise<ApiResponse<PetProjectResponse>> {
    return get(`/js-templates/${id}/project`)
  },

  /** Persist project files to object storage; DB stores metadata pointer only. */
  async saveProject(id: string, data: PetProjectPayload): Promise<ApiResponse<PetProjectSaveResult>> {
    return put(`/js-templates/${id}/project`, data)
  },
}

export default jsTemplateService
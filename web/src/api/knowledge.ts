import { get, post, del, ApiResponse } from '@/utils/request'

// 知识库基本信息
export interface KnowledgeBase {
    id: number
    user_id: number
    group_id?: number | null // 组织ID，如果设置则表示这是组织共享的知识库
    knowledge_key: string
    knowledge_name: string
    provider?: string
    created_at: string
    updated_at?: string
    update_at: string
    delete_at: string
}

// 创建知识库请求参数
export interface CreateKnowledgeBaseRequest {
    knowledgeName: string
    provider?: string
    file?: File
    groupId?: number | null // 组织ID，如果设置则创建为组织共享的知识库
}

// 上传文件到知识库请求参数
export interface UploadKnowledgeBaseRequest {
    file: File
    knowledgeKey: string
}

// 删除知识库请求参数
export interface DeleteKnowledgeBaseRequest {
    knowledgeKey: string
}
export interface KnowledgeInfo {
    name: string
    key: string
}

// 根据用户ID获取知识库列表响应
export type GetKnowledgeBaseByUserResponse = KnowledgeInfo[]


// 向知识库提问请求参数
export interface AskKnowledgeBaseRequest {
    knowledgeKey: string
    message: string
}

// 向知识库提问响应
export type AskKnowledgeBaseResponse = string

// 创建知识库
export const createKnowledgeBase = async (
    data: CreateKnowledgeBaseRequest
): Promise<ApiResponse<KnowledgeBase>> => {
    const formData = new FormData()
    formData.append('knowledgeName', data.knowledgeName)
    if (data.provider) {
        formData.append('provider', data.provider)
    }
    if (data.file) {
        formData.append('file', data.file)
    }
    if (data.groupId) {
        formData.append('group_id', data.groupId.toString())
    }
    return post<KnowledgeBase>('/knowledge/create', formData)
}

// 上传文件到知识库
// 在 knowledge.ts 中检查 uploadKnowledgeBase 函数
export const uploadKnowledgeBase = async (
    data: UploadKnowledgeBaseRequest
): Promise<ApiResponse<null>> => {
    const formData = new FormData()
    formData.append('file', data.file)
    formData.append('knowledgeKey', data.knowledgeKey) // 确保参数名匹配
    return post<null>('/knowledge/upload', formData)
}


// 删除知识库
export const deleteKnowledgeBase = async (
    knowledgeKey: string
): Promise<ApiResponse<string>> => {
    return del<string>('/knowledge/delete', {
        params: { knowledgeKey }
    })
}

// 根据用户ID获取知识库名称列表
export const getKnowledgeBaseByUser = async (): Promise<ApiResponse<GetKnowledgeBaseByUserResponse>> => {
    return get<GetKnowledgeBaseByUserResponse>(
        '/knowledge/get',
    )
}
// 向知识库提问（搜索/召回知识库文档）
export const askKnowledgeBase = async (
    params: AskKnowledgeBaseRequest
): Promise<ApiResponse<AskKnowledgeBaseResponse>> => {
    return get<AskKnowledgeBaseResponse>(
        '/knowledge/search',
        { params }
    )
}

// 列出知识库中的所有内容
export const listKnowledgeBaseContent = async (
    knowledgeKey: string
): Promise<ApiResponse<any>> => {
    return get<any>(
        '/knowledge/list',
        { params: { knowledgeKey } }
    )
}


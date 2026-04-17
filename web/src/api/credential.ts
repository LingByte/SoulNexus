import { get, post, del, ApiResponse } from '@/utils/request'

// 提供商的灵活配置类型
export interface ProviderConfig {
  provider: string
  [key: string]: any // 支持任意其他字段
}

// 密钥创建表单
export interface CreateCredentialForm {
  name: string
  llmProvider: string
  llmApiKey: string
  llmApiUrl: string
  expiresAt?: string
  tokenQuota?: number
  requestQuota?: number
  amountUsd?: number
  useNativeQuota?: boolean
  unlimitedQuota?: boolean
  
  // JSON格式配置
  asrConfig?: ProviderConfig // ASR配置,例如: {provider: "qiniu", apiKey: "...", baseUrl: "..."} 或 {provider: "qcloud", appId: "...", secretId: "...", secretKey: "..."}
  ttsConfig?: ProviderConfig // TTS配置
}

// 密钥信息（不包含敏感信息）
export interface Credential {
  id: number
  name: string
  apiKey: string
  apiSecret: string
  llmProvider: string
  userId: number
  asrProvider?: string
  ttsProvider?: string
  asrConfig?: ProviderConfig
  ttsConfig?: ProviderConfig
  status?: 'active' | 'banned' | 'suspended'
  expiresAt?: string
  usageCount?: number
  tokenQuota?: number
  tokenUsed?: number
  requestQuota?: number
  amountUsd?: number
  useNativeQuota?: boolean
  unlimitedQuota?: boolean
  createdAt: string
  updatedAt: string
}

// 创建密钥响应
export interface CreateCredentialResponse {
  id: number
  name: string
  apiKey: string
  apiSecret: string
}

// 获取用户密钥列表
export const fetchUserCredentials = async (): Promise<ApiResponse<Credential[]>> => {
  return get('/credentials/')
}

// 创建密钥
export const createCredential = async (data: CreateCredentialForm): Promise<ApiResponse<CreateCredentialResponse>> => {
  return post('/credentials/', data)
}

// 删除密钥
export const deleteCredential = async (id: number): Promise<ApiResponse<null>> => {
  return del(`/credentials/${id}`)
}

// 根据 apikey 和 apisecret 获取凭证信息（用于 controlpanel）
export interface CredentialDetail {
  llmProvider: string
  llmApiKey: string
  llmApiUrl: string
  asrProvider: string
  ttsProvider: string
}

export const getCredentialByKey = async (apiKey: string, apiSecret: string): Promise<ApiResponse<CredentialDetail | null>> => {
  return post('/credentials/by-key', { apiKey, apiSecret })
}

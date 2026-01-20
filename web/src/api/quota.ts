import { get, post, put, del, ApiResponse } from '@/utils/request'

// 配额类型
export type QuotaType = 
  | 'storage'
  | 'llm_tokens'
  | 'llm_calls'
  | 'api_calls'
  | 'call_duration'
  | 'call_count'
  | 'asr_duration'
  | 'asr_count'
  | 'tts_duration'
  | 'tts_count'

// 配额周期
export type QuotaPeriod = 'lifetime' | 'monthly' | 'yearly'

// 用户配额
export interface UserQuota {
  id: number
  userId: number
  quotaType: QuotaType
  totalQuota: number
  usedQuota: number
  period: QuotaPeriod
  resetAt?: string
  description?: string
  createdAt: string
  updatedAt: string
}

// 组织配额
export interface GroupQuota {
  id: number
  groupId: number
  quotaType: QuotaType
  totalQuota: number
  usedQuota: number
  period: QuotaPeriod
  resetAt?: string
  description?: string
  createdAt: string
  updatedAt: string
}

// 创建用户配额请求
export interface CreateUserQuotaRequest {
  quotaType: QuotaType
  totalQuota: number
  period?: QuotaPeriod
  description?: string
}

// 更新用户配额请求
export interface UpdateUserQuotaRequest {
  totalQuota?: number
  period?: QuotaPeriod
  description?: string
}

// 创建组织配额请求
export interface CreateGroupQuotaRequest {
  quotaType: QuotaType
  totalQuota: number
  period?: QuotaPeriod
  description?: string
}

// 更新组织配额请求
export interface UpdateGroupQuotaRequest {
  totalQuota?: number
  period?: QuotaPeriod
  description?: string
}

// 获取用户配额列表
export const getUserQuotas = async (): Promise<ApiResponse<UserQuota[]>> => {
  return get('/quota/user')
}

// 获取用户配额详情
export const getUserQuota = async (type: QuotaType): Promise<ApiResponse<UserQuota>> => {
  return get(`/quota/user/${type}`)
}

// 创建用户配额
export const createUserQuota = async (data: CreateUserQuotaRequest): Promise<ApiResponse<UserQuota>> => {
  return post('/quota/user', data)
}

// 更新用户配额
export const updateUserQuota = async (type: QuotaType, data: UpdateUserQuotaRequest): Promise<ApiResponse<UserQuota>> => {
  return put(`/quota/user/${type}`, data)
}

// 删除用户配额
export const deleteUserQuota = async (type: QuotaType): Promise<ApiResponse<null>> => {
  return del(`/quota/user/${type}`)
}

// 获取组织配额列表
export const getGroupQuotas = async (groupId: number): Promise<ApiResponse<GroupQuota[]>> => {
  return get(`/quota/group/${groupId}`)
}

// 获取组织配额详情
export const getGroupQuota = async (groupId: number, type: QuotaType): Promise<ApiResponse<GroupQuota>> => {
  return get(`/quota/group/${groupId}/${type}`)
}

// 创建组织配额
export const createGroupQuota = async (groupId: number, data: CreateGroupQuotaRequest): Promise<ApiResponse<GroupQuota>> => {
  return post(`/quota/group/${groupId}`, data)
}

// 更新组织配额
export const updateGroupQuota = async (groupId: number, type: QuotaType, data: UpdateGroupQuotaRequest): Promise<ApiResponse<GroupQuota>> => {
  return put(`/quota/group/${groupId}/${type}`, data)
}

// 删除组织配额
export const deleteGroupQuota = async (groupId: number, type: QuotaType): Promise<ApiResponse<null>> => {
  return del(`/quota/group/${groupId}/${type}`)
}

// 获取配额类型标签
export const getQuotaTypeLabel = (type: QuotaType): string => {
  const labels: Record<QuotaType, string> = {
    storage: '存储空间',
    llm_tokens: 'LLM Token',
    llm_calls: 'LLM 调用次数',
    api_calls: 'API 调用次数',
    call_duration: '通话时长',
    call_count: '通话次数',
    asr_duration: '语音识别时长',
    asr_count: '语音识别次数',
    tts_duration: '语音合成时长',
    tts_count: '语音合成次数',
  }
  return labels[type] || type
}

// 格式化配额值
export const formatQuotaValue = (type: QuotaType, value: number): string => {
  switch (type) {
    case 'storage':
      return formatBytes(value)
    case 'llm_tokens':
    case 'llm_calls':
    case 'api_calls':
    case 'call_count':
    case 'asr_count':
    case 'tts_count':
      return formatNumber(value)
    case 'call_duration':
    case 'asr_duration':
    case 'tts_duration':
      return formatDuration(value)
    default:
      return value.toString()
  }
}

// 格式化字节
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
}

// 格式化数字
const formatNumber = (n: number): string => {
  if (n < 1000) return n.toString()
  if (n < 1000000) return `${(n / 1000).toFixed(2)}K`
  return `${(n / 1000000).toFixed(2)}M`
}

// 格式化时长（秒）
const formatDuration = (seconds: number): string => {
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${(seconds / 60).toFixed(1)}分钟`
  return `${(seconds / 3600).toFixed(1)}小时`
}


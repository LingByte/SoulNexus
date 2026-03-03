import { get, post, put, del, ApiResponse } from '@/utils/request'
import type { 
  Voicemail, 
  VoicemailListParams, 
  VoicemailStats, 
  BatchOperationRequest 
} from '@/types/voicemail'

/**
 * 获取留言列表
 */
export const getVoicemails = async (params?: VoicemailListParams): Promise<ApiResponse<Voicemail[]>> => {
  return get('/voicemails', { params })
}

/**
 * 获取留言详情
 */
export const getVoicemailDetail = async (id: number): Promise<ApiResponse<Voicemail>> => {
  return get(`/voicemails/${id}`)
}

/**
 * 更新留言
 */
export const updateVoicemail = async (id: number, data: Partial<Voicemail>): Promise<ApiResponse<Voicemail>> => {
  return put(`/voicemails/${id}`, data)
}

/**
 * 删除留言
 */
export const deleteVoicemail = async (id: number): Promise<ApiResponse<void>> => {
  return del(`/voicemails/${id}`)
}

/**
 * 标记已读
 */
export const markVoicemailAsRead = async (id: number): Promise<ApiResponse<void>> => {
  return post(`/voicemails/${id}/mark-read`)
}

/**
 * 标记重要
 */
export const markVoicemailAsImportant = async (id: number, important: boolean): Promise<ApiResponse<void>> => {
  return post(`/voicemails/${id}/mark-important`, { important })
}

/**
 * 转录留言
 */
export const transcribeVoicemail = async (id: number): Promise<ApiResponse<void>> => {
  return post(`/voicemails/${id}/transcribe`)
}

/**
 * 生成摘要
 */
export const summarizeVoicemail = async (id: number): Promise<ApiResponse<void>> => {
  return post(`/voicemails/${id}/summarize`)
}

/**
 * 批量删除
 */
export const batchDeleteVoicemails = async (ids: number[]): Promise<ApiResponse<void>> => {
  return post('/voicemails/batch-delete', { ids })
}

/**
 * 批量标记已读
 */
export const batchMarkAsRead = async (ids: number[]): Promise<ApiResponse<void>> => {
  return post('/voicemails/batch-mark-read', { ids })
}

/**
 * 获取留言统计
 */
export const getVoicemailStats = async (): Promise<ApiResponse<VoicemailStats>> => {
  return get('/voicemails/stats')
}

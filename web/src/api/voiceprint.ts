import { get, post, put, del } from '@/utils/request'
import { ApiResponse } from '@/utils/request'

// 声纹记录类型
export interface VoiceprintRecord {
  id: number
  speaker_id: string
  assistant_id: string
  speaker_name: string
  description?: string
  created_at: string
  updated_at: string
  last_used?: string
  confidence?: number
}

// 声纹列表响应
export interface VoiceprintListResponse {
  total: number
  voiceprints: VoiceprintRecord[]
}

// 创建声纹请求
export interface CreateVoiceprintRequest {
  speaker_id: string
  assistant_id: string
  speaker_name: string
  description?: string
}

// 更新声纹请求
export interface UpdateVoiceprintRequest {
  speaker_name?: string
  description?: string
}

// 声纹识别响应
export interface VoiceprintIdentifyResponse {
  speaker_id: string
  score: number
  confidence: string
  is_match: boolean
}

// 声纹验证响应
export interface VoiceprintVerifyResponse {
  target_speaker_id: string
  identified_speaker_id: string
  score: number
  confidence: string
  is_match: boolean
  is_target_speaker: boolean
  verification_passed: boolean
}

// 获取声纹列表
export const getVoiceprints = async (assistantId: string): Promise<ApiResponse<VoiceprintListResponse>> => {
  return get(`/voiceprint?assistant_id=${assistantId}`)
}

// 创建声纹记录
export const createVoiceprint = async (data: CreateVoiceprintRequest): Promise<ApiResponse<VoiceprintRecord>> => {
  return post('/voiceprint', data)
}

// 注册声纹（上传音频）
export const registerVoiceprint = async (
  assistantId: string,
  speakerName: string,
  audioFile: File,
  description?: string
): Promise<ApiResponse<VoiceprintRecord>> => {
  const formData = new FormData()
  formData.append('assistant_id', assistantId)
  formData.append('speaker_name', speakerName)
  formData.append('audio_file', audioFile)
  if (description) {
    formData.append('description', description)
  }

  return post('/voiceprint/register', formData)
}

// 更新声纹记录
export const updateVoiceprint = async (id: number, data: UpdateVoiceprintRequest): Promise<ApiResponse<VoiceprintRecord>> => {
  return put(`/voiceprint/${id}`, data)
}

// 删除声纹记录
export const deleteVoiceprint = async (id: number): Promise<ApiResponse<void>> => {
  return del(`/voiceprint/${id}`)
}

// 声纹识别
export const identifyVoiceprint = async (
  assistantId: string,
  audioFile: File
): Promise<ApiResponse<VoiceprintIdentifyResponse>> => {
  const formData = new FormData()
  formData.append('assistant_id', assistantId)
  formData.append('audio_file', audioFile)

  return post('/voiceprint/identify', formData)
}

// 声纹验证
export const verifyVoiceprint = async (
  assistantId: string,
  speakerId: string,
  audioFile: File
): Promise<ApiResponse<VoiceprintVerifyResponse>> => {
  const formData = new FormData()
  formData.append('assistant_id', assistantId)
  formData.append('speaker_id', speakerId)
  formData.append('audio_file', audioFile)

  return post('/voiceprint/verify', formData)
}
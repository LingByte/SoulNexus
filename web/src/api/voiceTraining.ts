import { get, post } from '@/utils/request'

// 训练文本相关接口
export interface TrainingText {
  id: number
  text_id: number
  text_name: string
  language: string
  is_active: boolean
  created_at: string
  updated_at: string
  deleted_at: string | null
  text_segments: TrainingTextSegment[]
}

export interface TrainingTextSegment {
  id: number
  text_id: number
  seg_id: string
  seg_text: string
  created_at: string
  updated_at: string
  deleted_at: string | null
}

// 语音克隆相关接口
export interface VoiceClone {
  id: number
  user_id: number
  task_id: string
  task_name: string
  sex: number
  age_group: number
  language: string
  status: number
  text_id: number
  text_seg_id: number
  audio_url: string
  audio_duration: number
  audio_size: number
  train_vid: string
  asset_id: string
  failed_reason: string
  created_at: string
  updated_at: string
  deleted_at: string | null
}

// 合成历史相关接口
export interface SynthesisHistory {
  id: number
  user_id: number
  voice_clone_id: number
  text: string
  audio_url: string
  duration: number
  created_at: string
  updated_at: string
  deleted_at: string | null
}

// 创建训练任务请求
export interface CreateTaskRequest {
  task_name: string
  sex: number
  age_group: number
  language?: string
}

// 提交音频请求
export interface SubmitAudioRequest {
  task_id: string
  text_seg_id: number
  audio_file: File
}

// 查询任务状态请求
export interface QueryTaskRequest {
  task_id: string
}

// 语音合成请求
export interface SynthesizeRequest {
  voice_clone_id: number
  text: string
  language?: string
}

// 更新语音克隆请求
export interface UpdateVoiceCloneRequest {
  id: number
  task_name?: string
  sex?: number
  age_group?: number
  language?: string
}

// 删除语音克隆请求
export interface DeleteVoiceCloneRequest {
  id: number
}

// 获取训练文本列表
export const getTrainingTexts = () => {
  return get<TrainingText>('/voice/training-texts')
}

// 获取语音克隆列表
export const getVoiceClones = () => {
  return get<VoiceClone[]>('/voice/clones')
}

// 获取合成历史
export const getSynthesisHistory = () => {
  return get<SynthesisHistory[]>('/voice/synthesis/history')
}

// 创建训练任务
export const createTrainingTask = (data: CreateTaskRequest) => {
  return post<{ task_id: string }>('/voice/training/create', data)
}

// 提交音频文件
export const submitAudio = (data: SubmitAudioRequest) => {
  const formData = new FormData()
  formData.append('task_id', data.task_id)
  formData.append('text_seg_id', data.text_seg_id.toString())
  formData.append('audio_file', data.audio_file)
  
  return post('/voice/training/submit-audio', formData)
}

// 查询任务状态
export const queryTaskStatus = (data: QueryTaskRequest) => {
  return get<VoiceClone>(`/voice/training/status?task_id=${data.task_id}`)
}

// 语音合成
export const synthesizeVoice = (data: SynthesizeRequest) => {
  return post<{ audio_url: string }>('/voice/synthesize', data)
}

// 试听语音克隆
export const auditionVoiceClone = (id: number) => {
  return get<{ audio_url: string }>(`/voice/clones/${id}/audition`)
}

// 更新语音克隆
export const updateVoiceClone = (data: UpdateVoiceCloneRequest) => {
  return post('/voice/clones/update', data)
}

// 删除语音克隆
export const deleteVoiceClone = (data: DeleteVoiceCloneRequest) => {
  return post('/voice/clones/delete', data)
}

// 删除合成历史记录
export const deleteSynthesisRecord = (id: number) => {
  return post('/voice/synthesis/delete', { id })
}

// 留言类型定义

export interface Voicemail {
  id: number
  userId: number
  sipUserId?: number
  sipCallId?: number
  
  // 来电信息
  callerNumber: string
  callerName?: string
  callerLocation?: string
  
  // 音频信息
  audioPath: string
  audioUrl?: string
  audioFormat: string
  audioSize: number
  duration: number
  sampleRate: number
  channels: number
  
  // 留言内容
  transcribedText?: string
  summary?: string
  keywords?: string
  
  // 状态
  status: 'new' | 'read' | 'archived'
  isRead: boolean
  isImportant: boolean
  readAt?: string
  
  // 转录状态
  transcribeStatus: 'pending' | 'processing' | 'completed' | 'failed'
  transcribeError?: string
  transcribedAt?: string
  
  // 元数据
  metadata?: string
  notes?: string
  
  createdAt: string
  updatedAt: string
}

export interface VoicemailListParams {
  page?: number
  pageSize?: number
  status?: string
  isRead?: boolean
  isImportant?: boolean
  callerNumber?: string
  startDate?: string
  endDate?: string
}

export interface VoicemailStats {
  total: number
  unread: number
  important: number
  today: number
}

export interface BatchOperationRequest {
  ids: number[]
}

export interface TranscribeRequest {
  voicemailId: number
}

export interface SummarizeRequest {
  voicemailId: number
}

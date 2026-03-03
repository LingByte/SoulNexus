// 代接方案类型定义（对应后端 SipUser）

export interface Scheme {
  id: number
  userId?: number
  schemeName: string
  description?: string
  username: string
  
  // AI助手配置
  assistantId?: number
  assistant?: any // Assistant 对象
  autoAnswer: boolean
  autoAnswerDelay: number
  
  // AI回复配置
  openingMessage?: string
  keywordReplies?: KeywordReply[]
  fallbackMessage?: string
  
  // 录音配置
  recordingEnabled: boolean
  recordingMode: 'full' | 'message'
  recordingPath?: string
  
  // 留言配置
  messageEnabled: boolean
  messageDuration: number
  messagePrompt?: string
  
  // 绑定号码
  boundPhoneNumber?: string
  
  // 状态
  isActive: boolean
  enabled: boolean
  
  // 统计
  callCount: number
  messageCount: number
  totalCallDuration: number
  
  createdAt: string
  updatedAt: string
}

export interface KeywordReply {
  keyword: string
  reply: string
}

export interface CreateSchemeRequest {
  schemeName: string
  description?: string
  assistantId?: number
  autoAnswer: boolean
  autoAnswerDelay: number
  openingMessage?: string
  keywordReplies?: KeywordReply[]
  fallbackMessage?: string
  recordingEnabled: boolean
  recordingMode: 'full' | 'message'
  messageEnabled: boolean
  messageDuration: number
  messagePrompt?: string
  boundPhoneNumber?: string
}

export interface UpdateSchemeRequest extends Partial<CreateSchemeRequest> {
  enabled?: boolean
}

export interface SchemeStats {
  totalCalls: number
  todayCalls: number
  avgDuration: number
  successRate: number
}

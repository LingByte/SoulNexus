import request from '@/utils/request'

// AI自动接听相关接口

export interface BindAssistantRequest {
  sipUserId: number
  assistantId: number
  autoAnswer: boolean
  autoAnswerDelay: number
  welcomeMessage?: string
  welcomeAudioUrl?: string
}

export interface AICallSession {
  id: number
  callId: string
  sipUserId: number
  assistantId: number
  status: string
  startTime: string
  endTime?: string
  messages: string
  turnCount: number
  duration: number
  sipUser?: any
  assistant?: any
}

/**
 * 绑定AI助手到SIP用户
 */
export const bindAssistant = (data: BindAssistantRequest) => {
  return request({
    url: '/api/sip/users/bind-assistant',
    method: 'post',
    data,
  })
}

/**
 * 解绑AI助手
 */
export const unbindAssistant = (sipUserId: number) => {
  return request({
    url: `/api/sip/users/${sipUserId}/unbind-assistant`,
    method: 'post',
  })
}

/**
 * 获取AI通话会话列表
 */
export const getAICallSessions = (params?: {
  sipUserId?: number
  assistantId?: number
  limit?: number
}) => {
  return request<AICallSession[]>({
    url: '/api/sip/ai-sessions',
    method: 'get',
    params,
  })
}

/**
 * 获取AI通话会话详情
 */
export const getAICallSessionDetail = (sessionId: number) => {
  return request<AICallSession>({
    url: `/api/sip/ai-sessions/${sessionId}`,
    method: 'get',
  })
}

/**
 * 获取活跃的AI通话会话
 */
export const getActiveAICallSessions = () => {
  return request<AICallSession[]>({
    url: '/api/sip/ai-sessions/active',
    method: 'get',
  })
}

/**
 * 终止AI通话会话
 */
export const terminateAICallSession = (sessionId: number) => {
  return request({
    url: `/api/sip/ai-sessions/${sessionId}/terminate`,
    method: 'post',
  })
}

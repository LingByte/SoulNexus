import { get, post, ApiResponse } from '@/utils/request'

// SIP用户
export interface SipUser {
  id: number
  username: string
  displayName?: string
  alias?: string
  contact?: string
  contactIp?: string
  contactPort?: number
  status: 'registered' | 'unregistered' | 'expired'
  enabled: boolean
  registerCount: number
  callCount: number
  lastRegister?: string
  createdAt: string
  updatedAt: string
}

// 呼出请求
export interface MakeOutgoingCallRequest {
  targetUri: string
  userId?: number
  groupId?: number
  notes?: string
}

// 呼出响应
export interface MakeOutgoingCallResponse {
  callId: string
  status: string
  targetUri: string
}

// 呼出会话
export interface OutgoingSession {
  remoteRtpAddr: string
  callId: string
  targetUri: string
  status: string
  startTime: string
  answerTime?: string
  endTime?: string
  error?: string
}

// 通话记录
export interface SipCall {
  id: number
  callId: string
  direction: 'inbound' | 'outbound'
  status: 'calling' | 'ringing' | 'answered' | 'failed' | 'cancelled' | 'ended'
  fromUsername?: string
  fromUri?: string
  fromIp?: string
  toUsername?: string
  toUri?: string
  toIp?: string
  localRtpAddr?: string
  remoteRtpAddr?: string
  startTime: string
  answerTime?: string
  endTime?: string
  duration: number
  userId?: number
  groupId?: number
  errorCode?: number
  errorMessage?: string
  recordUrl?: string
  metadata?: string
  notes?: string
  createdAt: string
  updatedAt: string
}

// 获取SIP用户列表
export const getSipUsers = async (): Promise<ApiResponse<SipUser[]>> => {
  return get('/sip/users')
}

// 发起呼出
export const makeOutgoingCall = async (data: MakeOutgoingCallRequest): Promise<ApiResponse<MakeOutgoingCallResponse>> => {
  return post('/sip/calls/outgoing', data)
}

// 获取呼出状态
export const getOutgoingCallStatus = async (callId: string): Promise<ApiResponse<OutgoingSession>> => {
  return get(`/sip/calls/outgoing/${callId}`)
}

// 取消呼出
export const cancelOutgoingCall = async (callId: string): Promise<ApiResponse<void>> => {
  return post(`/sip/calls/outgoing/${callId}/cancel`)
}

// 挂断呼出（已接通的通话）
export const hangupOutgoingCall = async (callId: string): Promise<ApiResponse<void>> => {
  return post(`/sip/calls/outgoing/${callId}/hangup`)
}

// 获取通话历史
export const getCallHistory = async (params?: {
  userId?: number
  status?: string
  limit?: number
}): Promise<ApiResponse<SipCall[]>> => {
  return get('/sip/calls', params)
}


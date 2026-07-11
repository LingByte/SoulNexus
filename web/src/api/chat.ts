import { get, post, put, del, ApiResponse } from '@/utils/request'

// 聊天请求参数
export interface ChatRequest {
  agentId: number
  systemPrompt?: string
  speaker?: string
  language?: string
  apiKey?: string
  apiSecret?: string
  personaTag?: string
  temperature?: number
  maxTokens?: number
}

// 聊天响应
export interface ChatResponse {
  sessionId: string
  message: string
}

// 聊天会话日志摘要
export interface ChatSessionLogSummary {
  id: number
  sessionId: string
  agentId: number
  agentName: string
  chatType: string
  preview: string
  createdAt: string
  messageCount?: number // 该 session 下的消息数量
}

// 工具调用信息
export interface ToolCallInfo {
  id: string
  name: string
  arguments: string
}

// LLM Usage 信息
export interface LLMUsage {
  // Request Information
  model: string
  maxTokens?: number
  maxCompletionTokens?: number
  temperature?: number
  topP?: number
  frequencyPenalty?: number
  presencePenalty?: number
  stop?: string[]
  n?: number
  user?: string
  stream: boolean
  seed?: number

  // Response Information
  responseId?: string
  object?: string
  created?: number
  finishReason?: string
  promptTokens: number
  completionTokens: number
  totalTokens: number

  // Context Information
  systemPrompt?: string
  messageCount?: number

  // Timing Information
  startTime?: string // ISO 8601 format
  endTime?: string   // ISO 8601 format
  duration?: number  // Duration in milliseconds
  latencyMs?: number
  ttftMs?: number
  tps?: number
  queueTimeMs?: number
  latency_ms?: number
  ttft_ms?: number
  queue_time_ms?: number

  // Tool Call Information
  hasToolCalls?: boolean
  toolCallCount?: number
  toolCalls?: ToolCallInfo[]
}

// 聊天会话日志详情
export interface ChatSessionLogDetail {
  id: number
  sessionId: string
  agentId: number
  agentName: string
  chatType: string
  userMessage: string
  agentMessage: string
  audioUrl?: string
  duration?: number
  llmUsage?: LLMUsage // LLM使用信息
  createdAt: string
  updatedAt: string
}

// 聊天会话日志列表响应
export interface ChatSessionLogListResponse {
  logs: ChatSessionLogSummary[]
  nextCursor: number
  hasMoreData: boolean
  agentId?: number
}

// ===========================================================================
// Swipe / Branching API
// ===========================================================================

/** Message alternative (sibling reply under a user message) */
export interface MessageAlternative {
  id: string
  content: string
  branchIndex: number
  isActiveBranch: boolean
  tokenCount: number
  model: string
  provider: string
  createdAt: string
}

export interface AlternativesResponse {
  parentId: string
  alternatives: MessageAlternative[]
}

export interface RegenerateResponse {
  id: string
  content: string
  branchIndex: number
  isActiveBranch: boolean
  tokenCount: number
  model: string
  provider: string
  createdAt: string
}

export interface BranchResponse {
  branchSessionId: string
  title: string
  anchorMessageId: string
}

/** 获取某条 user message 下的所有候选 assistant 回复 */
export const getMessageAlternatives = async (messageId: string): Promise<ApiResponse<AlternativesResponse>> => {
  return get(`/chat/messages/${messageId}/alternatives`)
}

/** 切换到指定候选回复 (set as active branch) */
export const activateMessageAlternative = async (messageId: string): Promise<ApiResponse<{ messageId: string; isActiveBranch: boolean }>> => {
  return post(`/chat/messages/${messageId}/activate`)
}

/** 为某条 user message 生成新的候选回复 (regenerate) */
export const regenerateMessage = async (messageId: string, data?: {
  temperature?: number
  maxTokens?: number
  model?: string
}): Promise<ApiResponse<RegenerateResponse>> => {
  return post(`/chat/messages/${messageId}/regenerate`, data || {})
}

/** 从某条消息创建分支 (新时间线) */
export const branchFromMessage = async (messageId: string, data?: {
  title?: string
}): Promise<ApiResponse<BranchResponse>> => {
  return post(`/chat/messages/${messageId}/branch`, data || {})
}

// ===========================================================================
// World Info / Lorebook API
// ===========================================================================

export interface WorldInfoEntry {
  id: string
  agentId: number
  userId: string
  name: string
  content: string
  keys: string           // comma-separated trigger keywords
  regex: string          // optional regex trigger
  selective: boolean     // chain with previous entry
  constant: boolean      // always active
  depth: number          // recursion depth
  order: number          // insertion priority
  enabled: boolean
  groupId: number
  createdAt: string
  updatedAt: string
}

export interface ActivatedEntry {
  id: string
  name: string
  content: string
  order: number
}

/** 列出 World Info 条目 */
export const listWorldInfo = async (agentId?: number): Promise<ApiResponse<{ entries: WorldInfoEntry[]; total: number }>> => {
  const params = agentId ? `?agentId=${agentId}` : ''
  return get(`/chat/world-info${params}`)
}

/** 创建 World Info 条目 */
export const createWorldInfo = async (data: Partial<WorldInfoEntry>): Promise<ApiResponse<{ entry: WorldInfoEntry }>> => {
  return post('/chat/world-info', data)
}

/** 更新 World Info 条目 */
export const updateWorldInfo = async (id: string, data: Partial<WorldInfoEntry>): Promise<ApiResponse<{ entry: WorldInfoEntry }>> => {
  return put(`/chat/world-info/${id}`, data)
}

/** 删除 World Info 条目 */
export const deleteWorldInfo = async (id: string): Promise<ApiResponse<{ message: string }>> => {
  return del(`/chat/world-info/${id}`)
}

/** 激活匹配的 World Info 条目 */
export const activateWorldInfo = async (data: {
  agentId: number
  text: string
  maxScan?: number
}): Promise<ApiResponse<{ activated: ActivatedEntry[]; total: number }>> => {
  return post('/chat/world-info/activate', data)
}

/** 获取格式化的 World Info 注入文本 */
export const injectWorldInfo = async (data: {
  agentId: number
  text: string
  format?: 'prompt' | 'json'
}): Promise<ApiResponse<{ text?: string; entries?: ActivatedEntry[] }>> => {
  return post('/chat/world-info/inject', data)
}

// ===========================================================================
// User Persona API
// ===========================================================================

export interface UserPersona {
  id: string
  userId: string
  name: string
  description: string
  personality: string
  scenario: string
  avatarUrl: string
  isDefault: boolean
  createdAt: string
  updatedAt: string
}

/** 列出用户所有 Persona */
export const listPersonas = async (): Promise<ApiResponse<{ personas: UserPersona[]; total: number }>> => {
  return get('/chat/personas')
}

/** 创建 Persona */
export const createPersona = async (data: Partial<UserPersona>): Promise<ApiResponse<{ persona: UserPersona }>> => {
  return post('/chat/personas', data)
}

/** 更新 Persona */
export const updatePersona = async (id: string, data: Partial<UserPersona>): Promise<ApiResponse<{ persona: UserPersona }>> => {
  return put(`/chat/personas/${id}`, data)
}

/** 删除 Persona */
export const deletePersona = async (id: string): Promise<ApiResponse<{ message: string }>> => {
  return del(`/chat/personas/${id}`)
}

/** 设为默认 Persona */
export const setDefaultPersona = async (id: string): Promise<ApiResponse<{ message: string; personaId: string }>> => {
  return put(`/chat/personas/${id}/default`)
}

/** 获取 Persona 的 prompt 注入文本 */
export const injectPersona = async (id: string): Promise<ApiResponse<{ text: string; persona: UserPersona }>> => {
  return get(`/chat/personas/${id}/inject`)
}

// ===========================================================================
// Legacy API (unchanged)
// ===========================================================================

// 开始聊天会话
export const startChatSession = async (data: ChatRequest): Promise<ApiResponse<ChatResponse>> => {
  return post('/chat/start', data)
}

// 停止聊天会话
export const stopChatSession = async (sessionId: string): Promise<ApiResponse<{ message: string }>> => {
  return post('/chat/stop', { sessionId })
}

// 获取聊天会话日志列表
export const getChatSessionLogs = async (params: {
  pageSize?: number
  cursor?: string
}): Promise<ApiResponse<ChatSessionLogListResponse>> => {
  const queryParams = new URLSearchParams()
  if (params.pageSize) queryParams.append('pageSize', params.pageSize.toString())
  if (params.cursor) queryParams.append('cursor', params.cursor)
  
  const queryString = queryParams.toString()
  return get(`/chat/chat-session-log${queryString ? `?${queryString}` : ''}`)
}

// 获取聊天会话日志详情
export const getChatSessionLogDetail = async (id: number): Promise<ApiResponse<ChatSessionLogDetail>> => {
  return get(`/chat/chat-session-log/${id}`)
}

// 获取指定会话的所有聊天记录
export const getChatSessionLogsBySession = async (sessionId: string): Promise<ApiResponse<ChatSessionLogDetail[]>> => {
  return get(`/chat/chat-session-log/by-session/${sessionId}`)
}

/** @param agentId numeric agent id */
export const getChatSessionLogsByAssistant = async (agentId: number, params: {
  pageSize?: number
  cursor?: string
}): Promise<ApiResponse<ChatSessionLogListResponse>> => {
  const queryParams = new URLSearchParams()
  if (params.pageSize) queryParams.append('pageSize', params.pageSize.toString())
  if (params.cursor) queryParams.append('cursor', params.cursor)
  
  const queryString = queryParams.toString()
  return get(`/chat/chat-session-log/by-agent/${agentId}${queryString ? `?${queryString}` : ''}`)
}

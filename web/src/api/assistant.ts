import { get, post, put, del, ApiResponse } from '@/utils/request'
import { getApiBaseURL } from '@/config/apiConfig'

// 助手创建表单
export interface CreateAssistantForm {
  name: string
  groupId?: number | null // 组织ID，如果设置则创建为组织共享的助手
}

// 助手更新表单
export interface UpdateAssistantForm {
  name?: string
  systemPrompt?: string
  persona_tag?: string
  temperature?: number
  maxTokens?: number
  language?: string
  speaker?: string
  voiceCloneId?: number | null
  ttsProvider?: string
  apiKey?: string
  apiSecret?: string
  llmModel?: string // LLM模型名称
  enableVAD?: boolean // 是否启用VAD
  vadThreshold?: number // VAD阈值
  vadConsecutiveFrames?: number // VAD连续帧数
  jsSourceId?: string // loader.js 客户端 ID（勿与 JS 模板 jsSourceId 混用）
  boundJsTemplateSourceId?: string // 绑定的 JS 模板 jsSourceId
  enableJSONOutput?: boolean
  openingStatement?: string
  // 角色卡字段
  avatarUrl?: string
  description?: string
  personality?: string
  scenario?: string
  exampleDialogues?: string
  tags?: string
  creatorNote?: string
  specVersion?: string
  visibility?: string
}

// 助手信息 - 对应后端Assistant模型的完整字段
export interface Assistant {
  id: number
  userId: number
  groupId?: number | null // 组织ID，如果设置则表示这是组织共享的助手
  name: string
  systemPrompt: string
  personaTag: string
  temperature: number
  maxTokens: number
  language?: string
  jsSourceId: string
  boundJsTemplateSourceId?: string
  speaker?: string
  voiceCloneId?: number | null
  ttsProvider?: string
  apiKey?: string
  apiSecret?: string
  llmModel?: string // LLM模型名称
  enableVAD?: boolean // 是否启用VAD
  vadThreshold?: number // VAD阈值
  vadConsecutiveFrames?: number // VAD连续帧数
  createdAt: string
  updatedAt: string
  enableJSONOutput?: boolean
  openingStatement?: string
  // 角色卡字段
  avatarUrl?: string
  description?: string
  personality?: string
  scenario?: string
  exampleDialogues?: string
  tags?: string
  creatorNote?: string
  specVersion?: string
  visibility?: string
  downloadCount?: number
  rating?: number
  ratingCount?: number
  forkedFrom?: number
}

// 助手列表项 - 对应ListAssistants返回的字段
export interface AssistantListItem {
  id: number
  userId?: number
  groupId?: number | null
  name: string
  jsSourceId?: string
  personaTag?: string
  temperature?: number
  maxTokens?: number
  createdAt?: string
  updatedAt?: string
  // 角色卡字段（列表展示用）
  avatarUrl?: string
  description?: string
  tags?: string
  visibility?: string
  downloadCount?: number
  rating?: number
  specVersion?: string
  llmModel?: string
  speaker?: string
}

// 创建助手
export const createAssistant = async (data: CreateAssistantForm): Promise<ApiResponse<Assistant>> => {
  return post('/agents/add', data)
}

// 获取助手列表
export const getAssistantList = async (): Promise<ApiResponse<AssistantListItem[]>> => {
  return get('/agents')
}

// 获取助手详情
export const getAssistant = async (id: number): Promise<ApiResponse<Assistant>> => {
  return get(`/agents/${id}`)
}

// 更新助手
export const updateAssistant = async (id: number, data: UpdateAssistantForm): Promise<ApiResponse<Assistant>> => {
  return put(`/agents/${id}`, data)
}

// 删除助手
export const deleteAssistant = async (id: number): Promise<ApiResponse<null>> => {
  return del(`/agents/${id}`)
}

// 语音相关接口
export interface VoiceClone {
  id: number
  voice_name: string
  voice_description?: string
}

export interface OneShotRequest {
  agentId: number
  language?: string
  speaker?: string
  voiceCloneId?: number
  temperature?: number
  systemPrompt?: string
}

export interface OneShotTextV2Request {
  apiKey: string
  apiSecret: string
  text: string
  agentId?: number
  language?: string
  sessionId?: string
  systemPrompt?: string
  speaker?: string      // 音色编码
  voiceCloneId?: number // 训练音色ID（优先级高于speaker）
  temperature?: number  // 生成多样性 (0-2)
  maxTokens?: number   // 最大回复长度
  attachmentContent?: string
  attachmentName?: string
}

export interface OneShotResponse {
  text: string
  audioUrl?: string
  requestId?: string
}

// 获取用户音色列表
export const getVoiceClones = async (): Promise<ApiResponse<VoiceClone[]>> => {
  return get('/voice/clones')
}

// 一句话模式 - 文本输入（带TTS合成）
export const oneShotText = async (data: OneShotTextV2Request): Promise<ApiResponse<OneShotResponse>> => {
  return post('/voice/oneshot_text', data)
}

// 纯文本对话 - 文本输入（不进行TTS合成，用于调试）
export const plainText = async (data: OneShotTextV2Request): Promise<ApiResponse<{ text: string }>> => {
  return post('/voice/plain_text', data)
}

export interface ParsedAttachmentResult {
  fileName: string
  fileType: string
  content: string
}

export const parseAttachment = async (file: File): Promise<ApiResponse<ParsedAttachmentResult>> => {
  const formData = new FormData()
  formData.append('file', file)
  return post('/voice/parse_attachment', formData)
}

// 纯文本对话 - 流式接收（SSE）
export const plainTextStream = async (
  data: OneShotTextV2Request,
  onChunk: (text: string) => void,
  onComplete?: () => void,
  onError?: (error: string) => void
): Promise<void> => {
  try {
    // 获取API基础URL
    const baseURL = getApiBaseURL()
    const token = localStorage.getItem('auth_token') || localStorage.getItem('token') || ''
    
    const response = await fetch(`${baseURL}/voice/plain_text`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify(data),
    })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ msg: '请求失败' }))
      onError?.(errorData.msg || '请求失败')
      return
    }

    const reader = response.body?.getReader()
    if (!reader) {
      onError?.('无法读取响应流')
      return
    }

    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      
      if (done) {
        onComplete?.()
        break
      }

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const dataStr = line.slice(6).trim()
          if (dataStr === '[DONE]' || dataStr === '{"done": true}') {
            onComplete?.()
            return
          }

          try {
            const jsonData = JSON.parse(dataStr)
            if (jsonData.error) {
              onError?.(jsonData.error)
              return
            }
            if (jsonData.text) {
              onChunk(jsonData.text)
            }
          } catch (e) {
            // 忽略解析错误，继续处理下一行
            console.warn('Failed to parse SSE data:', dataStr, e)
          }
        }
      }
    }
  } catch (error: any) {
    onError?.(error.message || '流式请求失败')
  }
}

// 获取音频处理状态
export const getAudioStatus = async (requestId: string): Promise<ApiResponse<{ status: string; audioUrl?: string; text?: string }>> => {
  return get('/voice/audio_status', { params: { requestId } })
}

// 音色选项接口
export interface VoiceOption {
  id: string          // 音色编码
  name: string        // 音色名称
  description: string // 音色描述
  type: string        // 音色类型（男声/女声/童声等）
  language: string    // 支持的语言
  sampleRate?: string  // 音色采样率
  emotion?: string     // 音色情感
  scene?: string       // 推荐场景
}

export interface VoiceOptionsResponse {
  provider: string
  voices: VoiceOption[]
}

// 根据TTS Provider获取音色列表
export const getVoiceOptions = async (provider: string): Promise<ApiResponse<VoiceOptionsResponse>> => {
  return get('/voice/options', { params: { provider } })
}

// 语言选项接口
export interface LanguageOption {
  code: string        // 语言代码，如 zh-CN, en-US
  name: string        // 语言名称，如 中文、English
  nativeName: string  // 本地名称，如 中文、English
  configKey: string  // 配置字段名（不同平台可能不同），如 language, languageCode, lan
  description: string // 语言描述
}

export interface LanguageOptionsResponse {
  provider: string
  languages: LanguageOption[]
}

// 根据TTS Provider获取支持的语言列表
export const getLanguageOptions = async (provider: string): Promise<ApiResponse<LanguageOptionsResponse>> => {
  return get('/voice/language-options', { params: { provider } })
}

// 获取 FishSpeech 音色列表（自动从用户凭证中获取 API Key）
export interface FishSpeechVoicesResponse {
  provider: string
  voices: VoiceOption[]
}

export const getFishSpeechVoices = async (): Promise<ApiResponse<FishSpeechVoicesResponse>> => {
  return get('/voice/voice-options', { params: { provider: 'fishspeech' } })
}

// ==================== 角色卡导入/导出/头像 ====================

export interface ImportCharacterCardResponse {
  id: number
  name: string
  message?: string
}

// 导入角色卡（文件上传）
export const importCharacterCard = async (
  file: File,
  groupId?: number
): Promise<ApiResponse<ImportCharacterCardResponse>> => {
  const formData = new FormData()
  formData.append('file', file)
  if (groupId) formData.append('groupId', String(groupId))
  return post('/agents/import', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

// 导出角色卡（JSON 或 PNG）
export const exportCharacterCard = async (
  id: number,
  format: 'json' | 'png' = 'json'
): Promise<Blob> => {
  const baseURL = getApiBaseURL()
  const token = localStorage.getItem('auth_token') || localStorage.getItem('token') || ''
  const res = await fetch(`${baseURL}/agents/${id}/export?format=${format}`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  })
  if (!res.ok) throw new Error('导出失败')
  return res.blob()
}

// 上传智能体头像
export const uploadAgentAvatar = async (id: number, file: File): Promise<ApiResponse<{ avatarUrl: string }>> => {
  const formData = new FormData()
  formData.append('avatar', file)
  return post(`/agents/${id}/avatar`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

// ==================== 角色市场 API ====================

export interface MarketListParams {
  page?: number
  pageSize?: number
  search?: string
  sortBy?: 'download_count' | 'rating' | 'created_at'
}

export interface MarketAgent extends AssistantListItem {
  rating?: number
  ratingCount?: number
  downloadCount?: number
  forkedFrom?: number
}

export interface MarketListResponse {
  agents?: MarketAgent[]
  items?: MarketAgent[]
  total: number
  page: number
  pageSize: number
}

// 浏览市场角色（无需登录）
export const listMarketAgents = async (params?: MarketListParams): Promise<ApiResponse<MarketListResponse>> => {
  return get('/market/agents', { params })
}

// 查看公开角色详情（无需登录）
export const getMarketAgent = async (id: number): Promise<ApiResponse<Assistant>> => {
  return get(`/market/agents/${id}`)
}

// Fork 角色到当前组织（需登录）
export const forkMarketAgent = async (id: number): Promise<ApiResponse<{ id: number; name: string }>> => {
  return post(`/market/agents/${id}/fork`)
}

// 评分（需登录）
export const rateMarketAgent = async (id: number, rating: number): Promise<ApiResponse<{ rating: number; ratingCount: number }>> => {
  return post(`/market/agents/${id}/rate`, { score: rating })
}

export interface ShareInfo {
  url: string
  title: string
  description: string
  avatar: string
  rating: number
  ratingCount: number
  downloadCount: number
  agentId: number
}

// 获取分享链接（无需登录）
export const getMarketShareInfo = async (id: number): Promise<ApiResponse<ShareInfo>> => {
  return get(`/market/agents/${id}/share`)
}

// ==================== 预设模板 API ====================

export type PresetType = 'agent' | 'system_prompt' | 'voice' | 'knowledge'

export interface PresetTemplate {
  id: string
  groupId: number
  createdBy: number
  name: string
  description: string
  type: PresetType
  category: string
  tags: string
  visibility: 'private' | 'group' | 'public'
  content: string // JSON string
  useCount: number
  isBuiltin: boolean
  status: string
  createdAt: string
  updatedAt: string
}

export interface PresetListParams {
  type?: PresetType
  category?: string
  visibility?: string
  keyword?: string
  page?: number
  pageSize?: number
}

export interface PresetListResult {
  list: PresetTemplate[]
  total: number
  page: number
  pageSize: number
  totalPage: number
}

export interface CreatePresetReq {
  name: string
  description?: string
  type: PresetType
  category?: string
  tags?: string
  visibility?: string
  content: string
}

export interface UpdatePresetReq {
  name?: string
  description?: string
  category?: string
  tags?: string
  visibility?: string
  content?: string
}

export interface ApplyPresetReq {
  presetId: number
  variables?: Record<string, string>
  agentId?: number
}

export interface SystemPromptPresetPayload {
  systemPrompt: string
  personaTag?: string
  variables?: PresetVariable[]
}

export interface PresetVariable {
  name: string
  label: string
  defaultVal: string
  description: string
  required: boolean
}

// 列出可用预设模板（可选登录）
export const listPresets = async (params?: PresetListParams): Promise<ApiResponse<PresetListResult>> => {
  return get('/presets', { params })
}

// 获取单个预设模板详情（可选登录）
export const getPreset = async (id: number): Promise<ApiResponse<PresetTemplate>> => {
  return get(`/presets/${id}`)
}

// 创建新预设（需登录）
export const createPreset = async (data: CreatePresetReq): Promise<ApiResponse<PresetTemplate>> => {
  return post('/presets', data)
}

// 更新预设（需登录）
export const updatePreset = async (id: number, data: UpdatePresetReq): Promise<ApiResponse<PresetTemplate>> => {
  return put(`/presets/${id}`, data)
}

// 删除（归档）预设（需登录）
export const deletePreset = async (id: number): Promise<ApiResponse<null>> => {
  return del(`/presets/${id}`)
}

// 应用模板到 Agent（需登录）
export const applyPreset = async (id: number, data: ApplyPresetReq): Promise<ApiResponse<{ agentId?: number; systemPrompt?: string }>> => {
  return post(`/presets/${id}/apply`, data)
}

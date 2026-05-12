import { get, post, put, del, ApiResponse } from '@/utils/request'
import { getApiBaseURL } from '@/config/apiConfig'

// 助手创建表单
export interface CreateAssistantForm {
  name: string
  description?: string
  icon?: string
  groupId?: number | null // 组织ID，如果设置则创建为组织共享的助手
}

// 助手更新表单
export interface UpdateAssistantForm {
  name?: string
  description?: string
  icon?: string
  systemPrompt?: string
  persona_tag?: string
  temperature?: number
  maxTokens?: number
  speaker?: string
  voiceCloneId?: number | null
  ttsProvider?: string
  apiKey?: string
  apiSecret?: string
  llmModel?: string // LLM模型名称
  enableVAD?: boolean // 是否启用VAD
  vadThreshold?: number // VAD阈值
  vadConsecutiveFrames?: number // VAD连续帧数
  jsSourceId?: string // JS模板ID
    enableJSONOutput?: boolean
}

// 助手信息 - 对应后端Assistant模型的完整字段
export interface Assistant {
  id: number
  userId: number
  groupId?: number | null // 组织ID，如果设置则表示这是组织共享的助手
  name: string
  description: string
  icon: string
  systemPrompt: string
  personaTag: string
  temperature: number
  maxTokens: number
  jsSourceId: string
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
}

// 助手列表项 - 对应ListAssistants返回的字段
export interface AssistantListItem {
  id: number
  userId?: number
  groupId?: number | null
  name: string
  icon: string
  description: string
  jsSourceId?: string
  personaTag?: string
  temperature?: number
  maxTokens?: number
  createdAt?: string
  updatedAt?: string
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

// 上传 Agent 图标，返回可直接落到 agent.icon 字段的 URL。
export const uploadAssistantIcon = async (file: File): Promise<ApiResponse<{ url: string; key: string }>> => {
  const fd = new FormData()
  fd.append('icon', file)
  return post('/agents/icon/upload', fd, { headers: { 'Content-Type': 'multipart/form-data' } } as any)
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


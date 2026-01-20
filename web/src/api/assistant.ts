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
  language?: string
  speaker?: string
  voiceCloneId?: number | null
  knowledgeBaseId?: string | null
  ttsProvider?: string
  apiKey?: string
  apiSecret?: string
  llmModel?: string // LLM模型名称
  enableGraphMemory?: boolean
  enableVAD?: boolean // 是否启用VAD
  vadThreshold?: number // VAD阈值
  vadConsecutiveFrames?: number // VAD连续帧数
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
  language?: string
  speaker?: string
  voiceCloneId?: number | null
  knowledgeBaseId?: string | null
  ttsProvider?: string
  apiKey?: string
  apiSecret?: string
  llmModel?: string // LLM模型名称
  enableGraphMemory?: boolean // 是否启用基于图数据库的长期记忆
  enableVAD?: boolean // 是否启用VAD
  vadThreshold?: number // VAD阈值
  vadConsecutiveFrames?: number // VAD连续帧数
  createdAt: string
  updatedAt: string
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
  return post('/assistant/add', data)
}

// 获取助手列表
export const getAssistantList = async (): Promise<ApiResponse<AssistantListItem[]>> => {
  return get('/assistant')
}

// 获取助手详情
export const getAssistant = async (id: number): Promise<ApiResponse<Assistant>> => {
  return get(`/assistant/${id}`)
}

// 更新助手
export const updateAssistant = async (id: number, data: UpdateAssistantForm): Promise<ApiResponse<Assistant>> => {
  return put(`/assistant/${id}`, data)
}

// 更新助手JS模板
export const updateAssistantJS = async (id: number, jsSourceId: string): Promise<ApiResponse<any>> => {
  return put(`/assistant/${id}/js`, { jsSourceId })
}

// 删除助手
export const deleteAssistant = async (id: number): Promise<ApiResponse<null>> => {
  return del(`/assistant/${id}`)
}

// 语音相关接口
export interface VoiceClone {
  id: number
  voice_name: string
  voice_description?: string
}

export interface OneShotRequest {
  assistantId: number
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
  assistantId?: number
  language?: string
  sessionId?: string
  systemPrompt?: string
  speaker?: string      // 音色编码
  voiceCloneId?: number // 训练音色ID（优先级高于speaker）
  knowledgeBaseId?: string // 知识库ID（可选）
  temperature?: number  // 生成多样性 (0-2)
  maxTokens?: number   // 最大回复长度
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

// ========== Assistant Tools 相关接口 ==========

// 助手工具接口
export interface AssistantTool {
  id: number
  assistantId: number
  name: string
  description: string
  parameters: string // JSON Schema格式
  code?: string // 可选的代码标识
  webhookUrl?: string // Webhook URL（用于自定义工具执行）
  enabled: boolean
  createdAt: string
  updatedAt: string
}

// 创建工具表单
export interface CreateToolForm {
  name: string
  description: string
  parameters: string // JSON Schema格式
  code?: string
  webhookUrl?: string // Webhook URL（用于自定义工具执行）
  enabled?: boolean
}

// 更新工具表单
export interface UpdateToolForm {
  name?: string
  description?: string
  parameters?: string
  code?: string
  webhookUrl?: string // Webhook URL（用于自定义工具执行）
  enabled?: boolean
}

// 获取助手的所有工具
export const getAssistantTools = async (assistantId: number): Promise<ApiResponse<AssistantTool[]>> => {
  return get(`/assistant/${assistantId}/tools`)
}

// 创建工具
export const createAssistantTool = async (assistantId: number, data: CreateToolForm): Promise<ApiResponse<AssistantTool>> => {
  return post(`/assistant/${assistantId}/tools`, data)
}

// 更新工具
export const updateAssistantTool = async (assistantId: number, toolId: number, data: UpdateToolForm): Promise<ApiResponse<AssistantTool>> => {
  return put(`/assistant/${assistantId}/tools/${toolId}`, data)
}

// 删除工具
export const deleteAssistantTool = async (assistantId: number, toolId: number): Promise<ApiResponse<null>> => {
  return del(`/assistant/${assistantId}/tools/${toolId}`)
}

// 测试工具
export interface TestToolRequest {
  args: Record<string, any>
}

export interface TestToolResponse {
  result: string
  tool: AssistantTool
}

export const testAssistantTool = async (assistantId: number, toolId: number, args: Record<string, any>): Promise<ApiResponse<TestToolResponse>> => {
  return post(`/assistant/${assistantId}/tools/${toolId}/test`, { args })
}

// ========== Assistant Graph Data 相关接口 ==========

// 图节点
export interface GraphNode {
  id: string
  label: string
  type: string // Assistant, User, Conversation, Topic, Intent, Knowledge等
  props: Record<string, any>
}

// 图边（关系）
export interface GraphEdge {
  id: string
  source: string
  target: string
  type: string // HAS_CONVERSATION, WITH_ASSISTANT, DISCUSSES等
  props: Record<string, any>
}

// 图统计信息
export interface GraphStats {
  totalNodes: number
  totalEdges: number
  usersCount: number
  conversationsCount: number
  topicsCount: number
  intentsCount: number
  knowledgeCount: number
}

// 助手图数据
export interface AssistantGraphData {
  assistantId: number
  nodes: GraphNode[]
  edges: GraphEdge[]
  stats: GraphStats
}

// 获取助手图数据
export const getAssistantGraphData = async (assistantId: number): Promise<ApiResponse<AssistantGraphData>> => {
  return get(`/assistant/${assistantId}/graph`)
}
export type PlaygroundProtocol = 'openai' | 'anthropic'

export type PlaygroundRole = 'user' | 'assistant' | 'system'

export interface PlaygroundMessage {
  id: string
  role: PlaygroundRole
  content: string
  createdAt: number
  streaming?: boolean
  error?: string
  /** 推理阶段（thinking 开启且尚未输出正文） */
  thinking?: boolean
  usage?: PlaygroundMetrics
  rawResponse?: string
}

export interface PlaygroundMetrics {
  model?: string
  provider?: string
  input_tokens?: number
  output_tokens?: number
  total_tokens?: number
  latency_ms?: number
  ttft_ms?: number
  duration_ms?: number
}

export interface PlaygroundSettings {
  protocol: PlaygroundProtocol
  baseUrl: string
  apiKey: string
  anthropicVersion: string
  anthropicBeta: string
  model: string
  systemPrompt: string
  stream: boolean
  maxTokens: number
  temperature: number
  topP: number
  presencePenalty: number
  frequencyPenalty: number
  stopSequences: string
  seed: string
  n: number
  responseFormat: '' | 'json_object' | 'text'
  user: string
  enableThinking: boolean
  thinkingBudget: number
  anthropicThinkingEnabled: boolean
  anthropicThinkingBudget: number
  extraJson: string
  showRaw: boolean
}

export const DEFAULT_PLAYGROUND_SETTINGS: PlaygroundSettings = {
  protocol: 'openai',
  baseUrl: '',
  apiKey: '',
  anthropicVersion: '2023-06-01',
  anthropicBeta: '',
  model: 'qwen-plus',
  systemPrompt: '',
  stream: true,
  maxTokens: 1024,
  temperature: 0.7,
  topP: 1,
  presencePenalty: 0,
  frequencyPenalty: 0,
  stopSequences: '',
  seed: '',
  n: 1,
  responseFormat: '',
  user: '',
  enableThinking: false,
  thinkingBudget: 1024,
  anthropicThinkingEnabled: false,
  anthropicThinkingBudget: 1024,
  extraJson: '',
  showRaw: false,
}

export const PLAYGROUND_STORAGE_KEY = 'soulnexus_playground_v1'

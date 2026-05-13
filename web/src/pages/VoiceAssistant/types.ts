import type { LLMUsage } from '@/api/chat'

export interface Assistant {
  id: number
  name: string
  description: string
  jsSourceId: string
  language?: string
  active?: boolean
}

export interface ChatMessage {
  type: 'user' | 'agent'
  content: string
  timestamp: string
  id?: string
  audioUrl?: string
  isLoading?: boolean
  /** 来自会话日志恢复时附带，用于展示 LLM 详情 */
  llmUsage?: LLMUsage
}

export interface VoiceChatSession {
  id: number
  sessionId?: string
  content: string
  createdAt: string
  agentName?: string
  assistantName?: string // legacy client field
  chatType?: string
  messageCount?: number
}

export interface OnboardingStep {
  element: string
  text: string
  position: 'top' | 'bottom' | 'left' | 'right'
  isLast?: boolean
}

export type LineMode = 'webrtc' | 'websocket'


import type { LLMUsage } from '@/api/chat'

export interface Assistant {
  id: number
  name: string
  jsSourceId?: string
  boundJsTemplateSourceId?: string
  language?: string
  active?: boolean
}

/** MessageAlternative — a single candidate reply under a user message */
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

export interface ChatMessage {
  type: 'user' | 'agent'
  content: string
  timestamp: string
  id?: string
  audioUrl?: string
  isLoading?: boolean
  /** 来自会话日志恢复时附带，用于展示 LLM 详情 */
  llmUsage?: LLMUsage
  /** Swipe/Branch support */
  parentId?: string           // parent user message ID (for assistant messages)
  branchIndex?: number        // ordinal among siblings (0 = first)
  isActiveBranch?: boolean    // currently selected branch
  alternatives?: MessageAlternative[] // all sibling alternatives
  sessionId?: string          // for branch session reference
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


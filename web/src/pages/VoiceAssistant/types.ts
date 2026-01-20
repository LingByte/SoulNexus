export interface Assistant {
  id: number
  name: string
  description: string
  icon: string
  jsSourceId: string
  active?: boolean
}

export interface ChatMessage {
  type: 'user' | 'agent'
  content: string
  timestamp: string
  id?: string
  audioUrl?: string
}

export interface VoiceChatSession {
  id: number
  content: string
  createdAt: string
  assistantName?: string
  chatType?: string
}

export interface OnboardingStep {
  element: string
  text: string
  position: 'top' | 'bottom' | 'left' | 'right'
  isLast?: boolean
}

export type LineMode = 'webrtc' | 'websocket'


import type { ChatSessionLogDetail } from '@/api/chat'
import type { ChatMessage } from '@/pages/VoiceAssistant/types'

/**
 * 将会话日志（按条存的 user/assistant 轮次）展开为聊天区消息列表，用于恢复会话并继续对话。
 */
export function chatSessionLogsToChatMessages(logs: ChatSessionLogDetail[]): ChatMessage[] {
  if (!logs?.length) return []
  const sorted = [...logs].sort(
    (a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
  )
  const out: ChatMessage[] = []
  for (const log of sorted) {
    const ts = log.createdAt || new Date().toISOString()
    if (log.userMessage?.trim()) {
      out.push({
        type: 'user',
        content: log.userMessage.trim(),
        timestamp: ts,
        id: `resume-${log.id}-user`,
      })
    }
    if (log.agentMessage?.trim()) {
      out.push({
        type: 'agent',
        content: log.agentMessage.trim(),
        timestamp: ts,
        id: `resume-${log.id}-agent`,
        audioUrl: log.audioUrl,
        llmUsage: log.llmUsage,
      })
    }
  }
  return out
}

import React from 'react'
import { Spin, Empty } from '@arco-design/web-react'
import { Bot, MessageSquare, Volume2 } from 'lucide-react'
import { cn } from '@/utils/cn'
import { sanitizeReadableText } from '@/utils/string'

interface VoiceChatSession {
  id: number
  sessionId?: string
  content: string
  createdAt: string
  agentName?: string
  assistantName?: string
  chatType?: string
  messageCount?: number
}

interface ChatHistoryProps {
  chatHistory: VoiceChatSession[]
  loading: boolean
  error: string | null
  onSessionClick: (logId: number, sessionId?: string) => void
  className?: string
}

const chatTypeMap: Record<string, { icon: React.ReactNode; text: string; color: string }> = {
  realtime: { icon: <Volume2 className="w-3 h-3" />, text: '实时', color: '#7c3aed' },
  press:    { icon: <Volume2 className="w-3 h-3" />, text: '按住', color: '#0ea5e9' },
  text:     { icon: <MessageSquare className="w-3 h-3" />, text: '文本', color: '#16a34a' },
}

const ChatHistory: React.FC<ChatHistoryProps> = ({
  chatHistory,
  loading,
  error,
  onSessionClick,
  className = '',
}) => {
  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center py-10">
        <Spin size={20} tip="加载中..." />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center py-6 px-3">
        <span className="text-xs text-red-500">{error}</span>
      </div>
    )
  }

  if (chatHistory.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center py-10 px-3">
        <Empty
          icon={<Bot className="w-8 h-8 text-gray-300 dark:text-gray-600" />}
          description={<span className="text-xs text-gray-400 dark:text-gray-500">暂无历史会话</span>}
        />
      </div>
    )
  }

  return (
    <div className={cn('flex-1 overflow-y-auto px-2 py-2 space-y-1 custom-scrollbar', className)}>
      {chatHistory.map((session) => {
        const typeInfo = chatTypeMap[session.chatType ?? ''] ?? {
          icon: <MessageSquare className="w-3 h-3" />,
          text: '聊',
          color: '#6b7280',
        }
        const summary = sanitizeReadableText(session.content) || '…'

        return (
          <button
            key={session.id}
            type="button"
            onClick={() => onSessionClick(session.id, session.sessionId)}
            className={cn(
              'w-full text-left rounded-lg px-3 py-2 border border-transparent',
              'hover:bg-gray-50 dark:hover:bg-neutral-700/80 hover:border-gray-200 dark:hover:border-neutral-600',
              'transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-purple-500/40',
            )}
          >
            <div className="flex items-start gap-2 min-w-0">
              <div
                className="mt-0.5 shrink-0 w-5 h-5 rounded flex items-center justify-center"
                style={{ background: typeInfo.color + '18', color: typeInfo.color }}
              >
                {typeInfo.icon}
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-xs text-gray-800 dark:text-gray-100 line-clamp-2 leading-snug">
                  {summary}
                </p>
                <div className="mt-1 flex items-center justify-between gap-1">
                  <span
                    className="text-[10px] font-medium px-1.5 py-0.5 rounded"
                    style={{ background: typeInfo.color + '18', color: typeInfo.color }}
                  >
                    {typeInfo.text}
                  </span>
                  {session.messageCount != null && session.messageCount >= 2 && (
                    <span className="text-[10px] tabular-nums text-gray-400 dark:text-gray-500">
                      {session.messageCount} 条
                    </span>
                  )}
                </div>
              </div>
            </div>
          </button>
        )
      })}
    </div>
  )
}

export default ChatHistory

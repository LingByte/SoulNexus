import React from 'react'
import { motion } from 'framer-motion'
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

const ChatHistory: React.FC<ChatHistoryProps> = ({
  chatHistory,
  loading,
  error,
  onSessionClick,
  className = '',
}) => {
  const getChatTypeInfo = (chatType?: string) => {
    switch (chatType) {
      case 'realtime':
        return { icon: <MessageSquare className="w-3 h-3" />, text: '实时' }
      case 'press':
        return { icon: <Volume2 className="w-3 h-3" />, text: '按住' }
      case 'text':
        return { icon: <MessageSquare className="w-3 h-3" />, text: '文本' }
      default:
        return { icon: <MessageSquare className="w-3 h-3" />, text: '聊' }
    }
  }

  return (
    <div className={cn('flex-1 overflow-y-auto px-2 py-2 space-y-1 custom-scrollbar', className)}>
      {loading && (
        <div className="text-center text-xs text-gray-500 py-4">加载中...</div>
      )}

      {error && (
        <div className="text-center text-xs text-red-600 py-4">{error}</div>
      )}

      {chatHistory.length === 0 && !loading && (
        <div className="text-center text-gray-400 dark:text-gray-500 py-10 px-2">
          <Bot className="w-7 h-7 mx-auto mb-2 opacity-40" />
          <p className="text-xs font-medium text-gray-600 dark:text-gray-400">暂无历史会话</p>
        </div>
      )}

      {chatHistory.length > 0 &&
        !loading &&
        chatHistory.map((session, index) => {
          const chatTypeInfo = getChatTypeInfo(session.chatType)
          const summary = sanitizeReadableText(session.content) || '…'

          return (
            <motion.button
              type="button"
              key={session.id}
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: Math.min(index * 0.02, 0.2) }}
              onClick={() => onSessionClick(session.id, session.sessionId)}
              className={cn(
                'w-full text-left rounded-lg px-2 py-1.5 border border-transparent',
                'hover:bg-gray-100 dark:hover:bg-neutral-700/80 hover:border-gray-200 dark:hover:border-neutral-600',
                'transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-purple-500/40',
              )}
            >
              <div className="flex items-start gap-2 min-w-0">
                <span className="mt-0.5 shrink-0 text-gray-500 dark:text-gray-400">{chatTypeInfo.icon}</span>
                <div className="min-w-0 flex-1">
                  <p className="text-xs font-medium text-gray-800 dark:text-gray-100 line-clamp-3 leading-snug">
                    {summary}
                  </p>
                  <div className="mt-1 flex items-center justify-between gap-1.5">
                    <span className="text-[10px] text-gray-400 dark:text-gray-500">{chatTypeInfo.text}</span>
                    {session.messageCount != null && session.messageCount >= 2 && (
                      <span className="shrink-0 text-[10px] font-mono tabular-nums text-gray-400 dark:text-gray-500">
                        {session.messageCount}+
                      </span>
                    )}
                  </div>
                </div>
              </div>
            </motion.button>
          )
        })}
    </div>
  )
}

export default ChatHistory

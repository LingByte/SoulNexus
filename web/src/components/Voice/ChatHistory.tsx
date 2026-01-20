import React from 'react'
import { motion } from 'framer-motion'
import { Bot, MessageSquare, Volume2 } from 'lucide-react'
import { cn } from '@/utils/cn'

interface VoiceChatSession {
  id: number
  sessionId?: string
  content: string
  createdAt: string
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
  className = ''
}) => {
    const getChatTypeInfo = (chatType?: string) => {
    switch (chatType) {
      case 'realtime':
        return { icon: <MessageSquare className="w-3 h-3" />, text: '实时通话' }
      case 'press':
        return { icon: <Volume2 className="w-3 h-3" />, text: '按住说话' }
      case 'text':
        return { icon: <MessageSquare className="w-3 h-3" />, text: '文本聊天' }
      default:
        return { icon: <MessageSquare className="w-3 h-3" />, text: '聊天' }
    }
  }

  return (
    <div className={cn('flex-1 overflow-y-auto p-4 space-y-4 custom-scrollbar', className)}>
      {loading && (
        <div className="text-center text-gray-500">
          <p>加载中...</p>
        </div>
      )}

      {error && (
        <div className="text-center text-red-600">
          <p>{error}</p>
        </div>
      )}

      {chatHistory.length === 0 && !loading && (
        <div className="text-center text-gray-400 dark:text-gray-500 mt-20">
          <div className="flex justify-center mb-6">
            <div className="w-16 h-16 bg-gradient-to-br from-purple-100 to-indigo-100 dark:from-purple-900/30 dark:to-indigo-900/30 rounded-2xl flex items-center justify-center">
              <Bot className="w-8 h-8 text-purple-500 dark:text-purple-400" />
            </div>
          </div>
          <h3 className="text-lg font-semibold text-gray-600 dark:text-gray-300 mb-2">暂无历史会话</h3>
          <p className="text-sm text-gray-500 dark:text-gray-400 max-w-sm mx-auto">
            开始与AI助手对话，您的聊天记录将在这里显示
          </p>
        </div>
      )}

      {chatHistory.length > 0 && !loading && chatHistory.map((session, index) => {
        const chatTypeInfo = getChatTypeInfo(session.chatType)

        return (
          <motion.div
            key={session.id}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.1 }}
            onClick={() => onSessionClick(session.id, session.sessionId)}
            className="group p-4 cursor-pointer hover:bg-gradient-to-r hover:from-purple-50 hover:to-indigo-50 dark:hover:from-purple-900/20 dark:hover:to-indigo-900/20 rounded-xl transition-all duration-200 border border-gray-200 dark:border-neutral-600 hover:border-purple-200 dark:hover:border-purple-700 hover:shadow-md"
          >
            <div className="flex items-start justify-between mb-3">
              {/* 左侧部分，图标 + 助手名称和类型 */}
              <div className="flex items-center space-x-3">
                <div className="flex-shrink-0 w-8 h-8 bg-gradient-to-br from-purple-500 to-indigo-600 rounded-lg flex items-center justify-center">
                  {chatTypeInfo.icon}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center space-x-2">
                    <span className="font-medium text-gray-900 dark:text-white text-xs truncate max-w-[60px]" title={session.assistantName || '未知助手'}>
                      {session.assistantName && session.assistantName.length > 5 
                        ? `${session.assistantName.slice(0, 5)}...` 
                        : (session.assistantName || '未知助手')}
                    </span>
                    <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300">
                      {chatTypeInfo.text}
                    </span>
                  </div>
                </div>
              </div>
            </div>
            <div className="space-y-2">
              <div className="text-sm text-gray-700 dark:text-gray-300 line-clamp-2 leading-relaxed">
                {session.content || '新会话'}
              </div>

              {/* 添加一个小的指示器 */}
              <div className="flex items-center justify-between text-xs text-gray-400 dark:text-gray-500">
                <div className="flex items-center">
                  <div className="w-1 h-1 bg-purple-400 rounded-full mr-2"></div>
                  <span>点击查看完整对话</span>
                </div>
                {session.messageCount && session.messageCount > 1 && (
                  <span className="px-2 py-0.5 bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300 rounded-full text-xs font-medium">
                    {session.messageCount} 条消息
                  </span>
                )}
              </div>
            </div>
          </motion.div>
        )
      })}
    </div>
  )
}

export default ChatHistory

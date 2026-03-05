import React, { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { ChevronDown, Bot, MessageCircle, Users, Zap, Circle, User } from 'lucide-react'

interface LLMUsage {
  model: string
  totalTokens: number
  promptTokens: number
  completionTokens: number
  duration?: number
  hasToolCalls?: boolean
  toolCallCount?: number
  toolCalls?: ToolCallInfo[]
}

interface ToolCallInfo {
  name: string
  arguments?: string
}

interface ChatLog {
  id?: string
  userMessage?: string
  agentMessage?: string
  createdAt: string
  llmUsage?: LLMUsage
  assistantName?: string
}

interface ChatLogDetailProps {
  logs: ChatLog[]
  assistantName?: string
  assistantIcon?: string
  onClose: () => void
}

const ICON_MAP = {
  Bot: <Bot className="w-5 h-5" />,
  MessageCircle: <MessageCircle className="w-5 h-5" />,
  Users: <Users className="w-5 h-5" />,
  Zap: <Zap className="w-5 h-5" />,
  Circle: <Circle className="w-5 h-5" />
}

const ChatLogDetail: React.FC<ChatLogDetailProps> = ({
  logs,
  assistantName = 'AI助手',
  assistantIcon,
  onClose,
}) => {
  const [expandedLLMUsage, setExpandedLLMUsage] = useState<Set<number>>(new Set())

  const toggleLLMUsage = (index: number) => {
    const newSet = new Set(expandedLLMUsage)
    if (newSet.has(index)) {
      newSet.delete(index)
    } else {
      newSet.add(index)
    }
    setExpandedLLMUsage(newSet)
  }

  // 获取助手图标组件
  const getAssistantIconComponent = () => {
    if (assistantIcon && assistantIcon in ICON_MAP) {
      return ICON_MAP[assistantIcon as keyof typeof ICON_MAP]
    }
    return <Bot className="w-5 h-5" />
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-neutral-800 rounded-xl w-full max-w-2xl mx-4 shadow-xl">
        {/* 头部 */}
        <div className="flex justify-between items-center p-6 border-b border-gray-200 dark:border-neutral-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
            对话详情 ({logs?.length || 0} 条消息)
          </h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-700 dark:hover:text-white transition-colors"
          >
            ✕
          </button>
        </div>

        {/* 内容区域 */}
        <div className="max-h-[60vh] overflow-y-auto p-4 space-y-4 custom-scrollbar">
          {logs && Array.isArray(logs) ? (
            logs.map((log: ChatLog, index: number) => (
              <div key={log.id || index} className="space-y-3">
                {/* 分隔线（除了第一条） */}
                {index > 0 && (
                  <div className="border-t border-gray-300 dark:border-neutral-600 my-4"></div>
                )}

                {/* 用户消息 */}
                {log.userMessage && (
                  <div className="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-lg border border-blue-200 dark:border-blue-800">
                    <div className="flex items-center gap-3 mb-2">
                      <div className="w-8 h-8 bg-blue-500 rounded-full flex items-center justify-center text-white flex-shrink-0">
                        <User className="w-5 h-5" />
                      </div>
                      <div className="flex-1">
                        <span className="text-sm font-medium text-blue-700 dark:text-blue-300">用户</span>
                        <span className="ml-2 text-xs text-gray-400 dark:text-gray-500">
                          {new Date(log.createdAt).toLocaleString('zh-CN')}
                        </span>
                      </div>
                    </div>
                    <div className="text-sm text-gray-800 dark:text-gray-100 whitespace-pre-wrap ml-11">
                      {log.userMessage}
                    </div>
                  </div>
                )}

                {/* AI回复 + LLM 使用信息 */}
                {log.agentMessage && (
                  <div className="bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-800 overflow-hidden">
                    {/* 消息头部 */}
                    <div className="p-4">
                      <div className="flex items-center gap-3 mb-2">
                        <div className="w-8 h-8 bg-green-500 rounded-full flex items-center justify-center text-white flex-shrink-0">
                          {getAssistantIconComponent()}
                        </div>
                        <div className="flex-1 flex items-center justify-between">
                          <div>
                            <span className="text-sm font-medium text-green-700 dark:text-green-300">
                              {log.assistantName || assistantName}
                            </span>
                            <span className="ml-2 text-xs text-gray-400 dark:text-gray-500">
                              {new Date(log.createdAt).toLocaleString('zh-CN')}
                            </span>
                          </div>
                          {log.llmUsage && (
                            <button
                              onClick={() => toggleLLMUsage(index)}
                              className="ml-2 p-1 hover:bg-green-100 dark:hover:bg-green-900/30 rounded transition-colors flex-shrink-0"
                              title="查看 LLM 使用信息"
                            >
                              <motion.div
                                animate={{ rotate: expandedLLMUsage.has(index) ? 180 : 0 }}
                                transition={{ duration: 0.2 }}
                              >
                                <ChevronDown className="w-4 h-4 text-green-700 dark:text-green-300" />
                              </motion.div>
                            </button>
                          )}
                        </div>
                      </div>
                      <div className="text-sm text-gray-800 dark:text-gray-100 whitespace-pre-wrap ml-11">
                        {log.agentMessage}
                      </div>
                    </div>

                    {/* LLM 使用信息 - 折叠内容 */}
                    <AnimatePresence>
                      {log.llmUsage && expandedLLMUsage.has(index) && (
                        <motion.div
                          initial={{ height: 0, opacity: 0 }}
                          animate={{ height: 'auto', opacity: 1 }}
                          exit={{ height: 0, opacity: 0 }}
                          transition={{ duration: 0.2 }}
                          className="border-t border-green-200 dark:border-green-800 bg-green-100/50 dark:bg-green-900/10"
                        >
                          <div className="px-4 py-3 space-y-2">
                            {/* 基础信息 */}
                            <div className="grid grid-cols-2 gap-3">
                              <div className="flex justify-between">
                                <span className="text-xs font-medium text-green-700 dark:text-green-300">
                                  模型:
                                </span>
                                <span className="text-xs text-green-900 dark:text-green-100">
                                  {log.llmUsage.model}
                                </span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-xs font-medium text-green-700 dark:text-green-300">
                                  Total Tokens:
                                </span>
                                <span className="text-xs font-semibold text-green-900 dark:text-green-100">
                                  {log.llmUsage.totalTokens}
                                </span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-xs font-medium text-green-700 dark:text-green-300">
                                  Prompt:
                                </span>
                                <span className="text-xs text-green-900 dark:text-green-100">
                                  {log.llmUsage.promptTokens}
                                </span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-xs font-medium text-green-700 dark:text-green-300">
                                  Completion:
                                </span>
                                <span className="text-xs text-green-900 dark:text-green-100">
                                  {log.llmUsage.completionTokens}
                                </span>
                              </div>
                              {log.llmUsage.duration !== undefined && (
                                <div className="flex justify-between">
                                  <span className="text-xs font-medium text-green-700 dark:text-green-300">
                                    耗时:
                                  </span>
                                  <span className="text-xs text-green-900 dark:text-green-100">
                                    {log.llmUsage.duration >= 1000
                                      ? `${(log.llmUsage.duration / 1000).toFixed(2)}s`
                                      : `${log.llmUsage.duration}ms`}
                                  </span>
                                </div>
                              )}
                            </div>

                            {/* 工具调用信息 */}
                            {log.llmUsage.hasToolCalls && (
                              <div className="pt-2 border-t border-green-200 dark:border-green-800">
                                <div className="text-xs font-medium text-green-700 dark:text-green-300 mb-2">
                                  工具调用: {log.llmUsage.toolCallCount || 0} 个
                                </div>
                                {log.llmUsage.toolCalls && log.llmUsage.toolCalls.length > 0 && (
                                  <div className="space-y-1">
                                    {log.llmUsage.toolCalls.map((toolCall: ToolCallInfo, idx: number) => (
                                      <div
                                        key={idx}
                                        className="text-xs text-green-900 dark:text-green-100 bg-green-100 dark:bg-green-900/40 px-2 py-1 rounded"
                                      >
                                        <span className="font-medium">{toolCall.name}</span>
                                        {toolCall.arguments && (
                                          <span className="ml-1 text-green-700 dark:text-green-400">
                                            ({toolCall.arguments.length > 30
                                              ? `${toolCall.arguments.substring(0, 30)}...`
                                              : toolCall.arguments})
                                          </span>
                                        )}
                                      </div>
                                    ))}
                                  </div>
                                )}
                              </div>
                            )}
                          </div>
                        </motion.div>
                      )}
                    </AnimatePresence>
                  </div>
                )}
              </div>
            ))
          ) : (
            <div className="text-center py-8 text-gray-500 dark:text-gray-400">
              暂无对话记录
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default ChatLogDetail

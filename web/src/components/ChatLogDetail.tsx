import React, { useMemo, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { ChevronDown, Bot, MessageCircle, Users, Zap, Circle, User, Timer } from 'lucide-react'
import { sanitizeReadableText } from '@/utils/string'

interface LLMUsage {
  provider?: string
  model: string
  totalTokens: number
  promptTokens: number
  completionTokens: number
  total_tokens?: number
  input_tokens?: number
  output_tokens?: number
  duration?: number
  latencyMs?: number
  ttftMs?: number
  tps?: number
  queueTimeMs?: number
  latency_ms?: number
  ttft_ms?: number
  queue_time_ms?: number
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

  const normalizeMs = (usage: LLMUsage, key: 'latency' | 'ttft' | 'queue') => {
    if (key === 'latency') return usage.latencyMs ?? usage.latency_ms
    if (key === 'ttft') return usage.ttftMs ?? usage.ttft_ms
    return usage.queueTimeMs ?? usage.queue_time_ms
  }

  const getTotalTokens = (usage: LLMUsage) => usage.totalTokens ?? usage.total_tokens ?? 0
  const getPromptTokens = (usage: LLMUsage) => usage.promptTokens ?? usage.input_tokens ?? 0
  const getCompletionTokens = (usage: LLMUsage) => usage.completionTokens ?? usage.output_tokens ?? 0

  const totalStats = useMemo(() => {
    const list = logs.filter((x) => !!x.llmUsage).map((x) => x.llmUsage as LLMUsage)
    const totalTokens = list.reduce((acc, it) => acc + getTotalTokens(it), 0)
    return { count: list.length, totalTokens }
  }, [logs])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/55 backdrop-blur-[1px]">
      <div className="w-full max-w-4xl mx-4 rounded-2xl border border-white/15 bg-white dark:bg-neutral-900 shadow-2xl">
        {/* 头部 */}
        <div className="flex justify-between items-center p-5 border-b border-gray-200 dark:border-neutral-700">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">对话详情</h2>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {logs?.length || 0} 条消息 · {totalStats.count} 次LLM调用 · {totalStats.totalTokens} Tokens
            </p>
          </div>
          <button
            onClick={onClose}
            className="h-8 w-8 rounded-lg text-gray-400 hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-neutral-800 dark:hover:text-white transition-colors"
          >
            ✕
          </button>
        </div>

        {/* 内容区域 */}
        <div className="max-h-[70vh] overflow-y-auto p-5 space-y-4 custom-scrollbar bg-gradient-to-b from-transparent to-gray-50/30 dark:to-neutral-900/30">
          {logs && Array.isArray(logs) ? (
            logs.map((log: ChatLog, index: number) => (
              <div key={log.id || index} className="space-y-3">
                {/* 分隔线（除了第一条） */}
                {index > 0 && (
                  <div className="border-t border-gray-300 dark:border-neutral-600 my-4"></div>
                )}

                {/* 用户消息 */}
                {log.userMessage && (
                  <div className="rounded-xl border border-blue-200/70 bg-blue-50/80 dark:border-blue-900/40 dark:bg-blue-900/20 p-4">
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
                      {sanitizeReadableText(log.userMessage)}
                    </div>
                  </div>
                )}

                {/* AI回复 + LLM 使用信息 */}
                {log.agentMessage && (
                  <div className="rounded-xl border border-emerald-200/70 bg-emerald-50/80 dark:border-emerald-900/40 dark:bg-emerald-900/20 overflow-hidden">
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
                            <button onClick={() => toggleLLMUsage(index)} className="ml-2 p-1 hover:bg-green-100 dark:hover:bg-green-900/30 rounded transition-colors flex-shrink-0" title="查看 LLM 使用信息">
                              <motion.div animate={{ rotate: expandedLLMUsage.has(index) ? 180 : 0 }} transition={{ duration: 0.2 }}>
                                <ChevronDown className="w-4 h-4 text-green-700 dark:text-green-300" />
                              </motion.div>
                            </button>
                          )}
                        </div>
                      </div>
                      <div className="text-sm text-gray-800 dark:text-gray-100 whitespace-pre-wrap ml-11">
                        {sanitizeReadableText(log.agentMessage)}
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
                            <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                              <div className="rounded-lg bg-white/70 dark:bg-neutral-900/30 p-2">
                                <div className="text-[11px] text-gray-500 dark:text-gray-400">总Token</div>
                                <div className="text-sm font-semibold text-green-900 dark:text-green-100">{getTotalTokens(log.llmUsage)}</div>
                              </div>
                              <div className="rounded-lg bg-white/70 dark:bg-neutral-900/30 p-2">
                                <div className="text-[11px] text-gray-500 dark:text-gray-400">延迟</div>
                                <div className="text-sm font-semibold text-green-900 dark:text-green-100">{normalizeMs(log.llmUsage, 'latency') ?? '-'} ms</div>
                              </div>
                              <div className="rounded-lg bg-white/70 dark:bg-neutral-900/30 p-2">
                                <div className="text-[11px] text-gray-500 dark:text-gray-400">TTFT</div>
                                <div className="text-sm font-semibold text-green-900 dark:text-green-100">{normalizeMs(log.llmUsage, 'ttft') ?? '-'} ms</div>
                              </div>
                              <div className="rounded-lg bg-white/70 dark:bg-neutral-900/30 p-2">
                                <div className="text-[11px] text-gray-500 dark:text-gray-400">TPS</div>
                                <div className="text-sm font-semibold text-green-900 dark:text-green-100">{log.llmUsage.tps ?? '-'} </div>
                              </div>
                            </div>
                            {/* 基础信息 */}
                            <div className="grid grid-cols-2 gap-3 pt-1">
                              <div className="flex justify-between">
                                <span className="text-xs font-medium text-green-700 dark:text-green-300">
                                  Provider:
                                </span>
                                <span className="text-xs text-green-900 dark:text-green-100">
                                  {log.llmUsage.provider || '-'}
                                </span>
                              </div>
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
                                  {getTotalTokens(log.llmUsage)}
                                </span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-xs font-medium text-green-700 dark:text-green-300">
                                  Prompt:
                                </span>
                                <span className="text-xs text-green-900 dark:text-green-100">
                                  {getPromptTokens(log.llmUsage)}
                                </span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-xs font-medium text-green-700 dark:text-green-300">
                                  Completion:
                                </span>
                                <span className="text-xs text-green-900 dark:text-green-100">
                                  {getCompletionTokens(log.llmUsage)}
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
                              <div className="flex justify-between">
                                <span className="text-xs font-medium text-green-700 dark:text-green-300 flex items-center gap-1"><Timer className="w-3 h-3" /> 队列:</span>
                                <span className="text-xs text-green-900 dark:text-green-100">{normalizeMs(log.llmUsage, 'queue') ?? '-'} ms</span>
                              </div>
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

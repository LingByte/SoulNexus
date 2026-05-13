import React, { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Bot, User, Volume2, VolumeX, RefreshCw, ChevronDown } from 'lucide-react'
import { cn } from '@/utils/cn'
import MarkdownPreview from '@/components/UI/MarkdownPreview.tsx'
import { Typewriter } from '@/components/UX/MicroInteractions'
import { getApiBaseURL } from '@/config/apiConfig'
import type { ChatMessage } from '@/pages/VoiceAssistant/types'
import type { LLMUsage } from '@/api/chat'
import { formatChatMessageTime } from '@/utils/formatChatMessageTime'

interface ChatAreaProps {
  messages: ChatMessage[]
  isCalling: boolean
  className?: string
  isGlobalMuted?: boolean
  onMuteToggle?: (isMuted: boolean) => void
  onStopAudio?: () => void
  onNewSession?: () => void
  assistantName?: string
}

function displayTime(ts: string): string {
  if (!ts) return ''
  if (/^\d{4}-\d{2}-\d{2}T/.test(ts)) return formatChatMessageTime(ts)
  return ts
}

function getTotalTokens(u: LLMUsage): number {
  return u.totalTokens ?? (u as { total_tokens?: number }).total_tokens ?? 0
}
function getPromptTokens(u: LLMUsage): number {
  return u.promptTokens ?? (u as { input_tokens?: number }).input_tokens ?? 0
}
function getCompletionTokens(u: LLMUsage): number {
  return u.completionTokens ?? (u as { output_tokens?: number }).output_tokens ?? 0
}
function getLatencyMs(u: LLMUsage): number | undefined {
  return u.latencyMs ?? u.latency_ms
}

function LlmUsagePanel({ u }: { u: LLMUsage }) {
  return (
    <div className="space-y-2 text-xs">
      <div className="grid grid-cols-2 gap-1.5">
        <div className="rounded-md border border-gray-200/80 dark:border-neutral-600 p-1.5 dark:bg-neutral-900/40">
          <div className="text-[10px] text-gray-500">总 Tokens</div>
          <div className="font-semibold tabular-nums text-gray-900 dark:text-gray-100">{getTotalTokens(u)}</div>
        </div>
        <div className="rounded-md border border-gray-200/80 dark:border-neutral-600 p-1.5 dark:bg-neutral-900/40">
          <div className="text-[10px] text-gray-500">延迟</div>
          <div className="font-semibold tabular-nums text-gray-900 dark:text-gray-100">
            {getLatencyMs(u) != null ? `${getLatencyMs(u)} ms` : '—'}
          </div>
        </div>
        <div className="rounded-md border border-gray-200/80 dark:border-neutral-600 p-1.5 dark:bg-neutral-900/40">
          <div className="text-[10px] text-gray-500">Prompt</div>
          <div className="font-semibold tabular-nums text-gray-900 dark:text-gray-100">{getPromptTokens(u)}</div>
        </div>
        <div className="rounded-md border border-gray-200/80 dark:border-neutral-600 p-1.5 dark:bg-neutral-900/40">
          <div className="text-[10px] text-gray-500">Completion</div>
          <div className="font-semibold tabular-nums text-gray-900 dark:text-gray-100">{getCompletionTokens(u)}</div>
        </div>
      </div>
      <div className="space-y-1 text-[11px] text-gray-600 dark:text-gray-300 border-t border-gray-200/80 dark:border-neutral-600 pt-2">
        <div className="flex justify-between gap-2">
          <span className="text-gray-500">模型</span>
          <span className="font-mono text-right break-all">{u.model || '—'}</span>
        </div>
        <div className="flex justify-between gap-2">
          <span className="text-gray-500">Provider</span>
          <span className="text-right break-all">{(u as { provider?: string }).provider || '—'}</span>
        </div>
        {u.ttftMs != null || u.ttft_ms != null ? (
          <div className="flex justify-between gap-2">
            <span className="text-gray-500">TTFT</span>
            <span className="tabular-nums">{u.ttftMs ?? u.ttft_ms} ms</span>
          </div>
        ) : null}
        {u.duration != null ? (
          <div className="flex justify-between gap-2">
            <span className="text-gray-500">耗时</span>
            <span className="tabular-nums">
              {u.duration >= 1000 ? `${(u.duration / 1000).toFixed(2)} s` : `${u.duration} ms`}
            </span>
          </div>
        ) : null}
        {u.hasToolCalls ? (
          <div className="flex justify-between gap-2">
            <span className="text-gray-500">工具调用</span>
            <span>{u.toolCallCount ?? 0} 次</span>
          </div>
        ) : null}
      </div>
    </div>
  )
}

const ChatArea: React.FC<ChatAreaProps> = ({
  messages,
  isCalling,
  className = '',
  isGlobalMuted = false,
  onMuteToggle,
  onStopAudio,
  onNewSession,
  assistantName
}) => {
  const [typingMessages, setTypingMessages] = useState<Set<string>>(new Set())
  const [playingAudio, setPlayingAudio] = useState<string | null>(null)
  const [isMuted, setIsMuted] = useState(isGlobalMuted)
  const [expandedLlmMessageIds, setExpandedLlmMessageIds] = useState<Set<string>>(new Set())
  const currentAudioRef = useRef<HTMLAudioElement | null>(null)

  const toggleLlmExpand = (messageId: string) => {
    setExpandedLlmMessageIds((prev) => {
      const next = new Set(prev)
      if (next.has(messageId)) next.delete(messageId)
      else next.add(messageId)
      return next
    })
  }

  React.useEffect(() => {
    setIsMuted(isGlobalMuted)
  }, [isGlobalMuted])

  useEffect(() => {
    const lastMessage = messages[messages.length - 1]
    if (!lastMessage || lastMessage.type !== 'agent' || !lastMessage.id) return
    if (lastMessage.id.startsWith('resume-')) return
    setTypingMessages(prev => {
      if (prev.has(lastMessage.id!)) return prev
      const next = new Set(prev)
      next.add(lastMessage.id!)
      return next
    })
  }, [messages])

  const handleTypewriterComplete = (messageId: string) => {
    setTypingMessages(prev => {
      const newSet = new Set(prev)
      newSet.delete(messageId)
      return newSet
    })
  }

  const stopCurrentAudio = () => {
    if (currentAudioRef.current) {
      currentAudioRef.current.pause()
      currentAudioRef.current.currentTime = 0
      currentAudioRef.current = null
    }
    setPlayingAudio(null)
  }

  const stopAllAudio = () => {
    stopCurrentAudio()
    const audioElements = document.querySelectorAll('audio')
    audioElements.forEach(audio => {
      audio.pause()
      audio.currentTime = 0
    })
    onStopAudio?.()
  }

  React.useEffect(() => {
    if (onStopAudio) {
      ;(onStopAudio as any).stopAllAudio = stopAllAudio
    }
  }, [onStopAudio])

  const toggleGlobalMute = () => {
    const newMutedState = !isMuted
    setIsMuted(newMutedState)
    if (newMutedState) {
      stopCurrentAudio()
    }
    onMuteToggle?.(newMutedState)
  }

  const toggleAudio = (audioUrl: string, messageId: string) => {
    if (isMuted) {
      setIsMuted(false)
      onMuteToggle?.(false)
      setTimeout(() => playAudio(audioUrl, messageId), 100)
    } else if (playingAudio === messageId) {
      stopCurrentAudio()
    } else {
      playAudio(audioUrl, messageId)
    }
  }

  const playAudio = async (audioUrl: string, messageId: string) => {
    if (isMuted) return
    let url = audioUrl
    if (url.startsWith('/media/') || url.startsWith('/uploads/')) {
      const apiBaseURL = getApiBaseURL()
      const baseURL = apiBaseURL.replace('/api', '')
      url = url.replace('/media/', `${baseURL}/uploads/`)
    }

    stopCurrentAudio()

    const audioElements = document.querySelectorAll('audio')
    audioElements.forEach(audio => {
      if (audio !== currentAudioRef.current) {
        audio.pause()
        audio.currentTime = 0
      }
    })

    try {
      const audio = new Audio(url)
      currentAudioRef.current = audio
      setPlayingAudio(messageId)

      audio.onended = () => {
        setPlayingAudio(null)
        currentAudioRef.current = null
      }

      audio.onerror = () => {
        console.error('音频播放失败:', url)
        setPlayingAudio(null)
        currentAudioRef.current = null
      }

      await audio.play()
    } catch (error) {
      console.error('播放音频失败:', error)
      setPlayingAudio(null)
      currentAudioRef.current = null
    }
  }

  const renderAgentFooterAndLlm = (msg: ChatMessage, messageId: string, isPlaying: boolean) => {
    const open = msg.llmUsage ? expandedLlmMessageIds.has(messageId) : false
    const hasExtras = !!(msg.audioUrl || msg.llmUsage)
    return (
      <div className={cn('mt-2', hasExtras && 'border-t border-gray-200/80 dark:border-neutral-600 pt-2')}>
        <div className="flex items-center justify-between gap-2">
          <div className="text-xs text-gray-500 tabular-nums shrink-0 min-w-0">{displayTime(msg.timestamp)}</div>
          <div className="flex items-center gap-1.5 shrink-0">
            {msg.audioUrl && (
              <button
                type="button"
                onClick={() => toggleAudio(msg.audioUrl!, messageId)}
                className={cn(
                  'p-1 rounded transition-colors',
                  isMuted
                    ? 'text-red-500 hover:bg-red-100 dark:hover:bg-red-900'
                    : isPlaying
                      ? 'text-blue-500 hover:bg-blue-100 dark:hover:bg-blue-900'
                      : 'text-gray-500 hover:bg-gray-200 dark:hover:bg-gray-600'
                )}
                title={isMuted ? '点击取消静音并播放' : isPlaying ? '点击停止播放' : '点击播放音频'}
              >
                {isMuted ? (
                  <VolumeX className="w-4 h-4" />
                ) : isPlaying ? (
                  <div className="w-4 h-4 flex items-center justify-center">
                    <div className="w-2 h-2 bg-blue-500 rounded-full animate-pulse" />
                  </div>
                ) : (
                  <Volume2 className="w-4 h-4" />
                )}
              </button>
            )}
            {msg.llmUsage && (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation()
                  toggleLlmExpand(messageId)
                }}
                className="inline-flex items-center gap-0.5 rounded-md px-1.5 py-0.5 text-[11px] font-medium text-gray-600 hover:bg-gray-200/80 dark:text-gray-300 dark:hover:bg-neutral-600/60 transition-colors"
              >
                <span>{open ? '收起' : '查看更多'}</span>
                <ChevronDown className={cn('h-3.5 w-3.5 transition-transform duration-200', open && 'rotate-180')} />
              </button>
            )}
          </div>
        </div>
        <AnimatePresence initial={false}>
          {msg.llmUsage && open && (
            <motion.div
              key={`llm-${messageId}`}
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: 'auto', opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={{ duration: 0.28, ease: [0.16, 1, 0.3, 1] }}
              className="overflow-hidden"
            >
              <div className="pt-2">
                <div className="rounded-lg bg-white/70 p-2 dark:bg-neutral-900/55">
                  <LlmUsagePanel u={msg.llmUsage} />
                </div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    )
  }

  const renderMessageContent = (msg: ChatMessage, index: number) => {
    const messageId = msg.id || `msg-${index}`
    const isTyping = typingMessages.has(messageId)

    if (msg.type === 'agent') {
      const isPlaying = playingAudio === messageId

      if (msg.isLoading) {
        return (
          <div className="bg-gray-100 dark:bg-neutral-700 rounded-2xl p-3 text-sm">
            <div className="flex items-center gap-2">
              <div className="flex space-x-1">
                <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" />
                <div
                  className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"
                  style={{ animationDelay: '0.1s' }}
                />
                <div
                  className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"
                  style={{ animationDelay: '0.2s' }}
                />
              </div>
              <span className="text-gray-500 text-xs">AI正在思考中...</span>
            </div>
          </div>
        )
      }

      if (isTyping) {
        return (
          <div className="bg-gray-100 dark:bg-neutral-700 rounded-2xl p-3 text-sm">
            <Typewriter
              text={msg.content || ''}
              speed={30}
              delay={200}
              onComplete={() => handleTypewriterComplete(messageId)}
              className="prose prose-sm max-w-none dark:prose-invert"
            />
            {renderAgentFooterAndLlm(msg, messageId, isPlaying)}
          </div>
        )
      }

      return (
        <div className="bg-gray-100 dark:bg-neutral-700 rounded-2xl p-3 text-sm">
          <MarkdownPreview content={msg.content || ''} className="prose prose-sm max-w-none dark:prose-invert" />
          {renderAgentFooterAndLlm(msg, messageId, isPlaying)}
        </div>
      )
    }

    return (
      <div className="bg-purple-100 dark:bg-purple-900 rounded-2xl p-3 text-sm">
        <p className="whitespace-pre-wrap">{msg.content}</p>
        <div className="mt-1 text-xs text-purple-500 dark:text-purple-300 tabular-nums">{displayTime(msg.timestamp)}</div>
      </div>
    )
  }

  return (
    <div className={cn('flex-1 flex flex-col bg-white dark:bg-neutral-800 min-h-0 max-h-[92vh]', className)}>
      <div className="flex items-center justify-between p-3 border-b dark:border-neutral-700 bg-gray-50 dark:bg-neutral-900">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{assistantName || '未选择智能体'}</span>
          {isMuted && (
            <span className="text-xs text-red-500 bg-red-100 dark:bg-red-900 px-2 py-1 rounded">已静音</span>
          )}
        </div>
        <div className="flex items-center gap-2">
          {onNewSession && (
            <button
              type="button"
              onClick={onNewSession}
              className="flex items-center gap-1 px-3 py-1.5 text-xs font-medium text-gray-600 dark:text-gray-300 bg-white dark:bg-neutral-800 border border-gray-300 dark:border-neutral-600 rounded-lg hover:bg-gray-50 dark:hover:bg-neutral-700 transition-colors"
              title="开始新会话"
            >
              <RefreshCw className="w-3 h-3" />
              新会话
            </button>
          )}
          <button
            type="button"
            onClick={toggleGlobalMute}
            className={cn(
              'p-2 rounded-lg transition-colors',
              isMuted ? 'text-red-500 hover:bg-red-100 dark:hover:bg-red-900' : 'text-gray-500 hover:bg-gray-200 dark:hover:bg-gray-700'
            )}
            title={isMuted ? '取消静音' : '全局静音'}
          >
            {isMuted ? <VolumeX className="w-5 h-5" /> : <Volume2 className="w-5 h-5" />}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-4 custom-scrollbar">
        {!isCalling && messages.length === 0 ? (
          <div className="flex items-center justify-center h-full text-gray-400 dark:text-gray-500">
            <div className="text-center">
              <div className="flex justify-center mb-4">
                <Bot className="w-12 h-12 opacity-30" />
              </div>
              <p className="text-lg font-medium">暂无消息</p>
              <p className="text-sm">点击左侧语音按钮开始聊天</p>
            </div>
          </div>
        ) : (
          messages.map((msg, index) => (
            <motion.div
              key={msg.id || index}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: Math.min(index * 0.03, 0.3) }}
              className={cn('flex gap-3', msg.type === 'user' ? 'justify-end' : 'justify-start')}
            >
              {msg.type === 'agent' && (
                <div className="flex items-start gap-3 max-w-[75%]">
                  <div className="shrink-0 w-8 h-8 rounded-full bg-purple-100 dark:bg-purple-900 flex items-center justify-center">
                    <Bot className="w-4 h-4 text-purple-600 dark:text-purple-300" />
                  </div>
                  <div className="space-y-1 min-w-0 flex-1">{renderMessageContent(msg, index)}</div>
                </div>
              )}

              {msg.type === 'user' && (
                <div className="flex items-start gap-3 max-w-[75%] flex-row-reverse">
                  <div className="shrink-0 w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center">
                    <User className="w-4 h-4 text-blue-600 dark:text-blue-300" />
                  </div>
                  <div className="space-y-1 min-w-0">{renderMessageContent(msg, index)}</div>
                </div>
              )}
            </motion.div>
          ))
        )}
      </div>
    </div>
  )
}

export default ChatArea

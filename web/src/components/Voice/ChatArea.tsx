import React, { useState, useEffect, useRef } from 'react'
import { motion } from 'framer-motion'
import { Bot, User, Volume2, VolumeX, RefreshCw } from 'lucide-react'
import { cn } from '@/utils/cn'
import MarkdownPreview from '@/components/UI/MarkdownPreview.tsx'
import { Typewriter } from '@/components/UX/MicroInteractions'
import { getApiBaseURL } from '@/config/apiConfig'

interface ChatMessage {
  type: 'user' | 'agent'
  content: string
  timestamp: string
  id?: string
  audioUrl?: string
  isLoading?: boolean
}

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
  const currentAudioRef = useRef<HTMLAudioElement | null>(null)

  // 同步全局静音状态
  React.useEffect(() => {
    setIsMuted(isGlobalMuted)
  }, [isGlobalMuted])

  // 打字特效：仅对新增的 AI 消息触发一次，不因列表变化重置
  
  // 调试信息
  console.log('ChatArea渲染 - 消息数量:', messages.length)
  console.log('ChatArea渲染 - 消息内容:', messages)

  // 监听新增消息：仅当最后一条是全新 id 且未出现过时，加入打字集合
  useEffect(() => {
    const lastMessage = messages[messages.length - 1]
    if (!lastMessage || lastMessage.type !== 'agent' || !lastMessage.id) return
    setTypingMessages(prev => {
      if (prev.has(lastMessage.id!)) return prev
      const next = new Set(prev)
      next.add(lastMessage.id!)
      return next
    })
  }, [messages])

  // 打字机完成回调
  const handleTypewriterComplete = (messageId: string) => {
    setTypingMessages(prev => {
      const newSet = new Set(prev)
      newSet.delete(messageId)
      return newSet
    })
  }

  // 停止当前播放的音频
  const stopCurrentAudio = () => {
    if (currentAudioRef.current) {
      currentAudioRef.current.pause()
      currentAudioRef.current.currentTime = 0
      currentAudioRef.current = null
    }
    setPlayingAudio(null)
  }

  // 立即停止所有音频（供外部调用）
  const stopAllAudio = () => {
    stopCurrentAudio()
    // 停止页面上所有音频元素
    const audioElements = document.querySelectorAll('audio')
    audioElements.forEach(audio => {
      audio.pause()
      audio.currentTime = 0
    })
    onStopAudio?.()
  }

  // 暴露停止音频函数给父组件
  React.useEffect(() => {
    if (onStopAudio) {
      (onStopAudio as any).stopAllAudio = stopAllAudio
    }
  }, [onStopAudio])


  // 全局静音切换
  const toggleGlobalMute = () => {
    const newMutedState = !isMuted
    setIsMuted(newMutedState)
    if (newMutedState) {
      stopCurrentAudio()
    }
    onMuteToggle?.(newMutedState)
  }

  // 播放/停止音频
  const toggleAudio = (audioUrl: string, messageId: string) => {
    if (isMuted) {
      // 如果当前是静音状态，取消静音并播放
      setIsMuted(false)
      onMuteToggle?.(false)
      setTimeout(() => playAudio(audioUrl, messageId), 100)
    } else if (playingAudio === messageId) {
      // 如果当前正在播放这个音频，停止播放
      stopCurrentAudio()
    } else {
      // 播放音频
      playAudio(audioUrl, messageId)
    }
  }

  // 播放音频
  const playAudio = async (audioUrl: string, messageId: string) => {
    if (isMuted) return
    // 处理音频URL - 如果是相对路径，添加服务器基础URL
    if (audioUrl.startsWith('/media/') || audioUrl.startsWith('/uploads/')) {
      const apiBaseURL = getApiBaseURL()
      const baseURL = apiBaseURL.replace('/api', '')
      audioUrl = audioUrl.replace('/media/', `${baseURL}/uploads/`)
    }

      // 停止当前播放的音频
    stopCurrentAudio()
    
    // 停止页面上所有其他音频元素
    const audioElements = document.querySelectorAll('audio')
    audioElements.forEach(audio => {
      if (audio !== currentAudioRef.current) {
        audio.pause()
        audio.currentTime = 0
      }
    })
    
    try {
      const audio = new Audio(audioUrl)
      currentAudioRef.current = audio
      setPlayingAudio(messageId)
      
      audio.onended = () => {
        setPlayingAudio(null)
        currentAudioRef.current = null
      }
      
      audio.onerror = () => {
        console.error('音频播放失败:', audioUrl)
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

  // 渲染消息内容
  const renderMessageContent = (msg: ChatMessage, index: number) => {
    const messageId = msg.id || `msg-${index}`
    const isTyping = typingMessages.has(messageId)

    if (msg.type === 'agent') {
      // AI消息：新消息使用打字机效果，历史消息直接显示markdown
      const isPlaying = playingAudio === messageId
      
      // 显示loading状态
      if (msg.isLoading) {
        return (
          <div className="bg-gray-100 dark:bg-neutral-700 rounded-2xl p-3 text-sm">
            <div className="flex items-center gap-2">
              <div className="flex space-x-1">
                <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"></div>
                <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0.1s' }}></div>
                <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></div>
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
            <div className="flex items-center justify-between mt-2">
              <div className="text-xs text-gray-500">{msg.timestamp}</div>
              {msg.audioUrl && (
                <button
                  onClick={() => toggleAudio(msg.audioUrl!, messageId)}
                  className={`p-1 rounded transition-colors ${
                    isMuted 
                      ? 'text-red-500 hover:bg-red-100 dark:hover:bg-red-900' 
                      : isPlaying 
                        ? 'text-blue-500 hover:bg-blue-100 dark:hover:bg-blue-900' 
                        : 'text-gray-500 hover:bg-gray-200 dark:hover:bg-gray-600'
                  }`}
                  title={
                    isMuted 
                      ? "点击取消静音并播放" 
                      : isPlaying 
                        ? "点击停止播放" 
                        : "点击播放音频"
                  }
                >
                  {isMuted ? (
                    <VolumeX className="w-4 h-4" />
                  ) : isPlaying ? (
                    <div className="w-4 h-4 flex items-center justify-center">
                      <div className="w-2 h-2 bg-blue-500 rounded-full animate-pulse"></div>
                    </div>
                  ) : (
                    <Volume2 className="w-4 h-4" />
                  )}
                </button>
              )}
            </div>
          </div>
        )
      } else {
        return (
          <div className="bg-gray-100 dark:bg-neutral-700 rounded-2xl p-3 text-sm">
            <MarkdownPreview 
              content={msg.content || ''}
              className="prose prose-sm max-w-none dark:prose-invert"
            />
            <div className="flex items-center justify-between mt-2">
              <div className="text-xs text-gray-500">{msg.timestamp}</div>
              {msg.audioUrl && (
                <button
                  onClick={() => toggleAudio(msg.audioUrl!, messageId)}
                  className={`p-1 rounded transition-colors ${
                    isMuted 
                      ? 'text-red-500 hover:bg-red-100 dark:hover:bg-red-900' 
                      : isPlaying 
                        ? 'text-blue-500 hover:bg-blue-100 dark:hover:bg-blue-900' 
                        : 'text-gray-500 hover:bg-gray-200 dark:hover:bg-gray-600'
                  }`}
                  title={
                    isMuted 
                      ? "点击取消静音并播放" 
                      : isPlaying 
                        ? "点击停止播放" 
                        : "点击播放音频"
                  }
                >
                  {isMuted ? (
                    <VolumeX className="w-4 h-4" />
                  ) : isPlaying ? (
                    <div className="w-4 h-4 flex items-center justify-center">
                      <div className="w-2 h-2 bg-blue-500 rounded-full animate-pulse"></div>
                    </div>
                  ) : (
                    <Volume2 className="w-4 h-4" />
                  )}
                </button>
              )}
            </div>
          </div>
        )
      }
    } else {
      // 用户消息：直接显示文本
      return (
        <div className="bg-purple-100 dark:bg-purple-900 rounded-2xl p-3 text-sm">
          <p className="whitespace-pre-wrap">{msg.content}</p>
          <div className="mt-1 text-xs text-purple-500 dark:text-purple-300">{msg.timestamp}</div>
        </div>
      )
    }
  }

  return (
    <div className={cn('flex-1 flex flex-col bg-white dark:bg-neutral-800 min-h-0 max-h-[92vh]', className)}>
      {/* 全局静音控制栏 */}
      <div className="flex items-center justify-between p-3 border-b dark:border-neutral-700 bg-gray-50 dark:bg-neutral-900">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
            {assistantName || '未选择助手'}
          </span>
          {isMuted && (
            <span className="text-xs text-red-500 bg-red-100 dark:bg-red-900 px-2 py-1 rounded">
              已静音
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          {onNewSession && (
            <button
              onClick={onNewSession}
              className="flex items-center gap-1 px-3 py-1.5 text-xs font-medium text-gray-600 dark:text-gray-300 bg-white dark:bg-neutral-800 border border-gray-300 dark:border-neutral-600 rounded-lg hover:bg-gray-50 dark:hover:bg-neutral-700 transition-colors"
              title="开始新会话"
            >
              <RefreshCw className="w-3 h-3" />
              新会话
            </button>
          )}
          <button
            onClick={toggleGlobalMute}
            className={`p-2 rounded-lg transition-colors ${
              isMuted 
                ? 'text-red-500 hover:bg-red-100 dark:hover:bg-red-900' 
                : 'text-gray-500 hover:bg-gray-200 dark:hover:bg-gray-700'
            }`}
            title={isMuted ? "取消静音" : "全局静音"}
          >
            {isMuted ? (
              <VolumeX className="w-5 h-5" />
            ) : (
              <Volume2 className="w-5 h-5" />
            )}
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
              transition={{ delay: index * 0.1 }}
              className={cn('flex gap-3', msg.type === 'user' ? 'justify-end' : 'justify-start')}
            >
              {/* AI消息 - 左侧头像 */}
              {msg.type === 'agent' && (
                <div className="flex items-start gap-3 max-w-[75%]">
                  <div className="shrink-0 w-8 h-8 rounded-full bg-purple-100 dark:bg-purple-900 flex items-center justify-center">
                    <Bot className="w-4 h-4 text-purple-600 dark:text-purple-300" />
                  </div>
                  <div className="space-y-1">
                    {renderMessageContent(msg, index)}
                  </div>
                </div>
              )}

              {/* 用户消息 - 右侧头像 */}
              {msg.type === 'user' && (
                <div className="flex items-start gap-3 max-w-[75%] flex-row-reverse">
                  <div className="shrink-0 w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center">
                    <User className="w-4 h-4 text-blue-600 dark:text-blue-300" />
                  </div>
                  <div className="space-y-1">
                    {renderMessageContent(msg, index)}
                  </div>
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


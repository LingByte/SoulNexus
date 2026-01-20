import { useState, useRef } from 'react'
import { oneShotText, plainText } from '@/api/assistant'
import { showAlert } from '@/utils/notification'

export type TextMode = 'voice' | 'text'

interface UseVoiceAssistantOptions {
  apiKey: string
  apiSecret: string
  assistantId: number
  language: string
  systemPrompt: string
  selectedVoiceCloneId: number | null
  selectedSpeaker: string
  currentSessionId: string | null
  setCurrentSessionId: (id: string | null) => void
  addUserMessage: (content: string) => string
  addAIMessage: (content: string) => string
  addAILoadingMessage: () => string
  removeLoadingMessage: (id: string) => void
  pollAudioStatus: (requestId: string, messageId: string) => void
  textMode?: TextMode
  updateAIMessage?: (messageId: string, newText: string) => void
  selectedKnowledgeBase?: string | null // 选中的知识库ID
  temperature?: number // 生成多样性
  maxTokens?: number   // 最大回复长度
}

export const useVoiceAssistant = (options: UseVoiceAssistantOptions) => {
  const {
    apiKey,
    apiSecret,
    assistantId,
    language,
    systemPrompt,
    selectedVoiceCloneId,
    selectedSpeaker,
    currentSessionId,
    setCurrentSessionId,
    addUserMessage,
    addAIMessage,
    addAILoadingMessage,
    removeLoadingMessage,
    pollAudioStatus,
    textMode = 'voice',
    updateAIMessage,
    selectedKnowledgeBase,
    temperature,
    maxTokens,
  } = options

  const [inputValue, setInputValue] = useState('')
  const [isWaitingForResponse, setIsWaitingForResponse] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  // 处理错误信息提取
  const extractErrorMessage = (err: any): string => {
    if (err?.data && typeof err.data === 'string' && err.data.trim()) {
      return err.data
    } else if (err?.response?.data?.data && typeof err.response.data.data === 'string') {
      return err.response.data.data
    } else if (err?.msg) {
      return err.msg
    } else if (err?.response?.data?.msg) {
      return err.response.data.msg
    } else if (err?.message) {
      return err.message
    } else if (typeof err === 'string') {
      return err
    }
    return '请求失败'
  }


  // 发送文本消息
  const sendTextMessage = async (text: string) => {
    // 校验API密钥
    if (!apiKey || !apiSecret) {
      showAlert('请先在配置面板中设置API Key和API Secret', 'warning')
      return
    }

    // 生成或使用当前会话ID
    let sessionId = currentSessionId
    if (!sessionId) {
      sessionId = `text_${Date.now()}`
      setCurrentSessionId(sessionId)
    }

    // 先清空输入框
    setInputValue('')

    // 设置等待状态
    setIsWaitingForResponse(true)

    // 添加AI loading消息
    let loadingMessageId: string | undefined = undefined

    try {
      // 先添加用户消息到聊天记录
      addUserMessage(text)

      // 添加AI loading消息
      loadingMessageId = addAILoadingMessage()

      // 根据textMode选择使用不同的接口
      const requestData = {
        apiKey: apiKey,
        apiSecret: apiSecret,
        text: text,
        assistantId: assistantId || 1,
        language,
        sessionId: sessionId || `text_${Date.now()}`,
        systemPrompt,
        // 根据是否使用训练音色选择speaker或voiceCloneId（仅语音模式需要）
        ...(textMode === 'voice' && (selectedVoiceCloneId ? { voiceCloneId: selectedVoiceCloneId } : { speaker: selectedSpeaker })),
        // 如果选择了知识库，添加到请求中
        ...(selectedKnowledgeBase && { knowledgeBaseId: selectedKnowledgeBase }),
        // 传递 temperature 和 maxTokens
        ...(temperature !== undefined && { temperature }),
        ...(maxTokens !== undefined && { maxTokens }),
      }

      if (textMode === 'text') {
        // 纯文本模式：使用普通查询
        try {
          const response = await plainText(requestData)
          console.log('PlainText响应:', response)

          // 移除loading消息
          if (loadingMessageId) {
            removeLoadingMessage(loadingMessageId)
          }

          // 检查响应是否成功
          if (response.code !== 200) {
            let errorMsg = '请求失败'
            if (typeof response.data === 'string' && response.data.trim()) {
              errorMsg = response.data
            } else if (response.msg) {
              errorMsg = response.msg
            }
            showAlert(errorMsg, 'error', '请求失败')
            addAIMessage(`抱歉，处理您的请求时出现错误：${errorMsg}`)
            setIsWaitingForResponse(false)
            return
          }

          // 显示文本响应
          if (response.data?.text && response.data.text.trim()) {
            console.log('准备添加AI消息:', response.data.text)
            addAIMessage(response.data.text)
          } else {
            console.log('响应中没有有效text字段')
            addAIMessage('抱歉，未能获取到有效回复')
          }
          setIsWaitingForResponse(false)
        } catch (err: any) {
          console.error('文本发送失败:', err)
          // 移除loading消息（如果存在）
          if (loadingMessageId) {
            removeLoadingMessage(loadingMessageId)
          }

          const errorMsg = extractErrorMessage(err)
          showAlert(errorMsg, 'error', '请求失败')
          addAIMessage(`抱歉，处理您的请求时出现错误：${errorMsg}`)
          setIsWaitingForResponse(false)
        }
        return
      } else {
        // 语音模式：进行TTS合成
        const response = await oneShotText(requestData)
        console.log('OneShotText响应:', response)

        // 移除loading消息
        if (loadingMessageId) {
          removeLoadingMessage(loadingMessageId)
        }

        // 检查响应是否成功
        if (response.code !== 200) {
          let errorMsg = '请求失败'
          if (typeof response.data === 'string' && response.data.trim()) {
            errorMsg = response.data
          } else if (response.msg) {
            errorMsg = response.msg
          }
          showAlert(errorMsg, 'error', '请求失败')
          addAIMessage(`抱歉，处理您的请求时出现错误：${errorMsg}`)
          return
        }

        // 立即显示文本
        if (response.data?.text && response.data.text.trim()) {
          console.log('准备添加AI消息:', response.data.text)
          const messageId = addAIMessage(response.data.text)

          // 如果有requestId，开始轮询音频状态
          if ((response.data as any)?.requestId) {
            console.log('开始轮询音频状态:', (response.data as any).requestId)
            pollAudioStatus((response.data as any).requestId, messageId)
          }
        } else {
          console.log('响应中没有有效text字段，可能是function tools调用')
          // 对于function tools调用，暂时不显示消息，等待后续轮询结果
        }
      }
    } catch (err: any) {
      console.error('文本发送失败:', err)
      // 移除loading消息（如果存在）
      if (loadingMessageId) {
        removeLoadingMessage(loadingMessageId)
      }

      const errorMsg = extractErrorMessage(err)
      showAlert(errorMsg, 'error', '请求失败')
      addAIMessage(`抱歉，处理您的请求时出现错误：${errorMsg}`)
      setIsWaitingForResponse(false)
    } finally {
      // 语音模式下清除等待状态（纯文本模式已在各自分支中处理）
      if (textMode !== 'text') {
        setIsWaitingForResponse(false)
      }
    }
  }

  // 处理输入框回车
  const handleInputEnter = async (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      const value = inputValue.trim()
      if (!value) return
      await sendTextMessage(value)
    }
  }

  // 处理发送按钮点击
  const handleSendClick = async () => {
    const value = inputValue.trim()
    if (!value) return
    await sendTextMessage(value)
  }

  return {
    inputValue,
    setInputValue,
    isWaitingForResponse,
    inputRef,
    handleInputEnter,
    handleSendClick,
    sendTextMessage,
  }
}


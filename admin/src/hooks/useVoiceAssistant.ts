import { useState, useRef } from 'react'
import { post } from '@/utils/request'
import { showAlert } from '@/utils/notification'

interface UseVoiceAssistantOptions {
  apiKey: string
  apiSecret: string
  agentId: number
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
}

export const useVoiceAssistant = (options: UseVoiceAssistantOptions) => {
  const {
    apiKey,
    apiSecret,
    agentId,
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

      // 使用V2接口
      const requestData = {
        apiKey: apiKey,
        apiSecret: apiSecret,
        text: text,
        agentId: agentId || 1,
        language,
        sessionId: sessionId || `text_${Date.now()}`,
        systemPrompt,
        // 根据是否使用训练音色选择speaker或voiceCloneId
        ...(selectedVoiceCloneId ? { voiceCloneId: selectedVoiceCloneId } : { speaker: selectedSpeaker }),
      }

      const response = await post('/voice/oneshot_text', requestData)
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

      // 立即显示文本，不等待音频
      if (response.data?.text && response.data.text.trim()) {
        console.log('准备添加AI消息:', response.data.text)
        const messageId = addAIMessage(response.data.text)

        // 如果有requestId，开始轮询音频状态
        if ((response.data as any)?.requestId) {
          console.log('开始轮询音频状态:', (response.data as any).requestId)
          pollAudioStatus((response.data as any).requestId, messageId)
        } else {
          console.log('响应中没有requestId字段')
        }
      } else {
        console.log('响应中没有有效text字段，可能是function tools调用')
        // 对于function tools调用，暂时不显示消息，等待后续轮询结果
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
    } finally {
      // 清除等待状态
      setIsWaitingForResponse(false)
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


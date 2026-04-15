import { useState, useEffect } from 'react'
import { showAlert } from '@/utils/notification'

interface UseVoiceConnectionOptions {
  assistantId: number
  apiKey: string
  apiSecret: string
}

export const useVoiceConnection = (options: UseVoiceConnectionOptions) => {
  const {
    assistantId,
    apiKey,
    apiSecret,
  } = options

  const [socket, setSocket] = useState<WebSocket | null>(null)
  const [peerConnection, setPeerConnection] = useState<RTCPeerConnection | null>(null)
  const [localStream, setLocalStream] = useState<MediaStream | null>(null)
  const [callDuration, setCallDuration] = useState(0)
  const [callTimer, setCallTimer] = useState<NodeJS.Timeout | null>(null)

  // 清理函数
  useEffect(() => {
    return () => {
      if (callTimer) {
        clearInterval(callTimer)
      }
      if (socket && socket.readyState !== WebSocket.CLOSED) {
        socket.close()
      }
      if (peerConnection && peerConnection.connectionState !== 'closed') {
        peerConnection.close()
      }
      if (localStream) {
        localStream.getTracks().forEach(track => track.stop())
      }
    }
  }, [])

  // 开始通话计时器
  const startCallTimer = () => {
    const timer = setInterval(() => {
      setCallDuration(prev => prev + 1)
    }, 1000)
    setCallTimer(timer)
  }

  // 停止通话计时器
  const stopCallTimer = () => {
    if (callTimer) {
      clearInterval(callTimer)
      setCallTimer(null)
    }
    setCallDuration(0)
  }

  // 连接WebRTC
  const connectWebSocket = async () => {
    // WebRTC连接逻辑
    // 这里可以添加WebRTC相关的连接逻辑
    console.log('连接WebRTC')
  }

  // 开始通话
  const startCall = async () => {
    if (assistantId === 0) {
      showAlert('请先选择一个AI助手', 'warning')
      return false
    }

    // 校验API密钥
    if (!apiKey || !apiSecret) {
      showAlert('请先配置API密钥和密钥（右侧控制面板）', 'warning')
      return false
    }

    try {
      setCallDuration(0)
      startCallTimer()

      // 连接WebRTC
      await connectWebSocket()

      showAlert('通话已开始', 'success')
      return true
    } catch (err: any) {
      console.error('通话启动失败:', err)
      showAlert('通话启动失败', 'error')
      return false
    }
  }

  // 停止通话
  const stopCall = async () => {
    try {
      stopCallTimer()

      // 停止WebRTC连接
      if (peerConnection && peerConnection.connectionState !== 'closed') {
        peerConnection.close()
        setPeerConnection(null)
      }

      // 关闭WebSocket连接
      if (socket && socket.readyState !== WebSocket.CLOSED) {
        socket.close()
        setSocket(null)
      }

      // 停止本地音频流
      if (localStream) {
        localStream.getTracks().forEach(track => track.stop())
        setLocalStream(null)
      }

      showAlert('通话已结束', 'success')
      return true
    } catch (err: any) {
      console.error('终止通话失败:', err)
      showAlert('终止通话失败', 'error')
      return false
    }
  }

  return {
    socket,
    peerConnection,
    localStream,
    callDuration,
    startCall,
    stopCall,
    setPeerConnection,
  }
}


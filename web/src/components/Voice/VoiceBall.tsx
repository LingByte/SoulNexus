import React, { useRef, useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import { Mic, PhoneOff } from 'lucide-react'
import { cn } from '@/utils/cn'

interface VoiceBallProps {
  isCalling: boolean
  onToggleCall: () => void
  className?: string
}

const VoiceBall: React.FC<VoiceBallProps> = ({
  isCalling,
  onToggleCall,
  className = ''
}) => {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const audioContextRef = useRef<AudioContext | null>(null)
  const analyserRef = useRef<AnalyserNode | null>(null)
  const animationRef = useRef<number | null>(null)
  const [audioStream, setAudioStream] = useState<MediaStream | null>(null)

  // 初始化音频分析器
  useEffect(() => {
    if (isCalling) {
      initAudioContext()
    } else {
      stopAudioContext()
    }
  }, [isCalling])

  useEffect(() => {
    return () => {
      stopAudioContext()
    }
  }, [])

  const initAudioContext = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      setAudioStream(stream)

      const audioContext = new (window.AudioContext || (window as any).webkitAudioContext)()
      const analyser = audioContext.createAnalyser()
      analyser.fftSize = 256

      const source = audioContext.createMediaStreamSource(stream)
      source.connect(analyser)

      audioContextRef.current = audioContext
      analyserRef.current = analyser

      drawWaveform()
    } catch (err) {
      console.error('麦克风访问失败:', err)
    }
  }

  const stopAudioContext = () => {
    // 停止音频流
    if (audioStream) {
      audioStream.getTracks().forEach(track => {
        track.stop()
        track.enabled = false
      })
      setAudioStream(null)
    }

    // 关闭音频上下文
    if (audioContextRef.current) {
      if (audioContextRef.current.state !== 'closed') {
        audioContextRef.current.close()
          .then(() => {
            console.log('AudioContext成功关闭')
          })
          .catch(err => {
            console.error('关闭AudioContext失败:', err)
          })
      }
      audioContextRef.current = null
    }

    // 清理分析器引用
    analyserRef.current = null

    // 停止动画帧
    if (animationRef.current) {
      cancelAnimationFrame(animationRef.current)
      animationRef.current = null
    }
  }

  const drawWaveform = () => {
    if (!canvasRef.current || !analyserRef.current) return

    const canvas = canvasRef.current
    const ctx = canvas.getContext('2d')
    const analyser = analyserRef.current
    const bufferLength = analyser.frequencyBinCount
    const dataArray = new Uint8Array(bufferLength)

    const draw = () => {
      animationRef.current = requestAnimationFrame(draw)
      analyser.getByteFrequencyData(dataArray)

      // 清空画布
      ctx!.clearRect(0, 0, canvas.width, canvas.height)

      // 创建渐变背景
      const gradient = ctx!.createLinearGradient(0, 0, canvas.width, canvas.height)
      gradient.addColorStop(0, 'rgba(99, 102, 241, 0.1)')
      gradient.addColorStop(1, 'rgba(168, 85, 247, 0.1)')
      ctx!.fillStyle = gradient
      ctx!.fillRect(0, 0, canvas.width, canvas.height)

      // 绘制频率条
      const barWidth = (canvas.width / bufferLength) * 2.5
      let x = 0

      for (let i = 0; i < bufferLength; i++) {
        const barHeight = (dataArray[i] / 255) * canvas.height

        // 创建条形渐变
        const barGradient = ctx!.createLinearGradient(0, canvas.height, 0, canvas.height - barHeight)
        barGradient.addColorStop(0, '#6366f1')
        barGradient.addColorStop(1, '#a855f7')

        ctx!.fillStyle = barGradient
        ctx!.fillRect(
          x,
          canvas.height - barHeight,
          barWidth,
          barHeight
        )

        // 添加光晕效果
        ctx!.shadowColor = 'rgba(168, 85, 247, 0.5)'
        ctx!.shadowBlur = 10
        ctx!.shadowOffsetX = 0
        ctx!.shadowOffsetY = 0

        x += barWidth + 2
      }

      // 绘制动态粒子
      ctx!.shadowBlur = 15
      dataArray.forEach((value, i) => {
        if (i % 4 === 0 && value > 128) {
          const size = (value / 255) * 8
          ctx!.beginPath()
          ctx!.arc(
            Math.random() * canvas.width,
            canvas.height - (value / 255) * canvas.height,
            size,
            0,
            Math.PI * 2
          )
          ctx!.fillStyle = `hsla(${280 + (value / 255) * 40}, 80%, 70%, ${0.5 + (value / 255) * 0.5})`
          ctx!.fill()
        }
      })

      // 重置阴影
      ctx!.shadowBlur = 0
    }

    draw()
  }

  return (
    <div className={cn('relative group', className)}>
      <div className="absolute inset-0 bg-purple-300/40 blur-xl rounded-full animate-pulse"></div>
      <div className="relative flex items-center justify-center h-32 w-32 mx-auto
        bg-gradient-to-br from-purple-400 to-purple-600 rounded-full shadow-xl">
        <canvas
          ref={canvasRef}
          className="absolute w-full h-full rounded-full"
          width="128"
          height="128"
        />
        <motion.button
          onClick={onToggleCall}
          className={cn(
            'p-4 rounded-full transition-all shadow-lg z-10 relative overflow-hidden',
            isCalling
              ? 'bg-purple-500 hover:bg-purple-600 shadow-purple-400/50'
              : 'bg-gradient-to-br from-purple-300 to-purple-500 hover:from-purple-400 hover:to-purple-600'
          )}
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
        >
          <div className="absolute inset-0 bg-gradient-to-br from-white/30 to-transparent rounded-full" />
          {isCalling ? (
            <PhoneOff className="w-8 h-8 animate-pulse" />
          ) : (
            <Mic className="w-8 h-8 animate-bounce" />
          )}
        </motion.button>
      </div>
    </div>
  )
}

export default VoiceBall

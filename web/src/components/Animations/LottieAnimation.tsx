import { useRef, useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import lottie, { AnimationItem } from 'lottie-web'

interface LottieAnimationProps {
  animationData?: any
  path?: string
  loop?: boolean | number
  autoplay?: boolean
  speed?: number
  direction?: 1 | -1
  className?: string
  width?: number | string
  height?: number | string
  onComplete?: () => void
  onLoopComplete?: () => void
  onEnterFrame?: (frame: number) => void
  onSegmentStart?: () => void
  onDataReady?: () => void
  onDataFailed?: () => void
  onLoadedImages?: () => void
  onDOMLoaded?: () => void
  onDestroy?: () => void
  onError?: (error: any) => void
  renderer?: 'svg' | 'canvas' | 'html'
  rendererSettings?: {
    preserveAspectRatio?: string
    progressiveLoad?: boolean
    hideOnTransparent?: boolean
    className?: string
    id?: string
  }
  segments?: boolean | [number, number]
  initialSegment?: [number, number]
  quality?: 'high' | 'medium' | 'low'
  background?: string
  style?: React.CSSProperties
}

const LottieAnimation = ({
  animationData,
  path,
  loop = true,
  autoplay = true,
  speed = 1,
  direction = 1,
  className = '',
  width = '100%',
  height = '100%',
  onComplete,
  onLoopComplete,
  onEnterFrame,
  onSegmentStart,
  onDataReady,
  onDataFailed,
  onLoadedImages,
  onDOMLoaded,
  onDestroy,
  onError,
  renderer = 'svg',
  rendererSettings = {},
  segments = false,
  initialSegment,
  quality = 'high',
  background,
  style = {}
}: LottieAnimationProps) => {
  const containerRef = useRef<HTMLDivElement>(null)
  const animationRef = useRef<AnimationItem | null>(null)
  const [isLoaded, setIsLoaded] = useState(false)
  const [isPlaying, setIsPlaying] = useState(false)
  const [currentFrame, setCurrentFrame] = useState(0)
  const [totalFrames, setTotalFrames] = useState(0)
  const [duration, setDuration] = useState(0)

  // 初始化动画
  useEffect(() => {
    if (!containerRef.current) return

    const config = {
      container: containerRef.current,
      renderer,
      loop: typeof loop === 'number' ? loop : loop,
      autoplay,
      animationData,
      path,
      rendererSettings: {
        preserveAspectRatio: 'xMidYMid meet',
        progressiveLoad: true,
        hideOnTransparent: true,
        ...rendererSettings
      },
      segments: segments,
      initialSegment: initialSegment
    }

    try {
      animationRef.current = lottie.loadAnimation(config)

      // 设置事件监听器
      if (animationRef.current) {
        animationRef.current.addEventListener('complete', () => {
          setIsPlaying(false)
          onComplete?.()
        })

        animationRef.current.addEventListener('loopComplete', () => {
          onLoopComplete?.()
        })

        animationRef.current.addEventListener('enterFrame', () => {
          if (animationRef.current) {
            const frame = animationRef.current.currentFrame
            setCurrentFrame(frame)
            onEnterFrame?.(frame)
          }
        })

        animationRef.current.addEventListener('segmentStart', () => {
          onSegmentStart?.()
        })

        animationRef.current.addEventListener('data_ready', () => {
          setIsLoaded(true)
          if (animationRef.current) {
            setTotalFrames(animationRef.current.totalFrames)
            setDuration(animationRef.current.timeCompleted)
          }
          onDataReady?.()
        })

        animationRef.current.addEventListener('data_failed', () => {
          onDataFailed?.()
        })

        animationRef.current.addEventListener('loaded_images', () => {
          onLoadedImages?.()
        })

        animationRef.current.addEventListener('DOMLoaded', () => {
          onDOMLoaded?.()
        })

        animationRef.current.addEventListener('destroy', () => {
          onDestroy?.()
        })

        // 设置速度
        animationRef.current.setSpeed(speed)
        
        // 设置方向
        animationRef.current.setDirection(direction)

        // 设置质量 (lottie-web 5.x 版本不支持 setQuality 方法)
        // 质量通过 rendererSettings 控制

        setIsPlaying(autoplay)
      }
    } catch (error) {
      console.error('Lottie animation error:', error)
      onError?.(error)
    }

    return () => {
      if (animationRef.current) {
        animationRef.current.destroy()
        animationRef.current = null
      }
    }
  }, [animationData, path, loop, autoplay, speed, direction, renderer, rendererSettings, segments, initialSegment, quality, onComplete, onLoopComplete, onEnterFrame, onSegmentStart, onDataReady, onDataFailed, onLoadedImages, onDOMLoaded, onDestroy, onError])

  // 播放控制方法
  const play = () => {
    if (animationRef.current) {
      animationRef.current.play()
      setIsPlaying(true)
    }
  }

  const pause = () => {
    if (animationRef.current) {
      animationRef.current.pause()
      setIsPlaying(false)
    }
  }

  const stop = () => {
    if (animationRef.current) {
      animationRef.current.stop()
      setIsPlaying(false)
    }
  }

  const goToAndStop = (value: number, isFrame = true) => {
    if (animationRef.current) {
      animationRef.current.goToAndStop(value, isFrame)
    }
  }

  const goToAndPlay = (value: number, isFrame = true) => {
    if (animationRef.current) {
      animationRef.current.goToAndPlay(value, isFrame)
      setIsPlaying(true)
    }
  }

  const setSpeed = (newSpeed: number) => {
    if (animationRef.current) {
      animationRef.current.setSpeed(newSpeed)
    }
  }

  const setDirection = (newDirection: 1 | -1) => {
    if (animationRef.current) {
      animationRef.current.setDirection(newDirection)
    }
  }

  const playSegments = (segments: [number, number] | [number, number][], forceFlag = false) => {
    if (animationRef.current) {
      animationRef.current.playSegments(segments, forceFlag)
      setIsPlaying(true)
    }
  }

  const resize = () => {
    if (animationRef.current) {
      animationRef.current.resize()
    }
  }

  // 暴露控制方法
  useEffect(() => {
    if (containerRef.current) {
      (containerRef.current as any).lottieControls = {
        play,
        pause,
        stop,
        goToAndStop,
        goToAndPlay,
        setSpeed,
        setDirection,
        playSegments,
        resize,
        isPlaying: () => isPlaying,
        currentFrame: () => currentFrame,
        totalFrames: () => totalFrames,
        duration: () => duration
      }
    }
  }, [isPlaying, currentFrame, totalFrames, duration])

  return (
    <motion.div
      className={`relative ${className}`}
      style={{
        width,
        height,
        background,
        ...style
      }}
      initial={{ opacity: 0 }}
      animate={{ opacity: isLoaded ? 1 : 0 }}
      transition={{ duration: 0.3 }}
    >
      <div
        ref={containerRef}
        className="w-full h-full"
        style={{
          width: '100%',
          height: '100%'
        }}
      />
      
      {!isLoaded && (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-100 dark:bg-gray-800">
          <div className="text-gray-500 dark:text-gray-400">加载动画中...</div>
        </div>
      )}
    </motion.div>
  )
}

export default LottieAnimation

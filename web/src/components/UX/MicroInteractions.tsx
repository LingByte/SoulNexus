import { useState, useEffect, useRef } from 'react'
import { motion, useAnimation, useInView } from 'framer-motion'
import { cn } from '@/utils/cn.ts'
// 打字机效果组件
interface TypewriterProps {
    text: string
    speed?: number
    delay?: number
    className?: string
    onComplete?: () => void
}

export const Typewriter = ({
                               text,
                               speed = 50,
                               delay = 0,
                               className = "",
                               onComplete
                           }: TypewriterProps) => {
    const [displayText, setDisplayText] = useState('')
    const [currentIndex, setCurrentIndex] = useState(0)
    const [isComplete, setIsComplete] = useState(false)

    // 确保text是字符串类型
    const safeText = text || ''

    // 当text改变时重置打字机状态
    useEffect(() => {
        setDisplayText('')
        setCurrentIndex(0)
        setIsComplete(false)
    }, [safeText])

    useEffect(() => {
        if (currentIndex < safeText.length) {
            const timeout = setTimeout(() => {
                setDisplayText(prev => prev + safeText[currentIndex])
                setCurrentIndex(prev => prev + 1)
            }, currentIndex === 0 ? delay + speed : speed)

            return () => clearTimeout(timeout)
        } else if (onComplete && !isComplete) {
            onComplete()
            setIsComplete(true)
        }
    }, [currentIndex, safeText, speed, delay, onComplete, isComplete])

    return (
        <span className={cn('inline-block text-inherit', className)}>
      {currentIndex < safeText.length ? (
          <>
              {displayText}
              <motion.span
                  animate={{ opacity: [1, 0, 1] }}
                  transition={{ duration: 0.8, repeat: Infinity }}
                  className="ml-1 text-inherit"
              >
                  |
              </motion.span>
          </>
      ) : (
          <>
              {displayText}
              <motion.span
                  animate={{ opacity: [1, 0, 1] }}
                  transition={{ duration: 0.8, repeat: Infinity }}
                  className="ml-1 text-inherit"
              >
                  |
              </motion.span>
          </>
      )}
    </span>
    )
}

// 悬浮卡片效果
interface HoverCardProps {
  children: React.ReactNode
  className?: string
  intensity?: 'low' | 'medium' | 'high'
}

export const HoverCard = ({ 
  children, 
  className = "",
  intensity = 'medium' 
}: HoverCardProps) => {
  const [isHovered, setIsHovered] = useState(false)
  const cardRef = useRef<HTMLDivElement>(null)

  const intensityMap = {
    low: { scale: 1.02, shadow: '0 10px 25px -3px rgba(0, 0, 0, 0.1)' },
    medium: { scale: 1.05, shadow: '0 20px 40px -4px rgba(0, 0, 0, 0.15)' },
    high: { scale: 1.08, shadow: '0 25px 50px -5px rgba(0, 0, 0, 0.2)' }
  }

  const { scale, shadow } = intensityMap[intensity]

  return (
    <motion.div
      ref={cardRef}
      className={cn('transition-all duration-300', className)}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
      animate={{
        scale: isHovered ? scale : 1,
        boxShadow: isHovered ? shadow : '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
        y: isHovered ? -8 : 0
      }}
      transition={{ duration: 0.3, ease: "easeOut" }}
    >
      {children}
    </motion.div>
  )
}

// 加载骨架屏
interface SkeletonProps {
  className?: string
  lines?: number
  animated?: boolean
}

export const Skeleton = ({ 
  className = "",
  lines = 1,
  animated = true 
}: SkeletonProps) => {
  return (
    <div className={cn('space-y-2', className)}>
      {Array.from({ length: lines }).map((_, index) => (
        <div
          key={index}
          className={cn(
            'h-4 bg-gray-200 rounded',
            animated && 'animate-pulse'
          )}
          style={{
            width: `${Math.random() * 40 + 60}%`
          }}
        />
      ))}
    </div>
  )
}

// 进度指示器
interface ProgressIndicatorProps {
  current: number
  total: number
  className?: string
  showPercentage?: boolean
  animated?: boolean
}

export const ProgressIndicator = ({
  current,
  total,
  className = "",
  showPercentage = true,
  animated = true
}: ProgressIndicatorProps) => {
  const percentage = Math.round((current / total) * 100)

  return (
    <div className={cn('w-full', className)}>
      <div className="flex justify-between items-center mb-2">
        <span className="text-sm font-medium">
          进度
        </span>
        {showPercentage && (
          <span className="text-sm text-gray-500">
            {percentage}%
          </span>
        )}
      </div>
      <div className="w-full bg-gray-200 rounded-full h-2">
        <motion.div
          className="bg-gradient-to-r from-blue-500 to-purple-600 h-2 rounded-full"
          initial={{ width: 0 }}
          animate={{ width: `${percentage}%` }}
          transition={{ duration: animated ? 0.8 : 0, ease: "easeOut" }}
        />
      </div>
    </div>
  )
}

// 脉冲效果
interface PulseProps {
  children: React.ReactNode
  className?: string
  intensity?: 'low' | 'medium' | 'high'
  duration?: number
}

export const Pulse = ({ 
  children, 
  className = "",
  intensity = 'medium',
  duration = 2
}: PulseProps) => {
  const intensityMap = {
    low: 1.05,
    medium: 1.1,
    high: 1.15
  }

  return (
    <motion.div
      className={className}
      animate={{
        scale: [1, intensityMap[intensity], 1]
      }}
      transition={{
        duration,
        repeat: Infinity,
        ease: "easeInOut"
      }}
    >
      {children}
    </motion.div>
  )
}

// 滚动触发动画
interface ScrollRevealProps {
  children: React.ReactNode
  className?: string
  direction?: 'up' | 'down' | 'left' | 'right'
  delay?: number
  duration?: number
}

export const ScrollReveal = ({
  children,
  className = "",
  direction = 'up',
  delay = 0,
  duration = 0.6
}: ScrollRevealProps) => {
  const ref = useRef(null)
  const isInView = useInView(ref, { once: true, margin: "-100px" })
  const controls = useAnimation()

  const directionMap = {
    up: { y: 50, opacity: 0 },
    down: { y: -50, opacity: 0 },
    left: { x: 50, opacity: 0 },
    right: { x: -50, opacity: 0 }
  }

  useEffect(() => {
    if (isInView) {
      controls.start({
        x: 0,
        y: 0,
        opacity: 1,
        transition: {
          duration,
          delay,
          ease: "easeOut"
        }
      })
    }
  }, [isInView, controls, duration, delay])

  return (
    <motion.div
      ref={ref}
      className={className}
      initial={directionMap[direction]}
      animate={controls}
    >
      {children}
    </motion.div>
  )
}

// 磁吸效果
interface MagneticProps {
  children: React.ReactNode
  className?: string
  strength?: number
}

export const Magnetic = ({ 
  children, 
  className = "",
  strength = 0.3
}: MagneticProps) => {
  const ref = useRef<HTMLDivElement>(null)

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!ref.current) return

    const rect = ref.current.getBoundingClientRect()
    const x = e.clientX - rect.left - rect.width / 2
    const y = e.clientY - rect.top - rect.height / 2

    ref.current.style.transform = `translate(${x * strength}px, ${y * strength}px)`
  }

  const handleMouseLeave = () => {
    if (!ref.current) return
    ref.current.style.transform = 'translate(0px, 0px)'
  }

  return (
    <div
      ref={ref}
      className={cn('transition-transform duration-200 ease-out', className)}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
    >
      {children}
    </div>
  )
}

// 涟漪效果
interface RippleProps {
  children: React.ReactNode
  className?: string
  color?: string
  duration?: number
}

export const Ripple = ({ 
  children, 
  className = "",
  color = 'rgba(255, 255, 255, 0.6)',
  duration = 600
}: RippleProps) => {
  const [ripples, setRipples] = useState<Array<{ id: number; x: number; y: number }>>([])

  const handleClick = (e: React.MouseEvent) => {
    const rect = e.currentTarget.getBoundingClientRect()
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top

    const newRipple = {
      id: Date.now(),
      x,
      y
    }

    setRipples(prev => [...prev, newRipple])

    setTimeout(() => {
      setRipples(prev => prev.filter(ripple => ripple.id !== newRipple.id))
    }, duration)
  }

  return (
    <div
      className={cn('relative overflow-hidden', className)}
      onClick={handleClick}
    >
      {children}
      {ripples.map(ripple => (
        <motion.div
          key={ripple.id}
          className="absolute rounded-full pointer-events-none"
          style={{
            left: ripple.x,
            top: ripple.y,
            backgroundColor: color
          }}
          initial={{ scale: 0, opacity: 1 }}
          animate={{ scale: 4, opacity: 0 }}
          transition={{ duration: duration / 1000, ease: "easeOut" }}
        />
      ))}
    </div>
  )
}

// 粒子背景
interface ParticleBackgroundProps {
  className?: string
  particleCount?: number
  color?: string
  speed?: number
}

export const ParticleBackground = ({
  className = "",
  particleCount = 50,
  color = '#3b82f6',
  speed = 1
}: ParticleBackgroundProps) => {
  const [particles, setParticles] = useState<Array<{
    id: number
    x: number
    y: number
    vx: number
    vy: number
    size: number
  }>>([])

  useEffect(() => {
    const newParticles = Array.from({ length: particleCount }, (_, i) => ({
      id: i,
      x: Math.random() * window.innerWidth,
      y: Math.random() * window.innerHeight,
      vx: (Math.random() - 0.5) * speed,
      vy: (Math.random() - 0.5) * speed,
      size: Math.random() * 3 + 1
    }))
    setParticles(newParticles)
  }, [particleCount, speed])

  useEffect(() => {
    const animate = () => {
      setParticles(prev => prev.map(particle => {
        const newX = particle.x + particle.vx
        const newY = particle.y + particle.vy
        
        return {
          ...particle,
          x: newX > window.innerWidth ? 0 : newX < 0 ? window.innerWidth : newX,
          y: newY > window.innerHeight ? 0 : newY < 0 ? window.innerHeight : newY
        }
      }))
    }

    const interval = setInterval(animate, 16)
    return () => clearInterval(interval)
  }, [])

  return (
    <div className={cn('absolute inset-0 overflow-hidden', className)}>
      {particles.map(particle => (
        <motion.div
          key={particle.id}
          className="absolute rounded-full opacity-20"
          style={{
            left: particle.x,
            top: particle.y,
            width: particle.size,
            height: particle.size,
            backgroundColor: color
          }}
          animate={{
            scale: [1, 1.2, 1],
            opacity: [0.2, 0.4, 0.2]
          }}
          transition={{
            duration: 3,
            repeat: Infinity,
            ease: "easeInOut"
          }}
        />
      ))}
    </div>
  )
}

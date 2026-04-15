import React, { useState, useEffect, useRef } from 'react'
import { cn } from '@/utils/cn.ts'

interface EpicRatingEffectProps {
  score: number
  maxScore?: number
  size?: 'sm' | 'md' | 'lg' | 'xl'
  showAnimation?: boolean
  showParticles?: boolean
  showSound?: boolean
  className?: string
  onScoreChange?: (score: number) => void
}

interface Particle {
  id: number
  x: number
  y: number
  vx: number
  vy: number
  life: number
  maxLife: number
  color: string
  size: number
}

const EpicRatingEffect: React.FC<EpicRatingEffectProps> = ({
  score,
  maxScore = 100,
  size = 'md',
  showAnimation = true,
  showParticles = true,
  showSound = true,
  className
                                                           }) => {
  const [displayScore, setDisplayScore] = useState(0)
  const [isAnimating, setIsAnimating] = useState(false)
  const [particles, setParticles] = useState<Particle[]>([])
  const [showScore, setShowScore] = useState(false)
  const animationRef = useRef<number>()
  const particleIdRef = useRef(0)

  const sizeClasses = {
    sm: 'w-16 h-16 text-lg',
    md: 'w-24 h-24 text-2xl',
    lg: 'w-32 h-32 text-4xl',
    xl: 'w-40 h-40 text-6xl'
  }

  const getScoreColor = (score: number) => {
    const percentage = (score / maxScore) * 100
    if (percentage >= 90) return 'text-yellow-400'
    if (percentage >= 80) return 'text-green-400'
    if (percentage >= 70) return 'text-blue-400'
    if (percentage >= 60) return 'text-orange-400'
    return 'text-red-400'
  }

  const getScoreGlow = (score: number) => {
    const percentage = (score / maxScore) * 100
    if (percentage >= 90) return 'shadow-yellow-400/50'
    if (percentage >= 80) return 'shadow-green-400/50'
    if (percentage >= 70) return 'shadow-blue-400/50'
    if (percentage >= 60) return 'shadow-orange-400/50'
    return 'shadow-red-400/50'
  }

  const createParticle = (x: number, y: number, color: string): Particle => {
    const angle = Math.random() * Math.PI * 2
    const speed = 2 + Math.random() * 4
    return {
      id: particleIdRef.current++,
      x,
      y,
      vx: Math.cos(angle) * speed,
      vy: Math.sin(angle) * speed,
      life: 1,
      maxLife: 1,
      color,
      size: 2 + Math.random() * 4
    }
  }

  const createExplosion = (x: number, y: number) => {
    if (!showParticles) return
    
    const colors = ['#fbbf24', '#10b981', '#3b82f6', '#f59e0b', '#ef4444']
    const newParticles: Particle[] = []
    
    for (let i = 0; i < 20; i++) {
      newParticles.push(createParticle(x, y, colors[Math.floor(Math.random() * colors.length)]))
    }
    
    setParticles(prev => [...prev, ...newParticles])
  }

  const playSound = () => {
    if (!showSound) return
    
    // 创建音效
    const audioContext = new (window.AudioContext || (window as any).webkitAudioContext)()
    const oscillator = audioContext.createOscillator()
    const gainNode = audioContext.createGain()
    
    oscillator.connect(gainNode)
    gainNode.connect(audioContext.destination)
    
    const frequency = 440 + (score / maxScore) * 440 // 根据分数调整频率
    oscillator.frequency.setValueAtTime(frequency, audioContext.currentTime)
    oscillator.type = 'sine'
    
    gainNode.gain.setValueAtTime(0, audioContext.currentTime)
    gainNode.gain.linearRampToValueAtTime(0.1, audioContext.currentTime + 0.01)
    gainNode.gain.exponentialRampToValueAtTime(0.001, audioContext.currentTime + 0.5)
    
    oscillator.start(audioContext.currentTime)
    oscillator.stop(audioContext.currentTime + 0.5)
  }

  const animateScore = () => {
    if (!showAnimation) {
      setDisplayScore(score)
      return
    }

    setIsAnimating(true)
    setShowScore(true)
    
    const startTime = Date.now()
    const duration = 2000
    const startScore = displayScore
    
    const animate = () => {
      const elapsed = Date.now() - startTime
      const progress = Math.min(elapsed / duration, 1)
      
      // 使用缓动函数
      const easeOut = 1 - Math.pow(1 - progress, 3)
      const currentScore = startScore + (score - startScore) * easeOut
      
      setDisplayScore(Math.round(currentScore))
      
      if (progress < 1) {
        animationRef.current = requestAnimationFrame(animate)
      } else {
        setIsAnimating(false)
        createExplosion(50, 50) // 动画结束时创建爆炸效果
        playSound()
      }
    }
    
    animate()
  }

  const animateParticles = () => {
    setParticles(prev => 
      prev
        .map(particle => ({
          ...particle,
          x: particle.x + particle.vx,
          y: particle.y + particle.vy,
          life: particle.life - 0.02,
          vy: particle.vy + 0.1 // 重力效果
        }))
        .filter(particle => particle.life > 0)
    )
  }

  useEffect(() => {
    animateScore()
  }, [score])

  useEffect(() => {
    if (particles.length > 0) {
      const interval = setInterval(animateParticles, 16)
      return () => clearInterval(interval)
    }
  }, [particles])

  useEffect(() => {
    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current)
      }
    }
  }, [])

  const renderParticles = () => {
    if (!showParticles || particles.length === 0) return null

    return (
      <div className="absolute inset-0 pointer-events-none">
        {particles.map(particle => (
          <div
            key={particle.id}
            className="absolute rounded-full"
            style={{
              left: `${particle.x}%`,
              top: `${particle.y}%`,
              width: `${particle.size}px`,
              height: `${particle.size}px`,
              backgroundColor: particle.color,
              opacity: particle.life,
              transform: 'translate(-50%, -50%)',
              boxShadow: `0 0 ${particle.size * 2}px ${particle.color}`
            }}
          />
        ))}
      </div>
    )
  }

  return (
    <div className={cn('relative', className)}>
      {/* 主评分显示 */}
      <div className={cn(
        'relative flex items-center justify-center rounded-full border-4 border-white/20 bg-gradient-to-br from-slate-900 to-slate-800 backdrop-blur-sm',
        sizeClasses[size],
        getScoreColor(score),
        isAnimating && 'animate-pulse',
        `shadow-2xl ${getScoreGlow(score)}`
      )}>
        {/* 背景光晕 */}
        <div className={cn(
          'absolute inset-0 rounded-full opacity-30 blur-xl',
          getScoreColor(score).replace('text-', 'bg-')
        )} />
        
        {/* 分数显示 */}
        <div className="relative z-10 font-bold">
          {showScore ? displayScore : score}
        </div>
        
        {/* 分数条 */}
        <div className="absolute inset-0 rounded-full overflow-hidden">
          <div 
            className={cn(
              'h-full transition-all duration-1000 ease-out',
              getScoreColor(score).replace('text-', 'bg-')
            )}
            style={{ 
              width: `${(displayScore / maxScore) * 100}%`,
              opacity: 0.2
            }}
          />
        </div>
        
        {/* 旋转边框 */}
        <div className={cn(
          'absolute inset-0 rounded-full border-2 border-transparent',
          'bg-gradient-to-r from-yellow-400 via-green-400 via-blue-400 to-purple-400',
          'animate-spin',
          isAnimating ? 'opacity-100' : 'opacity-0'
        )} style={{ animationDuration: '2s' }} />
      </div>
      
      {/* 粒子效果 */}
      {renderParticles()}
      
      {/* 分数等级指示器 */}
      <div className="absolute -bottom-8 left-1/2 transform -translate-x-1/2">
        <div className={cn(
          'px-3 py-1 rounded-full text-xs font-semibold text-white',
          getScoreColor(score).replace('text-', 'bg-')
        )}>
          {displayScore >= maxScore * 0.9 ? 'S+' :
           displayScore >= maxScore * 0.8 ? 'S' :
           displayScore >= maxScore * 0.7 ? 'A' :
           displayScore >= maxScore * 0.6 ? 'B' : 'C'}
        </div>
      </div>
      
      {/* 冲击波效果 */}
      {isAnimating && (
        <div className="absolute inset-0 rounded-full border-2 border-white/50 animate-ping" />
      )}
    </div>
  )
}

export default EpicRatingEffect

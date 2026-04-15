import React, { useEffect, useRef, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

interface GlobalEffectsProps {
  children: React.ReactNode
  enableParticles?: boolean
  enableBackground?: boolean
  enableTransitions?: boolean
  enableSound?: boolean
  theme?: 'default' | 'neon' | 'cyber' | 'fire' | 'ice' | 'rainbow'
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
  rotation: number
  rotationSpeed: number
}

const GlobalEffects: React.FC<GlobalEffectsProps> = ({
  children,
  enableParticles = true,
  enableBackground = true,
  enableTransitions = true,
  enableSound = true,
  theme = 'default'
}) => {
  const [particles, setParticles] = useState<Particle[]>([])
  const [isLoaded, setIsLoaded] = useState(false)
  const animationRef = useRef<number>()
  const particleIdRef = useRef(0)

  const themes = {
    default: {
      colors: ['#3b82f6', '#1d4ed8', '#1e40af'],
      background: 'from-slate-900 via-blue-900 to-slate-900',
      particles: ['#3b82f6', '#1d4ed8', '#60a5fa']
    },
    neon: {
      colors: ['#00ff88', '#00d4aa', '#00a8cc'],
      background: 'from-black via-green-900 to-black',
      particles: ['#00ff88', '#00d4aa', '#00a8cc', '#00ff41']
    },
    cyber: {
      colors: ['#ff0080', '#ff4081', '#e91e63'],
      background: 'from-black via-pink-900 to-black',
      particles: ['#ff0080', '#ff4081', '#e91e63', '#f50057']
    },
    fire: {
      colors: ['#ff6b35', '#f7931e', '#ffd23f'],
      background: 'from-black via-orange-900 to-black',
      particles: ['#ff6b35', '#f7931e', '#ffd23f', '#ff4500']
    },
    ice: {
      colors: ['#00bcd4', '#26c6da', '#4dd0e1'],
      background: 'from-black via-cyan-900 to-black',
      particles: ['#00bcd4', '#26c6da', '#4dd0e1', '#00e5ff']
    },
    rainbow: {
      colors: ['#ff0000', '#ff8000', '#ffff00', '#00ff00', '#0080ff', '#8000ff'],
      background: 'from-red-900 via-purple-900 to-blue-900',
      particles: ['#ff0000', '#ff8000', '#ffff00', '#00ff00', '#0080ff', '#8000ff']
    }
  }

  const currentTheme = themes[theme] || themes.default

  const createParticle = (): Particle => {
    const angle = Math.random() * Math.PI * 2
    const speed = 0.5 + Math.random() * 2
    const color = currentTheme.particles[Math.floor(Math.random() * currentTheme.particles.length)]
    
    return {
      id: particleIdRef.current++,
      x: Math.random() * window.innerWidth,
      y: Math.random() * window.innerHeight,
      vx: Math.cos(angle) * speed,
      vy: Math.sin(angle) * speed,
      life: 1,
      maxLife: 200 + Math.random() * 300,
      color,
      size: 1 + Math.random() * 3,
      rotation: Math.random() * Math.PI * 2,
      rotationSpeed: (Math.random() - 0.5) * 0.1
    }
  }

  const animateParticles = () => {
    setParticles(prev => 
      prev
        .map(particle => ({
          ...particle,
          x: particle.x + particle.vx,
          y: particle.y + particle.vy,
          life: particle.life - 1,
          rotation: particle.rotation + particle.rotationSpeed,
          vx: particle.vx * 0.999, // 空气阻力
          vy: particle.vy * 0.999
        }))
        .filter(particle => particle.life > 0)
        .concat(Array.from({ length: 2 }, createParticle)) // 持续添加新粒子
    )
  }

  const playAmbientSound = () => {
    if (!enableSound) return
    
    try {
      const audioContext = new (window.AudioContext || (window as any).webkitAudioContext)()
      const oscillator = audioContext.createOscillator()
      const gainNode = audioContext.createGain()
      const filter = audioContext.createBiquadFilter()
      
      oscillator.connect(filter)
      filter.connect(gainNode)
      gainNode.connect(audioContext.destination)
      
      oscillator.frequency.setValueAtTime(60, audioContext.currentTime)
      oscillator.type = 'sine'
      
      filter.type = 'lowpass'
      filter.frequency.setValueAtTime(200, audioContext.currentTime)
      
      gainNode.gain.setValueAtTime(0, audioContext.currentTime)
      gainNode.gain.linearRampToValueAtTime(0.01, audioContext.currentTime + 1)
      
      oscillator.start(audioContext.currentTime)
      oscillator.stop(audioContext.currentTime + 10)
    } catch (error) {
      console.log('Audio not supported')
    }
  }

  useEffect(() => {
    setIsLoaded(true)
    
    if (enableParticles) {
      // 初始化粒子
      const initialParticles = Array.from({ length: 50 }, createParticle)
      setParticles(initialParticles)
    }

    if (enableSound) {
      playAmbientSound()
    }
  }, [enableParticles, enableSound])

  useEffect(() => {
    if (enableParticles) {
      // 开始动画循环
      const animate = () => {
        animateParticles()
        animationRef.current = requestAnimationFrame(animate)
      }
      animate()

      return () => {
        if (animationRef.current) {
          cancelAnimationFrame(animationRef.current)
        }
      }
    }
  }, [enableParticles])

  const renderParticles = () => {
    if (!enableParticles || particles.length === 0) return null

    return (
      <div className="fixed inset-0 pointer-events-none overflow-hidden z-0">
        {particles.map(particle => (
          <motion.div
            key={particle.id}
            className="absolute rounded-full"
            style={{
              left: particle.x,
              top: particle.y,
              width: particle.size,
              height: particle.size,
              backgroundColor: particle.color,
              opacity: particle.life / particle.maxLife,
              boxShadow: `0 0 ${particle.size * 2}px ${particle.color}`,
              filter: 'blur(0.5px)'
            }}
            animate={{
              rotate: particle.rotation,
              scale: [1, 1.2, 1],
            }}
            transition={{
              duration: 2,
              repeat: Infinity,
              ease: "easeInOut"
            }}
          />
        ))}
      </div>
    )
  }

  const renderBackground = () => {
    if (!enableBackground) return null

    return (
      <div className={`fixed inset-0 bg-gradient-to-br ${currentTheme.background} z-0`}>
        {/* 动态背景网格 */}
        <div className="absolute inset-0 opacity-20">
          <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/10 to-transparent animate-pulse" />
          <div className="absolute inset-0 bg-gradient-to-b from-transparent via-white/5 to-transparent animate-pulse" style={{ animationDelay: '1s' }} />
        </div>
        
        {/* 主题特效 */}
        {theme === 'rainbow' && (
          <div 
            className="absolute inset-0 opacity-30"
            style={{
              background: `conic-gradient(from 0deg, ${currentTheme.colors.join(', ')})`,
              animation: 'spin 20s linear infinite'
            }}
          />
        )}
        
        {/* 光晕效果 */}
        <div className="absolute top-1/4 left-1/4 w-96 h-96 rounded-full bg-blue-500/20 blur-3xl animate-pulse" />
        <div className="absolute bottom-1/4 right-1/4 w-96 h-96 rounded-full bg-purple-500/20 blur-3xl animate-pulse" style={{ animationDelay: '2s' }} />
      </div>
    )
  }

  return (
    <div className="relative min-h-screen">
      {/* 背景特效 */}
      {renderBackground()}
      
      {/* 粒子特效 */}
      {renderParticles()}
      
      {/* 内容区域 */}
      <AnimatePresence>
        {isLoaded && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.8, ease: "easeOut" }}
            className="relative z-10"
          >
            {children}
          </motion.div>
        )}
      </AnimatePresence>
      
      {/* 页面过渡特效 */}
      {enableTransitions && (
        <motion.div
          className="fixed inset-0 bg-gradient-to-r from-blue-600 via-purple-600 to-pink-600 z-50 pointer-events-none"
          initial={{ scaleX: 0 }}
          animate={{ scaleX: 1 }}
          transition={{ duration: 0.5, ease: "easeInOut" }}
          style={{ transformOrigin: "left" }}
        />
      )}
    </div>
  )
}

export default GlobalEffects

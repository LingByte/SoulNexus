import React, { useEffect, useRef, useState } from 'react'
import { motion } from 'framer-motion'

interface BackgroundEffectsProps {
  theme?: 'default' | 'neon' | 'cyber' | 'fire' | 'ice' | 'rainbow' | 'matrix' | 'stars'
  intensity?: 'low' | 'medium' | 'high'
  enableParticles?: boolean
  enableWaves?: boolean
  enableGrid?: boolean
  enableGlow?: boolean
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
  opacity: number
}

interface Wave {
  id: number
  x: number
  y: number
  radius: number
  maxRadius: number
  life: number
  maxLife: number
  color: string
  opacity: number
}

const BackgroundEffects: React.FC<BackgroundEffectsProps> = ({
  theme = 'default',
  intensity = 'medium',
  enableParticles = true,
  enableWaves = true,
  enableGrid = true,
  enableGlow = true
}) => {
  const [particles, setParticles] = useState<Particle[]>([])
  const [waves, setWaves] = useState<Wave[]>([])
  const animationRef = useRef<number>()
  const particleIdRef = useRef(0)
  const waveIdRef = useRef(0)

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
    },
    matrix: {
      colors: ['#00ff00', '#00cc00', '#009900'],
      background: 'from-black to-green-900',
      particles: ['#00ff00', '#00cc00', '#009900']
    },
    stars: {
      colors: ['#ffffff', '#f0f0f0', '#e0e0e0'],
      background: 'from-black via-purple-900 to-black',
      particles: ['#ffffff', '#f0f0f0', '#e0e0e0', '#c0c0c0']
    }
  }

  const intensitySettings = {
    low: { particleCount: 20, waveCount: 2, particleSpeed: 0.5 },
    medium: { particleCount: 50, waveCount: 5, particleSpeed: 1 },
    high: { particleCount: 100, waveCount: 10, particleSpeed: 2 }
  }

  const currentTheme = themes[theme]
  const settings = intensitySettings[intensity]

  const createParticle = (): Particle => {
    const angle = Math.random() * Math.PI * 2
    const speed = settings.particleSpeed * (0.5 + Math.random())
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
      opacity: 0.3 + Math.random() * 0.7
    }
  }

  const createWave = (): Wave => {
    const color = currentTheme.colors[Math.floor(Math.random() * currentTheme.colors.length)]
    return {
      id: waveIdRef.current++,
      x: Math.random() * window.innerWidth,
      y: Math.random() * window.innerHeight,
      radius: 0,
      maxRadius: 100 + Math.random() * 200,
      life: 1,
      maxLife: 100 + Math.random() * 200,
      color,
      opacity: 0.1 + Math.random() * 0.3
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
          vx: particle.vx * 0.999,
          vy: particle.vy * 0.999
        }))
        .filter(particle => particle.life > 0)
        .concat(Array.from({ length: 2 }, createParticle))
    )
  }

  const animateWaves = () => {
    setWaves(prev => 
      prev
        .map(wave => ({
          ...wave,
          radius: wave.radius + 1,
          life: wave.life - 1
        }))
        .filter(wave => wave.life > 0 && wave.radius < wave.maxRadius)
        .concat(Array.from({ length: 1 }, createWave))
    )
  }

  useEffect(() => {
    if (enableParticles) {
      const initialParticles = Array.from({ length: settings.particleCount }, createParticle)
      setParticles(initialParticles)
    }

    if (enableWaves) {
      const initialWaves = Array.from({ length: settings.waveCount }, createWave)
      setWaves(initialWaves)
    }
  }, [enableParticles, enableWaves, settings.particleCount, settings.waveCount])

  useEffect(() => {
    const animate = () => {
      animateParticles()
      animateWaves()
      animationRef.current = requestAnimationFrame(animate)
    }
    animate()

    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current)
      }
    }
  }, [enableParticles, enableWaves])

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
              opacity: particle.opacity * (particle.life / particle.maxLife),
              boxShadow: `0 0 ${particle.size * 2}px ${particle.color}`,
              filter: 'blur(0.5px)'
            }}
            animate={{
              scale: [1, 1.2, 1],
              opacity: [particle.opacity * 0.5, particle.opacity, particle.opacity * 0.5]
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

  const renderWaves = () => {
    if (!enableWaves || waves.length === 0) return null

    return (
      <div className="fixed inset-0 pointer-events-none z-0">
        {waves.map(wave => (
          <motion.div
            key={wave.id}
            className="absolute rounded-full border-2"
            style={{
              left: wave.x,
              top: wave.y,
              width: wave.radius,
              height: wave.radius,
              borderColor: wave.color,
              opacity: wave.opacity * (wave.life / wave.maxLife),
              boxShadow: `0 0 ${wave.radius / 4}px ${wave.color}`,
              transform: 'translate(-50%, -50%)'
            }}
            animate={{
              scale: [1, 1.1, 1],
              opacity: [wave.opacity * 0.5, wave.opacity, wave.opacity * 0.5]
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

  const renderGrid = () => {
    if (!enableGrid) return null

    return (
      <div className="fixed inset-0 pointer-events-none z-0">
        <div 
          className="w-full h-full opacity-20"
          style={{
            backgroundImage: `
              linear-gradient(${currentTheme.colors[0]}20 1px, transparent 1px),
              linear-gradient(90deg, ${currentTheme.colors[0]}20 1px, transparent 1px)
            `,
            backgroundSize: '50px 50px'
          }}
        />
      </div>
    )
  }

  const renderGlow = () => {
    if (!enableGlow) return null

    return (
      <div className="fixed inset-0 pointer-events-none z-0">
        {/* 中心光晕 */}
        <div 
          className="absolute top-1/4 left-1/4 w-96 h-96 rounded-full blur-3xl animate-pulse"
          style={{ backgroundColor: currentTheme.colors[0] }}
        />
        <div 
          className="absolute bottom-1/4 right-1/4 w-96 h-96 rounded-full blur-3xl animate-pulse"
          style={{ 
            backgroundColor: currentTheme.colors[1] || currentTheme.colors[0],
            animationDelay: '2s'
          }}
        />
        <div 
          className="absolute top-1/2 left-1/2 w-64 h-64 rounded-full blur-2xl animate-pulse"
          style={{ 
            backgroundColor: currentTheme.colors[2] || currentTheme.colors[0],
            animationDelay: '4s'
          }}
        />
      </div>
    )
  }

  return (
    <div className={`fixed inset-0 bg-gradient-to-br ${currentTheme.background} z-0`}>
      {/* 基础背景 */}
      <div className="absolute inset-0" />
      
      {/* 网格效果 */}
      {renderGrid()}
      
      {/* 粒子效果 */}
      {renderParticles()}
      
      {/* 波浪效果 */}
      {renderWaves()}
      
      {/* 光晕效果 */}
      {renderGlow()}
      
      {/* 主题特效 */}
      {theme === 'rainbow' && (
        <div 
          className="absolute inset-0 opacity-20"
          style={{
            background: `conic-gradient(from 0deg, ${currentTheme.colors.join(', ')})`,
            animation: 'spin 20s linear infinite'
          }}
        />
      )}
      
      {theme === 'matrix' && (
        <div className="absolute inset-0 opacity-10">
          <div className="absolute inset-0 bg-gradient-to-b from-transparent via-green-500/20 to-transparent animate-pulse" />
          <div className="absolute inset-0 bg-gradient-to-r from-transparent via-green-500/10 to-transparent animate-pulse" style={{ animationDelay: '1s' }} />
        </div>
      )}
      
      {theme === 'stars' && (
        <div className="absolute inset-0">
          {Array.from({ length: 100 }, (_, i) => (
            <motion.div
              key={i}
              className="absolute w-1 h-1 bg-white rounded-full"
              style={{
                left: `${Math.random() * 100}%`,
                top: `${Math.random() * 100}%`,
              }}
              animate={{
                opacity: [0, 1, 0],
                scale: [0, 1, 0]
              }}
              transition={{
                duration: 2 + Math.random() * 3,
                repeat: Infinity,
                delay: Math.random() * 2
              }}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export default BackgroundEffects

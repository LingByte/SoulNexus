import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { ReactNode } from 'react'

interface GlitchEffectProps {
  children: ReactNode
  className?: string
  intensity?: 'low' | 'medium' | 'high'
  duration?: number
  trigger?: 'hover' | 'click' | 'always' | 'once'
  disabled?: boolean
}

export const GlitchEffect = ({
  children,
  className = '',
  intensity = 'medium',
  duration = 0.5,
  trigger = 'hover',
  disabled = false
}: GlitchEffectProps) => {
  const [isGlitching, setIsGlitching] = useState(false)
  const [glitchKey, setGlitchKey] = useState(0)

  const intensityMap = {
    low: { offset: 2, blur: 0.5, hue: 10 },
    medium: { offset: 4, blur: 1, hue: 20 },
    high: { offset: 8, blur: 2, hue: 40 }
  }

  const { offset, blur, hue } = intensityMap[intensity]

  const triggerGlitch = () => {
    if (disabled) return
    
    setIsGlitching(true)
    setGlitchKey(prev => prev + 1)
    
    setTimeout(() => {
      setIsGlitching(false)
    }, duration * 1000)
  }

  useEffect(() => {
    if (trigger === 'always') {
      setIsGlitching(true)
    } else if (trigger === 'once') {
      triggerGlitch()
    }
  }, [trigger])

  const handleInteraction = () => {
    if (trigger === 'hover' || trigger === 'click') {
      triggerGlitch()
    }
  }

  const glitchVariants = {
    normal: {
      x: 0,
      y: 0,
      filter: 'hue-rotate(0deg) saturate(1) blur(0px)',
      opacity: 1
    },
    glitch: {
      x: [0, -offset, offset, -offset, offset, 0],
      y: [0, offset, -offset, offset, -offset, 0],
      filter: [
        'hue-rotate(0deg) saturate(1) blur(0px)',
        `hue-rotate(${hue}deg) saturate(2) blur(${blur}px)`,
        `hue-rotate(-${hue}deg) saturate(0.5) blur(${blur}px)`,
        `hue-rotate(${hue}deg) saturate(1.5) blur(${blur}px)`,
        'hue-rotate(0deg) saturate(1) blur(0px)'
      ],
      opacity: [1, 0.8, 0.6, 0.8, 1]
    }
  }

  return (
    <motion.div
      className={`relative inline-block ${className}`}
      onMouseEnter={trigger === 'hover' ? handleInteraction : undefined}
      onClick={trigger === 'click' ? handleInteraction : undefined}
      style={{ cursor: trigger === 'click' ? 'pointer' : 'default' }}
    >
      <AnimatePresence mode="wait">
        <motion.div
          key={glitchKey}
          variants={glitchVariants}
          initial="normal"
          animate={isGlitching ? 'glitch' : 'normal'}
          transition={{
            duration: isGlitching ? duration : 0.3,
            ease: isGlitching ? 'easeInOut' : 'easeOut'
          }}
        >
          {children}
        </motion.div>
      </AnimatePresence>

      {/* 故障线条效果 */}
      {isGlitching && (
        <motion.div
          className="absolute inset-0 pointer-events-none"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
        >
          {[...Array(3)].map((_, i) => (
            <motion.div
              key={i}
              className="absolute w-full h-0.5 bg-red-500"
              style={{
                top: `${20 + i * 30}%`,
                left: 0
              }}
              initial={{ x: '-100%', opacity: 0 }}
              animate={{ 
                x: ['100%', '-100%'],
                opacity: [0, 1, 0]
              }}
              transition={{
                duration: 0.1,
                delay: i * 0.05,
                repeat: 3,
                repeatDelay: 0.1
              }}
            />
          ))}
        </motion.div>
      )}

      {/* 数字雨效果 */}
      {isGlitching && (
        <motion.div
          className="absolute inset-0 pointer-events-none overflow-hidden"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
        >
          {[...Array(20)].map((_, i) => (
            <motion.div
              key={i}
              className="absolute text-green-400 font-mono text-xs"
              style={{
                left: `${Math.random() * 100}%`,
                top: '-10px'
              }}
              initial={{ y: -10, opacity: 0 }}
              animate={{ 
                y: '100vh',
                opacity: [0, 1, 0]
              }}
              transition={{
                duration: 2,
                delay: Math.random() * 2,
                ease: 'linear'
              }}
            >
              {Math.random().toString(36).substring(2, 8)}
            </motion.div>
          ))}
        </motion.div>
      )}
    </motion.div>
  )
}

export default GlitchEffect

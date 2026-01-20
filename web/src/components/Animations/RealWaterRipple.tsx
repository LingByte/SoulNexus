import { useState, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

interface RealWaterRippleProps {
  children: React.ReactNode
  className?: string
  intensity?: 'low' | 'medium' | 'high'
  color?: string
  duration?: number
  maxRipples?: number
}

interface Ripple {
  id: number
  x: number
  y: number
  size: number
  opacity: number
  life: number
  maxLife: number
}

export const RealWaterRipple = ({
  children,
  className = '',
  intensity = 'medium',
  color = 'rgba(59, 130, 246, 0.4)',
  duration = 1000,
  maxRipples = 3
}: RealWaterRippleProps) => {
  const [ripples, setRipples] = useState<Ripple[]>([])
  const containerRef = useRef<HTMLDivElement>(null)
  const rippleId = useRef(0)

  const intensityMap = {
    low: { maxSize: 80, speed: 0.8, count: 1 },
    medium: { maxSize: 120, speed: 1, count: 2 },
    high: { maxSize: 160, speed: 1.2, count: 3 }
  }

  const createRipple = (event: React.MouseEvent) => {
    if (!containerRef.current) return

    const rect = containerRef.current.getBoundingClientRect()
    const x = event.clientX - rect.left
    const y = event.clientY - rect.top

    const {speed, count } = intensityMap[intensity]

    // 限制涟漪数量
    if (ripples.length >= maxRipples) {
      setRipples(prev => prev.slice(1))
    }

    for (let i = 0; i < count; i++) {
      const newRipple: Ripple = {
        id: rippleId.current++,
        x: x + (Math.random() - 0.5) * 10,
        y: y + (Math.random() - 0.5) * 10,
        size: 0,
        opacity: 0.8,
        life: 0,
        maxLife: duration * speed
      }

      setRipples(prev => [...prev, newRipple])
    }
  }

  return (
    <div
      ref={containerRef}
      className={`relative overflow-hidden ${className}`}
      onMouseDown={createRipple}
    >
      {children}
      
      <AnimatePresence>
        {ripples.map(ripple => (
          <motion.div
            key={ripple.id}
            className="absolute pointer-events-none rounded-full"
            style={{
              left: ripple.x,
              top: ripple.y,
              width: ripple.size,
              height: ripple.size,
              background: `radial-gradient(circle, ${color} 0%, transparent 70%)`,
              boxShadow: `0 0 20px ${color}, inset 0 0 20px ${color}`,
              transform: 'translate(-50%, -50%)'
            }}
            initial={{ 
              scale: 0, 
              opacity: 0.8,
              width: 0,
              height: 0
            }}
            animate={{ 
              scale: 1,
              opacity: 0,
              width: 200,
              height: 200
            }}
            exit={{ 
              opacity: 0,
              scale: 1.2
            }}
            transition={{
              duration: duration / 1000,
              ease: [0.25, 0.46, 0.45, 0.94],
              opacity: {
                duration: duration / 1000,
                ease: 'easeOut'
              }
            }}
          />
        ))}
      </AnimatePresence>
    </div>
  )
}

export default RealWaterRipple

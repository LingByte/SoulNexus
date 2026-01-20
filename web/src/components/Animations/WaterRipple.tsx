import { useState, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

interface WaterRippleProps {
  children: React.ReactNode
  className?: string
  intensity?: 'low' | 'medium' | 'high'
  color?: string
  duration?: number
}

interface Ripple {
  id: number
  x: number
  y: number
  size: number
}

export const WaterRipple = ({
  children,
  className = '',
  intensity = 'medium',
  color = 'rgba(59, 130, 246, 0.3)',
  duration = 600
}: WaterRippleProps) => {
  const [ripples, setRipples] = useState<Ripple[]>([])
  const containerRef = useRef<HTMLDivElement>(null)
  const rippleId = useRef(0)

  const intensityMap = {
    low: { maxSize: 100, count: 1 },
    medium: { maxSize: 150, count: 2 },
    high: { maxSize: 200, count: 3 }
  }

  const createRipple = (event: React.MouseEvent) => {
    if (!containerRef.current) return

    const rect = containerRef.current.getBoundingClientRect()
    const x = event.clientX - rect.left
    const y = event.clientY - rect.top

    const { maxSize, count } = intensityMap[intensity]

    for (let i = 0; i < count; i++) {
      const newRipple: Ripple = {
        id: rippleId.current++,
        x: x + (Math.random() - 0.5) * 20,
        y: y + (Math.random() - 0.5) * 20,
        size: maxSize + Math.random() * 50
      }

      setRipples(prev => [...prev, newRipple])

      // 移除涟漪
      setTimeout(() => {
        setRipples(prev => prev.filter(ripple => ripple.id !== newRipple.id))
      }, duration)
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
              left: ripple.x - ripple.size / 2,
              top: ripple.y - ripple.size / 2,
              width: ripple.size,
              height: ripple.size,
              background: color,
              boxShadow: `0 0 20px ${color}`
            }}
            initial={{ scale: 0, opacity: 0.8 }}
            animate={{ scale: 1, opacity: 0 }}
            exit={{ opacity: 0 }}
            transition={{
              duration: duration / 1000,
              ease: 'easeOut'
            }}
          />
        ))}
      </AnimatePresence>
    </div>
  )
}

export default WaterRipple
